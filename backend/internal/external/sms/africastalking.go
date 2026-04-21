package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// AfricasTalkingConfig holds configuration for Africa's Talking SMS provider.
type AfricasTalkingConfig struct {
	APIKey    string `json:"api_key"`
	Username  string `json:"username"`
	Shortcode string `json:"shortcode"`
	BaseURL   string `json:"base_url"` // https://api.africastalking.com/version1 or sandbox
}

// AfricasTalkingProvider implements the SMS Provider interface using Africa's Talking API.
type AfricasTalkingProvider struct {
	cfg    AfricasTalkingConfig
	client *http.Client
	logger *slog.Logger
}

// NewAfricasTalkingProvider creates a new Africa's Talking SMS provider.
func NewAfricasTalkingProvider(cfg AfricasTalkingConfig, logger *slog.Logger) *AfricasTalkingProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.africastalking.com/version1"
	}
	return &AfricasTalkingProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (p *AfricasTalkingProvider) Name() string { return "africastalking" }

func (p *AfricasTalkingProvider) Send(ctx context.Context, phone, message string) (*SendResult, error) {
	p.logger.Info("sending SMS via Africa's Talking", slog.String("phone", phone))

	payload := fmt.Sprintf("username=%s&to=%s&message=%s",
		p.cfg.Username, phone, message,
	)
	if p.cfg.Shortcode != "" {
		payload += "&from=" + p.cfg.Shortcode
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.BaseURL+"/messaging", strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build AT request: %w", err)
	}
	req.Header.Set("apiKey", p.cfg.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return &SendResult{Provider: p.Name(), Success: false, Error: err.Error()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return &SendResult{
			Provider: p.Name(),
			Success:  false,
			Error:    fmt.Sprintf("AT API returned %d", resp.StatusCode),
		}, fmt.Errorf("africastalking API returned %d", resp.StatusCode)
	}

	var atResp struct {
		SMSMessageData struct {
			Message    string `json:"Message"`
			Recipients []struct {
				StatusCode int    `json:"statusCode"`
				Number     string `json:"number"`
				Status     string `json:"status"`
				MessageID  string `json:"messageId"`
			} `json:"Recipients"`
		} `json:"SMSMessageData"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&atResp); err != nil {
		return nil, fmt.Errorf("decode AT response: %w", err)
	}

	if len(atResp.SMSMessageData.Recipients) > 0 {
		r := atResp.SMSMessageData.Recipients[0]
		if r.StatusCode == 101 { // 101 = Sent to network
			return &SendResult{
				Provider:  p.Name(),
				MessageID: r.MessageID,
				Success:   true,
			}, nil
		}
		return &SendResult{
			Provider: p.Name(),
			Success:  false,
			Error:    fmt.Sprintf("AT status: %s (code %d)", r.Status, r.StatusCode),
		}, fmt.Errorf("AT send failed: %s", r.Status)
	}

	return &SendResult{Provider: p.Name(), Success: false, Error: "no recipients in response"}, fmt.Errorf("no recipients in AT response")
}

func (p *AfricasTalkingProvider) SendBulk(ctx context.Context, phones []string, message string) ([]SendResult, error) {
	// Africa's Talking supports comma-separated recipients in a single call
	p.logger.Info("sending bulk SMS via Africa's Talking", slog.Int("count", len(phones)))

	allPhones := strings.Join(phones, ",")

	payload := fmt.Sprintf("username=%s&to=%s&message=%s",
		p.cfg.Username, allPhones, message,
	)
	if p.cfg.Shortcode != "" {
		payload += "&from=" + p.cfg.Shortcode
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.BaseURL+"/messaging", strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build AT bulk request: %w", err)
	}
	req.Header.Set("apiKey", p.cfg.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AT bulk request failed: %w", err)
	}
	defer resp.Body.Close()

	var atResp struct {
		SMSMessageData struct {
			Recipients []struct {
				StatusCode int    `json:"statusCode"`
				Number     string `json:"number"`
				Status     string `json:"status"`
				MessageID  string `json:"messageId"`
			} `json:"Recipients"`
		} `json:"SMSMessageData"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&atResp)

	results := make([]SendResult, 0, len(atResp.SMSMessageData.Recipients))
	for _, r := range atResp.SMSMessageData.Recipients {
		results = append(results, SendResult{
			Provider:  p.Name(),
			MessageID: r.MessageID,
			Success:   r.StatusCode == 101,
			Error: func() string {
				if r.StatusCode != 101 {
					return r.Status
				}
				return ""
			}(),
		})
	}

	return results, nil
}
