package jambopay

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kibsoft/amy-mis/internal/external/payment"
)

// JamboPayConfig holds configuration for the JamboPay v2 API.
type JamboPayConfig struct {
	BaseURL      string `json:"base_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// JamboPayProvider implements the payment.Provider interface using JamboPay v2 API.
// Supports M-Pesa B2C, bank transfer, paybill, and till payouts.
type JamboPayProvider struct {
	cfg    JamboPayConfig
	client *http.Client
	logger *slog.Logger

	// Token cache
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

// NewJamboPayProvider creates a new JamboPay payment provider.
func NewJamboPayProvider(cfg JamboPayConfig, logger *slog.Logger) *JamboPayProvider {
	return &JamboPayProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (p *JamboPayProvider) Name() string { return "jambopay" }

// authenticate retrieves or refreshes the JamboPay OAuth2 token.
// JamboPay uses POST /auth/token with x-www-form-urlencoded client_credentials.
func (p *JamboPayProvider) authenticate(ctx context.Context) (string, error) {
	p.mu.RLock()
	if p.token != "" && time.Now().Before(p.expiresAt) {
		token := p.token
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	form := url.Values{
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"grant_type":    {"client_credentials"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.BaseURL+"/auth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("jambopay auth failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("jambopay auth returned %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode auth response: %w", err)
	}

	p.mu.Lock()
	p.token = tokenResp.AccessToken
	// Expire 60s early to avoid edge-case clock drift
	p.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	p.mu.Unlock()

	return tokenResp.AccessToken, nil
}

// doAuthenticatedRequest sends an authenticated request to JamboPay.
func (p *JamboPayProvider) doAuthenticatedRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	token, err := p.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var reqBody *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = strings.NewReader(string(b))
	} else {
		reqBody = strings.NewReader("")
	}

	req, err := http.NewRequestWithContext(ctx, method, p.cfg.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return p.client.Do(req)
}

// InitiatePayout initiates a JamboPay payout (M-Pesa B2C, bank, paybill, or till).
// POST /payout
func (p *JamboPayProvider) InitiatePayout(ctx context.Context, req payment.PayoutRequest) (*payment.PayoutResult, error) {
	p.logger.Info("initiating JamboPay payout",
		slog.String("channel", string(req.Channel)),
		slog.Int64("amount_cents", req.AmountCents),
		slog.String("order_id", req.OrderID),
	)

	// Convert cents to string amount (JamboPay expects string)
	amount := fmt.Sprintf("%.2f", float64(req.AmountCents)/100)

	// Build payTo based on channel
	payTo := map[string]string{}
	switch req.Channel {
	case payment.ChannelMobile:
		payTo["accountRef"] = req.RecipientName
		payTo["accountNumber"] = req.RecipientPhone
	case payment.ChannelBank:
		payTo["accountNumber"] = req.BankAccount
		payTo["accountRef"] = req.RecipientName
		payTo["bankCode"] = req.BankCode
	case payment.ChannelPaybill:
		payTo["accountNumber"] = req.PaybillNumber
		payTo["accountRef"] = req.PaybillRef
	default:
		return nil, fmt.Errorf("unsupported payout channel: %s", req.Channel)
	}

	payload := map[string]interface{}{
		"amount":      amount,
		"accountFrom": req.AccountFrom,
		"orderId":     req.OrderID,
		"provider":    string(req.Channel),
		"payTo":       payTo,
		"callBackUrl": req.CallbackURL,
		"narration":   req.Narration,
	}

	resp, err := p.doAuthenticatedRequest(ctx, http.MethodPost, "/payout", payload)
	if err != nil {
		return nil, fmt.Errorf("payout request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Status  int      `json:"status"`
			Message []string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("jambopay payout error %d: %v", resp.StatusCode, errResp.Message)
	}

	var payoutResp struct {
		Ref     string `json:"ref"`
		OrderID string `json:"orderId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payoutResp); err != nil {
		return nil, fmt.Errorf("decode payout response: %w", err)
	}

	return &payment.PayoutResult{
		Provider:    p.Name(),
		Reference:   payoutResp.Ref,
		OrderID:     payoutResp.OrderID,
		Status:      "pending_otp",
		RequiresOTP: true,
	}, nil
}

// VerifyPayout authorizes a pending payout with OTP.
// POST /payout/authorize
func (p *JamboPayProvider) VerifyPayout(ctx context.Context, req payment.PayoutVerifyRequest) (*payment.PayoutResult, error) {
	p.logger.Info("verifying JamboPay payout", slog.String("ref", req.Reference))

	payload := map[string]string{
		"ref": req.Reference,
		"otp": req.OTP,
	}

	resp, err := p.doAuthenticatedRequest(ctx, http.MethodPost, "/payout/authorize", payload)
	if err != nil {
		return nil, fmt.Errorf("verify payout failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("jambopay verify returned %d", resp.StatusCode)
	}

	return &payment.PayoutResult{
		Provider:  p.Name(),
		Reference: req.Reference,
		Status:    "completed",
	}, nil
}

// CheckBalance retrieves wallet account balance.
// GET /wallet/account?accountNo={accountNo}
func (p *JamboPayProvider) CheckBalance(ctx context.Context, accountNo string) (*payment.BalanceResult, error) {
	p.logger.Info("checking JamboPay balance", slog.String("account", accountNo))

	resp, err := p.doAuthenticatedRequest(ctx, http.MethodGet,
		"/wallet/account?accountNo="+accountNo, nil)
	if err != nil {
		return nil, fmt.Errorf("check balance failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jambopay balance returned %d", resp.StatusCode)
	}

	var acctResp struct {
		CurrentBalance float64 `json:"currentBalance"`
		Currency       string  `json:"currency"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&acctResp); err != nil {
		return nil, fmt.Errorf("decode balance response: %w", err)
	}

	return &payment.BalanceResult{
		Provider: p.Name(),
		Balance:  int64(acctResp.CurrentBalance * 100), // Convert to cents
		Currency: acctResp.Currency,
	}, nil
}
