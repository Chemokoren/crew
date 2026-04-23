package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/service"
)

// maxWebhookBodySize is the maximum allowed webhook payload size (1 MB).
const maxWebhookBodySize = 1 << 20

type WebhookHandler struct {
	webhookSvc     *service.WebhookService
	jamboPaySecret string
	perpaySecret   string
}

func NewWebhookHandler(webhookSvc *service.WebhookService, jamboPaySecret, perpaySecret string) *WebhookHandler {
	return &WebhookHandler{
		webhookSvc:     webhookSvc,
		jamboPaySecret: jamboPaySecret,
		perpaySecret:   perpaySecret,
	}
}

// verifySignature validates an HMAC-SHA256 signature against the payload.
// Returns true if:
//   - No secret is configured (development mode, gracefully skip verification)
//   - The provided signature matches the expected HMAC
func verifySignature(payload []byte, signature, secret string) bool {
	if secret == "" {
		return true // Skip verification if no secret configured (development)
	}
	if signature == "" {
		return false // Signature required when secret is configured
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// readWebhookBody reads the payload with a hard size limit to prevent DoS.
func readWebhookBody(c *gin.Context) ([]byte, error) {
	limited := http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodySize)
	defer c.Request.Body.Close()
	return io.ReadAll(limited)
}

// HandleJamboPay godoc
// @Summary HandleJamboPay
// @Description HandleJamboPay WebhookHandler
// @Tags Webhook
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/webhooks/jambopay [post]
func (h *WebhookHandler) HandleJamboPay(c *gin.Context) {
	payload, err := readWebhookBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body or payload too large"})
		return
	}

	// Verify HMAC-SHA256 signature from JamboPay
	signature := c.GetHeader("X-JamboPay-Signature")
	if !verifySignature(payload, signature, h.jamboPaySecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	if err := h.webhookSvc.ProcessJamboPayWebhook(c.Request.Context(), payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandlePerpay godoc
// @Summary HandlePerpay
// @Description HandlePerpay WebhookHandler
// @Tags Webhook
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/webhooks/perpay [post]
func (h *WebhookHandler) HandlePerpay(c *gin.Context) {
	payload, err := readWebhookBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body or payload too large"})
		return
	}

	// Verify HMAC-SHA256 signature from PerPay
	signature := c.GetHeader("X-Perpay-Signature")
	if !verifySignature(payload, signature, h.perpaySecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	if err := h.webhookSvc.ProcessPerpayWebhook(c.Request.Context(), payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
