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

// CrewMemberResponse represents a crew member response.
type CrewMemberResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
}

// EarningsSummary represents aggregated earnings data.
type EarningsSummary struct {
	TotalEarnedCents     int64  `json:"total_earned_cents"`
	TotalDeductionsCents int64  `json:"total_deductions_cents"`
	NetAmountCents       int64  `json:"net_amount_cents"`
	Currency             string `json:"currency"`
	AssignmentCount      int    `json:"assignment_count"`
}

// EarningItem represents a single earning record from the backend.
type EarningItem struct {
	ID              string `json:"id"`
	CrewMemberID    string `json:"crew_member_id"`
	AmountCents     int64  `json:"amount_cents"`
	DeductionsCents int64  `json:"deductions_cents"`
	EarningType     string `json:"earning_type"`
	Currency        string `json:"currency"`
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

// GetCrewMember looks up a crew member by ID.
func (c *Client) GetCrewMember(ctx context.Context, crewMemberID string) (*CrewMemberResponse, error) {
	resp, err := c.get(ctx, "/api/v1/crew/"+url.PathEscape(crewMemberID))
	if err != nil {
		return nil, err
	}

	var crew CrewMemberResponse
	if err := c.parseResponse(resp, &crew); err != nil {
		return nil, err
	}
	return &crew, nil
}

// SetPIN sets the transaction PIN for a user.
func (c *Client) SetPIN(ctx context.Context, phone, pin string) error {
	payload := map[string]string{"phone": phone, "pin": pin}
	resp, err := c.post(ctx, "/api/v1/auth/pin", payload)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// VerifyPIN verifies the transaction PIN for a user.
func (c *Client) VerifyPIN(ctx context.Context, phone, pin string) error {
	payload := map[string]string{"phone": phone, "pin": pin}
	resp, err := c.post(ctx, "/api/v1/auth/pin/verify", payload)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
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
// The backend's SummaryDashboard endpoint uses ?date=YYYY-MM-DD for daily summaries.
// For weekly/monthly, we query the earnings list with date_from/date_to and aggregate.
func (c *Client) GetEarningsSummary(ctx context.Context, crewMemberID, period string) (*EarningsSummary, error) {
	now := time.Now()
	var dateFrom string

	switch period {
	case "daily":
		dateFrom = now.Format("2006-01-02")
	case "weekly":
		dateFrom = now.AddDate(0, 0, -7).Format("2006-01-02")
	case "monthly":
		dateFrom = now.AddDate(0, -1, 0).Format("2006-01-02")
	default:
		dateFrom = now.Format("2006-01-02")
	}
	dateTo := now.Format("2006-01-02")

	// Use the earnings list endpoint with date filters and aggregate client-side.
	path := fmt.Sprintf("/api/v1/earnings?crew_member_id=%s&date_from=%s&date_to=%s&per_page=100",
		crewMemberID, dateFrom, dateTo)

	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	var earnings []EarningItem
	if err := c.parseListResponse(resp, &earnings); err != nil {
		// If no earnings found, return zero summary
		return &EarningsSummary{
			TotalEarnedCents: 0,
			Currency:         "KES",
			AssignmentCount:  0,
		}, nil
	}

	// Aggregate earnings
	summary := &EarningsSummary{Currency: "KES"}
	for _, e := range earnings {
		summary.TotalEarnedCents += e.AmountCents
		summary.TotalDeductionsCents += e.DeductionsCents
		summary.AssignmentCount++
	}
	summary.NetAmountCents = summary.TotalEarnedCents - summary.TotalDeductionsCents

	return summary, nil
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
// USSD withdrawals always go to the crew member's own phone via M-Pesa B2C.
func (c *Client) InitiateWithdrawal(ctx context.Context, crewMemberID string, amountCents int64, phone string) (*WithdrawalResult, error) {
	body := map[string]interface{}{
		"crew_member_id":  crewMemberID,
		"amount_cents":    amountCents,
		"channel":         "MOMO_B2C",
		"recipient_name":  "USSD Withdrawal",
		"recipient_phone": phone,
	}

	idempotencyKey := fmt.Sprintf("ussd-wd-%s-%d", crewMemberID, time.Now().UnixMilli())

	resp, err := c.postWithHeaders(ctx, fmt.Sprintf("/api/v1/wallets/%s/payout", crewMemberID), body, map[string]string{
		"Idempotency-Key": idempotencyKey,
	})
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

// LoanTierResponse represents a crew member's qualified loan tier.
type LoanTierResponse struct {
	Score        int     `json:"score"`
	Grade        string  `json:"grade"`
	MaxLoanKES   float64 `json:"max_loan_kes"`
	InterestRate float64 `json:"interest_rate"`
	MaxTenureDays int    `json:"max_tenure_days"`
	CooldownDays int     `json:"cooldown_days"`
	Description  string  `json:"description"`
}

// GetLoanTier retrieves the crew member's qualified loan tier.
func (c *Client) GetLoanTier(ctx context.Context, crewMemberID string) (*LoanTierResponse, error) {
	resp, err := c.get(ctx, fmt.Sprintf("/api/v1/loans/tier/%s", crewMemberID))
	if err != nil {
		return nil, err
	}

	var tier LoanTierResponse
	if err := c.parseResponse(resp, &tier); err != nil {
		return nil, err
	}
	return &tier, nil
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
	return c.postWithHeaders(ctx, path, body, nil)
}

func (c *Client) postWithHeaders(ctx context.Context, path string, body interface{}, extraHeaders map[string]string) (*http.Response, error) {
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
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

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
