package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
)

func newTestWalletService() (*WalletService, *mock.WalletRepo, *mock.CrewRepo) {
	walletRepo := mock.NewWalletRepo()
	crewRepo := mock.NewCrewRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewWalletService(walletRepo, crewRepo, logger), walletRepo, crewRepo
}

func setupCrewMember(t *testing.T, crewRepo *mock.CrewRepo) *models.CrewMember {
	t.Helper()
	crew := &models.CrewMember{
		CrewID:    "CRW-00001",
		FirstName: "John",
		LastName:  "Kamau",
		Role:      models.RoleDriver,
		KYCStatus: models.KYCVerified,
		IsActive:  true,
	}
	if err := crewRepo.Create(context.Background(), crew); err != nil {
		t.Fatalf("create crew member: %v", err)
	}
	return crew
}

func TestCreditWallet(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	tx, err := svc.Credit(ctx, CreditInput{
		CrewMemberID:   crew.ID,
		AmountCents:    150000, // KES 1,500
		Category:       models.TxCatEarning,
		IdempotencyKey: "earn-001",
		Reference:      "shift-2024-01-15",
		Description:    "Daily shift pay",
	})

	if err != nil {
		t.Fatalf("Credit: %v", err)
	}
	if tx.AmountCents != 150000 {
		t.Errorf("AmountCents = %d, want 150000", tx.AmountCents)
	}
	if tx.BalanceAfterCents != 150000 {
		t.Errorf("BalanceAfterCents = %d, want 150000", tx.BalanceAfterCents)
	}
	if tx.Status != models.TxCompleted {
		t.Errorf("Status = %q, want COMPLETED", tx.Status)
	}

	// Verify wallet balance
	wallet, err := svc.GetBalance(ctx, crew.ID)
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if wallet.BalanceCents != 150000 {
		t.Errorf("wallet balance = %d, want 150000", wallet.BalanceCents)
	}
}

func TestDebitWallet(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	// Credit first
	_, _ = svc.Credit(ctx, CreditInput{
		CrewMemberID:   crew.ID,
		AmountCents:    200000,
		Category:       models.TxCatEarning,
		IdempotencyKey: "credit-001",
	})

	// Then debit
	tx, err := svc.Debit(ctx, DebitInput{
		CrewMemberID:   crew.ID,
		AmountCents:    50000, // KES 500
		Category:       models.TxCatWithdrawal,
		IdempotencyKey: "withdraw-001",
		Description:    "M-Pesa withdrawal",
	})

	if err != nil {
		t.Fatalf("Debit: %v", err)
	}
	if tx.BalanceAfterCents != 150000 {
		t.Errorf("BalanceAfterCents = %d, want 150000", tx.BalanceAfterCents)
	}

	// Verify final balance
	wallet, _ := svc.GetBalance(ctx, crew.ID)
	if wallet.BalanceCents != 150000 {
		t.Errorf("wallet balance = %d, want 150000", wallet.BalanceCents)
	}
}

func TestDebitInsufficientBalance(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	// Credit a small amount
	_, _ = svc.Credit(ctx, CreditInput{
		CrewMemberID:   crew.ID,
		AmountCents:    10000,
		Category:       models.TxCatEarning,
		IdempotencyKey: "credit-small",
	})

	// Try to debit more than balance
	_, err := svc.Debit(ctx, DebitInput{
		CrewMemberID:   crew.ID,
		AmountCents:    50000,
		Category:       models.TxCatWithdrawal,
		IdempotencyKey: "overdraw-attempt",
	})

	if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestIdempotencyReplay(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	input := CreditInput{
		CrewMemberID:   crew.ID,
		AmountCents:    100000,
		Category:       models.TxCatEarning,
		IdempotencyKey: "idempotent-credit-001",
	}

	// First call
	tx1, err := svc.Credit(ctx, input)
	if err != nil {
		t.Fatalf("first Credit: %v", err)
	}

	// Second call with same idempotency key — should return same tx, NOT double-credit
	tx2, err := svc.Credit(ctx, input)
	if err != nil {
		t.Fatalf("second Credit: %v", err)
	}

	if tx1.ID != tx2.ID {
		t.Error("idempotent replay should return the same transaction")
	}

	// Balance should only reflect ONE credit
	wallet, _ := svc.GetBalance(ctx, crew.ID)
	if wallet.BalanceCents != 100000 {
		t.Errorf("balance = %d, want 100000 (idempotent replay should NOT double-credit)", wallet.BalanceCents)
	}
}

func TestCreditZeroAmount(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	_, err := svc.Credit(ctx, CreditInput{
		CrewMemberID: crew.ID,
		AmountCents:  0,
		Category:     models.TxCatEarning,
	})

	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for zero amount, got %v", err)
	}
}

func TestCreditNegativeAmount(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	_, err := svc.Credit(ctx, CreditInput{
		CrewMemberID: crew.ID,
		AmountCents:  -5000,
		Category:     models.TxCatEarning,
	})

	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation for negative amount, got %v", err)
	}
}

func TestAutoCreateWallet(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	// No wallet exists yet — credit should auto-create
	_, err := svc.Credit(ctx, CreditInput{
		CrewMemberID:   crew.ID,
		AmountCents:    50000,
		Category:       models.TxCatTopUp,
		IdempotencyKey: "topup-001",
	})
	if err != nil {
		t.Fatalf("Credit with auto-create: %v", err)
	}

	wallet, err := svc.GetBalance(ctx, crew.ID)
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if wallet.BalanceCents != 50000 {
		t.Errorf("balance = %d, want 50000", wallet.BalanceCents)
	}
}
