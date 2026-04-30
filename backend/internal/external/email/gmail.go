package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

// GmailConfig holds SMTP configuration for Gmail.
type GmailConfig struct {
	Host     string // SMTP host (e.g. smtp.gmail.com)
	Port     int    // SMTP port (e.g. 587)
	Username string // SMTP username (e.g. testineek@gmail.com)
	Password string // SMTP app password
	FromAddr string // Default sender address
	FromName string // Display name (e.g. "AMY MIS")
	UseTLS   bool   // Use STARTTLS (default: true)
}

// GmailProvider implements the email Provider interface using Gmail SMTP.
// Works with any SMTP server — not limited to Gmail.
type GmailProvider struct {
	config GmailConfig
	logger *slog.Logger
}

// NewGmailProvider creates a new Gmail SMTP email provider.
func NewGmailProvider(config GmailConfig, logger *slog.Logger) *GmailProvider {
	if config.Port == 0 {
		config.Port = 587
	}
	if config.FromName == "" {
		config.FromName = "AMY MIS"
	}
	return &GmailProvider{config: config, logger: logger}
}

func (p *GmailProvider) Name() string { return "gmail" }

// Send dispatches an email via SMTP.
func (p *GmailProvider) Send(ctx context.Context, msg Message) (*SendResult, error) {
	if len(msg.To) == 0 {
		return &SendResult{Provider: p.Name(), Success: false, Error: "no recipients"}, fmt.Errorf("no recipients")
	}

	// Build RFC 2822 message
	from := fmt.Sprintf("%s <%s>", p.config.FromName, p.config.FromAddr)
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(msg.To, ", ")
	headers["Subject"] = msg.Subject
	headers["MIME-Version"] = "1.0"

	var body string
	if msg.HTML != "" {
		headers["Content-Type"] = "text/html; charset=UTF-8"
		body = msg.HTML
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
		body = msg.Body
	}

	// Assemble raw message
	var rawMsg strings.Builder
	for k, v := range headers {
		rawMsg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	rawMsg.WriteString("\r\n")
	rawMsg.WriteString(body)

	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)
	auth := smtp.PlainAuth("", p.config.Username, p.config.Password, p.config.Host)

	var err error
	if p.config.UseTLS || p.config.Port == 587 {
		err = p.sendWithTLS(addr, auth, msg.To, rawMsg.String())
	} else {
		err = smtp.SendMail(addr, auth, p.config.FromAddr, msg.To, []byte(rawMsg.String()))
	}

	if err != nil {
		p.logger.Error("Gmail SMTP send failed",
			slog.String("to", strings.Join(msg.To, ",")),
			slog.String("error", err.Error()),
		)
		return &SendResult{
			Provider: p.Name(),
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	p.logger.Info("email sent via Gmail SMTP",
		slog.String("to", strings.Join(msg.To, ",")),
		slog.String("subject", msg.Subject),
	)

	return &SendResult{
		Provider: p.Name(),
		Success:  true,
	}, nil
}

// sendWithTLS handles STARTTLS connection (required by Gmail on port 587).
func (p *GmailProvider) sendWithTLS(addr string, auth smtp.Auth, to []string, body string) error {
	// Connect to SMTP server
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	// Upgrade to TLS
	tlsConfig := &tls.Config{ServerName: p.config.Host}
	if err := conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}

	// Authenticate
	if err := conn.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	// Set sender
	if err := conn.Mail(p.config.FromAddr); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := conn.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", recipient, err)
		}
	}

	// Send body
	wc, err := conn.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = wc.Write([]byte(body)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err = wc.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return conn.Quit()
}
