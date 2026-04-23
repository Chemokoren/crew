// Package backend provides an HTTP client for communicating with the AMY MIS backend API.
// All backend calls are non-blocking with strict timeouts to meet USSD latency requirements.
//
// Design:
//   - Context-based timeouts (< 1.5s per call)
//   - Connection pooling for high throughput
//   - Structured error types for circuit breaker integration
//   - JSON request/response serialization
package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Client communicates with the AMY MIS backend API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new backend API client with optimized HTTP transport.
func NewClient(baseURL, apiKey string, timeout time.Duration, logger *slog.Logger) *Client {
	transport := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     200,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   true,
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		logger: logger,
	}
}

// --- Response types (mirroring backend models) ---

// WalletResponse represents a wallet balance response.
type WalletResponse struct {
	ID           string `json:"id"`
	CrewMemberID string `json:"crew_member_id"`
	BalanceCents int64  `json:"balance_cents"`
	Currency     string `json:"currency"`
}

// UserResponse represents a user lookup response.
type UserResponse struct {
	ID           string `json:"id"`
	Phone        string `json:"phone"`
	CrewMemberID string `json:"crew_member_id"`
	IsActive     bool   `json:"is_active"`
}

// EarningsSummary represents aggregated earnings data.
type EarningsSummary struct {
	TotalEarnedCents     int64  `json:"total_earned_cents"`
	TotalDeductionsCents int64  `json:"total_deductions_cents"`
	NetAmountCents       int64  `json:"net_amount_cents"`
	Currency             string `json:"currency"`
	AssignmentCount      int    `json:"assignment_count"`
}

// TransactionResponse represents a wallet transaction.
type TransactionResponse struct {
	ID              string    `json:"id"`
	TransactionType string    `json:"transaction_type"`
	Category        string    `json:"category"`
	AmountCents     int64     `json:"amount_cents"`
	Currency        string    `json:"currency"`
	Reference       string    `json:"reference"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// WithdrawalResult represents the result of a withdrawal initiation.
type WithdrawalResult struct {
	TransactionID string `json:"transaction_id"`
	Reference     string `json:"reference"`
	Status        string `json:"status"`
}

// CreditScoreResponse represents a credit score.
type CreditScoreResponse struct {
	Score            int       `json:"score"`
	LastCalculatedAt time.Time `json:"last_calculated_at"`
}

// LoanResponse represents a loan application.
type LoanResponse struct {
	ID                   string `json:"id"`
	AmountRequestedCents int64  `json:"amount_requested_cents"`
	AmountApprovedCents  int64  `json:"amount_approved_cents"`
	Currency             string `json:"currency"`
	Status               string `json:"status"`
	TenureDays           int    `json:"tenure_days"`
}

// RegisterRequest holds the data for crew registration via USSD.
// Maps to the backend's dto.RegisterRequest fields.
type RegisterRequest struct {
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	NationalID string `json:"national_id"`
	Role       string `json:"role"`      // SystemRole: "CREW"
	CrewRole   string `json:"crew_role"` // CrewRole: "DRIVER", "CONDUCTOR", "RIDER"
}

// RegisterResponse holds the result of a crew registration.
type RegisterResponse struct {
	UserID       string `json:"user_id"`
	CrewMemberID string `json:"crew_member_id"`
	CrewID       string `json:"crew_id"`
}

// --- API methods ---

// GetUserByPhone looks up a user by their phone number (MSISDN).
func (c *Client) GetUserByPhone(ctx context.Context, phone string) (*UserResponse, error) {
	resp, err := c.get(ctx, "/api/v1/auth/lookup?phone="+url.QueryEscape(phone))
	if err != nil {
		return nil, err
	}

	var user UserResponse
	if err := c.parseResponse(resp, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetWalletBalance retrieves the wallet balance for a crew member.
func (c *Client) GetWalletBalance(ctx context.Context, crewMemberID string) (*WalletResponse, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/v1/wallets/%s", crewMemberID))
	if err != nil {
		return nil, err
	}

	var wallet WalletResponse
	if err := c.parseResponse(resp, &wallet); err != nil {
		return nil, err
	}
	return &wallet, nil
}

// GetEarningsSummary retrieves earnings summary for a period.
func (c *Client) GetEarningsSummary(ctx context.Context, crewMemberID, period string) (*EarningsSummary, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/v1/earnings/summary/%s?period=%s", crewMemberID, period))
	if err != nil {
		return nil, err
	}

	var summary EarningsSummary
	if err := c.parseResponse(resp, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

// GetLastTransaction retrieves the most recent wallet transaction.
func (c *Client) GetLastTransaction(ctx context.Context, crewMemberID string) (*TransactionResponse, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/v1/wallets/%s/transactions?per_page=1", crewMemberID))
	if err != nil {
		return nil, err
	}

	var txList []TransactionResponse
	if err := c.parseListResponse(resp, &txList); err != nil {
		return nil, err
	}
	if len(txList) == 0 {
		return nil, nil
	}
	return &txList[0], nil
}

// InitiateWithdrawal initiates a payout from the crew member's wallet.
func (c *Client) InitiateWithdrawal(ctx context.Context, crewMemberID string, amountCents int64) (*WithdrawalResult, error) {
	body := map[string]interface{}{
		"crew_member_id": crewMemberID,
		"amount_cents":   amountCents,
	}

	resp, err := c.post(ctx, fmt.Sprintf("/api/v1/wallets/%s/payout", crewMemberID), body)
	if err != nil {
		return nil, err
	}

	var result WithdrawalResult
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCreditScore retrieves a crew member's credit score.
func (c *Client) GetCreditScore(ctx context.Context, crewMemberID string) (*CreditScoreResponse, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/v1/credit/%s", crewMemberID))
	if err != nil {
		return nil, err
	}

	var score CreditScoreResponse
	if err := c.parseResponse(resp, &score); err != nil {
		return nil, err
	}
	return &score, nil
}

// GetLoans retrieves active loans for a crew member.
func (c *Client) GetLoans(ctx context.Context, crewMemberID string) ([]LoanResponse, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/v1/loans?crew_member_id=%s&per_page=5", crewMemberID))
	if err != nil {
		return nil, err
	}

	var loans []LoanResponse
	if err := c.parseListResponse(resp, &loans); err != nil {
		return nil, err
	}
	return loans, nil
}

// ApplyForLoan submits a new loan application.
func (c *Client) ApplyForLoan(ctx context.Context, crewMemberID string, amountCents int64, tenureDays int) (*LoanResponse, error) {
	body := map[string]interface{}{
		"crew_member_id": crewMemberID,
		"amount_cents":   amountCents,
		"tenure_days":    tenureDays,
	}

	resp, err := c.post(ctx, "/api/v1/loans", body)
	if err != nil {
		return nil, err
	}

	var loan LoanResponse
	if err := c.parseResponse(resp, &loan); err != nil {
		return nil, err
	}
	return &loan, nil
}

// RegisterCrew registers a new crew member via USSD self-registration.
func (c *Client) RegisterCrew(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	resp, err := c.post(ctx, "/api/v1/auth/register", req)
	if err != nil {
		return nil, err
	}

	var result RegisterResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// --- HTTP helpers ---

func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("backend GET %s: %w", path, err)
	}
	return resp, nil
}

func (c *Client) post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("backend POST %s: %w", path, err)
	}
	return resp, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "AMY-MIS-USSD/1.0")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// apiResponse wraps the standard backend response envelope.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *apiError       `json:"error,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *Client) parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiResp apiResponse
		if json.Unmarshal(body, &apiResp) == nil && apiResp.Error != nil {
			return fmt.Errorf("backend error [%d]: %s - %s", resp.StatusCode, apiResp.Error.Code, apiResp.Error.Message)
		}
		return fmt.Errorf("backend error [%d]: %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if !apiResp.Success {
		if apiResp.Error != nil {
			return fmt.Errorf("backend: %s", apiResp.Error.Message)
		}
		return fmt.Errorf("backend returned unsuccessful response")
	}

	if target != nil && apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, target); err != nil {
			return fmt.Errorf("unmarshal data: %w", err)
		}
	}
	return nil
}

func (c *Client) parseListResponse(resp *http.Response, target interface{}) error {
	return c.parseResponse(resp, target)
}
