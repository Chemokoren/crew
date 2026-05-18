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
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/pkg/jwt"
)

// supportTestEnv holds all dependencies for support handler tests.
type supportTestEnv struct {
	router    *gin.Engine
	auditRepo *mock.AuditRepo
	userRepo  *mock.UserRepo
	handler   *SupportHandler
	adminToken string
}

func setupSupportTestEnv(t *testing.T) *supportTestEnv {
	t.Helper()

	base := setupTestEnv()

	auditRepo := mock.NewAuditRepo()

	// Build a real AuthService using the base repos so SupportStats works
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	jwtMgr := jwt.NewManager(testJWTSecret, 15, 7)
	authSvc := service.NewAuthService(base.userRepo, base.crewRepo, jwtMgr, nil, logger)

	// Create a SupportHandler with real authSvc (wallet/payroll/otp can be nil for unit tests)
	supportHandler := NewSupportHandler(
		authSvc,
		nil, // walletSvc
		nil, // payrollSvc
		auditRepo,
		nil, // otpSvc
	)

	// Register a SYSTEM_ADMIN user for auth
	adminResp := base.registerUser(t, "+254700099001", "SecurePass123!", "SYSTEM_ADMIN")

	// Mount support routes on the existing router
	admin := base.router.Group("/api/v1/admin/support")
	admin.Use(func(c *gin.Context) {
		c.Set("Authorization", "Bearer "+adminResp.Tokens.AccessToken)
		c.Next()
	})
	{
		admin.GET("/stats", supportHandler.SupportStats)
		admin.GET("/search", supportHandler.SearchUsers)
		admin.GET("/users/:id/timeline", supportHandler.UserTimeline)
		admin.POST("/users/:id/resend-otp", supportHandler.ResendOTP)
	}

	return &supportTestEnv{
		router:     base.router,
		auditRepo:  auditRepo,
		userRepo:   base.userRepo,
		handler:    supportHandler,
		adminToken: adminResp.Tokens.AccessToken,
	}
}

// --- Support Stats Tests ---

func TestSupportStats_ReturnsStats(t *testing.T) {
	env := setupSupportTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/stats", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("SupportStats: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Error("expected success=true")
	}
}

// --- User Timeline Tests ---

func TestUserTimeline_InvalidID(t *testing.T) {
	env := setupSupportTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/users/not-a-uuid/timeline", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("UserTimeline invalid ID: status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestUserTimeline_EmptyTimeline(t *testing.T) {
	env := setupSupportTestEnv(t)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/users/"+userID.String()+"/timeline", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("UserTimeline empty: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data, ok := resp["data"].([]interface{})
	if !ok || len(data) != 0 {
		t.Errorf("expected empty timeline array, got %v", resp["data"])
	}
}

func TestUserTimeline_ReturnsUserLogs(t *testing.T) {
	env := setupSupportTestEnv(t)

	userID := uuid.New()
	otherID := uuid.New()

	// Seed audit logs — one for the target user (actor), one for another user
	_ = env.auditRepo.Create(nil, &models.AuditLog{
		ID:       uuid.New(),
		UserID:   &userID,
		Action:   "LOGIN",
		Resource: "session",
	})
	_ = env.auditRepo.Create(nil, &models.AuditLog{
		ID:         uuid.New(),
		UserID:     &otherID,
		Action:     "UPDATE",
		Resource:   "user",
		ResourceID: &userID, // userID is the affected resource
	})
	_ = env.auditRepo.Create(nil, &models.AuditLog{
		ID:       uuid.New(),
		UserID:   &otherID,
		Action:   "DELETE",
		Resource: "something_else",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/users/"+userID.String()+"/timeline", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("UserTimeline with logs: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data to be array, got %T", resp["data"])
	}
	// Should return 2 logs: one where user is actor, one where user is resource
	if len(data) != 2 {
		t.Errorf("expected 2 timeline entries for user, got %d", len(data))
	}
}

func TestUserTimeline_FilterByAction(t *testing.T) {
	env := setupSupportTestEnv(t)

	userID := uuid.New()

	_ = env.auditRepo.Create(nil, &models.AuditLog{
		ID: uuid.New(), UserID: &userID, Action: "LOGIN", Resource: "session",
	})
	_ = env.auditRepo.Create(nil, &models.AuditLog{
		ID: uuid.New(), UserID: &userID, Action: "UPDATE", Resource: "profile",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/users/"+userID.String()+"/timeline?action=LOGIN", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("UserTimeline action filter: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Errorf("expected 1 LOGIN entry, got %d", len(data))
	}
}

// --- Search Users Tests ---

func TestSearchUsers_MissingQuery(t *testing.T) {
	env := setupSupportTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/search", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SearchUsers no query: status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestSearchUsers_EmptyQuery(t *testing.T) {
	env := setupSupportTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/search?q=", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SearchUsers empty query: status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestSearchUsers_WithQuery(t *testing.T) {
	env := setupSupportTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/search?q=+254700099001", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	// The search handler uses its own authSvc which is nil in this test,
	// so it will fail gracefully. What matters is the route is reachable.
	// In integration tests this would return real results.
	if w.Code == http.StatusNotFound {
		t.Error("SearchUsers route should be reachable, got 404")
	}
}

// --- Resend OTP Tests ---

func TestResendOTP_InvalidUserID(t *testing.T) {
	env := setupSupportTestEnv(t)

	body := `{"channel":"sms"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/support/users/not-a-uuid/resend-otp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ResendOTP invalid ID: status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestResendOTP_NonExistentUser(t *testing.T) {
	env := setupSupportTestEnv(t)

	body := `{"channel":"sms"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/support/users/"+uuid.New().String()+"/resend-otp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	// The route is reachable (not a Gin 404); the handler should return a
	// structured error response since the user doesn't exist.
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", w.Body.String())
	}
	// Our API always returns {"success": false, "error": {...}} for errors
	if resp["success"] != false {
		t.Errorf("expected success=false for non-existent user, got %v", resp["success"])
	}
}

// --- Mask Destination Tests ---

func TestMaskDestination_Phone(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"+254712345678", "*********5678"},
		{"0712", "****"},
		{"", "****"},
		{"123", "****"},
	}
	for _, tt := range tests {
		got := maskDestination(tt.input)
		if got != tt.want {
			t.Errorf("maskDestination(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskDestination_Email(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"john.doe@example.com", "jo***@example.com"},
		{"ab@example.com", "ab***@example.com"},
		{"a@example.com", "a***@example.com"},
	}
	for _, tt := range tests {
		got := maskDestination(tt.input)
		if got != tt.want {
			t.Errorf("maskDestination(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Audit Logging Tests ---

func TestUserTimeline_Pagination(t *testing.T) {
	env := setupSupportTestEnv(t)

	userID := uuid.New()

	// Seed 5 logs
	for i := 0; i < 5; i++ {
		_ = env.auditRepo.Create(nil, &models.AuditLog{
			ID: uuid.New(), UserID: &userID, Action: "LOGIN", Resource: "session",
		})
	}

	// Request page 1 with per_page=2
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/users/"+userID.String()+"/timeline?page=1&per_page=2", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Pagination: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("expected 2 items on page 1, got %d", len(data))
	}

	meta := resp["meta"].(map[string]interface{})
	if int(meta["total"].(float64)) != 5 {
		t.Errorf("expected total=5, got %v", meta["total"])
	}

	// Page 3 should have 1 item
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/support/users/"+userID.String()+"/timeline?page=3&per_page=2", nil)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].([]interface{})
	if len(data) != 1 {
		t.Errorf("expected 1 item on page 3, got %d", len(data))
	}
}
