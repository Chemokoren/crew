package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/handler"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/jwt"
)

func setupSACCOTestEnv() (*gin.Engine, *mock.SACCOFloatRepo, uuid.UUID, string) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	jwtMgr := jwt.NewManager("test-secret-key-that-is-at-least-32-chars-long!", 15, 7)

	saccoRepo := mock.NewSACCORepo()
	membershipRepo := mock.NewMembershipRepo()
	floatRepo := mock.NewSACCOFloatRepo()

	auditRepo := mock.NewAuditRepo()
	auditSvc := service.NewAuditService(auditRepo, logger)

	saccoSvc := service.NewSACCOService(saccoRepo, membershipRepo, floatRepo, auditSvc, logger)
	saccoHandler := handler.NewSACCOHandler(saccoSvc)

	router := gin.New()
	secured := router.Group("/api/v1")
	secured.Use(middleware.JWTAuth(jwtMgr))

	saccos := secured.Group("/saccos")
	{
		saccos.GET("/:id/float", saccoHandler.GetFloat)
		saccos.POST("/:id/float/credit", saccoHandler.CreditFloat)
		saccos.POST("/:id/float/debit", saccoHandler.DebitFloat)
		saccos.GET("/:id/float/transactions", saccoHandler.ListFloatTransactions)
	}

	// Create a sacco and float directly for testing
	ctx := context.Background()
	sacco := &models.SACCO{
		ID:   uuid.New(),
		Name: "Test SACCO",
	}
	_ = saccoRepo.Create(ctx, sacco)
	_, _ = floatRepo.GetOrCreate(ctx, sacco.ID)

	// Create an admin token
	pair, _ := jwtMgr.GenerateTokenPair(uuid.New(), "+254700000000", "SACCO_ADMIN", nil, nil)
	token := pair.AccessToken

	return router, floatRepo, sacco.ID, token
}

func TestGetFloat(t *testing.T) {
	router, _, saccoID, token := setupSACCOTestEnv()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/saccos/"+saccoID.String()+"/float", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})
	if data["balance_cents"].(float64) != 0 {
		t.Errorf("expected balance 0, got %v", data["balance_cents"])
	}
}

func TestCreditFloat(t *testing.T) {
	router, _, saccoID, token := setupSACCOTestEnv()

	body := `{"amount_cents": 10000, "idempotency_key": "DEP-001-KEY", "reference": "DEP-001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/saccos/"+saccoID.String()+"/float/credit", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})
	if data["amount_cents"].(float64) != 10000 {
		t.Errorf("expected amount 10000, got %v", data["amount_cents"])
	}
}

func TestDebitFloat(t *testing.T) {
	router, _, saccoID, token := setupSACCOTestEnv()

	// First credit to have balance
	bodyCredit := `{"amount_cents": 10000, "idempotency_key": "DEP-001-KEY", "reference": "DEP-001"}`
	reqCredit := httptest.NewRequest(http.MethodPost, "/api/v1/saccos/"+saccoID.String()+"/float/credit", bytes.NewBufferString(bodyCredit))
	reqCredit.Header.Set("Content-Type", "application/json")
	reqCredit.Header.Set("Authorization", "Bearer "+token)
	wCredit := httptest.NewRecorder()
	router.ServeHTTP(wCredit, reqCredit)

	// Now debit
	bodyDebit := `{"amount_cents": 3000, "idempotency_key": "PAY-001-KEY", "reference": "PAY-001"}`
	reqDebit := httptest.NewRequest(http.MethodPost, "/api/v1/saccos/"+saccoID.String()+"/float/debit", bytes.NewBufferString(bodyDebit))
	reqDebit.Header.Set("Content-Type", "application/json")
	reqDebit.Header.Set("Authorization", "Bearer "+token)
	wDebit := httptest.NewRecorder()
	router.ServeHTTP(wDebit, reqDebit)

	if wDebit.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", wDebit.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(wDebit.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})
	if data["balance_after_cents"].(float64) != 7000 {
		t.Errorf("expected balance 7000, got %v", data["balance_after_cents"])
	}
}

func TestListFloatTransactions(t *testing.T) {
	router, _, saccoID, token := setupSACCOTestEnv()

	// Credit
	bodyCredit := `{"amount_cents": 10000, "idempotency_key": "DEP-002-KEY", "reference": "DEP-002"}`
	reqCredit := httptest.NewRequest(http.MethodPost, "/api/v1/saccos/"+saccoID.String()+"/float/credit", bytes.NewBufferString(bodyCredit))
	reqCredit.Header.Set("Content-Type", "application/json")
	reqCredit.Header.Set("Authorization", "Bearer "+token)
	wCredit := httptest.NewRecorder()
	router.ServeHTTP(wCredit, reqCredit)

	// List
	req := httptest.NewRequest(http.MethodGet, "/api/v1/saccos/"+saccoID.String()+"/float/transactions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	dataRaw, ok := resp["data"]
	if !ok || dataRaw == nil {
		t.Fatalf("expected data in response, got: %s", w.Body.String())
	}
	data, ok := dataRaw.([]interface{})
	if !ok {
		t.Fatalf("expected data to be []interface{}, got %T in response: %s", dataRaw, w.Body.String())
	}
	
	if len(data) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(data))
	}
}
