package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"log/slog"
	"os"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- Manager tests ---

type mockProvider struct {
	name      string
	sendErr   error
	sendCount int
	mu        sync.Mutex
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Send(ctx context.Context, phone, message string) (*SendResult, error) {
	m.mu.Lock()
	m.sendCount++
	m.mu.Unlock()
	if m.sendErr != nil {
		return &SendResult{Provider: m.name, Success: false, Error: m.sendErr.Error()}, m.sendErr
	}
	return &SendResult{Provider: m.name, Success: true, MessageID: "mock-msg-001"}, nil
}
func (m *mockProvider) SendBulk(ctx context.Context, phones []string, message string) ([]SendResult, error) {
	results := make([]SendResult, len(phones))
	for i, p := range phones {
		r, _ := m.Send(ctx, p, message)
		results[i] = *r
	}
	return results, nil
}

func TestManagerSendPrimary(t *testing.T) {
	primary := &mockProvider{name: "primary"}
	fallback := &mockProvider{name: "fallback"}
	mgr := NewManager(testLogger(), primary, fallback)

	result, err := mgr.Send(context.Background(), "+254712345678", "Hello")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if result.Provider != "primary" {
		t.Errorf("Provider = %q, want primary", result.Provider)
	}
	if primary.sendCount != 1 {
		t.Errorf("primary.sendCount = %d, want 1", primary.sendCount)
	}
	if fallback.sendCount != 0 {
		t.Errorf("fallback should not be called when primary succeeds")
	}
}

func TestManagerFallbackOnPrimaryFailure(t *testing.T) {
	primary := &mockProvider{name: "primary", sendErr: fmt.Errorf("network timeout")}
	fallback := &mockProvider{name: "fallback"}
	mgr := NewManager(testLogger(), primary, fallback)

	result, err := mgr.Send(context.Background(), "+254712345678", "Hello")
	if err != nil {
		t.Fatalf("Send should succeed via fallback: %v", err)
	}
	if result.Provider != "fallback" {
		t.Errorf("Provider = %q, want fallback", result.Provider)
	}
}

func TestManagerAllProvidersFail(t *testing.T) {
	p1 := &mockProvider{name: "p1", sendErr: fmt.Errorf("fail1")}
	p2 := &mockProvider{name: "p2", sendErr: fmt.Errorf("fail2")}
	mgr := NewManager(testLogger(), p1, p2)

	_, err := mgr.Send(context.Background(), "+254712345678", "Hello")
	if err == nil {
		t.Error("Send should fail when all providers fail")
	}
}

func TestManagerSetPrimary(t *testing.T) {
	p1 := &mockProvider{name: "optimize"}
	p2 := &mockProvider{name: "africastalking"}
	mgr := NewManager(testLogger(), p1, p2)

	if err := mgr.SetPrimary("africastalking"); err != nil {
		t.Fatalf("SetPrimary: %v", err)
	}

	result, _ := mgr.Send(context.Background(), "+254712345678", "Test")
	if result.Provider != "africastalking" {
		t.Errorf("After SetPrimary, Provider = %q, want africastalking", result.Provider)
	}
}

func TestManagerSetPrimaryNotFound(t *testing.T) {
	mgr := NewManager(testLogger(), &mockProvider{name: "optimize"})
	if err := mgr.SetPrimary("nonexistent"); err == nil {
		t.Error("SetPrimary should fail for unknown provider")
	}
}

func TestManagerSendBulk(t *testing.T) {
	primary := &mockProvider{name: "primary"}
	mgr := NewManager(testLogger(), primary)

	phones := []string{"+254700000001", "+254700000002", "+254700000003"}
	results, err := mgr.SendBulk(context.Background(), phones, "Bulk test")
	if err != nil {
		t.Fatalf("SendBulk: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("results count = %d, want 3", len(results))
	}
}

// --- Optimize provider tests (with httptest server) ---

func TestOptimizeSendSuccess(t *testing.T) {
	// Mock OAuth token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-jwt-token",
		})
	}))
	defer tokenServer.Close()

	// Mock SMS send server
	smsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "JWT test-jwt-token" {
			t.Errorf("Authorization = %q, want JWT test-jwt-token", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer smsServer.Close()

	provider := NewOptimizeProvider(OptimizeConfig{
		ClientID:           "test-client",
		ClientSecret:       "test-secret",
		TokenURL:           tokenServer.URL,
		SMSURL:             smsServer.URL,
		SenderID:           "AMY-MIS",
		CallbackURL:        "https://example.com/callback",
		TokenExpirySeconds: 3600,
	}, testLogger())

	result, err := provider.Send(context.Background(), "+254712345678", "Test message")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !result.Success {
		t.Error("Send should succeed")
	}
	if result.Provider != "optimize" {
		t.Errorf("Provider = %q, want optimize", result.Provider)
	}
}

func TestOptimizeTokenCaching(t *testing.T) {
	callCount := 0
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "cached-token"})
	}))
	defer tokenServer.Close()

	smsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer smsServer.Close()

	provider := NewOptimizeProvider(OptimizeConfig{
		ClientID: "c", ClientSecret: "s",
		TokenURL: tokenServer.URL, SMSURL: smsServer.URL,
		TokenExpirySeconds: 3600,
	}, testLogger())

	// Send twice — token should only be fetched once
	provider.Send(context.Background(), "+254700000001", "msg1")
	provider.Send(context.Background(), "+254700000002", "msg2")

	if callCount != 1 {
		t.Errorf("Token fetched %d times, want 1 (should be cached)", callCount)
	}
}

func TestOptimizeAuthFailure(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer tokenServer.Close()

	provider := NewOptimizeProvider(OptimizeConfig{
		ClientID: "bad", ClientSecret: "bad",
		TokenURL: tokenServer.URL, SMSURL: "http://unused",
	}, testLogger())

	_, err := provider.Send(context.Background(), "+254712345678", "test")
	if err == nil {
		t.Error("Send should fail on auth failure")
	}
}

func TestOptimizeName(t *testing.T) {
	p := NewOptimizeProvider(OptimizeConfig{}, testLogger())
	if p.Name() != "optimize" {
		t.Errorf("Name = %q, want optimize", p.Name())
	}
}

// --- Africa's Talking provider tests ---

func TestAfricasTalkingSendSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apiKey") == "" {
			t.Error("apiKey header should be set")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"SMSMessageData": map[string]interface{}{
				"Message": "Sent to 1/1",
				"Recipients": []map[string]interface{}{
					{"statusCode": 101, "number": "+254712345678", "status": "Success", "messageId": "ATXid_123"},
				},
			},
		})
	}))
	defer server.Close()

	provider := NewAfricasTalkingProvider(AfricasTalkingConfig{
		APIKey:   "test-key",
		Username: "sandbox",
		BaseURL:  server.URL,
	}, testLogger())

	result, err := provider.Send(context.Background(), "+254712345678", "Test")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !result.Success {
		t.Error("Send should succeed")
	}
	if result.MessageID != "ATXid_123" {
		t.Errorf("MessageID = %q, want ATXid_123", result.MessageID)
	}
}

func TestAfricasTalkingSendFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"SMSMessageData": map[string]interface{}{
				"Recipients": []map[string]interface{}{
					{"statusCode": 403, "number": "+254712345678", "status": "InvalidPhoneNumber"},
				},
			},
		})
	}))
	defer server.Close()

	provider := NewAfricasTalkingProvider(AfricasTalkingConfig{
		APIKey: "key", Username: "sandbox", BaseURL: server.URL,
	}, testLogger())

	result, err := provider.Send(context.Background(), "+254712345678", "Test")
	if err == nil {
		t.Error("Send should fail for invalid phone")
	}
	if result.Success {
		t.Error("result.Success should be false")
	}
}

func TestAfricasTalkingName(t *testing.T) {
	p := NewAfricasTalkingProvider(AfricasTalkingConfig{}, testLogger())
	if p.Name() != "africastalking" {
		t.Errorf("Name = %q, want africastalking", p.Name())
	}
}

func TestAfricasTalkingBulkSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"SMSMessageData": map[string]interface{}{
				"Recipients": []map[string]interface{}{
					{"statusCode": 101, "number": "+254700000001", "status": "Success", "messageId": "msg1"},
					{"statusCode": 101, "number": "+254700000002", "status": "Success", "messageId": "msg2"},
				},
			},
		})
	}))
	defer server.Close()

	provider := NewAfricasTalkingProvider(AfricasTalkingConfig{
		APIKey: "key", Username: "sandbox", BaseURL: server.URL,
	}, testLogger())

	results, err := provider.SendBulk(context.Background(),
		[]string{"+254700000001", "+254700000002"}, "Bulk")
	if err != nil {
		t.Fatalf("SendBulk: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results = %d, want 2", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("recipient %s should succeed", r.MessageID)
		}
	}
}
