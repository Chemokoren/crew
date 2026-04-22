package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/handler/dto"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/types"
)

func setupApiTestEnv() (*gin.Engine, *WalletHandler, *CrewHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	crewRepo := mock.NewCrewRepo()
	walletRepo := mock.NewWalletRepo()
	auditSvc := service.NewAuditService(mock.NewAuditRepo(), logger)
	walletSvc := service.NewWalletService(walletRepo, crewRepo, auditSvc, logger)
	crewSvc := service.NewCrewService(crewRepo, nil, logger)

	walletHandler := NewWalletHandler(walletSvc)
	crewHandler := NewCrewHandler(crewSvc)

	return router, walletHandler, crewHandler
}

// mockAuthMiddleware creates a middleware that injects test claims
func mockAuthMiddleware(role types.SystemRole, crewID *uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := &jwt.Claims{
			UserID:     uuid.New(),
			SystemRole: role,
		}
		if crewID != nil {
			claims.CrewMemberID = crewID
		}
		c.Set(middleware.AuthUserKey, claims)
		c.Next()
	}
}

func TestCrewHandler_Create(t *testing.T) {
	router, _, crewHandler := setupApiTestEnv()

	router.POST("/crew", mockAuthMiddleware(types.RoleSystemAdmin, nil), crewHandler.Create)

	reqBody := dto.CreateCrewRequest{
		NationalID: "12345678",
		FirstName:  "Test",
		LastName:   "User",
		Role:       models.RoleDriver,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/crew", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestWalletHandler_GetBalance(t *testing.T) {
	router, walletHandler, crewHandler := setupApiTestEnv()

	// Use CrewService to create a valid crew member, then use its ID for the wallet
	crewID := uuid.New()
	
	router.GET("/wallets/:crew_member_id", mockAuthMiddleware(types.RoleCrewUser, &crewID), walletHandler.GetBalance)

	// In real tests we'd seed the mock DB, but let's just trigger it
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/wallets/"+crewID.String(), nil)

	router.ServeHTTP(w, req)

	// Since we mockAuthMiddleware with RoleCrewUser and matching crewID, it should pass auth
	// But Wallet might not exist, returning 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for missing wallet, got %d", w.Code)
	}
	_ = crewHandler
}
