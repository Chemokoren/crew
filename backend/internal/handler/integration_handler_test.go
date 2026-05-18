package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/config"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestIntegrationHandler creates a handler with a sample config and mock repos.
func newTestIntegrationHandler() (*IntegrationHandler, *mock.SystemSettingRepo, *mock.WebhookEventRepo) {
	cfg := &config.Config{
		PaymentJamboPayEnabled:   true,
		JamboPayClientID:         "client_id",
		PaymentPrimaryProvider:   "jambopay",
		SMSOptimizeEnabled:       true,
		SMSClientID:              "sms_client",
		SMSPrimaryProvider:       "optimize",
		SMSATEnabled:             false,
		ATAPIKey:                 "",
		WhatsAppMetaEnabled:      false,
		WhatsAppPhoneNumberID:    "",
		WhatsAppPrimaryProvider:  "meta",
		IdentityIPRSEnabled:      true,
		IPRSClientID:             "iprs_client",
		IdentityPrimaryProvider:  "iprs",
		PayrollPerpayEnabled:     true,
		PerpayClientID:           "perpay_client",
		PayrollPrimaryProvider:   "perpay",
		EmailGmailEnabled:        true,
		EmailHostUser:            "admin@example.com",
		EmailPrimaryProvider:     "gmail",
		StorageMinIOEnabled:      true,
		MinIOEndpoint:            "localhost:9000",
		StoragePrimaryProvider:   "minio",
	}
	settingsRepo := mock.NewSystemSettingRepo()
	webhookRepo := mock.NewWebhookEventRepo()

	h := NewIntegrationHandler(cfg, settingsRepo, webhookRepo)
	return h, settingsRepo, webhookRepo
}

func setupIntegrationRouter(h *IntegrationHandler) *gin.Engine {
	r := gin.New()
	admin := r.Group("/api/v1/admin")
	{
		admin.GET("/integrations", h.ListIntegrations)
		admin.PUT("/integrations/:slug/toggle", h.ToggleIntegration)
		admin.GET("/integrations/webhooks", h.ListWebhookLogs)
		admin.GET("/integrations/api-keys", h.ListAPIKeys)
		admin.POST("/integrations/api-keys", h.GenerateAPIKey)
		admin.DELETE("/integrations/api-keys/:slug", h.RevokeAPIKey)
	}
	return r
}

// --- Integration Listing Tests ---

func TestListIntegrations(t *testing.T) {
	h, _, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/integrations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                `json:"success"`
		Data    []IntegrationStatus `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if len(resp.Data) != 8 {
		t.Fatalf("expected 8 integrations, got %d", len(resp.Data))
	}

	// Check JamboPay is active and primary
	jambopay := resp.Data[0]
	if jambopay.Slug != "jambopay" {
		t.Errorf("expected first integration slug=jambopay, got %s", jambopay.Slug)
	}
	if jambopay.Status != "active" {
		t.Errorf("expected jambopay status=active, got %s", jambopay.Status)
	}
	if !jambopay.Primary {
		t.Error("expected jambopay to be primary")
	}
	if !jambopay.Enabled {
		t.Error("expected jambopay to be enabled")
	}

	// Check Africa's Talking is unconfigured (no API key in test config)
	at := resp.Data[2]
	if at.Slug != "africastalking" {
		t.Errorf("expected third integration slug=africastalking, got %s", at.Slug)
	}
	if at.Status != "unconfigured" {
		t.Errorf("expected africastalking status=unconfigured, got %s", at.Status)
	}
}

func TestListIntegrationsReturnsCorrectStatusTypes(t *testing.T) {
	h, _, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/integrations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Data []IntegrationStatus `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	validStatuses := map[string]bool{"active": true, "inactive": true, "unconfigured": true}
	for _, p := range resp.Data {
		if !validStatuses[p.Status] {
			t.Errorf("integration %s has invalid status: %s", p.Slug, p.Status)
		}
	}
}

// --- Toggle Integration Tests ---

func TestToggleIntegration(t *testing.T) {
	h, settingsRepo, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	// Disable SMS integration
	body := `{"enabled": false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/integrations/sms/toggle", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Slug    string `json:"slug"`
			Enabled bool   `json:"enabled"`
			Message string `json:"message"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.Slug != "sms" {
		t.Errorf("slug = %s, want sms", resp.Data.Slug)
	}
	if resp.Data.Enabled != false {
		t.Error("expected enabled=false")
	}

	// Verify the setting was persisted
	setting, err := settingsRepo.Get(nil, "integration.sms_status")
	if err != nil {
		t.Fatalf("expected setting to be stored, got err: %v", err)
	}
	if setting.Value != "inactive" {
		t.Errorf("stored value = %s, want 'inactive'", setting.Value)
	}
}

func TestToggleIntegrationEnable(t *testing.T) {
	h, _, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	body := `{"enabled": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/integrations/jambopay/toggle", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Enabled bool `json:"enabled"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Data.Enabled {
		t.Error("expected enabled=true")
	}
}

func TestToggleIntegrationBadRequest(t *testing.T) {
	h, _, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	// Invalid JSON
	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/integrations/sms/toggle", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// --- Webhook Logs Tests ---

func TestListWebhookLogs_Empty(t *testing.T) {
	h, _, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/integrations/webhooks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestListWebhookLogs_WithEvents(t *testing.T) {
	h, _, webhookRepo := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	// Seed some webhook events
	webhookRepo.Create(nil, &models.WebhookEvent{
		Source:    models.WebhookJamboPay,
		EventType: "payment.received",
		ExternalRef: "JP-001",
		Payload:    json.RawMessage(`{"amount":5000}`),
	})
	webhookRepo.Create(nil, &models.WebhookEvent{
		Source:    models.WebhookIPRS,
		EventType: "kyc.verified",
		ExternalRef: "IPRS-002",
		Payload:    json.RawMessage(`{"status":"approved"}`),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/integrations/webhooks?limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []webhookLogEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data) < 2 {
		t.Fatalf("expected at least 2 webhook events, got %d", len(resp.Data))
	}
}

// --- API Key Tests ---

func TestGenerateAPIKey(t *testing.T) {
	h, _, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	body := `{"name": "Mobile App"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/integrations/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool       `json:"success"`
		Data    APIKeyEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Data.Name != "Mobile App" {
		t.Errorf("name = %s, want 'Mobile App'", resp.Data.Name)
	}
	if !strings.HasPrefix(resp.Data.Key, "amy_") {
		t.Errorf("key should start with 'amy_', got %s", resp.Data.Key[:8])
	}
	if len(resp.Data.Key) != 68 { // amy_ (4) + 64 hex chars
		t.Errorf("key length = %d, want 68", len(resp.Data.Key))
	}
	if resp.Data.Masked == resp.Data.Key {
		t.Error("masked should not equal the full key")
	}
}

func TestGenerateAPIKey_MissingName(t *testing.T) {
	h, _, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/integrations/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestListAPIKeys(t *testing.T) {
	h, settingsRepo, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	// Seed API keys via the settings repo
	settingsRepo.Set(nil, &models.SystemSetting{
		Key:       "apikey.mobile-app",
		Value:     "amy_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		ValueType: "string",
		Category:  "apikey",
		Label:     "Mobile App",
	})
	settingsRepo.Set(nil, &models.SystemSetting{
		Key:       "apikey.web-dashboard",
		Value:     "amy_fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		ValueType: "string",
		Category:  "apikey",
		Label:     "Web Dashboard",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/integrations/api-keys", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []APIKeyEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 API keys, got %d", len(resp.Data))
	}

	// Keys should be masked
	for _, k := range resp.Data {
		if !strings.Contains(k.Masked, "••••••••") {
			t.Errorf("key %s should be masked, got %s", k.Name, k.Masked)
		}
	}
}

func TestRevokeAPIKey(t *testing.T) {
	h, settingsRepo, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	// Seed a key
	settingsRepo.Set(nil, &models.SystemSetting{
		Key:       "apikey.test-key",
		Value:     "amy_test1234567890test1234567890test1234567890test1234567890abcdefgh",
		ValueType: "string",
		Category:  "apikey",
		Label:     "Test Key",
	})

	// Verify it exists
	keys, _ := settingsRepo.GetByPrefix(nil, "apikey.")
	if len(keys) != 1 {
		t.Fatalf("expected 1 key before revoke, got %d", len(keys))
	}

	// Revoke it
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/integrations/api-keys/test-key", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	keys, _ = settingsRepo.GetByPrefix(nil, "apikey.")
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys after revoke, got %d", len(keys))
	}
}

func TestRevokeAPIKey_NotFound(t *testing.T) {
	h, _, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/integrations/api-keys/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 since the key doesn't exist
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body = %s", w.Code, w.Body.String())
	}
}

func TestGenerateAndListAPIKey_Roundtrip(t *testing.T) {
	h, _, _ := newTestIntegrationHandler()
	r := setupIntegrationRouter(h)

	// 1. Generate a key
	body := `{"name": "Integration Service"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/integrations/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("generate: status = %d, want 201; body = %s", w.Code, w.Body.String())
	}

	var genResp struct {
		Data APIKeyEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &genResp)
	fullKey := genResp.Data.Key

	// 2. List keys — should contain the generated key (masked)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/integrations/api-keys", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var listResp struct {
		Data []APIKeyEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &listResp)

	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 key, got %d", len(listResp.Data))
	}
	if listResp.Data[0].Name != "Integration Service" {
		t.Errorf("name = %s, want 'Integration Service'", listResp.Data[0].Name)
	}
	// Listed key should be the full key (since it's stored in settings)
	if listResp.Data[0].Key != fullKey {
		t.Errorf("listed key should match generated key")
	}

	// 3. Revoke the key
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/integrations/api-keys/integration-service", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("revoke: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	// 4. List should be empty
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/integrations/api-keys", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Data) != 0 {
		t.Fatalf("expected 0 keys after revoke, got %d", len(listResp.Data))
	}
}

// --- Helper Function Tests ---

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"amy_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "amy_0123••••••••cdef"},
		{"short", "••••••••"},
		{"exactly12ch", "••••••••"},
		{"amy_longerthan12", "amy_long••••••••an12"},
	}

	for _, tt := range tests {
		got := maskAPIKey(tt.input)
		if got != tt.expected {
			t.Errorf("maskAPIKey(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSanitizeSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Mobile App", "mobile-app"},
		{"Web Dashboard", "web-dashboard"},
		{"Integration Service", "integration-service"},
		{"test_key", "test-key"},
		{"UPPER CASE", "upper-case"},
		{"special@#$chars", "specialchars"},
		{"key-123", "key-123"},
	}

	for _, tt := range tests {
		got := sanitizeSlug(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeSlug(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestResolveIntegrationStatus(t *testing.T) {
	tests := []struct {
		enabled    bool
		configured bool
		expected   string
	}{
		{true, true, "active"},
		{false, true, "inactive"},
		{true, false, "unconfigured"},
		{false, false, "unconfigured"},
	}

	for _, tt := range tests {
		got := resolveIntegrationStatus(tt.enabled, tt.configured)
		if got != tt.expected {
			t.Errorf("resolveIntegrationStatus(%v, %v) = %q, want %q", tt.enabled, tt.configured, got, tt.expected)
		}
	}
}
