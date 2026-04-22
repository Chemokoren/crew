package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/payment"
	"github.com/kibsoft/amy-mis/internal/models"
)

// PayoutService orchestrates the process of moving funds from the internal wallet to external accounts (M-Pesa, Bank, etc.).
type PayoutService struct {
	walletSvc  *WalletService
	payManager *payment.Manager
	auditSvc   *AuditService
	logger     *slog.Logger
}

// NewPayoutService creates a new PayoutService.
func NewPayoutService(walletSvc *WalletService, payManager *payment.Manager, auditSvc *AuditService, logger *slog.Logger) *PayoutService {
	return &PayoutService{
		walletSvc:  walletSvc,
		payManager: payManager,
		auditSvc:   auditSvc,
		logger:     logger,
	}
}

// PayoutInput holds the parameters needed to process a payout.
type PayoutInput struct {
	CrewMemberID   uuid.UUID
	AmountCents    int64
	Channel        payment.PayoutChannel
	RecipientName  string
	RecipientPhone string
	BankAccount    string
	BankCode       string
	PaybillNumber  string
	PaybillRef     string
	IdempotencyKey string
}

// InitiatePayout handles the business logic for withdrawing funds from a wallet.
func (s *PayoutService) InitiatePayout(ctx context.Context, input PayoutInput) (*payment.PayoutResult, error) {
	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: payout amount must be positive", ErrValidation)
	}

	if s.payManager == nil {
		return nil, fmt.Errorf("payout provider is not configured")
	}

	// 1. Debit wallet first
	tx, err := s.walletSvc.Debit(ctx, DebitInput{
		CrewMemberID:   input.CrewMemberID,
		AmountCents:    input.AmountCents,
		Category:       models.TxCatWithdrawal,
		IdempotencyKey: "payout-" + input.IdempotencyKey,
		Description:    fmt.Sprintf("Payout via %s", input.Channel),
	})
	if err != nil {
		return nil, fmt.Errorf("debit for payout failed: %w", err)
	}

	// 2. Initiate payout via Payment Manager
	req := payment.PayoutRequest{
		AmountCents:    input.AmountCents,
		AccountFrom:    "wallet-" + input.CrewMemberID.String(),
		OrderID:        input.IdempotencyKey,
		Channel:        input.Channel,
		RecipientName:  input.RecipientName,
		RecipientPhone: input.RecipientPhone,
		BankAccount:    input.BankAccount,
		BankCode:       input.BankCode,
		PaybillNumber:  input.PaybillNumber,
		PaybillRef:     input.PaybillRef,
		Narration:      "Crew Wallet Payout",
	}

	result, err := s.payManager.InitiatePayout(ctx, req)
	if err != nil {
		s.logger.Error("payout provider failed, initiating automatic reversal",
			slog.String("tx_id", tx.ID.String()),
			slog.String("error", err.Error()),
		)

		// Automatically reverse the debit to prevent lost funds
		_, reverseErr := s.walletSvc.Credit(ctx, CreditInput{
			CrewMemberID:   input.CrewMemberID,
			AmountCents:    input.AmountCents,
			Category:       models.TxCatReversal,
			IdempotencyKey: "rev-payout-" + input.IdempotencyKey,
			Reference:      tx.ID.String(),
			Description:    "Automatic reversal: payout provider failed",
		})
		if reverseErr != nil {
			// Critical: reversal also failed — requires manual intervention
			s.logger.Error("CRITICAL: payout reversal failed, manual reconciliation required",
				slog.String("original_tx_id", tx.ID.String()),
				slog.Int64("amount_cents", input.AmountCents),
				slog.String("crew_member_id", input.CrewMemberID.String()),
				slog.String("reversal_error", reverseErr.Error()),
			)
			s.auditSvc.Log(ctx, input.CrewMemberID, "PAYOUT_REVERSAL_FAILED", "payout", &tx.ID, nil,
				map[string]interface{}{"amount_cents": input.AmountCents, "error": reverseErr.Error()}, "", "")
		} else {
			s.logger.Info("payout debit reversed successfully",
				slog.String("original_tx_id", tx.ID.String()),
				slog.Int64("amount_cents", input.AmountCents),
			)
			s.auditSvc.Log(ctx, input.CrewMemberID, "PAYOUT_REVERSED", "payout", &tx.ID, nil,
				map[string]interface{}{"amount_cents": input.AmountCents, "reason": "provider_failure"}, "", "")
		}

		return nil, fmt.Errorf("payout initiation failed: %w", err)
	}

	s.logger.Info("payout initiated successfully",
		slog.String("tx_id", tx.ID.String()),
		slog.String("provider_ref", result.Reference),
	)

	// Audit trail for payout initiation
	s.auditSvc.Log(ctx, input.CrewMemberID, "PAYOUT_INITIATED", "payout", &tx.ID, nil, result, "", "")

	return result, nil
}
