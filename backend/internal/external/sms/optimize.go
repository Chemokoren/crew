package sms

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
)

// OptimizeConfig holds configuration for the Optimize SMS provider.
type OptimizeConfig struct {
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	TokenURL       string `json:"token_url"`
	SMSURL         string `json:"sms_url"`
	SenderID       string `json:"sender_id"`
	CallbackURL    string `json:"callback_url"`
	TokenExpirySeconds int `json:"token_expiry_seconds"` // Default 3600
}

// OptimizeProvider implements the SMS Provider interface using the Optimize SMS API.
// Ported from the Python SMSService reference implementation.
type OptimizeProvider struct {
	cfg    OptimizeConfig
	client *http.Client
	logger *slog.Logger

	// Token cache (thread-safe)
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
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
func (p *OptimizeProvider) getToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	if p.token != "" && time.Now().Before(p.expiresAt) {
		token := p.token
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	// Request new token
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
		return "", fmt.Errorf("token request returned %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	// Cache token
	p.mu.Lock()
	p.token = tokenResp.AccessToken
	p.expiresAt = time.Now().Add(time.Duration(p.cfg.TokenExpirySeconds) * time.Second)
	p.mu.Unlock()

	return tokenResp.AccessToken, nil
}

func (p *OptimizeProvider) Send(ctx context.Context, phone, message string) (*SendResult, error) {
	p.logger.Info("sending SMS via Optimize", slog.String("phone", phone))

	token, err := p.getToken(ctx)
	if err != nil {
		return &SendResult{Provider: p.Name(), Success: false, Error: err.Error()}, err
	}

	payload := map[string]string{
		"contact":     phone,
		"message":     message,
		"sender_name": p.cfg.SenderID,
		"callback":    p.cfg.CallbackURL,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.SMSURL, strings.NewReader(string(body)))
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return &SendResult{
			Provider: p.Name(),
			Success:  false,
			Error:    fmt.Sprintf("Optimize API returned %d", resp.StatusCode),
		}, fmt.Errorf("optimize SMS API returned %d", resp.StatusCode)
	}

	var smsResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&smsResp)

	p.logger.Info("SMS sent via Optimize", slog.String("phone", phone))

	return &SendResult{
		Provider: p.Name(),
		Success:  true,
	}, nil
}

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
