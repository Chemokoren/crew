package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OptimizeConfig holds configuration for the Optimize SMS provider.
type OptimizeConfig struct {
	ClientID           string `json:"client_id"`
	ClientSecret       string `json:"client_secret"`
	TokenURL           string `json:"token_url"`
	SMSURL             string `json:"sms_url"`
	SenderID           string `json:"sender_id"`
	CallbackURL        string `json:"callback_url"`
	TokenExpirySeconds int    `json:"token_expiry_seconds"` // Default 3600
}

// OptimizeProvider implements the SMS Provider interface using the Optimize SMS API.
// Ported from the Python SMSService reference implementation.
//
// Concurrency optimizations:
//   - Token is cached with RWMutex for many readers / single writer.
//   - singleflight semantics via tokenRefreshMu prevent thundering herd on token refresh.
//   - Automatic token refresh on 401 (stale cache) with a single retry.
type OptimizeProvider struct {
	cfg    OptimizeConfig
	client *http.Client
	logger *slog.Logger

	// Token cache (thread-safe)
	mu        sync.RWMutex
	token     string
	expiresAt time.Time

	// Prevents concurrent token refresh stampede
	tokenRefreshMu sync.Mutex
}

// NewOptimizeProvider creates a new Optimize SMS provider.
func NewOptimizeProvider(cfg OptimizeConfig, logger *slog.Logger) *OptimizeProvider {
	if cfg.TokenExpirySeconds <= 0 {
		cfg.TokenExpirySeconds = 3600
	}
	return &OptimizeProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (p *OptimizeProvider) Name() string { return "optimize" }

// getToken retrieves a cached token or requests a new one via OAuth2 client_credentials.
// Thread-safe with singleflight semantics to prevent thundering herd.
func (p *OptimizeProvider) getToken(ctx context.Context) (string, error) {
	// Fast path: read-lock to check cached token
	p.mu.RLock()
	if p.token != "" && time.Now().Before(p.expiresAt) {
		token := p.token
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	// Slow path: acquire refresh lock so only one goroutine refreshes
	return p.refreshToken(ctx)
}

// refreshToken acquires the refresh mutex and fetches a new token.
// If another goroutine already refreshed while we waited, uses the new cached value.
func (p *OptimizeProvider) refreshToken(ctx context.Context) (string, error) {
	p.tokenRefreshMu.Lock()
	defer p.tokenRefreshMu.Unlock()

	// Double-check: another goroutine may have refreshed while we waited
	p.mu.RLock()
	if p.token != "" && time.Now().Before(p.expiresAt) {
		token := p.token
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	p.logger.Info("requesting new Optimize SMS token")

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in response")
	}

	// Cache token with a safety margin (refresh 60s before actual expiry)
	ttl := time.Duration(p.cfg.TokenExpirySeconds) * time.Second
	if ttl > 60*time.Second {
		ttl -= 60 * time.Second
	}

	p.mu.Lock()
	p.token = tokenResp.AccessToken
	p.expiresAt = time.Now().Add(ttl)
	p.mu.Unlock()

	p.logger.Info("Optimize SMS token refreshed",
		slog.Duration("ttl", ttl),
	)

	return tokenResp.AccessToken, nil
}

// invalidateToken clears the cached token, forcing the next call to refresh.
func (p *OptimizeProvider) invalidateToken() {
	p.mu.Lock()
	p.token = ""
	p.expiresAt = time.Time{}
	p.mu.Unlock()
}

// Send dispatches an SMS to a single recipient.
// On 401 Unauthorized, it automatically refreshes the token and retries once.
func (p *OptimizeProvider) Send(ctx context.Context, phone, message string) (*SendResult, error) {
	p.logger.Info("sending SMS via Optimize", slog.String("phone", phone))

	result, err := p.doSend(ctx, phone, message)
	if err != nil && result != nil && result.Error != "" &&
		strings.Contains(result.Error, "401") {
		// Token may be stale — invalidate, refresh, and retry once
		p.logger.Warn("Optimize returned 401, refreshing token and retrying",
			slog.String("phone", phone),
		)
		p.invalidateToken()
		return p.doSend(ctx, phone, message)
	}
	return result, err
}

// doSend performs the actual SMS API call.
func (p *OptimizeProvider) doSend(ctx context.Context, phone, message string) (*SendResult, error) {
	token, err := p.getToken(ctx)
	if err != nil {
		return &SendResult{Provider: p.Name(), Success: false, Error: err.Error()}, err
	}

	// Build payload — sender_name is optional; omit if not configured
	// to avoid 400 "Sender name not available" errors for unregistered IDs.
	payload := map[string]string{
		"contact": phone,
		"message": message,
	}
	if p.cfg.SenderID != "" {
		payload["sender_name"] = p.cfg.SenderID
	}
	if p.cfg.CallbackURL != "" {
		payload["callback"] = p.cfg.CallbackURL
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.SMSURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build SMS request: %w", err)
	}
	req.Header.Set("Authorization", "JWT "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return &SendResult{Provider: p.Name(), Success: false, Error: err.Error()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return &SendResult{
			Provider: p.Name(),
			Success:  false,
			Error:    fmt.Sprintf("Optimize API returned %d", resp.StatusCode),
		}, fmt.Errorf("optimize SMS API returned %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("Optimize API returned %d: %s", resp.StatusCode, string(respBody))
		return &SendResult{
			Provider: p.Name(),
			Success:  false,
			Error:    errMsg,
		}, fmt.Errorf("%s", errMsg)
	}

	var smsResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&smsResp)

	msgID, _ := smsResp["message_id"].(string)
	p.logger.Info("SMS sent via Optimize",
		slog.String("phone", phone),
		slog.String("message_id", msgID),
	)

	return &SendResult{
		Provider:  p.Name(),
		MessageID: msgID,
		Success:   true,
	}, nil
}

// SendBulk dispatches the same message to multiple recipients.
// Uses individual Send calls — override if the provider supports batch API.
func (p *OptimizeProvider) SendBulk(ctx context.Context, phones []string, message string) ([]SendResult, error) {
	results := make([]SendResult, 0, len(phones))
	for _, phone := range phones {
		result, err := p.Send(ctx, phone, message)
		if err != nil {
			results = append(results, SendResult{Provider: p.Name(), Success: false, Error: err.Error()})
			continue
		}
		results = append(results, *result)
	}
	return results, nil
}
