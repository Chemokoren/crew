// Package service – TransactionService provides atomic, idempotent financial
// operations that span multiple repositories (float + wallet).
// All operations run inside a single DB transaction so that either both
// sides commit or neither does.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// TransactionService handles composite financial operations that must be atomic.
type TransactionService struct {
	txMgr     *database.TxManager
	floatRepo repository.OrganizationFloatRepository
	walletSvc *WalletService
	auditSvc  *AuditService
	logger    *slog.Logger
}

// NewTransactionService creates a new TransactionService.
func NewTransactionService(
	txMgr *database.TxManager,
	floatRepo repository.OrganizationFloatRepository,
	walletSvc *WalletService,
	auditSvc *AuditService,
	logger *slog.Logger,
) *TransactionService {
	return &TransactionService{
		txMgr:     txMgr,
		floatRepo: floatRepo,
		walletSvc: walletSvc,
		auditSvc:  auditSvc,
		logger:    logger,
	}
}

// ──────────────────────────────────────────────
// Employee Payout (float → wallet, atomic)
// ──────────────────────────────────────────────

// EmployeePayoutInput holds the parameters for an atomic employee payout.
type EmployeePayoutInput struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	CrewMemberID   uuid.UUID `json:"crew_member_id" binding:"required"`
	GrossCents     int64     `json:"gross_cents" binding:"required,min=1"`
	NetCents       int64     `json:"net_cents" binding:"required,min=1"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required"`
	Description    string    `json:"description"`
}

// EmployeePayoutResult is the response from an atomic employee payout.
type EmployeePayoutResult struct {
	FloatTx  *models.OrganizationFloatTransaction `json:"float_transaction"`
	WalletTx *models.WalletTransaction            `json:"wallet_transaction"`
}

// EmployeePayout atomically debits the org float (gross amount) and credits
// the employee wallet (net amount). If either operation fails, the entire
// transaction is rolled back.
//
// Idempotency: the same idempotency_key will return the same result without
// re-executing. Two derived keys are used internally:
//   - "{key}" for the float debit
//   - "{key}:wallet" for the wallet credit
func (s *TransactionService) EmployeePayout(ctx context.Context, input EmployeePayoutInput) (*EmployeePayoutResult, error) {
	if input.GrossCents <= 0 || input.NetCents <= 0 {
		return nil, fmt.Errorf("%w: gross and net amounts must be positive", ErrValidation)
	}
	if input.NetCents > input.GrossCents {
		return nil, fmt.Errorf("%w: net amount cannot exceed gross amount", ErrValidation)
	}

	var result EmployeePayoutResult

	err := s.txMgr.RunInTx(ctx, func(txCtx context.Context) error {
		// 1. Get org float record
		sf, err := s.floatRepo.GetOrCreate(txCtx, input.OrganizationID)
		if err != nil {
			return fmt.Errorf("get org float: %w", err)
		}

		// 2. Debit org float by GROSS amount
		floatTx, err := s.floatRepo.DebitFloat(
			txCtx, sf.ID, sf.Version, input.GrossCents,
			input.IdempotencyKey,
			fmt.Sprintf("Employee payout | %s", input.Description),
		)
		if err != nil {
			return fmt.Errorf("debit org float: %w", err)
		}
		result.FloatTx = floatTx

		// 3. Credit employee wallet by NET amount
		walletTx, err := s.walletSvc.Credit(txCtx, CreditInput{
			CrewMemberID:   input.CrewMemberID,
			AmountCents:    input.NetCents,
			Category:       models.TxCatEarning,
			IdempotencyKey: input.IdempotencyKey + ":wallet",
			Description:    input.Description,
		})
		if err != nil {
			return fmt.Errorf("credit employee wallet: %w", err)
		}
		result.WalletTx = walletTx

		return nil
	})

	if err != nil {
		return nil, err
	}

	s.logger.Info("employee payout completed",
		slog.String("org_id", input.OrganizationID.String()),
		slog.String("crew_member_id", input.CrewMemberID.String()),
		slog.Int64("gross_cents", input.GrossCents),
		slog.Int64("net_cents", input.NetCents),
		slog.String("idempotency_key", input.IdempotencyKey),
	)

	return &result, nil
}

// ──────────────────────────────────────────────
// Wallet-to-Wallet Transfer (atomic)
// ──────────────────────────────────────────────

// WalletTransferInput holds the parameters for an atomic wallet-to-wallet transfer.
type WalletTransferInput struct {
	FromCrewMemberID uuid.UUID `json:"from_crew_member_id" binding:"required"`
	ToCrewMemberID   uuid.UUID `json:"to_crew_member_id" binding:"required"`
	AmountCents      int64     `json:"amount_cents" binding:"required,min=1"`
	IdempotencyKey   string    `json:"idempotency_key" binding:"required"`
	Description      string    `json:"description"`
}

// WalletTransferResult is the response from an atomic wallet transfer.
type WalletTransferResult struct {
	DebitTx  *models.WalletTransaction `json:"debit_transaction"`
	CreditTx *models.WalletTransaction `json:"credit_transaction"`
}

// WalletTransfer atomically debits the sender and credits the recipient.
// If either side fails, the entire transfer is rolled back.
func (s *TransactionService) WalletTransfer(ctx context.Context, input WalletTransferInput) (*WalletTransferResult, error) {
	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: transfer amount must be positive", ErrValidation)
	}
	if input.FromCrewMemberID == input.ToCrewMemberID {
		return nil, fmt.Errorf("%w: cannot transfer to yourself", ErrValidation)
	}

	var result WalletTransferResult

	err := s.txMgr.RunInTx(ctx, func(txCtx context.Context) error {
		// 1. Debit sender
		debitTx, err := s.walletSvc.Debit(txCtx, DebitInput{
			CrewMemberID:   input.FromCrewMemberID,
			AmountCents:    input.AmountCents,
			Category:       models.TxCatWithdrawal,
			IdempotencyKey: input.IdempotencyKey + ":debit",
			Description:    fmt.Sprintf("Transfer out: %s", input.Description),
		})
		if err != nil {
			return fmt.Errorf("debit sender: %w", err)
		}
		result.DebitTx = debitTx

		// 2. Credit recipient
		creditTx, err := s.walletSvc.Credit(txCtx, CreditInput{
			CrewMemberID:   input.ToCrewMemberID,
			AmountCents:    input.AmountCents,
			Category:       models.TxCatEarning,
			IdempotencyKey: input.IdempotencyKey + ":credit",
			Description:    fmt.Sprintf("Transfer in: %s", input.Description),
		})
		if err != nil {
			return fmt.Errorf("credit recipient: %w", err)
		}
		result.CreditTx = creditTx

		return nil
	})

	if err != nil {
		return nil, err
	}

	s.logger.Info("wallet transfer completed",
		slog.String("from", input.FromCrewMemberID.String()),
		slog.String("to", input.ToCrewMemberID.String()),
		slog.Int64("amount_cents", input.AmountCents),
		slog.String("idempotency_key", input.IdempotencyKey),
	)

	return &result, nil
}
