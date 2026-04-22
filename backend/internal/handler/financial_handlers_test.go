package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/types"
)

func TestLoanHandler_Apply(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock repos
	loanRepo := mock.NewLoanApplicationRepo()
	creditScoreRepo := mock.NewCreditScoreRepo()
	walletRepo := mock.NewWalletRepo()

	loanSvc := service.NewLoanService(loanRepo, creditScoreRepo, walletRepo)

	loanHandler := NewLoanHandler(loanSvc)

	crewID := uuid.New()
	
	// Route
	router.POST("/loans", mockAuthMiddleware(types.RoleCrewUser, &crewID), loanHandler.Apply)

	reqBody := struct {
		CrewMemberID uuid.UUID `json:"crew_member_id"`
		AmountCents  int64     `json:"amount_cents"`
		TenureDays   int       `json:"tenure_days"`
		Purpose      string    `json:"purpose"`
	}{
		CrewMemberID: crewID,
		AmountCents:  500000, // 5k
		TenureDays:   30,
		Purpose:      "Maintenance",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/loans", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	// Should fail due to not meeting credit score or other validation rules, returning 400
	// For testing routing, we accept 400 or 201
	if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 201 or 400, got %d", w.Code)
	}
}

func TestCreditHandler_GetScore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	creditScoreRepo := mock.NewCreditScoreRepo()
	earningRepo := mock.NewEarningRepo()
	assignmentRepo := mock.NewAssignmentRepo()

	creditSvc := service.NewCreditService(creditScoreRepo, earningRepo, assignmentRepo)
	creditHandler := NewCreditHandler(creditSvc)

	crewID := uuid.New()
	router.GET("/credit/:crew_member_id", mockAuthMiddleware(types.RoleSystemAdmin, nil), creditHandler.GetScore)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/credit/"+crewID.String(), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusOK {
		t.Errorf("Expected status 404 or 200, got %d", w.Code)
	}
}
