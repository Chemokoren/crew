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

func setupApiTestEnv() (*gin.Engine, *WalletHandler, *CrewHandler, *AssignmentHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	crewRepo := mock.NewCrewRepo()
	walletRepo := mock.NewWalletRepo()
	assignmentRepo := mock.NewAssignmentRepo()
	earningRepo := mock.NewEarningRepo()
	notifRepo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()

	auditSvc := service.NewAuditService(mock.NewAuditRepo(), logger)
	walletSvc := service.NewWalletService(walletRepo, crewRepo, auditSvc, logger)
	crewSvc := service.NewCrewService(crewRepo, nil, nil, logger)
	prefRepo := mock.NewNotificationPreferenceRepo()
	notifSvc := service.NewNotificationService(notifRepo, prefRepo, userRepo, nil, logger)
	assignmentSvc := service.NewAssignmentService(assignmentRepo, earningRepo, walletSvc, notifSvc, nil, logger)

	walletHandler := NewWalletHandler(walletSvc, 10000)
	crewHandler := NewCrewHandler(crewSvc, notifSvc)
	assignmentHandler := NewAssignmentHandler(assignmentSvc)

	return router, walletHandler, crewHandler, assignmentHandler
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
	router, _, crewHandler, _ := setupApiTestEnv()

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

func TestCrewHandler_GetByID(t *testing.T) {
	router, _, crewHandler, _ := setupApiTestEnv()
	router.GET("/crew/:id", mockAuthMiddleware(types.RoleSystemAdmin, nil), crewHandler.GetByID)

	crewID := uuid.New()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/crew/"+crewID.String(), nil)
	router.ServeHTTP(w, req)

	// Will return 404 since it's not seeded, but tests the routing and execution
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestWalletHandler_GetBalance(t *testing.T) {
	router, walletHandler, crewHandler, _ := setupApiTestEnv()

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

func TestAssignmentHandler_Create(t *testing.T) {
	router, _, _, assignmentHandler := setupApiTestEnv()
	router.POST("/assignments", mockAuthMiddleware(types.RoleSystemAdmin, nil), assignmentHandler.Create)

	vid := uuid.New()
	reqBody := dto.CreateAssignmentRequest{
		CrewMemberID:     uuid.New(),
		VehicleID:        &vid,
		OrganizationID:          uuid.New(),
		ShiftDate:        "2023-10-10",
		ShiftStart:       "2023-10-10T08:00:00Z",
		EarningModel:     models.EarningFixed,
		FixedAmountCents: 1000,
		WorkType:         models.WorkTypeShift,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/assignments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

// --- KYC Unverification Handler Tests ---

func TestCrewHandler_UpdateKYC_Verify(t *testing.T) {
	router, _, crewHandler, _ := setupApiTestEnv()
	router.POST("/crew", mockAuthMiddleware(types.RoleSystemAdmin, nil), crewHandler.Create)
	router.PUT("/crew/:id/kyc", mockAuthMiddleware(types.RoleSystemAdmin, nil), crewHandler.UpdateKYC)

	// Create a crew member first
	createBody, _ := json.Marshal(dto.CreateCrewRequest{
		NationalID: "98765432",
		FirstName:  "Hannah",
		LastName:   "Wambui",
		Role:       models.RoleDriver,
	})
	cw := httptest.NewRecorder()
	cReq, _ := http.NewRequest("POST", "/crew", bytes.NewBuffer(createBody))
	cReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(cw, cReq)

	if cw.Code != http.StatusCreated {
		t.Fatalf("Create crew failed: status=%d body=%s", cw.Code, cw.Body.String())
	}

	var createResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(cw.Body.Bytes(), &createResp)

	// Verify KYC
	kycBody, _ := json.Marshal(dto.UpdateKYCRequest{KYCStatus: models.KYCVerified})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/crew/"+createResp.Data.ID+"/kyc", bytes.NewBuffer(kycBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Verify KYC failed: status=%d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["kyc_status"] != "VERIFIED" {
		t.Errorf("expected kyc_status=VERIFIED, got %v", data["kyc_status"])
	}
}

func TestCrewHandler_UpdateKYC_UnverifyWithReason(t *testing.T) {
	router, _, crewHandler, _ := setupApiTestEnv()
	router.POST("/crew", mockAuthMiddleware(types.RoleSystemAdmin, nil), crewHandler.Create)
	router.PUT("/crew/:id/kyc", mockAuthMiddleware(types.RoleSystemAdmin, nil), crewHandler.UpdateKYC)

	// Create
	createBody, _ := json.Marshal(dto.CreateCrewRequest{
		NationalID: "44443333",
		FirstName:  "Ian",
		LastName:   "Kamau",
		Role:       models.RoleConductor,
	})
	cw := httptest.NewRecorder()
	cReq, _ := http.NewRequest("POST", "/crew", bytes.NewBuffer(createBody))
	cReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(cw, cReq)
	var createResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(cw.Body.Bytes(), &createResp)

	// Verify first
	kycBody, _ := json.Marshal(dto.UpdateKYCRequest{KYCStatus: models.KYCVerified})
	vw := httptest.NewRecorder()
	vReq, _ := http.NewRequest("PUT", "/crew/"+createResp.Data.ID+"/kyc", bytes.NewBuffer(kycBody))
	vReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(vw, vReq)

	// Now unverify with reason
	unverifyBody, _ := json.Marshal(map[string]string{
		"kyc_status": "PENDING",
		"reason":     "Expired ID document — requires re-upload",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/crew/"+createResp.Data.ID+"/kyc", bytes.NewBuffer(unverifyBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Unverify KYC failed: status=%d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["kyc_status"] != "PENDING" {
		t.Errorf("expected kyc_status=PENDING, got %v", data["kyc_status"])
	}
	if data["kyc_verified_at"] != nil {
		t.Error("expected kyc_verified_at to be nil after unverification")
	}
}

func TestCrewHandler_UpdateKYC_RejectWithReason(t *testing.T) {
	router, _, crewHandler, _ := setupApiTestEnv()
	router.POST("/crew", mockAuthMiddleware(types.RoleSystemAdmin, nil), crewHandler.Create)
	router.PUT("/crew/:id/kyc", mockAuthMiddleware(types.RoleSystemAdmin, nil), crewHandler.UpdateKYC)

	// Create
	createBody, _ := json.Marshal(dto.CreateCrewRequest{
		NationalID: "77778888",
		FirstName:  "Jane",
		LastName:   "Njoki",
		Role:       models.RoleDriver,
	})
	cw := httptest.NewRecorder()
	cReq, _ := http.NewRequest("POST", "/crew", bytes.NewBuffer(createBody))
	cReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(cw, cReq)
	var createResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(cw.Body.Bytes(), &createResp)

	// Reject with reason
	rejectBody, _ := json.Marshal(map[string]string{
		"kyc_status": "REJECTED",
		"reason":     "Blurry documents — cannot verify identity",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/crew/"+createResp.Data.ID+"/kyc", bytes.NewBuffer(rejectBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Reject KYC failed: status=%d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["kyc_status"] != "REJECTED" {
		t.Errorf("expected kyc_status=REJECTED, got %v", data["kyc_status"])
	}
}
