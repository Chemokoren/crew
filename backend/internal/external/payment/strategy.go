// Package payment provides a Strategy-pattern payment abstraction.
// JamboPay is the default provider; additional providers can be added
// by implementing the Provider interface.
package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kibsoft/amy-mis/pkg/retry"
)

// ErrNotImplemented is returned by providers that don't support a specific operation.
var ErrNotImplemented = errors.New("operation not implemented by this provider")

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

// CollectionRequest holds data for initiating a mobile money collection (STK push).
type CollectionRequest struct {
	AmountCents int64  `json:"amount_cents"`
	AccountTo   string `json:"account_to"`    // Collection account receiving funds
	OrderID     string `json:"order_id"`      // Idempotency key
	Provider    string `json:"provider"`      // "MPESA", "AIRTEL_MONEY"
	PhoneNumber string `json:"phone_number"` // Phone to push STK to
	Description string `json:"description"`
	CallbackURL string `json:"callback_url"`
}

// CollectionResult holds the response from a collection initiation.
type CollectionResult struct {
	Provider  string `json:"provider"`
	Reference string `json:"reference"` // Provider's transaction reference
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`    // "pending", "completed", "failed"
}

// BankVerificationRequest holds data for verifying a bank transfer reference.
type BankVerificationRequest struct {
	BankRef     string `json:"bank_ref"`      // Transaction reference from the bank
	BankCode    string `json:"bank_code"`      // Bank identifier (kcb, equity, coop, etc.)
	AmountCents int64  `json:"amount_cents"`   // Expected amount
	AccountTo   string `json:"account_to"`     // Receiving account number
}

// BankVerificationResult holds the result of a bank transfer verification.
type BankVerificationResult struct {
	Verified    bool   `json:"verified"`     // Whether the reference was successfully verified
	Provider    string `json:"provider"`     // Provider that verified
	Reference   string `json:"reference"`    // Verified bank reference
	Status      string `json:"status"`       // "VERIFIED", "NOT_FOUND", "MISMATCH", "UNAVAILABLE"
	Message     string `json:"message"`      // Human-readable status detail
	AmountCents int64  `json:"amount_cents"` // Actual amount found (if verified)
}

// Provider defines the contract for payment/payout providers.
type Provider interface {
	Name() string
	InitiatePayout(ctx context.Context, req PayoutRequest) (*PayoutResult, error)
	VerifyPayout(ctx context.Context, req PayoutVerifyRequest) (*PayoutResult, error)
	CheckBalance(ctx context.Context, accountNo string) (*BalanceResult, error)
	InitiateCollection(ctx context.Context, req CollectionRequest) (*CollectionResult, error)
	// VerifyBankTransfer checks a bank transfer reference against the bank's records.
	// Returns ErrNotImplemented if the provider does not support bank verification.
	VerifyBankTransfer(ctx context.Context, req BankVerificationRequest) (*BankVerificationResult, error)
}

// Manager orchestrates payment providers with fallback support
// and automatic exponential-backoff retry on transient network errors.
type Manager struct {
	providers   []Provider
	mu          sync.RWMutex
	logger      *slog.Logger
	retryPolicy retry.Policy
}

// NewManager creates a payment manager with the given providers.
// An optional retry.Policy can be passed; if omitted, DefaultPolicy() is used.
func NewManager(logger *slog.Logger, providers ...Provider) *Manager {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	logger.Info("payment manager initialized", slog.Any("providers", names))
	return &Manager{
		providers:   providers,
		logger:      logger,
		retryPolicy: retry.DefaultPolicy(),
	}
}

// SetRetryPolicy updates the retry policy for all future calls.
func (m *Manager) SetRetryPolicy(p retry.Policy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retryPolicy = p
	m.logger.Info("payment retry policy updated",
		slog.Int("max_attempts", p.MaxAttempts),
		slog.Duration("initial_delay", p.InitialDelay),
		slog.Duration("max_delay", p.MaxDelay),
	)
}

// InitiatePayout dispatches a payout using the primary provider with retry.
func (m *Manager) InitiatePayout(ctx context.Context, req PayoutRequest) (*PayoutResult, error) {
	m.mu.RLock()
	providers := m.providers
	policy := m.retryPolicy
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		result, err := retry.Do(ctx, m.logger, "payment.InitiatePayout/"+p.Name(), policy,
			retry.IsNetworkError,
			func(ctx context.Context) (*PayoutResult, error) {
				return p.InitiatePayout(ctx, req)
			},
		)
		if err == nil {
			return result, nil
		}
		lastErr = err
		m.logger.Warn("payment provider failed", slog.String("provider", p.Name()), slog.String("error", err.Error()))
	}
	return nil, fmt.Errorf("all payment providers failed: %w", lastErr)
}

// VerifyPayout authorizes a pending payout with retry.
func (m *Manager) VerifyPayout(ctx context.Context, req PayoutVerifyRequest) (*PayoutResult, error) {
	m.mu.RLock()
	primary := m.providers[0]
	policy := m.retryPolicy
	m.mu.RUnlock()

	return retry.Do(ctx, m.logger, "payment.VerifyPayout/"+primary.Name(), policy,
		retry.IsNetworkError,
		func(ctx context.Context) (*PayoutResult, error) {
			return primary.VerifyPayout(ctx, req)
		},
	)
}

// CheckBalance retrieves account balance from the primary provider with retry.
func (m *Manager) CheckBalance(ctx context.Context, accountNo string) (*BalanceResult, error) {
	m.mu.RLock()
	primary := m.providers[0]
	policy := m.retryPolicy
	m.mu.RUnlock()

	return retry.Do(ctx, m.logger, "payment.CheckBalance/"+primary.Name(), policy,
		retry.IsNetworkError,
		func(ctx context.Context) (*BalanceResult, error) {
			return primary.CheckBalance(ctx, accountNo)
		},
	)
}

// InitiateCollection dispatches a mobile money collection with retry.
func (m *Manager) InitiateCollection(ctx context.Context, req CollectionRequest) (*CollectionResult, error) {
	m.mu.RLock()
	primary := m.providers[0]
	policy := m.retryPolicy
	m.mu.RUnlock()

	return retry.Do(ctx, m.logger, "payment.InitiateCollection/"+primary.Name(), policy,
		retry.IsNetworkError,
		func(ctx context.Context) (*CollectionResult, error) {
			return primary.InitiateCollection(ctx, req)
		},
	)
}

// VerifyBankTransfer checks a bank transfer reference against providers.
// It tries each provider in order; providers that return ErrNotImplemented are skipped.
// If no provider supports verification, returns a result with Status="UNAVAILABLE".
func (m *Manager) VerifyBankTransfer(ctx context.Context, req BankVerificationRequest) (*BankVerificationResult, error) {
	m.mu.RLock()
	providers := m.providers
	m.mu.RUnlock()

	for _, p := range providers {
		result, err := p.VerifyBankTransfer(ctx, req)
		if err != nil {
			if errors.Is(err, ErrNotImplemented) {
				m.logger.Debug("provider does not support bank verification", slog.String("provider", p.Name()))
				continue
			}
			m.logger.Warn("bank verification failed", slog.String("provider", p.Name()), slog.String("error", err.Error()))
			continue
		}
		return result, nil
	}

	// No provider could verify — return UNAVAILABLE (not an error)
	return &BankVerificationResult{
		Verified: false,
		Status:   "UNAVAILABLE",
		Message:  "No payment provider supports bank transfer verification",
	}, nil
}

// SetPrimary reorders providers so the named provider becomes primary.
// This allows runtime switching without a restart.
func (m *Manager) SetPrimary(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.providers {
		if p.Name() == name {
			m.providers = append([]Provider{p}, append(m.providers[:i], m.providers[i+1:]...)...)
			m.logger.Info("payment primary provider switched", slog.String("provider", name))
			return nil
		}
	}
	return fmt.Errorf("payment provider %q not found", name)
}

// ProviderNames returns the names of all registered providers in order (primary first).
func (m *Manager) ProviderNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, len(m.providers))
	for i, p := range m.providers {
		names[i] = p.Name()
	}
	return names
}

