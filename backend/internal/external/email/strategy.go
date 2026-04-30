// Package email provides a Strategy-pattern email abstraction.
// Multiple providers can be registered (Gmail SMTP, SendGrid, Twilio, etc.)
// and the active one is selected at runtime via configuration.
// The engine is reusable across all email use cases: OTP, support, notifications.
package email

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// SendResult holds the outcome of an email send attempt.
type SendResult struct {
	Provider  string `json:"provider"`
	MessageID string `json:"message_id,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// Message represents a structured email message.
type Message struct {
	To      []string // Recipient addresses
	Subject string   // Email subject line
	Body    string   // Plain text body
	HTML    string   // HTML body (optional, takes priority if set)
}

// Provider defines the contract every email provider must implement.
// Adding a new email vendor (SendGrid, Mailgun, etc.) requires only
// implementing this interface and registering it with the Manager.
type Provider interface {
	// Name returns a unique identifier for this provider (e.g. "gmail", "sendgrid").
	Name() string

	// Send dispatches a single email message.
	Send(ctx context.Context, msg Message) (*SendResult, error)
}

// Manager orchestrates email providers using the Strategy pattern.
// It maintains an ordered list of providers and uses the primary (first)
// with automatic fallback to subsequent providers on failure.
type Manager struct {
	providers []Provider
	mu        sync.RWMutex
	logger    *slog.Logger
}

// NewManager creates an email manager with the given providers.
// The first provider is the primary; subsequent providers are fallbacks.
func NewManager(logger *slog.Logger, providers ...Provider) *Manager {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	logger.Info("email manager initialized",
		slog.Any("providers", names),
		slog.String("primary", names[0]),
	)
	return &Manager{providers: providers, logger: logger}
}

// Send dispatches an email using the primary provider.
// Falls back to the next provider in the chain on failure.
func (m *Manager) Send(ctx context.Context, msg Message) (*SendResult, error) {
	m.mu.RLock()
	providers := m.providers
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		result, err := p.Send(ctx, msg)
		if err == nil && result.Success {
			return result, nil
		}

		lastErr = err
		if err != nil {
			m.logger.Warn("email provider failed, trying next",
				slog.String("provider", p.Name()),
				slog.Any("to", msg.To),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil, fmt.Errorf("all email providers failed: %w", lastErr)
}

// SendToOne is a convenience method for sending to a single recipient.
func (m *Manager) SendToOne(ctx context.Context, to, subject, body string) (*SendResult, error) {
	return m.Send(ctx, Message{
		To:      []string{to},
		Subject: subject,
		Body:    body,
	})
}

// SendHTML sends an HTML email to a single recipient.
func (m *Manager) SendHTML(ctx context.Context, to, subject, htmlBody string) (*SendResult, error) {
	return m.Send(ctx, Message{
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
	})
}

// SetPrimary reorders providers so the named provider becomes primary.
// This allows runtime switching without a restart.
func (m *Manager) SetPrimary(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.providers {
		if p.Name() == name {
			m.providers = append([]Provider{p}, append(m.providers[:i], m.providers[i+1:]...)...)
			m.logger.Info("email primary provider switched", slog.String("provider", name))
			return nil
		}
	}
	return fmt.Errorf("email provider %q not found", name)
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
