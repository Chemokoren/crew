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
	"github.com/kibsoft/amy-mis/internal/handler/dto"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/types"
)

const testJWTSecret = "test-secret-key-that-is-at-least-32-chars-long!"

func init() {
	gin.SetMode(gin.TestMode)
}

// testEnv holds all test dependencies.
type testEnv struct {
	router     *gin.Engine
	jwtManager *jwt.Manager
	userRepo   *mock.UserRepo
	crewRepo   *mock.CrewRepo
	walletRepo *mock.WalletRepo
}

func setupTestEnv() *testEnv {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtMgr := jwt.NewManager(testJWTSecret, 15, 7)

	userRepo := mock.NewUserRepo()
	crewRepo := mock.NewCrewRepo()
	walletRepo := mock.NewWalletRepo()

	authSvc := service.NewAuthService(userRepo, crewRepo, jwtMgr, nil, logger)
	crewSvc := service.NewCrewService(crewRepo, nil, logger) // Passing nil for identity.Provider in tests
	walletSvc := service.NewWalletService(walletRepo, crewRepo, logger)

	authHandler := NewAuthHandler(authSvc)
	crewHandler := NewCrewHandler(crewSvc)
	walletHandler := NewWalletHandler(walletSvc)

	router := gin.New()

	// Public
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
	}

	// Secured
	secured := router.Group("/api/v1")
	secured.Use(middleware.JWTAuth(jwtMgr))
	{
		secured.GET("/auth/me", authHandler.Me)

		crew := secured.Group("/crew")
		crew.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			crew.POST("", crewHandler.Create)
			crew.GET("", crewHandler.List)
			crew.GET("/:id", crewHandler.GetByID)
		}

		wallets := secured.Group("/wallets")
		{
			wallets.GET("/:crew_member_id", walletHandler.GetBalance)
			wallets.POST("/credit", walletHandler.Credit)
		}
	}

	return &testEnv{
		router:     router,
		jwtManager: jwtMgr,
		userRepo:   userRepo,
		crewRepo:   crewRepo,
		walletRepo: walletRepo,
	}
}

func (e *testEnv) registerUser(t *testing.T, phone, password string, role types.SystemRole) dto.AuthResponse {
	t.Helper()
	body := dto.RegisterRequest{
		Phone:    phone,
		Password: password,
		Role:     role,
	}
	if role == types.RoleCrewUser {
		body.FirstName = "Test"
		body.LastName = "User"
		body.NationalID = "12345678"
		body.CrewRole = models.RoleDriver
	}

	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("register failed: status=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data dto.AuthResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Data
}

// --- Auth Handler Tests ---

func TestRegisterEndpoint(t *testing.T) {
	env := setupTestEnv()

	body := `{"phone":"+254712345678","password":"SecurePass123!","role":"SACCO_ADMIN"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Error("expected success=true")
	}
	data := resp["data"].(map[string]interface{})
	if data["tokens"] == nil {
		t.Error("expected tokens in response")
	}
}

func TestRegisterDuplicatePhone(t *testing.T) {
	env := setupTestEnv()

	body := `{"phone":"+254712345678","password":"SecurePass123!","role":"SACCO_ADMIN"}`

	// First registration
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	// Second registration — same phone
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 Conflict; body = %s", w.Code, w.Body.String())
	}
}

func TestRegisterMissingFields(t *testing.T) {
	env := setupTestEnv()

	body := `{"phone":"+254712345678"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestLoginEndpoint(t *testing.T) {
	env := setupTestEnv()
	env.registerUser(t, "+254712345678", "SecurePass123!", types.RoleSaccoAdmin)

	body := `{"phone":"+254712345678","password":"SecurePass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	tokens := data["tokens"].(map[string]interface{})
	if tokens["access_token"] == "" {
		t.Error("expected access_token")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	env := setupTestEnv()
	env.registerUser(t, "+254712345678", "SecurePass123!", types.RoleSaccoAdmin)

	body := `{"phone":"+254712345678","password":"WrongPassword!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401; body = %s", w.Code, w.Body.String())
	}
}

func TestRefreshEndpoint(t *testing.T) {
	env := setupTestEnv()
	authResp := env.registerUser(t, "+254712345678", "SecurePass123!", types.RoleSaccoAdmin)

	body, _ := json.Marshal(dto.RefreshRequest{RefreshToken: authResp.Tokens.RefreshToken})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestMeEndpoint(t *testing.T) {
	env := setupTestEnv()
	authResp := env.registerUser(t, "+254712345678", "SecurePass123!", types.RoleSaccoAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+authResp.Tokens.AccessToken)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["phone"] != "+254712345678" {
		t.Errorf("phone = %v, want +254712345678", data["phone"])
	}
}

func TestMeWithoutToken(t *testing.T) {
	env := setupTestEnv()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestMeWithInvalidToken(t *testing.T) {
	env := setupTestEnv()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- RBAC Tests ---

func TestCrewEndpointRequiresAdminRole(t *testing.T) {
	env := setupTestEnv()

	// Register as CREW user (should NOT have access to /crew endpoints)
	crewAuth := env.registerUser(t, "+254700000001", "SecurePass123!", types.RoleCrewUser)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/crew", nil)
	req.Header.Set("Authorization", "Bearer "+crewAuth.Tokens.AccessToken)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("CREW user accessing /crew: status = %d, want 403", w.Code)
	}
}

func TestCrewEndpointAllowsSaccoAdmin(t *testing.T) {
	env := setupTestEnv()

	adminAuth := env.registerUser(t, "+254700000002", "SecurePass123!", types.RoleSaccoAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/crew", nil)
	req.Header.Set("Authorization", "Bearer "+adminAuth.Tokens.AccessToken)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("SACCO_ADMIN accessing /crew: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

// --- Crew Handler Tests ---

func TestCreateCrewMember(t *testing.T) {
	env := setupTestEnv()
	adminAuth := env.registerUser(t, "+254700000002", "SecurePass123!", types.RoleSaccoAdmin)

	body := `{"national_id":"87654321","first_name":"Jane","last_name":"Wanjiku","role":"CONDUCTOR"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/crew", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminAuth.Tokens.AccessToken)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["full_name"] != "Jane Wanjiku" {
		t.Errorf("full_name = %v, want 'Jane Wanjiku'", data["full_name"])
	}
	if data["kyc_status"] != "PENDING" {
		t.Errorf("kyc_status = %v, want PENDING", data["kyc_status"])
	}
}
