package jambopay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"
	"os"

	"github.com/kibsoft/amy-mis/internal/external/payment"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// walletAccountListResp returns the real JamboPay paginated account response shape.
func walletAccountListResp(balance int64, currency string) map[string]interface{} {
	return map[string]interface{}{
		"pageIndex": 1,
		"pageSize":  10,
		"count":     1,
		"data": []map[string]interface{}{
			{
				"accountNo":      "ACC-001",
				"currentBalance": balance, // already in minor units
				"bookBalance":    balance,
				"currency":       currency,
				"accountType":    "Business",
				"isActive":       true,
			},
		},
	}
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Auth endpoint — matches accounts.jambopay.com/v2/auth/token
	mux.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("auth method = %s, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "jp-test-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})

	// Payout endpoint
	mux.HandleFunc("/payout", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer jp-test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"ref":     "JP-REF-001",
			"orderId": "ORD-001",
		})
	})

	// Verify payout
	mux.HandleFunc("/payout/authorize", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
	})

	// Balance — paginated list (matches real API shape)
	mux.HandleFunc("/wallet/account", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(walletAccountListResp(1502600, "KES")) // KES 15,026.00
	})

	return httptest.NewServer(mux)
}

// newMockProvider creates a provider pointing at both base URL and auth URL
// to the same mock server (the mock handles /auth/token for both).
func newMockProvider(t *testing.T, server *httptest.Server) *JamboPayProvider {
	t.Helper()
	return NewJamboPayProvider(JamboPayConfig{
		BaseURL:       server.URL,
		AuthURL:       server.URL,
		ClientID:      "test",
		ClientSecret:  "test",
		CollectionAccount: "COLL-001", // collection account
		PayoutAccount:     "PAY-001",  // payout / merchant wallet
	}, testLogger())
}

func TestJamboPayName(t *testing.T) {
	p := NewJamboPayProvider(JamboPayConfig{}, testLogger())
	if p.Name() != "jambopay" {
		t.Errorf("Name = %q, want jambopay", p.Name())
	}
}

func TestJamboPayInitiatePayoutMobile(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()
	p := newMockProvider(t, server)

	result, err := p.InitiatePayout(context.Background(), payment.PayoutRequest{
		AmountCents:    150000,
		AccountFrom:    "ACC-001",
		OrderID:        "ORD-001",
		Channel:        payment.ChannelMobile,
		RecipientName:  "John Kamau",
		RecipientPhone: "0712345678",
		CallbackURL:    "https://example.com/callback",
		Narration:      "Salary payout",
	})
	if err != nil {
		t.Fatalf("InitiatePayout: %v", err)
	}
	if result.Reference != "JP-REF-001" {
		t.Errorf("Reference = %q, want JP-REF-001", result.Reference)
	}
	if !result.RequiresOTP {
		t.Error("JamboPay payouts require OTP verification")
	}
	if result.Status != "pending_otp" {
		t.Errorf("Status = %q, want pending_otp", result.Status)
	}
}

func TestJamboPayInitiatePayoutBank(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()
	p := newMockProvider(t, server)

	result, err := p.InitiatePayout(context.Background(), payment.PayoutRequest{
		AmountCents: 500000,
		AccountFrom: "ACC-001",
		OrderID:     "ORD-002",
		Channel:     payment.ChannelBank,
		BankAccount: "1234567890",
		BankCode:    "11",
		Narration:   "Bank transfer",
	})
	if err != nil {
		t.Fatalf("InitiatePayout bank: %v", err)
	}
	if result.Provider != "jambopay" {
		t.Errorf("Provider = %q, want jambopay", result.Provider)
	}
}

func TestJamboPayUnsupportedChannel(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()
	p := newMockProvider(t, server)

	_, err := p.InitiatePayout(context.Background(), payment.PayoutRequest{
		Channel: "CRYPTO",
	})
	if err == nil {
		t.Error("should fail for unsupported channel")
	}
}

func TestJamboPayVerifyPayout(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()
	p := newMockProvider(t, server)

	result, err := p.VerifyPayout(context.Background(), payment.PayoutVerifyRequest{
		Reference: "JP-REF-001",
		OTP:       "123456",
	})
	if err != nil {
		t.Fatalf("VerifyPayout: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want completed", result.Status)
	}
}

func TestJamboPayCheckBalance(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()
	p := newMockProvider(t, server)

	result, err := p.CheckBalance(context.Background(), "ACC-001")
	if err != nil {
		t.Fatalf("CheckBalance: %v", err)
	}
	// Mock returns 1502600 minor units = KES 15,026.00
	if result.Balance != 1502600 {
		t.Errorf("Balance = %d, want 1502600", result.Balance)
	}
	if result.Currency != "KES" {
		t.Errorf("Currency = %q, want KES", result.Currency)
	}
}

func TestJamboPayTokenCaching(t *testing.T) {
	authCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		authCalls++
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "cached-token", "expires_in": 3600,
		})
	})
	mux.HandleFunc("/wallet/account", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(walletAccountListResp(0, "KES"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	p := NewJamboPayProvider(JamboPayConfig{
		BaseURL:       server.URL,
		AuthURL:       server.URL,
		ClientID:      "c",
		ClientSecret:  "s",
		CollectionAccount: "COLL-001",
		PayoutAccount:     "PAY-001",
	}, testLogger())

	p.CheckBalance(context.Background(), "A")
	p.CheckBalance(context.Background(), "B")

	if authCalls != 1 {
		t.Errorf("Auth called %d times, want 1 (token should be cached)", authCalls)
	}
}

func TestJamboPayAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := NewJamboPayProvider(JamboPayConfig{
		BaseURL:       server.URL,
		AuthURL:       server.URL,
		ClientID:      "bad",
		ClientSecret:  "bad",
	}, testLogger())

	_, err := p.CheckBalance(context.Background(), "ACC")
	if err == nil {
		t.Error("should fail on auth failure")
	}
}
