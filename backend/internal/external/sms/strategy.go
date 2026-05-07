// Package sms provides a Strategy-pattern SMS abstraction.
// Multiple providers can be registered and the active one is selected at runtime.
// Providers can also be run in parallel for redundancy (fallback chain).
package sms

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kibsoft/amy-mis/pkg/retry"
)

// SendResult holds the outcome of an SMS send attempt.
type SendResult struct {
	Provider  string `json:"provider"`
	MessageID string `json:"message_id,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// Provider defines the contract every SMS provider must implement.
// Adding a new SMS vendor requires only implementing this interface
// and registering it with the Manager.
type Provider interface {
	// Name returns a unique identifier for this provider (e.g. "optimize", "africastalking").
	Name() string

	// Send dispatches an SMS to the given phone number.
	// Phone must be in E.164 format (e.g. "+254712345678").
	Send(ctx context.Context, phone, message string) (*SendResult, error)

	// SendBulk dispatches the same message to multiple recipients.
	// Default implementations can loop over Send, but providers may override
	// with batch APIs for efficiency.
	SendBulk(ctx context.Context, phones []string, message string) ([]SendResult, error)
}

// Manager orchestrates SMS providers using the Strategy pattern.
// It maintains an ordered list of providers and uses the primary (first)
// with automatic fallback to subsequent providers on failure.
// Transient network errors are retried with exponential backoff.
type Manager struct {
	providers   []Provider
	mu          sync.RWMutex
	logger      *slog.Logger
	retryPolicy retry.Policy
}

// NewManager creates an SMS manager with the given providers.
// The first provider is the primary; subsequent providers are fallbacks.
func NewManager(logger *slog.Logger, providers ...Provider) *Manager {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}

	logger.Info("SMS manager initialized",
		slog.Any("providers", names),
		slog.String("primary", names[0]),
	)

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
}

// Send dispatches an SMS using the primary provider.
// Falls back to the next provider in the chain on failure.
// Each provider attempt is retried with exponential backoff on network errors.
func (m *Manager) Send(ctx context.Context, phone, message string) (*SendResult, error) {
	m.mu.RLock()
	providers := m.providers
	policy := m.retryPolicy
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		result, err := retry.Do(ctx, m.logger, "sms.Send/"+p.Name(), policy,
			retry.IsNetworkError,
			func(ctx context.Context) (*SendResult, error) {
				res, err := p.Send(ctx, phone, message)
				if err != nil {
					return nil, err
				}
				if !res.Success {
					return nil, fmt.Errorf("sms send failed: %s", res.Error)
				}
				return res, nil
			},
		)
		if err == nil {
			return result, nil
		}

		lastErr = err
		m.logger.Warn("SMS provider failed, trying next",
			slog.String("provider", p.Name()),
			slog.String("phone", phone),
			slog.String("error", err.Error()),
		)
	}

	return nil, fmt.Errorf("all SMS providers failed: %w", lastErr)
}

// SendBulk dispatches the same message to multiple recipients using the primary provider.
func (m *Manager) SendBulk(ctx context.Context, phones []string, message string) ([]SendResult, error) {
	m.mu.RLock()
	providers := m.providers
	policy := m.retryPolicy
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		results, err := retry.Do(ctx, m.logger, "sms.SendBulk/"+p.Name(), policy,
			retry.IsNetworkError,
			func(ctx context.Context) ([]SendResult, error) {
				return p.SendBulk(ctx, phones, message)
			},
		)
		if err == nil {
			return results, nil
		}
		lastErr = err
		m.logger.Warn("SMS bulk provider failed, trying next",
			slog.String("provider", p.Name()),
			slog.String("error", err.Error()),
		)
	}

	return nil, fmt.Errorf("all SMS providers failed for bulk send: %w", lastErr)
}

// SetPrimary reorders providers so the named provider becomes primary.
// This allows runtime switching without a restart.
func (m *Manager) SetPrimary(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.providers {
		if p.Name() == name {
			// Move to front
			m.providers = append([]Provider{p}, append(m.providers[:i], m.providers[i+1:]...)...)
			m.logger.Info("SMS primary provider switched", slog.String("provider", name))
			return nil
		}
	}

	return fmt.Errorf("SMS provider %q not found", name)
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

