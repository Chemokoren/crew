package retry_test

import (
	"context"
	"errors"
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
