// Package identity provides a Strategy-pattern identity verification abstraction.
// IPRS (Integrated Population Registration System) is the default provider
// for Kenya national ID verification.
package identity

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// CitizenDetails holds the result of a national ID lookup.
type CitizenDetails struct {
	Provider     string `json:"provider"`
	IDNumber     string `json:"id_number"`
	SerialNumber string `json:"serial_number,omitempty"`
	FirstName    string `json:"first_name"`
	MiddleName   string `json:"middle_name,omitempty"`
	LastName     string `json:"last_name"`
	Gender       string `json:"gender"`
	DateOfBirth  string `json:"date_of_birth"`
	PlaceOfBirth string `json:"place_of_birth,omitempty"`
	Citizenship  string `json:"citizenship,omitempty"`
	Photo        string `json:"photo,omitempty"` // Base64 encoded
	Verified     bool   `json:"verified"`
}

// VerifyRequest holds the data for verifying a citizen's identity.
type VerifyRequest struct {
	IDNumber     string `json:"id_number" validate:"required"`
	SerialNumber string `json:"serial_number"` // Optional, improves accuracy
}

// Provider defines the contract for identity verification providers.
type Provider interface {
	Name() string
	VerifyCitizen(ctx context.Context, req VerifyRequest) (*CitizenDetails, error)
}

// Manager orchestrates identity verification providers.
type Manager struct {
	providers []Provider
	mu        sync.RWMutex
	logger    *slog.Logger
}

// NewManager creates an identity verification manager.
func NewManager(logger *slog.Logger, providers ...Provider) *Manager {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	logger.Info("identity manager initialized", slog.Any("providers", names))
	return &Manager{providers: providers, logger: logger}
}

// VerifyCitizen verifies a citizen's identity using the primary provider.
func (m *Manager) VerifyCitizen(ctx context.Context, req VerifyRequest) (*CitizenDetails, error) {
	m.mu.RLock()
	providers := m.providers
	m.mu.RUnlock()

	var lastErr error
	for _, p := range providers {
		result, err := p.VerifyCitizen(ctx, req)
		if err == nil {
			return result, nil
		}
		lastErr = err
		m.logger.Warn("identity provider failed",
			slog.String("provider", p.Name()),
			slog.String("error", err.Error()),
		)
	}
	return nil, fmt.Errorf("all identity providers failed: %w", lastErr)
}

// Name returns the name of the manager (implements Provider interface).
func (m *Manager) Name() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.providers) > 0 {
		return "identity-manager[" + m.providers[0].Name() + "]"
	}
	return "identity-manager"
}

// SetPrimary reorders providers so the named provider becomes primary.
// This allows runtime switching without a restart.
func (m *Manager) SetPrimary(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.providers {
		if p.Name() == name {
			m.providers = append([]Provider{p}, append(m.providers[:i], m.providers[i+1:]...)...)
			m.logger.Info("identity primary provider switched", slog.String("provider", name))
			return nil
		}
	}
	return fmt.Errorf("identity provider %q not found", name)
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
