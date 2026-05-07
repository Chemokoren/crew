// Package retry provides a configurable exponential-backoff retry mechanism
// for external integration calls (payment, SMS, payroll, identity).
//
// Features:
//   - Exponential backoff: delay doubles on each attempt (500ms → 1s → 2s → …)
//   - Jitter: ±25% randomisation prevents thundering-herd effects
//   - Context-aware: respects cancellation and deadline propagation
//   - Configurable: attempts, delays, and a predicate to decide which errors are retryable
//   - Idempotent-safe: callers must ensure the wrapped operation is safe to repeat
package retry

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"
)

// Policy defines the retry behaviour for an integration call.
type Policy struct {
	MaxAttempts  int           // Total attempts (including the first). Min 1.
	InitialDelay time.Duration // Delay after the first failure. Doubles each retry.
	MaxDelay     time.Duration // Cap on the backoff delay.
}

// DefaultPolicy returns sensible defaults:
// 3 attempts, 500ms initial delay, 5s max delay.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:  3,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     5 * time.Second,
	}
}

// Do executes fn with exponential-backoff retries.
// It retries on any error unless isRetryable returns false.
// If isRetryable is nil, all errors are retried.
// Returns the result and the last error encountered.
func Do[T any](ctx context.Context, logger *slog.Logger, opName string, p Policy, isRetryable func(error) bool, fn func(ctx context.Context) (T, error)) (T, error) {
	if p.MaxAttempts < 1 {
		p.MaxAttempts = 1
	}
	if p.InitialDelay <= 0 {
		p.InitialDelay = 500 * time.Millisecond
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = 5 * time.Second
	}

	delay := p.InitialDelay
	var lastErr error
	var zero T

	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return zero, fmt.Errorf("%s: context cancelled after %d attempt(s): %w", opName, attempt-1, ctx.Err())
		}

		result, err := fn(ctx)
		if err == nil {
			if attempt > 1 {
				logger.Info("retry succeeded",
					slog.String("op", opName),
					slog.Int("attempt", attempt),
				)
			}
			return result, nil
		}

		lastErr = err

		// Check if the error is retryable
		if isRetryable != nil && !isRetryable(err) {
			return zero, fmt.Errorf("%s: non-retryable error: %w", opName, err)
		}

		// Don't sleep after the last attempt
		if attempt == p.MaxAttempts {
			break
		}

		// Add jitter: ±25%
		jittered := addJitter(delay)

		logger.Warn("retrying after failure",
			slog.String("op", opName),
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", p.MaxAttempts),
			slog.Duration("next_delay", jittered),
			slog.String("error", err.Error()),
		)

		// Wait or respect context cancellation
		timer := time.NewTimer(jittered)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, fmt.Errorf("%s: context cancelled during backoff: %w", opName, ctx.Err())
		case <-timer.C:
		}

		// Double the delay, capped at max
		delay *= 2
		if delay > p.MaxDelay {
			delay = p.MaxDelay
		}
	}

	return zero, fmt.Errorf("%s: all %d attempts failed: %w", opName, p.MaxAttempts, lastErr)
}

// DoVoid is like Do but for functions that don't return a value.
func DoVoid(ctx context.Context, logger *slog.Logger, opName string, p Policy, isRetryable func(error) bool, fn func(ctx context.Context) error) error {
	_, err := Do(ctx, logger, opName, p, isRetryable, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(ctx)
	})
	return err
}

// IsNetworkError returns true for errors that are typically transient
// (timeouts, connection refused, DNS errors).
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	transientPatterns := []string{
		"timeout",
		"i/o timeout",
		"connection refused",
		"connection reset",
		"no such host",
		"temporary failure",
		"dial tcp",
		"TLS handshake timeout",
		"server misbehaving",
		"network is unreachable",
		"EOF",
	}
	for _, p := range transientPatterns {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// addJitter adds ±25% random jitter to a duration.
func addJitter(d time.Duration) time.Duration {
	// jitter range: 75% to 125% of d
	factor := 0.75 + rand.Float64()*0.5
	return time.Duration(float64(d) * factor)
}
