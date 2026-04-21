// Package payment provides a Strategy-pattern payment abstraction.
// JamboPay is the default provider; additional providers can be added
// by implementing the Provider interface.
package payment

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// PayoutChannel identifies how money is sent.
type PayoutChannel string

const (
	ChannelMobile  PayoutChannel = "MOMO_B2C" // Mobile money (M-Pesa B2C)
	ChannelBank    PayoutChannel = "BANK"      // Bank transfer
	ChannelPaybill PayoutChannel = "MOMO_B2B"  // Paybill / Till
)

// PayoutRequest holds the data for initiating a payout.
type PayoutRequest struct {
	AmountCents    int64         `json:"amount_cents"`
	AccountFrom    string        `json:"account_from"`     // Source wallet/account
	OrderID        string        `json:"order_id"`         // Idempotency reference
	Channel        PayoutChannel `json:"channel"`
	RecipientName  string        `json:"recipient_name"`   // payTo.accountRef
	RecipientPhone string        `json:"recipient_phone"`  // For mobile payouts
	BankAccount    string        `json:"bank_account"`     // For bank payouts
	BankCode       string        `json:"bank_code"`        // For bank payouts
	PaybillNumber  string        `json:"paybill_number"`   // For paybill payouts
	PaybillRef     string        `json:"paybill_ref"`      // For paybill payouts
	CallbackURL    string        `json:"callback_url"`
	Narration      string        `json:"narration"`
}

// PayoutResult holds the response from a payout initiation.
type PayoutResult struct {
	Provider    string `json:"provider"`
	Reference   string `json:"reference"`    // Provider's transaction reference
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`       // "pending_otp", "completed", "failed"
	RequiresOTP bool   `json:"requires_otp"` // Whether OTP verification is needed
}

// PayoutVerifyRequest holds the data for verifying/authorizing a payout with OTP.
type PayoutVerifyRequest struct {
	Reference string `json:"reference"`
	OTP       string `json:"otp"`
}

// BalanceResult holds wallet/account balance information.
type BalanceResult struct {
	Provider string `json:"provider"`
	Balance  int64  `json:"balance_cents"`
	Currency string `json:"currency"`
}

// Provider defines the contract for payment/payout providers.
type Provider interface {
	Name() string
	InitiatePayout(ctx context.Context, req PayoutRequest) (*PayoutResult, error)
	VerifyPayout(ctx context.Context, req PayoutVerifyRequest) (*PayoutResult, error)
	CheckBalance(ctx context.Context, accountNo string) (*BalanceResult, error)
}

// Manager orchestrates payment providers with fallback support.
type Manager struct {
	providers []Provider
	mu        sync.RWMutex
	logger    *slog.Logger
}

// NewManager creates a payment manager with the given providers.
func NewManager(logger *slog.Logger, providers ...Provider) *Manager {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	logger.Info("payment manager initialized", slog.Any("providers", names))
	return &Manager{providers: providers, logger: logger}
}

// InitiatePayout dispatches a payout using the primary provider.
func (m *Manager) InitiatePayout(ctx context.Context, req PayoutRequest) (*PayoutResult, error) {
	m.mu.RLock()
	providers := m.providers
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		result, err := p.InitiatePayout(ctx, req)
		if err == nil {
			return result, nil
		}
		lastErr = err
		m.logger.Warn("payment provider failed", slog.String("provider", p.Name()), slog.String("error", err.Error()))
	}
	return nil, fmt.Errorf("all payment providers failed: %w", lastErr)
}

// VerifyPayout authorizes a pending payout.
func (m *Manager) VerifyPayout(ctx context.Context, req PayoutVerifyRequest) (*PayoutResult, error) {
	m.mu.RLock()
	primary := m.providers[0]
	m.mu.RUnlock()
	return primary.VerifyPayout(ctx, req)
}

// CheckBalance retrieves account balance from the primary provider.
func (m *Manager) CheckBalance(ctx context.Context, accountNo string) (*BalanceResult, error) {
	m.mu.RLock()
	primary := m.providers[0]
	m.mu.RUnlock()
	return primary.CheckBalance(ctx, accountNo)
}
