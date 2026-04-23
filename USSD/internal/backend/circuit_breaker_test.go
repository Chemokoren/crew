package backend

import (
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	// Should start closed
	if cb.State() != CircuitClosed {
		t.Errorf("initial state should be CLOSED, got %s", cb.State())
	}

	// Should allow requests
	if !cb.Allow() {
		t.Error("CLOSED circuit should allow requests")
	}

	// Should still be closed after fewer failures than threshold
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitClosed {
		t.Errorf("should still be CLOSED after 2 failures (threshold=3), got %s", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("should be OPEN after 3 failures, got %s", cb.State())
	}

	if cb.Allow() {
		t.Error("OPEN circuit should NOT allow requests")
	}
}

func TestCircuitBreaker_SuccessResetsFaliures(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // Reset failures

	if cb.Failures() != 0 {
		t.Errorf("success should reset failure count, got %d", cb.Failures())
	}

	// Should still be closed
	if cb.State() != CircuitClosed {
		t.Errorf("should still be CLOSED, got %s", cb.State())
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatalf("should be OPEN, got %s", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open on next Allow()
	if !cb.Allow() {
		t.Error("should allow request after timeout (half-open)")
	}

	if cb.State() != CircuitHalfOpen {
		t.Errorf("should be HALF_OPEN after timeout, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToClosedOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure()

	time.Sleep(60 * time.Millisecond)
	cb.Allow() // Triggers HALF_OPEN

	// Simulate successful test requests
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Errorf("should be CLOSED after successful test requests, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToOpenOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure()

	time.Sleep(60 * time.Millisecond)
	cb.Allow() // Triggers HALF_OPEN

	cb.RecordFailure() // Failure in half-open → back to open

	if cb.State() != CircuitOpen {
		t.Errorf("should be OPEN after failure in half-open, got %s", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Second)

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatalf("should be OPEN, got %s", cb.State())
	}

	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("should be CLOSED after reset, got %s", cb.State())
	}

	if cb.Failures() != 0 {
		t.Errorf("failures should be 0 after reset, got %d", cb.Failures())
	}
}

func TestCircuitBreaker_StateString(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{CircuitClosed, "CLOSED"},
		{CircuitOpen, "OPEN"},
		{CircuitHalfOpen, "HALF_OPEN"},
		{CircuitState(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
