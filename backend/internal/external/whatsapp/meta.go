package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// MetaConfig holds configuration for the Meta (Facebook) WhatsApp Cloud API.
type MetaConfig struct {
	PhoneNumberID string // WhatsApp Business Phone Number ID
	AccessToken   string // Permanent access token
	APIVersion    string // API version (default: v18.0)
}

// MetaProvider implements the WhatsApp Provider interface using Meta Cloud API.
type MetaProvider struct {
	config MetaConfig
	client *http.Client
	logger *slog.Logger
}

// NewMetaProvider creates a new Meta WhatsApp Cloud API provider.
func NewMetaProvider(config MetaConfig, logger *slog.Logger) *MetaProvider {
	if config.APIVersion == "" {
		config.APIVersion = "v18.0"
	}
	return &MetaProvider{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

func (p *MetaProvider) Name() string { return "meta" }

// Send dispatches a WhatsApp text message via the Meta Cloud API.
func (p *MetaProvider) Send(ctx context.Context, phone, message string) (*SendResult, error) {
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages",
		p.config.APIVersion, p.config.PhoneNumberID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                phone,
		"type":              "text",
		"text": map[string]string{
			"body": message,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.AccessToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return &SendResult{Provider: p.Name(), Success: false, Error: err.Error()}, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("WhatsApp API error: %d — %s", resp.StatusCode, string(respBody))
		p.logger.Error("Meta WhatsApp send failed",
			slog.String("phone", phone),
			slog.Int("status", resp.StatusCode),
			slog.String("response", string(respBody)),
		)
		return &SendResult{Provider: p.Name(), Success: false, Error: errMsg}, fmt.Errorf(errMsg)
	}

	// Parse response for message ID
	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	_ = json.Unmarshal(respBody, &result)

	messageID := ""
	if len(result.Messages) > 0 {
		messageID = result.Messages[0].ID
	}

	p.logger.Info("WhatsApp message sent via Meta",
		slog.String("phone", phone),
		slog.String("message_id", messageID),
	)

	return &SendResult{
		Provider:  p.Name(),
		MessageID: messageID,
		Success:   true,
	}, nil
}
