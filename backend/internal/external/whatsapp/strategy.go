// Package whatsapp provides a Strategy-pattern WhatsApp messaging abstraction.
// Supports multiple providers: Meta Cloud API (default), Twilio, etc.
package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// SendResult holds the outcome of a WhatsApp send attempt.
type SendResult struct {
	Provider  string `json:"provider"`
	MessageID string `json:"message_id,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// Provider defines the contract every WhatsApp provider must implement.
// Adding a new WhatsApp vendor (Twilio, Vonage, etc.) requires only
// implementing this interface and registering it with the Manager.
type Provider interface {
	// Name returns a unique identifier for this provider (e.g. "meta", "twilio").
	Name() string

	// Send dispatches a WhatsApp message to the given phone number.
	// Phone must be in E.164 format (e.g. "+254712345678").
	Send(ctx context.Context, phone, message string) (*SendResult, error)
}

// Manager orchestrates WhatsApp providers using the Strategy pattern.
type Manager struct {
	providers []Provider
	mu        sync.RWMutex
	logger    *slog.Logger
}

// NewManager creates a WhatsApp manager with the given providers.
func NewManager(logger *slog.Logger, providers ...Provider) *Manager {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	logger.Info("WhatsApp manager initialized",
		slog.Any("providers", names),
		slog.String("primary", names[0]),
	)
	return &Manager{providers: providers, logger: logger}
}

// Send dispatches a WhatsApp message using the primary provider.
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
			m.logger.Warn("WhatsApp provider failed, trying next",
				slog.String("provider", p.Name()),
				slog.String("phone", phone),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil, fmt.Errorf("all WhatsApp providers failed: %w", lastErr)
}

// SetPrimary reorders providers so the named provider becomes primary.
func (m *Manager) SetPrimary(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.providers {
		if p.Name() == name {
			m.providers = append([]Provider{p}, append(m.providers[:i], m.providers[i+1:]...)...)
			m.logger.Info("WhatsApp primary provider switched", slog.String("provider", name))
			return nil
		}
	}
	return fmt.Errorf("WhatsApp provider %q not found", name)
}

// ProviderNames returns the names of all registered providers in order.
func (m *Manager) ProviderNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, len(m.providers))
	for i, p := range m.providers {
		names[i] = p.Name()
	}
	return names
}
