// Package payroll provides a Strategy-pattern payroll processing abstraction.
// PerPay is the default provider. The interface is designed for minimal
// future modification when swapping or adding payroll providers.
package payroll

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// PayComponent represents a single pay element (salary, bonus, overtime).
type PayComponent struct {
	ID          string  `json:"id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// Deduction represents a single deduction (insurance, SACCO contribution, tax).
type Deduction struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Type   string  `json:"type"`
	PreTax bool    `json:"pre_tax"`
}

// SubmitRequest holds the data for submitting a payroll request.
type SubmitRequest struct {
	EmployeeID        string         `json:"employee_id"`
	FullName          string         `json:"full_name"`
	EmployeePIN       string         `json:"employee_pin"`   // KRA PIN (e.g. "A123456789K")
	Currency          string         `json:"currency"`
	PayPeriodStart    string         `json:"pay_period_start_date"` // YYYY-MM-DD
	PayPeriodEnd      string         `json:"pay_period_end_date"`   // YYYY-MM-DD
	PayComponents     []PayComponent `json:"pay_components"`
	Deductions        []Deduction    `json:"deductions"`
	IdempotencyKey    string         `json:"-"` // Set via header
	CorrelationID     string         `json:"-"` // Set via header (optional)
}

// SubmitResult holds the response from a payroll submission.
type SubmitResult struct {
	Provider      string `json:"provider"`
	CorrelationID string `json:"correlation_id"`
	Status        string `json:"status"` // "received", "processing", "completed", "failed"
	StatusURL     string `json:"status_url"`
}

// StatusResult holds the current processing status of a payroll request.
type StatusResult struct {
	Provider      string  `json:"provider"`
	CorrelationID string  `json:"correlation_id"`
	Status        string  `json:"status"`
	CurrentStep   string  `json:"current_step,omitempty"`
	GrossPay      float64 `json:"gross_pay,omitempty"`
	NetPay        float64 `json:"net_pay,omitempty"`
	Deductions    float64 `json:"total_deductions,omitempty"`
	ErrorCode     string  `json:"error_code,omitempty"`
	ErrorMessage  string  `json:"error_message,omitempty"`
}

// Provider defines the contract for payroll processing providers.
type Provider interface {
	Name() string
	SubmitPayroll(ctx context.Context, req SubmitRequest) (*SubmitResult, error)
	GetStatus(ctx context.Context, correlationID string) (*StatusResult, error)
}

// Manager orchestrates payroll providers with the Strategy pattern.
type Manager struct {
	providers []Provider
	mu        sync.RWMutex
	logger    *slog.Logger
}

// NewManager creates a payroll manager.
func NewManager(logger *slog.Logger, providers ...Provider) *Manager {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	logger.Info("payroll manager initialized", slog.Any("providers", names))
	return &Manager{providers: providers, logger: logger}
}

// SubmitPayroll submits a payroll request using the primary provider.
// Falls back to the next provider in the chain on failure.
func (m *Manager) SubmitPayroll(ctx context.Context, req SubmitRequest) (*SubmitResult, error) {
	m.mu.RLock()
	providers := m.providers
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		result, err := p.SubmitPayroll(ctx, req)
		if err == nil {
			return result, nil
		}
		lastErr = err
		m.logger.Warn("payroll provider failed, trying next",
			slog.String("provider", p.Name()),
			slog.String("error", err.Error()),
		)
	}
	return nil, fmt.Errorf("all payroll providers failed: %w", lastErr)
}

// GetStatus queries processing status from the primary provider.
// Falls back to the next provider in the chain on failure.
func (m *Manager) GetStatus(ctx context.Context, correlationID string) (*StatusResult, error) {
	m.mu.RLock()
	providers := m.providers
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		result, err := p.GetStatus(ctx, correlationID)
		if err == nil {
			return result, nil
		}
		lastErr = err
		m.logger.Warn("payroll status check failed, trying next",
			slog.String("provider", p.Name()),
			slog.String("error", err.Error()),
		)
	}
	return nil, fmt.Errorf("all payroll providers failed for status check: %w", lastErr)
}

// SetPrimary reorders providers so the named provider becomes primary.
func (m *Manager) SetPrimary(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.providers {
		if p.Name() == name {
			m.providers = append([]Provider{p}, append(m.providers[:i], m.providers[i+1:]...)...)
			m.logger.Info("payroll primary provider switched", slog.String("provider", name))
			return nil
		}
	}
	return fmt.Errorf("payroll provider %q not found", name)
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
