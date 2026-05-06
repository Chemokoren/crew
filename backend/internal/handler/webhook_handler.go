package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/service"
)

// maxWebhookBodySize is the maximum allowed webhook payload size (1 MB).
const maxWebhookBodySize = 1 << 20

// ChecksumVerifier is a function that verifies a JamboPay callback checksum.
// Injected from the JamboPayProvider so the handler doesn't import jambopay directly.
type ChecksumVerifier func(ref, amount, checksum string) bool

type WebhookHandler struct {
	webhookSvc      *service.WebhookService
	verifyChecksum  ChecksumVerifier // JamboPay SHA256 checksum verifier
	perpaySecret    string           // PerPay HMAC-SHA256 secret
}

func NewWebhookHandler(
	webhookSvc *service.WebhookService,
	verifyChecksum ChecksumVerifier,
	perpaySecret string,
) *WebhookHandler {
	return &WebhookHandler{
		webhookSvc:     webhookSvc,
		verifyChecksum: verifyChecksum,
		perpaySecret:   perpaySecret,
	}
}

// readWebhookBody reads the payload with a hard size limit to prevent DoS.
func readWebhookBody(c *gin.Context) ([]byte, error) {
	limited := http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodySize)
	defer c.Request.Body.Close()
	return io.ReadAll(limited)
}

// verifyHMAC validates an HMAC-SHA256 signature against the raw payload.
// Used for PerPay which sends an X-Perpay-Signature header.
// Returns true when no secret is configured (development bypass).
func verifyHMAC(payload []byte, signature, secret string) bool {
	if secret == "" {
		return true
	}
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// HandleJamboPay processes JamboPay v2 payout/transfer callbacks.
//
// JamboPay sends a JSON body:
//
//	{"status":"string","amount":"string","ref":"string","orderId":"string","description":"string","checksum":"string"}
//
// Checksum is validated as SHA256(ref + amount + client_id + client_secret).
// JamboPay does NOT use HMAC headers — checksum is embedded in the body.
//
// @Summary HandleJamboPay
// @Description JamboPay v2 payout/transfer callback
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

	// Parse just enough to extract checksum fields
	var cb struct {
		Ref      string `json:"ref"`
		Amount   string `json:"amount"`
		Checksum string `json:"checksum"`
	}
	if err := json.Unmarshal(payload, &cb); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	// Verify SHA256 checksum embedded in the callback body
	// Spec: SHA256(ref + amount + client_id + client_secret)
	if h.verifyChecksum != nil && !h.verifyChecksum(cb.Ref, cb.Amount, cb.Checksum) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid callback checksum"})
		return
	}

	if err := h.webhookSvc.ProcessJamboPayWebhook(c.Request.Context(), payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandlePerpay processes PerPay payroll callbacks.
// PerPay uses an HMAC-SHA256 signature in the X-Perpay-Signature header.
//
// @Summary HandlePerpay
// @Description PerPay payroll callback
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

	signature := c.GetHeader("X-Perpay-Signature")
	if !verifyHMAC(payload, signature, h.perpaySecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
		return
	}

	if err := h.webhookSvc.ProcessPerpayWebhook(c.Request.Context(), payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
