// Package external provides shared types and utilities for all external integrations.
// Every integration in AMY MIS follows the Strategy pattern:
//
//   1. A Provider interface defines the vendor contract
//   2. A Manager orchestrates providers with primary/fallback + enable/disable
//   3. Each vendor implements the Provider interface
//   4. Providers are registered via configuration flags
//
// This allows zero-downtime vendor switching from environment variables alone.
package external

// ProviderStatus describes the enabled/disabled state of an integration provider.
type ProviderStatus struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Primary bool   `json:"primary"`
}
