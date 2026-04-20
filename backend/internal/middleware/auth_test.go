package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/types"
)

func init() { gin.SetMode(gin.TestMode) }

const testSecret = "test-secret-key-that-is-at-least-32-chars-long!"

func TestJWTAuthMissingHeader(t *testing.T) {
	jwtMgr := jwt.NewManager(testSecret, 15, 7)
	r := gin.New()
	r.Use(JWTAuth(jwtMgr))
	r.GET("/t", func(c *gin.Context) { c.JSON(200, nil) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/t", nil))
	if w.Code != 401 { t.Errorf("got %d want 401", w.Code) }
}

func TestJWTAuthInvalidToken(t *testing.T) {
	jwtMgr := jwt.NewManager(testSecret, 15, 7)
	r := gin.New()
	r.Use(JWTAuth(jwtMgr))
	r.GET("/t", func(c *gin.Context) { c.JSON(200, nil) })
	req := httptest.NewRequest("GET", "/t", nil)
	req.Header.Set("Authorization", "Bearer bad.token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("got %d want 401", w.Code) }
}

func TestJWTAuthValid(t *testing.T) {
	jwtMgr := jwt.NewManager(testSecret, 15, 7)
	r := gin.New()
	r.Use(JWTAuth(jwtMgr))
	r.GET("/t", func(c *gin.Context) {
		cl := GetClaims(c)
		if cl == nil { t.Fatal("nil claims") }
		c.JSON(200, nil)
	})
	tok, _ := jwtMgr.GenerateTokenPair([16]byte{1}, "+254", types.RoleSaccoAdmin, nil, nil)
	req := httptest.NewRequest("GET", "/t", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Errorf("got %d want 200", w.Code) }
}

func TestRequireRoleDenied(t *testing.T) {
	jwtMgr := jwt.NewManager(testSecret, 15, 7)
	r := gin.New()
	r.Use(JWTAuth(jwtMgr))
	r.Use(RequireRole(types.RoleSystemAdmin))
	r.GET("/t", func(c *gin.Context) { c.JSON(200, nil) })
	tok, _ := jwtMgr.GenerateTokenPair([16]byte{1}, "+254", types.RoleCrewUser, nil, nil)
	req := httptest.NewRequest("GET", "/t", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 { t.Errorf("got %d want 403", w.Code) }
}

func TestRequireRoleAllowed(t *testing.T) {
	jwtMgr := jwt.NewManager(testSecret, 15, 7)
	r := gin.New()
	r.Use(JWTAuth(jwtMgr))
	r.Use(RequireRole(types.RoleSaccoAdmin))
	r.GET("/t", func(c *gin.Context) { c.JSON(200, nil) })
	tok, _ := jwtMgr.GenerateTokenPair([16]byte{1}, "+254", types.RoleSaccoAdmin, nil, nil)
	req := httptest.NewRequest("GET", "/t", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Errorf("got %d want 200", w.Code) }
}
