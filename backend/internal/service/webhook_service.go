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
	walletRepo  repository.WalletRepository // For JamboPay reversals if needed
	payrollRepo repository.PayrollRepository
	logger      *slog.Logger
}

func NewWebhookService(
	webhookRepo repository.WebhookEventRepository,
	payoutSvc *PayoutService,
	payrollSvc *PayrollService,
	walletRepo repository.WalletRepository,
	payrollRepo repository.PayrollRepository,
	logger *slog.Logger,
) *WebhookService {
	return &WebhookService{
		webhookRepo: webhookRepo,
		payoutSvc:   payoutSvc,
		payrollSvc:  payrollSvc,
		walletRepo:  walletRepo,
		payrollRepo: payrollRepo,
		logger:      logger,
	}
}

// ProcessJamboPayWebhook processes a callback from JamboPay.
func (s *WebhookService) ProcessJamboPayWebhook(ctx context.Context, payload []byte) error {
	// Parse the generic JSON payload
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid jambopay json: %w", err)
	}

	// Extract standard fields
	// Let's assume JamboPay sends "reference" or "order_id" and "status"
	ref, _ := data["reference"].(string)
	if ref == "" {
		ref, _ = data["order_id"].(string)
	}
	status, _ := data["status"].(string)

	event := &models.WebhookEvent{
		Source:      models.WebhookJamboPay,
		EventType:   "PAYOUT_STATUS_UPDATE",
		ExternalRef: ref,
		Payload:     payload,
	}

	if err := s.webhookRepo.Create(ctx, event); err != nil {
		return fmt.Errorf("failed to save webhook event: %w", err)
	}

	if ref == "" || status == "" {
		s.logger.Warn("JamboPay webhook missing reference or status", slog.Any("data", data))
		return nil // Nothing to act on
	}

	s.logger.Info("processing JamboPay webhook", slog.String("ref", ref), slog.String("status", status))

	// Find the WalletTransaction by IdempotencyKey (which is order_id/ref)
	tx, err := s.walletRepo.GetByIdempotencyKey(ctx, "payout-"+ref)
	if err != nil {
		s.logger.Error("failed to find payout transaction", slog.String("ref", ref), slog.Any("err", err))
		return nil 
	}

	if tx.Status == models.TxCompleted || tx.Status == models.TxFailed || tx.Status == models.TxReversed {
		s.logger.Info("payout already in terminal state", slog.String("tx_id", tx.ID.String()))
		return s.webhookRepo.MarkProcessed(ctx, event.ID)
	}

	switch status {
	case "COMPLETED", "SUCCESS":
		tx.Status = models.TxCompleted
		if err := s.walletRepo.UpdateTransaction(ctx, tx); err != nil {
			return fmt.Errorf("update tx to completed: %w", err)
		}
	case "FAILED", "REVERSED":
		tx.Status = models.TxFailed
		if err := s.walletRepo.UpdateTransaction(ctx, tx); err != nil {
			return fmt.Errorf("update tx to failed: %w", err)
		}
		// REVERSE THE DEBIT!
		s.logger.Info("reversing failed payout", slog.String("tx_id", tx.ID.String()))
		wallet, err := s.walletRepo.GetWalletByID(ctx, tx.WalletID)
		if err != nil {
			return fmt.Errorf("failed to get wallet for reversal: %w", err)
		}
		_, err = s.payoutSvc.walletSvc.Credit(ctx, CreditInput{
			CrewMemberID:   wallet.CrewMemberID,
			AmountCents:    tx.AmountCents,
			Category:       models.TxCatReversal,
			IdempotencyKey: "rev-" + tx.IdempotencyKey,
			Description:    "Reversal for failed payout " + tx.Reference,
		})
		if err != nil {
			return fmt.Errorf("failed to reverse debit: %w", err)
		}
	}

	return s.webhookRepo.MarkProcessed(ctx, event.ID)
}

// ProcessPerpayWebhook processes a callback from Perpay.
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

	// Typically, we check if ALL entries for a run are completed, 
	// but here we can just log or fetch run by ref.
	if ref != "" && status == "COMPLETED" {
		// e.g. update PayrollRun if ref is tied to it.
		// For simplicity, mark processed.
	}

	return s.webhookRepo.MarkProcessed(ctx, event.ID)
}
