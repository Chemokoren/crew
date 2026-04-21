package perpay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"
	"os"

	"github.com/kibsoft/amy-mis/internal/external/payroll"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/issue", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["x_client_id"] == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "pp-jwt-token",
			"token_type":   "Bearer",
			"expires_in":   900,
		})
	})

	mux.HandleFunc("/payroll/v1/payroll/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer pp-jwt-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Check idempotency replay
		if r.Header.Get("Idempotency-Key") == "duplicate-key" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"correlation_id": "existing-corr-id",
				"status":         "received",
			})
			return
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"correlation_id": "550e8400-e29b-41d4-a716-446655440000",
			"status":         "received",
			"status_url":     "/payroll/v1/requests/550e8400-e29b-41d4-a716-446655440000/status",
		})
	})

	mux.HandleFunc("/payroll/v1/requests/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"correlation_id": "550e8400-e29b-41d4-a716-446655440000",
			"status":         "completed",
			"result_summary": map[string]interface{}{
				"gross_pay":        75000.00,
				"net_pay":          62500.00,
				"total_deductions": 12500.00,
			},
		})
	})

	return httptest.NewServer(mux)
}

func TestPerPayName(t *testing.T) {
	p := NewPerPayProvider(PerPayConfig{}, testLogger())
	if p.Name() != "perpay" {
		t.Errorf("Name = %q, want perpay", p.Name())
	}
}

func TestPerPaySubmitPayroll(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	p := NewPerPayProvider(PerPayConfig{
		BaseURL: server.URL, ClientID: "test", ClientSecret: "test",
	}, testLogger())

	result, err := p.SubmitPayroll(context.Background(), payroll.SubmitRequest{
		EmployeeID:    "EMP-12345",
		FullName:      "Jane Doe",
		EmployeePIN:   "A123456789K",
		Currency:      "KES",
		PayPeriodStart: "2026-02-01",
		PayPeriodEnd:   "2026-02-28",
		PayComponents: []payroll.PayComponent{
			{ID: "base_salary", Amount: 75000.00, Description: "Base monthly salary"},
		},
		Deductions: []payroll.Deduction{
			{ID: "sacco_contribution", Amount: 10000.00, Type: "cooperative_contribution", PreTax: true},
		},
	})
	if err != nil {
		t.Fatalf("SubmitPayroll: %v", err)
	}
	if result.CorrelationID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("CorrelationID = %q", result.CorrelationID)
	}
	if result.Status != "received" {
		t.Errorf("Status = %q, want received", result.Status)
	}
	if result.Provider != "perpay" {
		t.Errorf("Provider = %q, want perpay", result.Provider)
	}
}

func TestPerPayIdempotencyReplay(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	p := NewPerPayProvider(PerPayConfig{
		BaseURL: server.URL, ClientID: "test", ClientSecret: "test",
	}, testLogger())

	result, err := p.SubmitPayroll(context.Background(), payroll.SubmitRequest{
		EmployeeID:     "EMP-12345",
		FullName:       "Jane Doe",
		Currency:       "KES",
		PayPeriodStart: "2026-02-01",
		PayPeriodEnd:   "2026-02-28",
		IdempotencyKey: "duplicate-key",
	})
	if err != nil {
		t.Fatalf("SubmitPayroll idempotent: %v", err)
	}
	if result.CorrelationID != "existing-corr-id" {
		t.Errorf("should return cached correlation ID, got %q", result.CorrelationID)
	}
}

func TestPerPayGetStatusCompleted(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	p := NewPerPayProvider(PerPayConfig{
		BaseURL: server.URL, ClientID: "test", ClientSecret: "test",
	}, testLogger())

	result, err := p.GetStatus(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want completed", result.Status)
	}
	if result.GrossPay != 75000.00 {
		t.Errorf("GrossPay = %.2f, want 75000.00", result.GrossPay)
	}
	if result.NetPay != 62500.00 {
		t.Errorf("NetPay = %.2f, want 62500.00", result.NetPay)
	}
	if result.Deductions != 12500.00 {
		t.Errorf("Deductions = %.2f, want 12500.00", result.Deductions)
	}
}

func TestPerPayTokenCaching(t *testing.T) {
	authCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/issue", func(w http.ResponseWriter, r *http.Request) {
		authCalls++
		json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "t", "expires_in": 900})
	})
	mux.HandleFunc("/payroll/v1/requests/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"correlation_id": "x", "status": "received"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	p := NewPerPayProvider(PerPayConfig{
		BaseURL: server.URL, ClientID: "c", ClientSecret: "s",
	}, testLogger())

	p.GetStatus(context.Background(), "a")
	p.GetStatus(context.Background(), "b")

	if authCalls != 1 {
		t.Errorf("Auth called %d times, want 1", authCalls)
	}
}
