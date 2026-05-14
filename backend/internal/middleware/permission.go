package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/rbac"
	"github.com/kibsoft/amy-mis/internal/service"
	pkgjwt "github.com/kibsoft/amy-mis/pkg/jwt"
)

// PermissionChecker is the interface the middleware uses to check permissions.
// Implemented by service.RBACService.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, permKey string) bool
}

// ContextualPermissionChecker can evaluate request-aware dynamic policies.
type ContextualPermissionChecker interface {
	HasPermissionWithContext(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, permKey string, evalCtx rbac.EvaluationContext) bool
}

// PermissionDeniedAuditor records denied permission checks.
type PermissionDeniedAuditor interface {
	AuditPermissionDenied(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, permKeys []string, method, path, reason, ipAddress, userAgent string)
}

// permCheckerKey is the Gin context key for the permission checker.
const permCheckerKey = "perm_checker"

// InjectPermissionChecker stores the PermissionChecker in Gin context for downstream use.
func InjectPermissionChecker(checker PermissionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(permCheckerKey, checker)
		c.Next()
	}
}

// RequirePermission returns middleware that checks the user has ALL specified permissions.
// Falls back to 403 Forbidden if any permission is missing.
func RequirePermission(permKeys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := getClaims(c)
		if claims == nil {
			abortForbidden(c, "Authentication required")
			return
		}

		checker := getChecker(c)

		for _, key := range permKeys {
			if !hasPermission(c, checker, claims, key) {
				abortForbiddenWithAudit(c, checker, claims, permKeys, "Missing permission: "+key)
				return
			}
		}

		c.Next()
	}
}

// RequireAnyPermission returns middleware that checks the user has at least ONE of the specified permissions.
func RequireAnyPermission(permKeys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := getClaims(c)
		if claims == nil {
			abortForbidden(c, "Authentication required")
			return
		}

		checker := getChecker(c)

		for _, key := range permKeys {
			if hasPermission(c, checker, claims, key) {
				c.Next()
				return
			}
		}

		abortForbiddenWithAudit(c, checker, claims, permKeys, "Insufficient permissions")
	}
}

// hasPermission checks if a user has a specific permission through the dynamic
// RBAC system. The user's system_role is injected into the request context so
// that the RBACService can resolve the matching RBAC system role and its
// database-stored permissions — no hardcoded role-to-permission maps.
func hasPermission(c *gin.Context, checker PermissionChecker, claims *pkgjwt.Claims, key string) bool {
	evalCtx := rbac.EvaluationContext{
		CurrentTime: timeNow(),
		IPAddress:   c.ClientIP(),
		Timezone:    "Africa/Nairobi",
	}

	if checker == nil || claims.UserID == uuid.Nil {
		return false
	}

	// Inject the user's system_role into the context so the RBAC service
	// can resolve system-role permissions from the database.
	ctx := service.SetSystemRoleInContext(c.Request.Context(), claims.SystemRole)

	if contextual, ok := checker.(ContextualPermissionChecker); ok {
		return contextual.HasPermissionWithContext(ctx, claims.UserID, claims.OrganizationID, key, evalCtx)
	}
	return checker.HasPermission(ctx, claims.UserID, claims.OrganizationID, key)
}

func getClaims(c *gin.Context) *pkgjwt.Claims {
	val, exists := c.Get(AuthUserKey)
	if !exists {
		return nil
	}
	claims, ok := val.(*pkgjwt.Claims)
	if !ok {
		return nil
	}
	return claims
}

func getChecker(c *gin.Context) PermissionChecker {
	val, exists := c.Get(permCheckerKey)
	if !exists {
		return nil
	}
	checker, ok := val.(PermissionChecker)
	if !ok {
		return nil
	}
	return checker
}

func abortForbidden(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "FORBIDDEN",
			"message": message,
		},
	})
}

func abortForbiddenWithAudit(c *gin.Context, checker PermissionChecker, claims *pkgjwt.Claims, permKeys []string, message string) {
	if auditor, ok := checker.(PermissionDeniedAuditor); ok && claims != nil && claims.UserID != uuid.Nil {
		auditor.AuditPermissionDenied(
			c.Request.Context(),
			claims.UserID,
			claims.OrganizationID,
			permKeys,
			c.Request.Method,
			c.FullPath(),
			message,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
	}
	abortForbidden(c, message)
}

var timeNow = func() time.Time { return time.Now() }
