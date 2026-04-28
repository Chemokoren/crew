package crb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Provider defines the interface for Credit Reference Bureau integration.
// Both TransUnion Kenya and Metropol implement this interface.
type Provider interface {
	// Name returns the CRB provider name.
	Name() string

	// GetCreditReport retrieves a credit report for a person by national ID.
	GetCreditReport(ctx context.Context, nationalID string) (*CreditReport, error)

	// SubmitLoanData reports loan performance data to the CRB.
	SubmitLoanData(ctx context.Context, data LoanReportData) error
}

// CreditReport represents a CRB credit report.
type CreditReport struct {
	NationalID      string            `json:"national_id"`
	FullName        string            `json:"full_name"`
	CRBScore        int               `json:"crb_score"`         // 0-900 range typically
	TotalLoans      int               `json:"total_loans"`
	ActiveLoans     int               `json:"active_loans"`
	DefaultedLoans  int               `json:"defaulted_loans"`
	TotalExposure   int64             `json:"total_exposure_kes"`
	HighestDefault  int64             `json:"highest_default_kes"`
	CreditHistory   []CreditHistoryItem `json:"credit_history"`
	LastQueried     time.Time         `json:"last_queried"`
	ProviderName    string            `json:"provider_name"`
	RawResponse     json.RawMessage   `json:"raw_response,omitempty"`
}

// CreditHistoryItem represents a single loan from CRB history.
type CreditHistoryItem struct {
	LenderName    string    `json:"lender_name"`
	AmountKES     int64     `json:"amount_kes"`
	Status        string    `json:"status"` // "PERFORMING", "DEFAULTED", "CLOSED"
	DisbursedAt   time.Time `json:"disbursed_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	DaysPastDue   int       `json:"days_past_due"`
}

// LoanReportData is what we submit to CRBs about our loans.
type LoanReportData struct {
	CrewMemberID   uuid.UUID `json:"crew_member_id"`
	NationalID     string    `json:"national_id"`
	LoanID         uuid.UUID `json:"loan_id"`
	AmountCents    int64     `json:"amount_cents"`
	Status         string    `json:"status"`
	DisbursedAt    time.Time `json:"disbursed_at"`
	DueAt          time.Time `json:"due_at"`
	RepaidAt       *time.Time `json:"repaid_at,omitempty"`
	DaysPastDue    int       `json:"days_past_due"`
}

// --- TransUnion Kenya Provider ---

// TransUnionConfig holds the config for TransUnion Kenya CRB API.
type TransUnionConfig struct {
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	ClientID     string `json:"client_id"`
	InfinityCode string `json:"infinity_code"`
}

type transUnionProvider struct {
	config TransUnionConfig
	client *http.Client
	logger *slog.Logger
}

// NewTransUnionProvider creates a TransUnion Kenya CRB provider.
func NewTransUnionProvider(config TransUnionConfig, logger *slog.Logger) Provider {
	return &transUnionProvider{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

func (p *transUnionProvider) Name() string { return "transunion" }

func (p *transUnionProvider) GetCreditReport(ctx context.Context, nationalID string) (*CreditReport, error) {
	url := fmt.Sprintf("%s/api/v2/credit-report/%s", p.config.BaseURL, nationalID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("transunion: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("X-Client-ID", p.config.ClientID)
	req.Header.Set("X-Infinity-Code", p.config.InfinityCode)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("transunion: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		p.logger.Error("transunion: CRB query failed",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)
		return nil, fmt.Errorf("transunion: HTTP %d", resp.StatusCode)
	}

	var report CreditReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, fmt.Errorf("transunion: parse response: %w", err)
	}
	report.ProviderName = "transunion"
	report.LastQueried = time.Now()
	report.RawResponse = body
	return &report, nil
}

func (p *transUnionProvider) SubmitLoanData(ctx context.Context, data LoanReportData) error {
	p.logger.Info("transunion: loan data submission",
		slog.String("loan_id", data.LoanID.String()),
		slog.String("status", data.Status),
	)
	// TODO: Implement actual submission when TransUnion data submission API is available
	return nil
}

// --- Metropol Kenya Provider ---

// MetropolConfig holds the config for Metropol Kenya CRB API.
type MetropolConfig struct {
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	PublicKey    string `json:"public_key"`
}

type metropolProvider struct {
	config MetropolConfig
	client *http.Client
	logger *slog.Logger
}

// NewMetropolProvider creates a Metropol Kenya CRB provider.
func NewMetropolProvider(config MetropolConfig, logger *slog.Logger) Provider {
	return &metropolProvider{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

func (p *metropolProvider) Name() string { return "metropol" }

func (p *metropolProvider) GetCreditReport(ctx context.Context, nationalID string) (*CreditReport, error) {
	url := fmt.Sprintf("%s/api/identity/verify/%s", p.config.BaseURL, nationalID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("metropol: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("metropol: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metropol: HTTP %d", resp.StatusCode)
	}

	var report CreditReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, fmt.Errorf("metropol: parse response: %w", err)
	}
	report.ProviderName = "metropol"
	report.LastQueried = time.Now()
	report.RawResponse = body
	return &report, nil
}

func (p *metropolProvider) SubmitLoanData(ctx context.Context, data LoanReportData) error {
	p.logger.Info("metropol: loan data submission",
		slog.String("loan_id", data.LoanID.String()),
		slog.String("status", data.Status),
	)
	// TODO: Implement actual submission when Metropol data submission API is available
	return nil
}

// --- CRB Manager (Failover) ---

// Manager wraps multiple CRB providers with failover support.
type Manager struct {
	providers []Provider
	logger    *slog.Logger
}

// NewManager creates a CRB manager with failover across providers.
func NewManager(logger *slog.Logger, providers ...Provider) *Manager {
	return &Manager{
		providers: providers,
		logger:    logger,
	}
}

// GetCreditReport tries each provider in order until one succeeds.
func (m *Manager) GetCreditReport(ctx context.Context, nationalID string) (*CreditReport, error) {
	var lastErr error
	for _, p := range m.providers {
		report, err := p.GetCreditReport(ctx, nationalID)
		if err != nil {
			m.logger.Warn("CRB provider failed, trying next",
				slog.String("provider", p.Name()),
				slog.String("error", err.Error()),
			)
			lastErr = err
			continue
		}
		return report, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all CRB providers failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no CRB providers configured")
}

// SubmitLoanData submits to all configured providers (best-effort).
func (m *Manager) SubmitLoanData(ctx context.Context, data LoanReportData) {
	for _, p := range m.providers {
		if err := p.SubmitLoanData(ctx, data); err != nil {
			m.logger.Error("CRB loan submission failed",
				slog.String("provider", p.Name()),
				slog.String("error", err.Error()),
			)
		}
	}
}
