package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/service"
)

func setupTransactionService(t *testing.T) *service.TransactionService {
	t.Helper()
	// TransactionService with nil txMgr — only validation paths can be tested
	// (RunInTx panics with nil, but validation errors return before that point)
	txSvc := service.NewTransactionService(nil, nil, nil, nil, nil)
	return txSvc
}

// ── Employee Payout Tests ────────────────────────────────

func TestEmployeePayout_ValidationErrors(t *testing.T) {
	txSvc := setupTransactionService(t)
	ctx := context.Background()

	tests := []struct {
		name  string
		input service.EmployeePayoutInput
	}{
		{
			name: "zero gross",
			input: service.EmployeePayoutInput{
				OrganizationID: uuid.New(),
				CrewMemberID:   uuid.New(),
				GrossCents:     0,
				NetCents:       100,
				IdempotencyKey: uuid.New().String(),
			},
		},
		{
			name: "zero net",
			input: service.EmployeePayoutInput{
				OrganizationID: uuid.New(),
				CrewMemberID:   uuid.New(),
				GrossCents:     100,
				NetCents:       0,
				IdempotencyKey: uuid.New().String(),
			},
		},
		{
			name: "negative gross",
			input: service.EmployeePayoutInput{
				OrganizationID: uuid.New(),
				CrewMemberID:   uuid.New(),
				GrossCents:     -500,
				NetCents:       100,
				IdempotencyKey: uuid.New().String(),
			},
		},
		{
			name: "net exceeds gross",
			input: service.EmployeePayoutInput{
				OrganizationID: uuid.New(),
				CrewMemberID:   uuid.New(),
				GrossCents:     100,
				NetCents:       200,
				IdempotencyKey: uuid.New().String(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := txSvc.EmployeePayout(ctx, tc.input)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

// ── Wallet Transfer Tests ────────────────────────────────

func TestWalletTransfer_ValidationErrors(t *testing.T) {
	txSvc := setupTransactionService(t)
	ctx := context.Background()

	sameID := uuid.New()
	tests := []struct {
		name  string
		input service.WalletTransferInput
	}{
		{
			name: "zero amount",
			input: service.WalletTransferInput{
				FromCrewMemberID: uuid.New(),
				ToCrewMemberID:   uuid.New(),
				AmountCents:      0,
				IdempotencyKey:   uuid.New().String(),
			},
		},
		{
			name: "negative amount",
			input: service.WalletTransferInput{
				FromCrewMemberID: uuid.New(),
				ToCrewMemberID:   uuid.New(),
				AmountCents:      -500,
				IdempotencyKey:   uuid.New().String(),
			},
		},
		{
			name: "self transfer",
			input: service.WalletTransferInput{
				FromCrewMemberID: sameID,
				ToCrewMemberID:   sameID,
				AmountCents:      100,
				IdempotencyKey:   uuid.New().String(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := txSvc.WalletTransfer(ctx, tc.input)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}
