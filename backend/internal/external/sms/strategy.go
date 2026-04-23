// Package sms provides a Strategy-pattern SMS abstraction.
// Multiple providers can be registered and the active one is selected at runtime.
// Providers can also be run in parallel for redundancy (fallback chain).
package sms

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
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
type Manager struct {
	providers []Provider
	mu        sync.RWMutex
	logger    *slog.Logger
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
		providers: providers,
		logger:    logger,
	}
}

// Send dispatches an SMS using the primary provider.
// Falls back to the next provider in the chain on failure.
func (m *Manager) Send(ctx context.Context, phone, message string) (*SendResult, error) {
	m.mu.RLock()
	providers := m.providers
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		result, err := p.Send(ctx, phone, message)
		if err == nil && result.Success {
			return result, nil
		}

		lastErr = err
		if err != nil {
			m.logger.Warn("SMS provider failed, trying next",
				slog.String("provider", p.Name()),
				slog.String("phone", phone),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil, fmt.Errorf("all SMS providers failed: %w", lastErr)
}

// SendBulk dispatches the same message to multiple recipients using the primary provider.
func (m *Manager) SendBulk(ctx context.Context, phones []string, message string) ([]SendResult, error) {
	m.mu.RLock()
	providers := m.providers
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		results, err := p.SendBulk(ctx, phones, message)
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
