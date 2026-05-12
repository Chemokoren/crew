// Package service handles webhook callbacks from JamboPay and PerPay.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

type WebhookService struct {
	webhookRepo repository.WebhookEventRepository
	payoutSvc   *PayoutService
	payrollSvc  *PayrollService
	orgSvc      *OrganizationService // For float top-up confirmation
	walletRepo  repository.WalletRepository // For JamboPay reversals
	payrollRepo repository.PayrollRepository
	logger      *slog.Logger
}

func NewWebhookService(
	webhookRepo repository.WebhookEventRepository,
	payoutSvc *PayoutService,
	payrollSvc *PayrollService,
	orgSvc *OrganizationService,
	walletRepo repository.WalletRepository,
	payrollRepo repository.PayrollRepository,
	logger *slog.Logger,
) *WebhookService {
	return &WebhookService{
		webhookRepo: webhookRepo,
		payoutSvc:   payoutSvc,
		payrollSvc:  payrollSvc,
		orgSvc:      orgSvc,
		walletRepo:  walletRepo,
		payrollRepo: payrollRepo,
		logger:      logger,
	}
}

// jamboPayCallback represents the official JamboPay v2 callback payload.
// Spec: POST to callbackUrl with:
//   {"status":"string","amount":"string","ref":"string","orderId":"string","description":"string","checksum":"string"}
// Checksum: SHA256(ref + amount + client_id + client_secret)
type jamboPayCallback struct {
	Status      string `json:"status"`
	Amount      string `json:"amount"`
	Ref         string `json:"ref"`       // Provider transaction reference
	OrderID     string `json:"orderId"`   // Our idempotency key / order ID
	Description string `json:"description"`
	Checksum    string `json:"checksum"`  // SHA256(ref+amount+client_id+client_secret)
}

// ProcessJamboPayWebhook processes a JamboPay v2 callback.
// Checksum verification is done at the handler level (WebhookHandler) using
// JamboPayProvider.VerifyCallbackChecksum before this is called.
func (s *WebhookService) ProcessJamboPayWebhook(ctx context.Context, payload []byte) error {
	var cb jamboPayCallback
	if err := json.Unmarshal(payload, &cb); err != nil {
		return fmt.Errorf("invalid jambopay callback json: %w", err)
	}

	// Use ref as external reference, fall back to orderId
	ref := cb.Ref
	if ref == "" {
		ref = cb.OrderID
	}

	// Persist the raw webhook event (idempotency-safe)
	event := &models.WebhookEvent{
		Source:      models.WebhookJamboPay,
		EventType:   "PAYOUT_STATUS_UPDATE",
		ExternalRef: ref,
		Payload:     payload,
	}
	if err := s.webhookRepo.Create(ctx, event); err != nil {
		return fmt.Errorf("failed to save webhook event: %w", err)
	}

	if ref == "" || cb.Status == "" {
		s.logger.Warn("JamboPay callback missing ref or status",
			slog.String("raw_ref", cb.Ref),
			slog.String("order_id", cb.OrderID),
		)
		return nil
	}

	s.logger.Info("processing JamboPay callback",
		slog.String("ref", ref),
		slog.String("status", cb.Status),
		slog.String("amount", cb.Amount),
	)

	// ——— 1. Check if this is a collection/float top-up callback ———
	// OrderID maps to the idempotency key we used when creating the pending tx
	if s.orgSvc != nil && cb.OrderID != "" {
		pendingTx, floatErr := s.orgSvc.GetFloatTxByIdempotencyKey(ctx, cb.OrderID)
		if floatErr == nil && pendingTx.Status == models.TxPending {
			switch cb.Status {
			case "SUCCESS", "COMPLETED", "COMPLETE":
				_, confirmErr := s.orgSvc.ConfirmPendingTopUp(ctx, pendingTx.ID, models.SyncCallback)
				if confirmErr != nil {
					s.logger.Error("CRITICAL: failed to confirm float top-up",
						slog.String("tx_id", pendingTx.ID.String()),
						slog.String("error", confirmErr.Error()),
					)
				} else {
					s.logger.Info("float top-up confirmed via callback",
						slog.String("tx_id", pendingTx.ID.String()),
						slog.String("amount", cb.Amount),
						slog.String("ref", ref),
					)
				}
			case "FAILED", "FAILURE", "REVERSED", "ERROR":
				_ = s.orgSvc.FailPendingTopUp(ctx, pendingTx.ID, "payment "+cb.Status+": "+cb.Description, models.SyncCallback)
				s.logger.Info("float top-up failed via callback",
					slog.String("tx_id", pendingTx.ID.String()),
					slog.String("status", cb.Status),
				)
			default:
				s.logger.Info("intermediate float top-up callback status — no action",
					slog.String("status", cb.Status),
				)
			}
			return s.webhookRepo.MarkProcessed(ctx, event.ID)
		}
	}

	// ——— 2. Check if this is a payout callback ———
	// Locate the internal wallet transaction via idempotency key
	// Convention: we use "payout-{orderID}" as the idempotency key when debiting
	tx, err := s.walletRepo.GetByIdempotencyKey(ctx, "payout-"+cb.OrderID)
	if err != nil {
		// Not found is non-fatal — could be a transfer callback, not a payout
		s.logger.Warn("wallet transaction not found for JamboPay callback",
			slog.String("ref", ref),
			slog.String("order_id", cb.OrderID),
			slog.String("err", err.Error()),
		)
		return s.webhookRepo.MarkProcessed(ctx, event.ID)
	}

	// Skip if already in a terminal state (idempotency guard)
	if tx.Status == models.TxCompleted || tx.Status == models.TxFailed || tx.Status == models.TxReversed {
		s.logger.Info("payout already in terminal state — skipping duplicate callback",
			slog.String("tx_id", tx.ID.String()),
			slog.String("status", string(tx.Status)),
		)
		return s.webhookRepo.MarkProcessed(ctx, event.ID)
	}

	switch cb.Status {
	case "SUCCESS", "COMPLETED", "COMPLETE":
		tx.Status = models.TxCompleted
		if err := s.walletRepo.UpdateTransaction(ctx, tx); err != nil {
			return fmt.Errorf("mark tx completed: %w", err)
		}
		s.logger.Info("payout completed successfully",
			slog.String("tx_id", tx.ID.String()),
			slog.String("ref", ref),
		)

	case "FAILED", "FAILURE", "REVERSED", "ERROR":
		tx.Status = models.TxFailed
		if err := s.walletRepo.UpdateTransaction(ctx, tx); err != nil {
			return fmt.Errorf("mark tx failed: %w", err)
		}
		s.logger.Info("payout failed — initiating reversal", slog.String("tx_id", tx.ID.String()))

		// Credit back the debited amount (automatic reversal)
		wallet, err := s.walletRepo.GetWalletByID(ctx, tx.WalletID)
		if err != nil {
			return fmt.Errorf("fetch wallet for reversal: %w", err)
		}
		_, reverseErr := s.payoutSvc.walletSvc.Credit(ctx, CreditInput{
			CrewMemberID:   wallet.CrewMemberID,
			AmountCents:    tx.AmountCents,
			Category:       models.TxCatReversal,
			IdempotencyKey: "rev-" + tx.IdempotencyKey,
			Description:    "Reversal for failed JamboPay payout " + ref,
		})
		if reverseErr != nil {
			s.logger.Error("CRITICAL: payout reversal failed — manual reconciliation required",
				slog.String("tx_id", tx.ID.String()),
				slog.Int64("amount_cents", tx.AmountCents),
				slog.String("reversal_error", reverseErr.Error()),
			)
		} else {
			s.logger.Info("payout debit reversed successfully",
				slog.String("tx_id", tx.ID.String()),
			)
		}

	default:
		s.logger.Info("received intermediate JamboPay status — no action",
			slog.String("status", cb.Status),
			slog.String("ref", ref),
		)
	}

	return s.webhookRepo.MarkProcessed(ctx, event.ID)
}

// ProcessPerpayWebhook processes a callback from Perpay payroll provider.
func (s *WebhookService) ProcessPerpayWebhook(ctx context.Context, payload []byte) error {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid perpay json: %w", err)
	}

	ref, _ := data["correlation_id"].(string)
	status, _ := data["status"].(string)

	event := &models.WebhookEvent{
		Source:      models.WebhookPerpay,
		EventType:   "PAYROLL_STATUS_UPDATE",
		ExternalRef: ref,
		Payload:     payload,
	}
	if err := s.webhookRepo.Create(ctx, event); err != nil {
		return fmt.Errorf("failed to save perpay event: %w", err)
	}

	s.logger.Info("processing Perpay webhook", slog.String("ref", ref), slog.String("status", status))

	if ref != "" && status == "COMPLETED" {
		// Future: update PayrollRun.Status to COMPLETED by external ref
	}

	return s.webhookRepo.MarkProcessed(ctx, event.ID)
}
