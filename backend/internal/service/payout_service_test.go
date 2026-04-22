package service_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/payment"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
)

type MockPaymentProvider struct {
	ShouldFail bool
}

func (m *MockPaymentProvider) Name() string { return "mock_payment" }
func (m *MockPaymentProvider) InitiatePayout(ctx context.Context, req payment.PayoutRequest) (*payment.PayoutResult, error) {
	if m.ShouldFail {
		return nil, os.ErrNotExist
	}
	return &payment.PayoutResult{
		Provider:  m.Name(),
		Reference: "MOCK_REF",
		OrderID:   req.OrderID,
		Status:    "completed",
	}, nil
}
func (m *MockPaymentProvider) VerifyPayout(ctx context.Context, req payment.PayoutVerifyRequest) (*payment.PayoutResult, error) {
	return nil, nil
}
func (m *MockPaymentProvider) CheckBalance(ctx context.Context, accountNo string) (*payment.BalanceResult, error) {
	return nil, nil
}

func TestPayoutService_InitiatePayout(t *testing.T) {
	crewRepo := mock.NewCrewRepo()
	walletRepo := mock.NewWalletRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditRepo := mock.NewAuditRepo()
	auditSvc := service.NewAuditService(auditRepo, logger)
	walletSvc := service.NewWalletService(walletRepo, crewRepo, auditSvc, logger)

	mockProvider := &MockPaymentProvider{ShouldFail: false}
	payManager := payment.NewManager(logger, mockProvider)
	payoutSvc := service.NewPayoutService(walletSvc, payManager, auditSvc, logger)

	crew := &models.CrewMember{ID: uuid.New(), CrewID: "CRW-01", KYCStatus: models.KYCVerified}
	crewRepo.Create(context.Background(), crew)
	_, err := walletSvc.Credit(context.Background(), service.CreditInput{
		CrewMemberID:   crew.ID,
		AmountCents:    5000,
		Category:       models.TxCatTopUp,
		IdempotencyKey: "test-credit",
	})
	if err != nil {
		t.Fatalf("failed to setup test: %v", err)
	}

	res, err := payoutSvc.InitiatePayout(context.Background(), service.PayoutInput{
		CrewMemberID:   crew.ID,
		AmountCents:    2000,
		Channel:        payment.ChannelMobile,
		RecipientName:  "Jane",
		RecipientPhone: "0712345678",
		IdempotencyKey: "test-payout-1",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Reference != "MOCK_REF" {
		t.Errorf("expected MOCK_REF, got %s", res.Reference)
	}

	wallet, _ := walletSvc.GetBalance(context.Background(), crew.ID)
	if wallet.BalanceCents != 3000 {
		t.Errorf("expected balance 3000, got %d", wallet.BalanceCents)
	}
}

func TestPayoutService_InitiatePayout_FailsIfInsufficientBalance(t *testing.T) {
	crewRepo := mock.NewCrewRepo()
	walletRepo := mock.NewWalletRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auditRepo := mock.NewAuditRepo()
	auditSvc := service.NewAuditService(auditRepo, logger)
	walletSvc := service.NewWalletService(walletRepo, crewRepo, auditSvc, logger)

	mockProvider := &MockPaymentProvider{ShouldFail: false}
	payManager := payment.NewManager(logger, mockProvider)
	payoutSvc := service.NewPayoutService(walletSvc, payManager, auditSvc, logger)

	crew := &models.CrewMember{ID: uuid.New(), CrewID: "CRW-01", KYCStatus: models.KYCVerified}
	crewRepo.Create(context.Background(), crew)
	
	// No credit, balance is 0.
	
	_, err := payoutSvc.InitiatePayout(context.Background(), service.PayoutInput{
		CrewMemberID:   crew.ID,
		AmountCents:    2000,
		Channel:        payment.ChannelMobile,
		RecipientName:  "Jane",
		RecipientPhone: "0712345678",
		IdempotencyKey: "test-payout-insufficient",
	})

	if err == nil {
		t.Fatalf("expected error due to insufficient balance")
	}
}
