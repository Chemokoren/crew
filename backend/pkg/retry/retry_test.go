package retry_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kibsoft/amy-mis/pkg/retry"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestDo_ImmediateSuccess(t *testing.T) {
	calls := 0
	result, err := retry.Do(context.Background(), testLogger(), "test_op", retry.DefaultPolicy(), nil,
		func(ctx context.Context) (string, error) {
			calls++
			return "ok", nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected 'ok', got %q", result)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDo_RetryThenSucceed(t *testing.T) {
	calls := 0
	result, err := retry.Do(context.Background(), testLogger(), "test_op",
		retry.Policy{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
		nil,
		func(ctx context.Context) (string, error) {
			calls++
			if calls < 3 {
				return "", errors.New("dial tcp: i/o timeout")
			}
			return "recovered", nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "recovered" {
		t.Fatalf("expected 'recovered', got %q", result)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_AllAttemptsFail(t *testing.T) {
	calls := 0
	_, err := retry.Do(context.Background(), testLogger(), "test_op",
		retry.Policy{MaxAttempts: 2, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
		nil,
		func(ctx context.Context) (string, error) {
			calls++
			return "", errors.New("connection refused")
		},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestDo_NonRetryableError(t *testing.T) {
	calls := 0
	_, err := retry.Do(context.Background(), testLogger(), "test_op",
		retry.Policy{MaxAttempts: 5, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
		func(err error) bool { return false }, // nothing is retryable
		func(ctx context.Context) (string, error) {
			calls++
			return "", errors.New("bad request")
		},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call (non-retryable), got %d", calls)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	_, err := retry.Do(ctx, testLogger(), "test_op",
		retry.Policy{MaxAttempts: 5, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
		nil,
		func(ctx context.Context) (string, error) {
			return "", errors.New("should not be called")
		},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{errors.New("dial tcp: lookup accounts.jambopay.com: i/o timeout"), true},
		{errors.New("connection refused"), true},
		{errors.New("TLS handshake timeout"), true},
		{errors.New("unexpected EOF"), true},
		{errors.New("bad request: invalid amount"), false},
		{errors.New("unauthorized"), false},
		{nil, false},
	}

	for _, tc := range tests {
		got := retry.IsNetworkError(tc.err)
		if got != tc.expected {
			t.Errorf("IsNetworkError(%v) = %v, want %v", tc.err, got, tc.expected)
		}
	}
}

func TestDoVoid_Works(t *testing.T) {
	calls := 0
	err := retry.DoVoid(context.Background(), testLogger(), "test_void",
		retry.Policy{MaxAttempts: 2, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
		nil,
		func(ctx context.Context) error {
			calls++
			if calls < 2 {
				return errors.New("timeout")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestDo_MaxAttemptsOne_NoRetry(t *testing.T) {
	calls := 0
	_, err := retry.Do(context.Background(), testLogger(), "test_no_retry",
		retry.Policy{MaxAttempts: 1, InitialDelay: 10 * time.Millisecond, MaxDelay: 50 * time.Millisecond},
		nil,
		func(ctx context.Context) (string, error) {
			calls++
			return "", errors.New("fail")
		},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Fatalf("MaxAttempts=1 should call exactly once, got %d", calls)
	}
}

func TestDo_ZeroPolicy_Normalizes(t *testing.T) {
	// Zero values should be normalized to defaults and not panic
	calls := 0
	_, err := retry.Do(context.Background(), testLogger(), "test_zero_policy",
		retry.Policy{}, // all zeros
		nil,
		func(ctx context.Context) (string, error) {
			calls++
			return "ok", nil
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// MaxAttempts=0 normalizes to 1
	if calls != 1 {
		t.Fatalf("expected 1 call with zero policy, got %d", calls)
	}
}

func TestDo_ContextTimeoutDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	calls := 0
	_, err := retry.Do(ctx, testLogger(), "test_timeout_backoff",
		retry.Policy{MaxAttempts: 5, InitialDelay: 200 * time.Millisecond, MaxDelay: 1 * time.Second},
		nil,
		func(ctx context.Context) (string, error) {
			calls++
			return "", errors.New("network error")
		},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should fail after 1st attempt because the 200ms backoff exceeds 30ms ctx deadline
	if calls != 1 {
		t.Fatalf("expected 1 call before context timeout, got %d", calls)
	}
}

func TestDefaultPolicy_Values(t *testing.T) {
	p := retry.DefaultPolicy()
	if p.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", p.MaxAttempts)
	}
	if p.InitialDelay != 500*time.Millisecond {
		t.Errorf("InitialDelay = %v, want 500ms", p.InitialDelay)
	}
	if p.MaxDelay != 5*time.Second {
		t.Errorf("MaxDelay = %v, want 5s", p.MaxDelay)
	}
}

func TestIsNetworkError_WrappedErrors(t *testing.T) {
	// Test that wrapped errors are still detected
	inner := errors.New("dial tcp 196.50.21.127:443: i/o timeout")
	wrapped := fmt.Errorf("jambopay auth: %w", inner)
	if !retry.IsNetworkError(wrapped) {
		t.Error("expected wrapped i/o timeout to be detected as network error")
	}
}

