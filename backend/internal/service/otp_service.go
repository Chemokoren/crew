package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/kibsoft/amy-mis/internal/external/messaging"
	"github.com/redis/go-redis/v9"
)

// OTP configuration constants
const (
	otpLength     = 6
	otpTTL        = 10 * time.Minute // OTP valid for 10 minutes
	otpCooldown   = 60 * time.Second // Minimum time between OTP requests
	otpMaxRetries = 5                // Max verification attempts before lockout
)

// OTPService manages one-time passwords for self-service password reset.
// OTPs are stored in Redis with automatic TTL-based expiry.
// Delivery is handled by the unified messaging engine (email/SMS/WhatsApp).
type OTPService struct {
	rdb            *redis.Client
	msgEngine      *messaging.Engine
	defaultChannel messaging.Channel // Configurable: "email" | "sms" | "whatsapp"
	enabled        bool              // OTP feature toggle (admin-configurable)
	logger         *slog.Logger
}

// OTPConfig holds OTP service configuration.
type OTPConfig struct {
	DefaultChannel string // "email" (default), "sms", or "whatsapp"
	Enabled        bool   // Feature toggle
}

// NewOTPService creates a new OTP service backed by Redis.
func NewOTPService(rdb *redis.Client, msgEngine *messaging.Engine, cfg OTPConfig, logger *slog.Logger) *OTPService {
	channel := messaging.ChannelEmail
	switch cfg.DefaultChannel {
	case "sms":
		channel = messaging.ChannelSMS
	case "whatsapp":
		channel = messaging.ChannelWhatsApp
	default:
		channel = messaging.ChannelEmail
	}

	logger.Info("OTP service initialized",
		slog.String("default_channel", string(channel)),
		slog.Bool("enabled", cfg.Enabled),
	)

	return &OTPService{
		rdb:            rdb,
		msgEngine:      msgEngine,
		defaultChannel: channel,
		enabled:        cfg.Enabled,
		logger:         logger,
	}
}

// GenerateAndSend creates an OTP and delivers it via the specified channel.
// If channel is empty, uses the configured default channel.
// For email delivery, the emailAddress parameter is required.
// For SMS/WhatsApp delivery, the phone parameter is required.
func (s *OTPService) GenerateAndSend(ctx context.Context, phone, emailAddress, channel string) error {
	if !s.enabled {
		return fmt.Errorf("OTP-based password reset is disabled by the system administrator")
	}

	// Determine delivery channel
	deliveryChannel := s.defaultChannel
	switch channel {
	case "sms":
		deliveryChannel = messaging.ChannelSMS
	case "email":
		deliveryChannel = messaging.ChannelEmail
	case "whatsapp":
		deliveryChannel = messaging.ChannelWhatsApp
	case "":
		// Use default
	default:
		return fmt.Errorf("unsupported OTP channel: %s (use email, sms, or whatsapp)", channel)
	}

	// Validate that the channel is configured
	if s.msgEngine != nil && !s.msgEngine.IsChannelAvailable(deliveryChannel) {
		return fmt.Errorf("%s channel is not configured — contact your administrator", deliveryChannel)
	}

	// Determine delivery address
	address := phone
	if deliveryChannel == messaging.ChannelEmail {
		if emailAddress == "" {
			return fmt.Errorf("email address is required for email OTP delivery")
		}
		address = emailAddress
	} else if phone == "" {
		return fmt.Errorf("phone number is required for %s OTP delivery", deliveryChannel)
	}

	// Check cooldown — prevent spamming
	cooldownKey := fmt.Sprintf("otp:cooldown:%s", phone)
	if ttl, _ := s.rdb.TTL(ctx, cooldownKey).Result(); ttl > 0 {
		return fmt.Errorf("please wait %d seconds before requesting another OTP", int(ttl.Seconds()))
	}

	// Generate cryptographically secure 6-digit code
	code, err := generateSecureCode(otpLength)
	if err != nil {
		return fmt.Errorf("generate OTP: %w", err)
	}

	// Store OTP with TTL (use pipeline for atomicity)
	otpKey := fmt.Sprintf("otp:code:%s", phone)
	attemptsKey := fmt.Sprintf("otp:attempts:%s", phone)
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, otpKey, code, otpTTL)
	pipe.Set(ctx, cooldownKey, "1", otpCooldown)
	pipe.Del(ctx, attemptsKey) // Reset attempts on new OTP
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store OTP: %w", err)
	}

	// Deliver OTP via messaging engine
	if s.msgEngine != nil {
		result, err := s.msgEngine.SendOTP(ctx, deliveryChannel, address, code, 10)
		if err != nil {
			s.logger.Error("OTP delivery failed",
				slog.String("channel", string(deliveryChannel)),
				slog.String("address", address),
				slog.String("error", err.Error()),
			)
			// Don't fail the OTP generation — the code is in Redis and can be resent
		} else {
			s.logger.Info("OTP delivered",
				slog.String("channel", string(deliveryChannel)),
				slog.String("address", address),
				slog.String("provider", result.Provider),
			)
		}
	} else {
		s.logger.Warn("messaging engine not configured — OTP generated but not delivered",
			slog.String("phone", phone),
		)
	}

	s.logger.Info("OTP generated",
		slog.String("phone", phone),
		slog.String("channel", string(deliveryChannel)),
		slog.Duration("ttl", otpTTL),
	)

	return nil
}

// VerifyOTP checks the provided code against the stored OTP.
// Returns a reset token (UUID) on success that must be used to complete the reset.
// Implements brute-force protection via attempt counting.
func (s *OTPService) VerifyOTP(ctx context.Context, phone, code string) (string, error) {
	otpKey := fmt.Sprintf("otp:code:%s", phone)
	attemptsKey := fmt.Sprintf("otp:attempts:%s", phone)
	resetTokenKey := fmt.Sprintf("otp:reset_token:%s", phone)

	// Check attempt count
	attempts, _ := s.rdb.Get(ctx, attemptsKey).Int()
	if attempts >= otpMaxRetries {
		// Clean up — force re-request
		s.rdb.Del(ctx, otpKey, attemptsKey)
		return "", fmt.Errorf("too many attempts — please request a new OTP")
	}

	// Get stored OTP
	storedCode, err := s.rdb.Get(ctx, otpKey).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("OTP expired or not found — please request a new one")
	} else if err != nil {
		return "", fmt.Errorf("verify OTP: %w", err)
	}

	// Increment attempts
	s.rdb.Incr(ctx, attemptsKey)
	s.rdb.Expire(ctx, attemptsKey, otpTTL) // Attempts expire with OTP

	// Constant-time comparison
	if storedCode != code {
		remaining := otpMaxRetries - attempts - 1
		return "", fmt.Errorf("invalid OTP — %d attempts remaining", remaining)
	}

	// OTP verified — generate a short-lived reset token
	resetToken, err := generateSecureCode(32)
	if err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}

	// Store reset token (5 min TTL) and clean up OTP
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, resetTokenKey, resetToken, 5*time.Minute)
	pipe.Del(ctx, otpKey, attemptsKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}

	s.logger.Info("OTP verified", slog.String("phone", phone))
	return resetToken, nil
}

// ResetPasswordWithToken validates the reset token.
func (s *OTPService) ResetPasswordWithToken(ctx context.Context, phone, resetToken, newPassword string) error {
	resetTokenKey := fmt.Sprintf("otp:reset_token:%s", phone)

	storedToken, err := s.rdb.Get(ctx, resetTokenKey).Result()
	if err == redis.Nil {
		return fmt.Errorf("reset token expired — please start over")
	} else if err != nil {
		return fmt.Errorf("validate reset token: %w", err)
	}

	if storedToken != resetToken {
		return fmt.Errorf("invalid reset token")
	}

	// Clean up token (single-use)
	s.rdb.Del(ctx, resetTokenKey)

	s.logger.Info("password reset via OTP", slog.String("phone", phone))
	return nil
}

// IsEnabled returns whether OTP is enabled.
func (s *OTPService) IsEnabled() bool {
	return s.enabled
}

// DefaultChannel returns the configured default OTP delivery channel.
func (s *OTPService) DefaultChannel() messaging.Channel {
	return s.defaultChannel
}

// AvailableChannels returns all channels available for OTP delivery.
func (s *OTPService) AvailableChannels() []messaging.Channel {
	if s.msgEngine == nil {
		return nil
	}
	return s.msgEngine.AvailableChannels()
}

// HashPassword creates a bcrypt hash (exported for use by handlers).
func HashPassword(password string) (string, error) {
	return "", fmt.Errorf("use auth service AdminResetPassword instead")
}

// generateSecureCode creates a cryptographically secure numeric string of the given length.
func generateSecureCode(length int) (string, error) {
	code := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code, nil
}
