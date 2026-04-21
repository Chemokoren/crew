package payment

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"log/slog"
	"os"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// mockPaymentProvider implements payment.Provider for testing.
type mockPaymentProvider struct {
	name       string
	payoutErr  error
	balanceErr error
	mu         sync.Mutex
	callCount  int
}

func (m *mockPaymentProvider) Name() string { return m.name }

func (m *mockPaymentProvider) InitiatePayout(ctx context.Context, req PayoutRequest) (*PayoutResult, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	if m.payoutErr != nil {
		return nil, m.payoutErr
	}
	return &PayoutResult{
		Provider:    m.name,
		Reference:   "REF-001",
		OrderID:     req.OrderID,
		Status:      "pending_otp",
		RequiresOTP: true,
	}, nil
}

func (m *mockPaymentProvider) VerifyPayout(ctx context.Context, req PayoutVerifyRequest) (*PayoutResult, error) {
	return &PayoutResult{
		Provider:  m.name,
		Reference: req.Reference,
		Status:    "completed",
	}, nil
}

func (m *mockPaymentProvider) CheckBalance(ctx context.Context, accountNo string) (*BalanceResult, error) {
	if m.balanceErr != nil {
		return nil, m.balanceErr
	}
	return &BalanceResult{
		Provider: m.name,
		Balance:  1500000,
		Currency: "KES",
	}, nil
}

func TestManagerInitiatePayoutPrimary(t *testing.T) {
	primary := &mockPaymentProvider{name: "primary"}
	fallback := &mockPaymentProvider{name: "fallback"}
	mgr := NewManager(testLogger(), primary, fallback)

	result, err := mgr.InitiatePayout(context.Background(), PayoutRequest{
		AmountCents: 100000, OrderID: "ORD-001", Channel: ChannelMobile,
	})
	if err != nil {
		t.Fatalf("InitiatePayout: %v", err)
	}
	if result.Provider != "primary" {
		t.Errorf("Provider = %q, want primary", result.Provider)
	}
	if fallback.callCount != 0 {
		t.Error("fallback should not be called when primary succeeds")
	}
}

func TestManagerInitiatePayoutFallback(t *testing.T) {
	primary := &mockPaymentProvider{name: "primary", payoutErr: fmt.Errorf("timeout")}
	fallback := &mockPaymentProvider{name: "fallback"}
	mgr := NewManager(testLogger(), primary, fallback)

	result, err := mgr.InitiatePayout(context.Background(), PayoutRequest{
		AmountCents: 50000, Channel: ChannelBank,
	})
	if err != nil {
		t.Fatalf("should fallback: %v", err)
	}
	if result.Provider != "fallback" {
		t.Errorf("Provider = %q, want fallback", result.Provider)
	}
}

func TestManagerInitiatePayoutAllFail(t *testing.T) {
	p1 := &mockPaymentProvider{name: "p1", payoutErr: fmt.Errorf("fail")}
	p2 := &mockPaymentProvider{name: "p2", payoutErr: fmt.Errorf("also fail")}
	mgr := NewManager(testLogger(), p1, p2)

	_, err := mgr.InitiatePayout(context.Background(), PayoutRequest{})
	if err == nil {
		t.Error("should fail when all providers fail")
	}
}

func TestManagerVerifyPayout(t *testing.T) {
	primary := &mockPaymentProvider{name: "jambopay"}
	mgr := NewManager(testLogger(), primary)

	result, err := mgr.VerifyPayout(context.Background(), PayoutVerifyRequest{
		Reference: "REF-001", OTP: "123456",
	})
	if err != nil {
		t.Fatalf("VerifyPayout: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want completed", result.Status)
	}
}

func TestManagerCheckBalance(t *testing.T) {
	primary := &mockPaymentProvider{name: "jambopay"}
	mgr := NewManager(testLogger(), primary)

	result, err := mgr.CheckBalance(context.Background(), "ACC-001")
	if err != nil {
		t.Fatalf("CheckBalance: %v", err)
	}
	if result.Balance != 1500000 {
		t.Errorf("Balance = %d, want 1500000", result.Balance)
	}
	if result.Currency != "KES" {
		t.Errorf("Currency = %q, want KES", result.Currency)
	}
}

func TestPayoutChannelConstants(t *testing.T) {
	tests := []struct {
		channel PayoutChannel
		want    string
	}{
		{ChannelMobile, "MOMO_B2C"},
		{ChannelBank, "BANK"},
		{ChannelPaybill, "MOMO_B2B"},
	}
	for _, tt := range tests {
		if string(tt.channel) != tt.want {
			t.Errorf("channel %v = %q, want %q", tt.channel, string(tt.channel), tt.want)
		}
	}
}
