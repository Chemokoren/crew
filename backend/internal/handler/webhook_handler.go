package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/service"
)

type WebhookHandler struct {
	webhookSvc *service.WebhookService
}

func NewWebhookHandler(webhookSvc *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{webhookSvc: webhookSvc}
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
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}
	defer c.Request.Body.Close()

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
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}
	defer c.Request.Body.Close()

	if err := h.webhookSvc.ProcessPerpayWebhook(c.Request.Context(), payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
