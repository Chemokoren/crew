package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// WalletService manages crew member wallets and financial transactions.
type WalletService struct {
	walletRepo repository.WalletRepository
	crewRepo   repository.CrewRepository
	auditSvc   *AuditService
	logger     *slog.Logger
}

// NewWalletService creates a new WalletService.
func NewWalletService(
	walletRepo repository.WalletRepository,
	crewRepo repository.CrewRepository,
	auditSvc *AuditService,
	logger *slog.Logger,
) *WalletService {
	return &WalletService{
		walletRepo: walletRepo,
		crewRepo:   crewRepo,
		auditSvc:   auditSvc,
		logger:     logger,
	}
}

// GetOrCreateWallet returns the wallet for a crew member, creating one if needed.
func (s *WalletService) GetOrCreateWallet(ctx context.Context, crewMemberID uuid.UUID) (*models.Wallet, error) {
	wallet, err := s.walletRepo.GetByCrewMemberID(ctx, crewMemberID)
	if err == nil {
		return wallet, nil
	}
	if err != ErrNotFound {
		return nil, fmt.Errorf("get wallet: %w", err)
	}

	// Verify crew member exists
	if _, err := s.crewRepo.GetByID(ctx, crewMemberID); err != nil {
		return nil, fmt.Errorf("crew member not found: %w", err)
	}

	// Create wallet
	wallet = &models.Wallet{
		CrewMemberID: crewMemberID,
		BalanceCents: 0,
		Currency:     "KES",
		Version:      0,
		IsActive:     true,
	}

	if err := s.walletRepo.Create(ctx, wallet); err != nil {
		return nil, fmt.Errorf("create wallet: %w", err)
	}

	s.logger.Info("wallet created",
		slog.String("crew_member_id", crewMemberID.String()),
		slog.String("wallet_id", wallet.ID.String()),
	)

	return wallet, nil
}

// CreditInput holds parameters for crediting a wallet.
type CreditInput struct {
	CrewMemberID   uuid.UUID
	AmountCents    int64
	Category       models.TransactionCategory
	IdempotencyKey string
	Reference      string
	Description    string
}

// Credit adds funds to a crew member's wallet with full idempotency.
func (s *WalletService) Credit(ctx context.Context, input CreditInput) (*models.WalletTransaction, error) {
	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: credit amount must be positive", ErrValidation)
	}

	wallet, err := s.GetOrCreateWallet(ctx, input.CrewMemberID)
	if err != nil {
		return nil, err
	}

	tx, err := s.walletRepo.CreditWallet(
		ctx,
		wallet.ID,
		wallet.Version,
		input.AmountCents,
		input.Category,
		input.IdempotencyKey,
		input.Reference,
		input.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("credit wallet: %w", err)
	}

	s.logger.Info("wallet credited",
		slog.String("wallet_id", wallet.ID.String()),
		slog.Int64("amount_cents", input.AmountCents),
		slog.String("category", string(input.Category)),
		slog.String("idempotency_key", input.IdempotencyKey),
	)

	// Log audit trail
	s.auditSvc.Log(ctx, input.CrewMemberID, "CREDIT", "wallet", &wallet.ID, nil, tx, "", "")

	return tx, nil
}

// DebitInput holds parameters for debiting a wallet.
type DebitInput struct {
	CrewMemberID   uuid.UUID
	AmountCents    int64
	Category       models.TransactionCategory
	IdempotencyKey string
	Reference      string
	Description    string
}

// Debit removes funds from a crew member's wallet. Returns ErrInsufficientBalance if overdraw.
func (s *WalletService) Debit(ctx context.Context, input DebitInput) (*models.WalletTransaction, error) {
	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: debit amount must be positive", ErrValidation)
	}

	wallet, err := s.GetOrCreateWallet(ctx, input.CrewMemberID)
	if err != nil {
		return nil, err
	}

	tx, err := s.walletRepo.DebitWallet(
		ctx,
		wallet.ID,
		wallet.Version,
		input.AmountCents,
		input.Category,
		input.IdempotencyKey,
		input.Reference,
		input.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("debit wallet: %w", err)
	}

	s.logger.Info("wallet debited",
		slog.String("wallet_id", wallet.ID.String()),
		slog.Int64("amount_cents", input.AmountCents),
		slog.String("category", string(input.Category)),
		slog.String("idempotency_key", input.IdempotencyKey),
	)

	// Log audit trail
	s.auditSvc.Log(ctx, input.CrewMemberID, "DEBIT", "wallet", &wallet.ID, nil, tx, "", "")

	return tx, nil
}

// GetBalance returns the current wallet balance for a crew member.
func (s *WalletService) GetBalance(ctx context.Context, crewMemberID uuid.UUID) (*models.Wallet, error) {
	return s.GetOrCreateWallet(ctx, crewMemberID)
}

// GetTransactions returns paginated transaction history for a crew member.
func (s *WalletService) GetTransactions(ctx context.Context, crewMemberID uuid.UUID, filter repository.TxFilter, page, perPage int) ([]models.WalletTransaction, int64, error) {
	wallet, err := s.GetOrCreateWallet(ctx, crewMemberID)
	if err != nil {
		return nil, 0, err
	}
	return s.walletRepo.GetTransactions(ctx, wallet.ID, filter, page, perPage)
}
