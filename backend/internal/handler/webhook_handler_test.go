package handler_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/handler"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
)

func TestWebhookHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	webhookRepo := mock.NewWebhookEventRepo()
	walletRepo := mock.NewWalletRepo()

	webhookSvc := service.NewWebhookService(webhookRepo, nil, nil, walletRepo, nil, logger)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)

	r := gin.New()
	r.POST("/webhooks/jambopay", webhookHandler.HandleJamboPay)
	r.POST("/webhooks/perpay", webhookHandler.HandlePerpay)

	t.Run("JamboPay Webhook Success", func(t *testing.T) {
		payload := `{"order_id": "test-order", "status": "COMPLETED"}`
		req := httptest.NewRequest(http.MethodPost, "/webhooks/jambopay", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("JamboPay Webhook Invalid JSON", func(t *testing.T) {
		payload := `{invalid_json}`
		req := httptest.NewRequest(http.MethodPost, "/webhooks/jambopay", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			// json.Unmarshal in ProcessJamboPayWebhook returns an error, which gives 500
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("PerPay Webhook Success", func(t *testing.T) {
		payload := `{"correlation_id": "corr-123", "status": "COMPLETED"}`
		req := httptest.NewRequest(http.MethodPost, "/webhooks/perpay", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}
