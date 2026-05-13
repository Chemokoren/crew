package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pkgjwt "github.com/kibsoft/amy-mis/pkg/jwt"
)

// PermissionChecker is the interface the middleware uses to check permissions.
// Implemented by service.RBACService.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, permKey string) bool
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
		if checker == nil {
			// No checker injected — deny by default
			abortForbidden(c, "Permission system unavailable")
			return
		}

		for _, key := range permKeys {
			if !checker.HasPermission(c.Request.Context(), claims.UserID, claims.OrganizationID, key) {
				abortForbidden(c, "Missing permission: "+key)
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
		if checker == nil {
			abortForbidden(c, "Permission system unavailable")
			return
		}

		for _, key := range permKeys {
			if checker.HasPermission(c.Request.Context(), claims.UserID, claims.OrganizationID, key) {
				c.Next()
				return
			}
		}

		abortForbidden(c, "Insufficient permissions")
	}
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
