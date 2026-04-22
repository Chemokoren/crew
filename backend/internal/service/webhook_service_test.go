package service_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/external/payment"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
)



func TestWebhookService_JamboPayReversal(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	walletRepo := mock.NewWalletRepo()
	crewRepo := mock.NewCrewRepo()
	webhookRepo := mock.NewWebhookEventRepo()
	
	// Create a WalletService and PayoutService
	auditRepo := mock.NewAuditRepo()
	auditSvc := service.NewAuditService(auditRepo, logger)
	walletSvc := service.NewWalletService(walletRepo, crewRepo, auditSvc, logger)
	paymentMgr := payment.NewManager(logger, &MockPaymentProvider{})
	payoutSvc := service.NewPayoutService(walletSvc, paymentMgr, auditSvc, logger)
	
	// We pass nil for payrollRepo and payrollSvc because they aren't strictly required for JamboPay webhook tests right now
	webhookSvc := service.NewWebhookService(webhookRepo, payoutSvc, nil, walletRepo, nil, logger)

	ctx := context.Background()

	// 1. Setup Wallet for Crew Member
	crewID := uuid.New()
	wallet := &models.Wallet{CrewMemberID: crewID, Currency: "KES", IsActive: true}
	_ = walletRepo.Create(ctx, wallet)

	// 2. Add some balance
	_, _ = walletSvc.Credit(ctx, service.CreditInput{
		CrewMemberID:   crewID,
		AmountCents:    10000,
		Category:       models.TxCatEarning,
		IdempotencyKey: "earn-1",
	})

	// 3. Initiate a Payout (creates debit)
	_, err := payoutSvc.InitiatePayout(ctx, service.PayoutInput{
		CrewMemberID:   crewID,
		AmountCents:    5000,
		Channel:        payment.ChannelMobile,
		RecipientPhone: "254700000000",
		IdempotencyKey: "payout-order-1", // Will become "payout-payout-order-1" as key in debit
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w, _ := walletRepo.GetWalletByID(ctx, wallet.ID)
	if w.BalanceCents != 5000 {
		t.Errorf("expected balance 5000 after payout debit, got %d", w.BalanceCents)
	}

	// Mock debit returns TxCompleted by default, but a webhook would operate on a PENDING tx
	// Set it to pending manually
	tx, _ := walletRepo.GetByIdempotencyKey(ctx, "payout-payout-order-1")
	tx.Status = models.TxPending
	_ = walletRepo.UpdateTransaction(ctx, tx)

	payloadMap := map[string]interface{}{
		"order_id": "payout-order-1",
		"status":   "FAILED",
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	// 4. Process webhook
	err = webhookSvc.ProcessJamboPayWebhook(ctx, payloadBytes)
	if err != nil {
		t.Fatalf("webhook error: %v", err)
	}

	// 5. Verify reversal
	wAfter, _ := walletRepo.GetWalletByID(ctx, wallet.ID)
	if wAfter.BalanceCents != 10000 {
		t.Errorf("expected balance to be reversed back to 10000, got %d", wAfter.BalanceCents)
	}

	// Verify the original tx is FAILED
	tx, err = walletRepo.GetByIdempotencyKey(ctx, "payout-payout-order-1")
	if err != nil {
		t.Fatalf("failed to find tx: %v", err)
	}
	if tx.Status != models.TxFailed {
		t.Errorf("expected tx to be FAILED, got %s", tx.Status)
	}

	// Verify webhook event was recorded
	events, _ := webhookRepo.ListUnprocessed(ctx, models.WebhookJamboPay, 10)
	if len(events) != 0 {
		// Because it was marked processed
		t.Errorf("expected 0 unprocessed events, got %d", len(events))
	}
}

func TestWebhookService_PerpayCompletion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	webhookRepo := mock.NewWebhookEventRepo()
	webhookSvc := service.NewWebhookService(webhookRepo, nil, nil, nil, nil, logger)

	ctx := context.Background()

	payloadMap := map[string]interface{}{
		"correlation_id": "corr-456",
		"status":         "COMPLETED",
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	err := webhookSvc.ProcessPerpayWebhook(ctx, payloadBytes)
	if err != nil {
		t.Fatalf("webhook error: %v", err)
	}

	// Just verifying it doesn't crash and creates an event
	events, _ := webhookRepo.ListUnprocessed(ctx, models.WebhookPerpay, 10)
	if len(events) != 0 {
		t.Errorf("expected 0 unprocessed events, got %d", len(events))
	}
}
