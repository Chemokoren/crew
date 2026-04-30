// Package messaging provides a unified multi-channel messaging engine.
// It orchestrates email, SMS, and WhatsApp providers through a single API.
// This engine is reusable across ALL messaging use cases: OTP, support,
// notifications, system alerts — avoiding code duplication.
package messaging

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kibsoft/amy-mis/internal/external/email"
	"github.com/kibsoft/amy-mis/internal/external/sms"
	"github.com/kibsoft/amy-mis/internal/external/whatsapp"
)

// Channel represents a messaging delivery channel.
type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

// DeliveryResult holds the outcome of a message delivery attempt.
type DeliveryResult struct {
	Channel   Channel `json:"channel"`
	Provider  string  `json:"provider"`
	MessageID string  `json:"message_id,omitempty"`
	Success   bool    `json:"success"`
	Error     string  `json:"error,omitempty"`
}

// Engine is the unified messaging engine that dispatches messages
// across any configured channel: email, SMS, or WhatsApp.
// All channel managers are optional — the engine gracefully handles
// unconfigured channels by returning descriptive errors.
type Engine struct {
	emailMgr    *email.Manager
	smsMgr      *sms.Manager
	whatsappMgr *whatsapp.Manager
	logger      *slog.Logger
}

// NewEngine creates a unified messaging engine.
// Pass nil for any manager that isn't configured.
func NewEngine(
	emailMgr *email.Manager,
	smsMgr *sms.Manager,
	whatsappMgr *whatsapp.Manager,
	logger *slog.Logger,
) *Engine {
	channels := []string{}
	if emailMgr != nil {
		channels = append(channels, "email")
	}
	if smsMgr != nil {
		channels = append(channels, "sms")
	}
	if whatsappMgr != nil {
		channels = append(channels, "whatsapp")
	}
	logger.Info("messaging engine initialized", slog.Any("channels", channels))

	return &Engine{
		emailMgr:    emailMgr,
		smsMgr:      smsMgr,
		whatsappMgr: whatsappMgr,
		logger:      logger,
	}
}

// Send dispatches a message via the specified channel.
// For email: address must be an email address. For SMS/WhatsApp: a phone number.
func (e *Engine) Send(ctx context.Context, channel Channel, address, subject, body string) (*DeliveryResult, error) {
	switch channel {
	case ChannelEmail:
		return e.sendEmail(ctx, address, subject, body)
	case ChannelSMS:
		return e.sendSMS(ctx, address, body)
	case ChannelWhatsApp:
		return e.sendWhatsApp(ctx, address, body)
	default:
		return nil, fmt.Errorf("unsupported messaging channel: %s", channel)
	}
}

// SendHTML dispatches an HTML email via the email channel.
func (e *Engine) SendHTML(ctx context.Context, to, subject, htmlBody string) (*DeliveryResult, error) {
	if e.emailMgr == nil {
		return nil, fmt.Errorf("email channel not configured")
	}
	result, err := e.emailMgr.SendHTML(ctx, to, subject, htmlBody)
	if err != nil {
		return &DeliveryResult{Channel: ChannelEmail, Success: false, Error: err.Error()}, err
	}
	return &DeliveryResult{
		Channel:   ChannelEmail,
		Provider:  result.Provider,
		MessageID: result.MessageID,
		Success:   result.Success,
	}, nil
}

// SendOTP sends a one-time password via the specified channel.
// This is a convenience method that formats the OTP message appropriately
// for each channel.
func (e *Engine) SendOTP(ctx context.Context, channel Channel, address, code string, ttlMinutes int) (*DeliveryResult, error) {
	switch channel {
	case ChannelEmail:
		subject := "AMY MIS — Your Password Reset Code"
		html := buildOTPEmailHTML(code, ttlMinutes)
		return e.SendHTML(ctx, address, subject, html)

	case ChannelSMS:
		message := fmt.Sprintf("AMY MIS: Your password reset code is %s. Valid for %d minutes. Do not share this code.", code, ttlMinutes)
		return e.sendSMS(ctx, address, message)

	case ChannelWhatsApp:
		message := fmt.Sprintf("🔐 *AMY MIS*\n\nYour password reset code is: *%s*\n\nThis code is valid for %d minutes.\nDo not share this code with anyone.", code, ttlMinutes)
		return e.sendWhatsApp(ctx, address, message)

	default:
		return nil, fmt.Errorf("unsupported OTP channel: %s", channel)
	}
}

// IsChannelAvailable checks if a specific channel is configured and ready.
func (e *Engine) IsChannelAvailable(channel Channel) bool {
	switch channel {
	case ChannelEmail:
		return e.emailMgr != nil
	case ChannelSMS:
		return e.smsMgr != nil
	case ChannelWhatsApp:
		return e.whatsappMgr != nil
	default:
		return false
	}
}

// AvailableChannels returns all configured channels.
func (e *Engine) AvailableChannels() []Channel {
	var channels []Channel
	if e.emailMgr != nil {
		channels = append(channels, ChannelEmail)
	}
	if e.smsMgr != nil {
		channels = append(channels, ChannelSMS)
	}
	if e.whatsappMgr != nil {
		channels = append(channels, ChannelWhatsApp)
	}
	return channels
}

// --- Internal channel dispatchers ---

func (e *Engine) sendEmail(ctx context.Context, to, subject, body string) (*DeliveryResult, error) {
	if e.emailMgr == nil {
		return nil, fmt.Errorf("email channel not configured")
	}
	result, err := e.emailMgr.SendToOne(ctx, to, subject, body)
	if err != nil {
		return &DeliveryResult{Channel: ChannelEmail, Success: false, Error: err.Error()}, err
	}
	return &DeliveryResult{
		Channel:   ChannelEmail,
		Provider:  result.Provider,
		MessageID: result.MessageID,
		Success:   result.Success,
	}, nil
}

func (e *Engine) sendSMS(ctx context.Context, phone, body string) (*DeliveryResult, error) {
	if e.smsMgr == nil {
		return nil, fmt.Errorf("SMS channel not configured")
	}
	result, err := e.smsMgr.Send(ctx, phone, body)
	if err != nil {
		return &DeliveryResult{Channel: ChannelSMS, Success: false, Error: err.Error()}, err
	}
	return &DeliveryResult{
		Channel:   ChannelSMS,
		Provider:  result.Provider,
		MessageID: result.MessageID,
		Success:   result.Success,
	}, nil
}

func (e *Engine) sendWhatsApp(ctx context.Context, phone, body string) (*DeliveryResult, error) {
	if e.whatsappMgr == nil {
		return nil, fmt.Errorf("WhatsApp channel not configured")
	}
	result, err := e.whatsappMgr.Send(ctx, phone, body)
	if err != nil {
		return &DeliveryResult{Channel: ChannelWhatsApp, Success: false, Error: err.Error()}, err
	}
	return &DeliveryResult{
		Channel:   ChannelWhatsApp,
		Provider:  result.Provider,
		MessageID: result.MessageID,
		Success:   result.Success,
	}, nil
}

// buildOTPEmailHTML generates a branded HTML email for OTP delivery.
func buildOTPEmailHTML(code string, ttlMinutes int) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"></head>
<body style="margin:0;padding:0;background-color:#0a0e1a;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="max-width:480px;margin:40px auto;background:#111827;border-radius:12px;border:1px solid rgba(255,255,255,0.06);">
  <tr><td style="padding:32px 32px 0;text-align:center;">
    <div style="font-size:28px;font-weight:800;color:#fff;">⚡ AMY<span style="color:#00d2ff;">MIS</span></div>
    <p style="color:#9ca3af;font-size:14px;margin:4px 0 24px;">Workforce Financial Operating System</p>
  </td></tr>
  <tr><td style="padding:0 32px;">
    <div style="background:rgba(0,210,255,0.04);border:1px solid rgba(0,210,255,0.12);border-radius:8px;padding:24px;text-align:center;">
      <p style="color:#d1d5db;font-size:14px;margin:0 0 16px;">Your password reset code is:</p>
      <div style="font-size:36px;font-weight:800;letter-spacing:12px;color:#00d2ff;font-family:monospace;padding:12px 0;">%s</div>
      <p style="color:#6b7280;font-size:12px;margin:16px 0 0;">This code expires in %d minutes</p>
    </div>
  </td></tr>
  <tr><td style="padding:24px 32px 32px;">
    <p style="color:#9ca3af;font-size:13px;line-height:1.5;margin:0;">
      If you didn't request a password reset, you can safely ignore this email. Your account remains secure.
    </p>
    <hr style="border:none;border-top:1px solid rgba(255,255,255,0.06);margin:20px 0;">
    <p style="color:#4b5563;font-size:11px;text-align:center;margin:0;">
      This is an automated message from AMY MIS. Do not reply to this email.<br>
      &copy; 2026 AMY MIS. All rights reserved.
    </p>
  </td></tr>
</table>
</body>
</html>`, code, ttlMinutes)
}
