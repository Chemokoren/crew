package perpay

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kibsoft/amy-mis/internal/external/payroll"
)

// PerPayConfig holds configuration for the PerPay 3rd-Party Payroll Adapter API.
type PerPayConfig struct {
	BaseURL      string `json:"base_url"`      // https://api.netcom.app or staging
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// PerPayProvider implements the payroll.Provider interface using the PerPay API.
// Based on the openapi.yaml spec (3rd-Party Payroll Adapter API v1.0.0).
type PerPayProvider struct {
	cfg    PerPayConfig
	client *http.Client
	logger *slog.Logger

	// Token cache (JWT, 15min TTL)
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

// NewPerPayProvider creates a new PerPay payroll provider.
func NewPerPayProvider(cfg PerPayConfig, logger *slog.Logger) *PerPayProvider {
	return &PerPayProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (p *PerPayProvider) Name() string { return "perpay" }

// authenticate retrieves or refreshes the PerPay JWT token.
// POST /auth/issue with x_client_id and x_client_secret as JSON body.
func (p *PerPayProvider) authenticate(ctx context.Context) (string, error) {
	p.mu.RLock()
	if p.token != "" && time.Now().Before(p.expiresAt) {
		token := p.token
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	payload := map[string]string{
		"x_client_id":     p.cfg.ClientID,
		"x_client_secret": p.cfg.ClientSecret,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.BaseURL+"/auth/issue", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("build perpay auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("perpay auth failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("perpay auth returned %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"` // 900 seconds (15 min)
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode perpay auth response: %w", err)
	}

	p.mu.Lock()
	p.token = tokenResp.AccessToken
	p.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	p.mu.Unlock()

	return tokenResp.AccessToken, nil
}

// SubmitPayroll submits a payroll request to the PerPay async processing pipeline.
// POST /payroll/v1/payroll/submit → returns 202 Accepted with correlation_id.
func (p *PerPayProvider) SubmitPayroll(ctx context.Context, req payroll.SubmitRequest) (*payroll.SubmitResult, error) {
	p.logger.Info("submitting payroll to PerPay",
		slog.String("employee_id", req.EmployeeID),
		slog.String("period", req.PayPeriodStart+" → "+req.PayPeriodEnd),
	)

	token, err := p.authenticate(ctx)
	if err != nil {
		return nil, fmt.Errorf("perpay auth: %w", err)
	}

	payload := map[string]interface{}{
		"employee_id":           req.EmployeeID,
		"full_name":             req.FullName,
		"employee_pin":          req.EmployeePIN,
		"currency":              req.Currency,
		"pay_period_start_date": req.PayPeriodStart,
		"pay_period_end_date":   req.PayPeriodEnd,
		"pay_components":        req.PayComponents,
		"deductions":            req.Deductions,
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.BaseURL+"/payroll/v1/payroll/submit", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("build perpay request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	if req.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	}
	if req.CorrelationID != "" {
		httpReq.Header.Set("X-Correlation-ID", req.CorrelationID)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("perpay submit failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle idempotency replay (409)
	if resp.StatusCode == http.StatusConflict {
		var cached struct {
			CorrelationID string `json:"correlation_id"`
			Status        string `json:"status"`
			StatusURL     string `json:"status_url"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&cached)
		return &payroll.SubmitResult{
			Provider:      p.Name(),
			CorrelationID: cached.CorrelationID,
			Status:        cached.Status,
			StatusURL:     cached.StatusURL,
		}, nil
	}

	if resp.StatusCode != http.StatusAccepted {
		var errResp struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("perpay submit error %d: %s - %s",
			resp.StatusCode, errResp.ErrorCode, errResp.Message)
	}

	var submitResp struct {
		CorrelationID string `json:"correlation_id"`
		Status        string `json:"status"`
		StatusURL     string `json:"status_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return nil, fmt.Errorf("decode perpay response: %w", err)
	}

	p.logger.Info("payroll submitted to PerPay",
		slog.String("correlation_id", submitResp.CorrelationID),
	)

	return &payroll.SubmitResult{
		Provider:      p.Name(),
		CorrelationID: submitResp.CorrelationID,
		Status:        submitResp.Status,
		StatusURL:     submitResp.StatusURL,
	}, nil
}

// GetStatus queries the processing status of a previously submitted payroll request.
// GET /payroll/v1/requests/{correlation_id}/status
func (p *PerPayProvider) GetStatus(ctx context.Context, correlationID string) (*payroll.StatusResult, error) {
	token, err := p.authenticate(ctx)
	if err != nil {
		return nil, fmt.Errorf("perpay auth: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.cfg.BaseURL+"/payroll/v1/requests/"+correlationID+"/status", nil)
	if err != nil {
		return nil, fmt.Errorf("build status request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("perpay status request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("correlation ID %s not found", correlationID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("perpay status returned %d", resp.StatusCode)
	}

	var statusResp struct {
		CorrelationID string `json:"correlation_id"`
		Status        string `json:"status"`
		CurrentStep   string `json:"current_step"`
		ResultSummary *struct {
			GrossPay        float64 `json:"gross_pay"`
			NetPay          float64 `json:"net_pay"`
			TotalDeductions float64 `json:"total_deductions"`
		} `json:"result_summary"`
		Error *struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("decode status response: %w", err)
	}

	result := &payroll.StatusResult{
		Provider:      p.Name(),
		CorrelationID: statusResp.CorrelationID,
		Status:        statusResp.Status,
		CurrentStep:   statusResp.CurrentStep,
	}
	if statusResp.ResultSummary != nil {
		result.GrossPay = statusResp.ResultSummary.GrossPay
		result.NetPay = statusResp.ResultSummary.NetPay
		result.Deductions = statusResp.ResultSummary.TotalDeductions
	}
	if statusResp.Error != nil {
		result.ErrorCode = statusResp.Error.ErrorCode
		result.ErrorMessage = statusResp.Error.Message
	}

	return result, nil
}
