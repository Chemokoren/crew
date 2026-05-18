package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/config"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// IntegrationStatus represents the health/config state of an external provider.
type IntegrationStatus struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Configured  bool   `json:"configured"`
	Primary     bool   `json:"primary"`
	Status      string `json:"status"` // "active", "inactive", "unconfigured"
}

// APIKeyEntry is the JSON shape returned when listing API keys.
type APIKeyEntry struct {
	Name    string `json:"name"`
	Key     string `json:"key"`
	Masked  string `json:"masked"`
	Created string `json:"created"`
}

// IntegrationHandler serves integration health, webhook logs, API keys, and toggle endpoints.
type IntegrationHandler struct {
	cfg          *config.Config
	settingsRepo repository.SystemSettingRepository
	webhookRepo  repository.WebhookEventRepository
}

// NewIntegrationHandler creates a new IntegrationHandler.
func NewIntegrationHandler(
	cfg *config.Config,
	settingsRepo repository.SystemSettingRepository,
	webhookRepo repository.WebhookEventRepository,
) *IntegrationHandler {
	return &IntegrationHandler{
		cfg:          cfg,
		settingsRepo: settingsRepo,
		webhookRepo:  webhookRepo,
	}
}

// ListIntegrations returns the health/config status of all external integrations.
// @Summary List integration health status
// @Description Returns real-time status of all external provider integrations
// @Tags Integrations
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/integrations [get]
func (h *IntegrationHandler) ListIntegrations(c *gin.Context) {
	integrations := h.buildIntegrationStatuses()
	SuccessResponse(c, http.StatusOK, integrations)
}

// ToggleIntegration enables or disables a provider via system settings.
// @Summary Toggle integration status
// @Description Enables or disables an integration provider
// @Tags Integrations
// @Accept json
// @Produce json
// @Param slug path string true "Integration slug"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/integrations/{slug}/toggle [put]
func (h *IntegrationHandler) ToggleIntegration(c *gin.Context) {
	slug := c.Param("slug")
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// Persist the toggle as a system setting (integration.<slug>_status)
	key := "integration." + slug + "_status"
	value := "inactive"
	if req.Enabled {
		value = "active"
	}

	setting := &models.SystemSetting{
		Key:       key,
		Value:     value,
		ValueType: "string",
		Category:  "integration",
		Label:     "Integration " + slug + " status",
	}

	if err := h.settingsRepo.Set(c.Request.Context(), setting); err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{
		"slug":    slug,
		"enabled": req.Enabled,
		"message": "Integration " + slug + " " + value,
	})
}

// ListWebhookLogs returns recent webhook events.
// @Summary List recent webhook events
// @Description Returns recent webhook callback events from all providers
// @Tags Integrations
// @Produce json
// @Param limit query int false "Max results" default(50)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/integrations/webhooks [get]
func (h *IntegrationHandler) ListWebhookLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Get recent webhook events from all sources
	var allEvents []webhookLogEntry

	sources := []models.WebhookSource{models.WebhookJamboPay, models.WebhookPerpay, models.WebhookIPRS}
	for _, source := range sources {
		events, err := h.webhookRepo.ListUnprocessed(c.Request.Context(), source, limit)
		if err != nil {
			continue
		}
		for _, e := range events {
			statusCode := 200
			if e.ErrorMessage != "" {
				statusCode = 500
			}
			if e.IsProcessed {
				statusCode = 200
			}
			allEvents = append(allEvents, webhookLogEntry{
				ID:         e.ID.String(),
				Time:       e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Provider:   string(e.Source),
				Event:      e.EventType,
				StatusCode: statusCode,
				Response:   truncateStr(e.ErrorMessage, 100),
				Processed:  e.IsProcessed,
			})
		}
	}

	// Limit total results
	if len(allEvents) > limit {
		allEvents = allEvents[:limit]
	}

	SuccessResponse(c, http.StatusOK, allEvents)
}

// --- API Key Management ---

// ListAPIKeys returns all API keys stored as system settings with the "apikey." prefix.
// @Summary List API keys
// @Description Returns all service API keys (values are masked for security)
// @Tags Integrations
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/integrations/api-keys [get]
func (h *IntegrationHandler) ListAPIKeys(c *gin.Context) {
	settings, err := h.settingsRepo.GetByPrefix(c.Request.Context(), "apikey.")
	if err != nil {
		MapServiceError(c, err)
		return
	}

	keys := make([]APIKeyEntry, 0, len(settings))
	for _, s := range settings {
		keys = append(keys, APIKeyEntry{
			Name:    s.Label,
			Key:     s.Value,
			Masked:  maskAPIKey(s.Value),
			Created: s.CreatedAt.Format(time.RFC3339),
		})
	}

	SuccessResponse(c, http.StatusOK, keys)
}

// GenerateAPIKey creates a new API key and stores it as a system setting.
// @Summary Generate a new API key
// @Description Creates a cryptographically random API key and stores it
// @Tags Integrations
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/admin/integrations/api-keys [post]
func (h *IntegrationHandler) GenerateAPIKey(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "name is required")
		return
	}

	// Generate a cryptographically secure API key (32 bytes = 64 hex chars)
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate key"})
		return
	}
	apiKey := "amy_" + hex.EncodeToString(keyBytes)

	// Store as system setting with apikey. prefix
	slug := sanitizeSlug(req.Name)
	setting := &models.SystemSetting{
		Key:       "apikey." + slug,
		Value:     apiKey,
		ValueType: "string",
		Category:  "apikey",
		Label:     req.Name,
	}

	if err := h.settingsRepo.Set(c.Request.Context(), setting); err != nil {
		MapServiceError(c, err)
		return
	}

	// Return the full key only on creation — subsequent listings will mask it
	SuccessResponse(c, http.StatusCreated, APIKeyEntry{
		Name:    req.Name,
		Key:     apiKey,
		Masked:  maskAPIKey(apiKey),
		Created: time.Now().UTC().Format(time.RFC3339),
	})
}

// RevokeAPIKey deletes an API key by its slug.
// @Summary Revoke an API key
// @Description Permanently deletes an API key
// @Tags Integrations
// @Produce json
// @Param slug path string true "API key slug"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/integrations/api-keys/{slug} [delete]
func (h *IntegrationHandler) RevokeAPIKey(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		BadRequest(c, "API key slug is required")
		return
	}

	key := "apikey." + slug
	if err := h.settingsRepo.Delete(c.Request.Context(), key); err != nil {
		MapServiceError(c, err)
		return
	}

	SuccessResponse(c, http.StatusOK, gin.H{"message": "API key revoked", "slug": slug})
}

// --- Internal helpers ---

func (h *IntegrationHandler) buildIntegrationStatuses() []IntegrationStatus {
	return []IntegrationStatus{
		{
			Slug: "jambopay", Name: "M-Pesa (JamboPay)", Icon: "account_balance", Type: "Payment",
			Description: "STK push, C2B collections, and B2C payouts",
			Enabled:     h.cfg.PaymentJamboPayEnabled,
			Configured:  h.cfg.JamboPayClientID != "",
			Primary:     h.cfg.PaymentPrimaryProvider == "jambopay",
			Status:      resolveIntegrationStatus(h.cfg.PaymentJamboPayEnabled, h.cfg.JamboPayClientID != ""),
		},
		{
			Slug: "sms", Name: "SMS Gateway (Optimize)", Icon: "sms", Type: "Messaging",
			Description: "Optimize SMS provider for OTP and notifications",
			Enabled:     h.cfg.SMSOptimizeEnabled,
			Configured:  h.cfg.SMSClientID != "",
			Primary:     h.cfg.SMSPrimaryProvider == "optimize",
			Status:      resolveIntegrationStatus(h.cfg.SMSOptimizeEnabled, h.cfg.SMSClientID != ""),
		},
		{
			Slug: "africastalking", Name: "Africa's Talking", Icon: "dialpad", Type: "Messaging",
			Description: "Africa's Talking USSD + SMS fallback",
			Enabled:     h.cfg.SMSATEnabled,
			Configured:  h.cfg.ATAPIKey != "",
			Primary:     h.cfg.SMSPrimaryProvider == "africastalking",
			Status:      resolveIntegrationStatus(h.cfg.SMSATEnabled, h.cfg.ATAPIKey != ""),
		},
		{
			Slug: "whatsapp", Name: "WhatsApp Cloud API", Icon: "chat", Type: "Messaging",
			Description: "Meta WhatsApp Business messaging",
			Enabled:     h.cfg.WhatsAppMetaEnabled,
			Configured:  h.cfg.WhatsAppPhoneNumberID != "",
			Primary:     h.cfg.WhatsAppPrimaryProvider == "meta",
			Status:      resolveIntegrationStatus(h.cfg.WhatsAppMetaEnabled, h.cfg.WhatsAppPhoneNumberID != ""),
		},
		{
			Slug: "iprs", Name: "IPRS (KYC)", Icon: "verified_user", Type: "Identity",
			Description: "National ID verification via IPRS API",
			Enabled:     h.cfg.IdentityIPRSEnabled,
			Configured:  h.cfg.IPRSClientID != "",
			Primary:     h.cfg.IdentityPrimaryProvider == "iprs",
			Status:      resolveIntegrationStatus(h.cfg.IdentityIPRSEnabled, h.cfg.IPRSClientID != ""),
		},
		{
			Slug: "perpay", Name: "PerPay Payroll", Icon: "payments", Type: "Payroll",
			Description: "External payroll submission and reconciliation",
			Enabled:     h.cfg.PayrollPerpayEnabled,
			Configured:  h.cfg.PerpayClientID != "",
			Primary:     h.cfg.PayrollPrimaryProvider == "perpay",
			Status:      resolveIntegrationStatus(h.cfg.PayrollPerpayEnabled, h.cfg.PerpayClientID != ""),
		},
		{
			Slug: "email", Name: "Email (SMTP)", Icon: "email", Type: "Messaging",
			Description: "Gmail SMTP for transactional emails",
			Enabled:     h.cfg.EmailGmailEnabled,
			Configured:  h.cfg.EmailHostUser != "",
			Primary:     h.cfg.EmailPrimaryProvider == "gmail",
			Status:      resolveIntegrationStatus(h.cfg.EmailGmailEnabled, h.cfg.EmailHostUser != ""),
		},
		{
			Slug: "minio", Name: "MinIO Storage", Icon: "cloud_upload", Type: "Storage",
			Description: "Object storage for documents and files",
			Enabled:     h.cfg.StorageMinIOEnabled,
			Configured:  h.cfg.MinIOEndpoint != "",
			Primary:     h.cfg.StoragePrimaryProvider == "minio",
			Status:      resolveIntegrationStatus(h.cfg.StorageMinIOEnabled, h.cfg.MinIOEndpoint != ""),
		},
	}
}

func resolveIntegrationStatus(enabled, configured bool) string {
	if !configured {
		return "unconfigured"
	}
	if !enabled {
		return "inactive"
	}
	return "active"
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}

// maskAPIKey shows only the prefix and last 4 chars, masking the rest.
func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return "••••••••"
	}
	return key[:8] + "••••••••" + key[len(key)-4:]
}

// sanitizeSlug creates a lowercase slug from a name, replacing spaces with hyphens.
func sanitizeSlug(name string) string {
	slug := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		switch {
		case c >= 'a' && c <= 'z':
			slug = append(slug, c)
		case c >= 'A' && c <= 'Z':
			slug = append(slug, c+32) // toLower
		case c >= '0' && c <= '9':
			slug = append(slug, c)
		case c == ' ' || c == '-' || c == '_':
			slug = append(slug, '-')
		}
	}
	return string(slug)
}

// webhookLogEntry is the JSON shape returned to the frontend.
type webhookLogEntry struct {
	ID         string `json:"id"`
	Time       string `json:"time"`
	Provider   string `json:"provider"`
	Event      string `json:"event"`
	StatusCode int    `json:"statusCode"`
	Response   string `json:"response"`
	Processed  bool   `json:"processed"`
}

