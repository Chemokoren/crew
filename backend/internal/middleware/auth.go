package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/types"
)

const (
	// AuthUserKey is the context key for the authenticated user claims.
	AuthUserKey = "auth_user"
)

// JWTAuth validates the Bearer token and injects claims into the Gin context.
func JWTAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortUnauthorized(c, "Authorization header required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			abortUnauthorized(c, "Authorization header must be: Bearer <token>")
			return
		}

		claims, err := jwtManager.VerifyToken(parts[1])
		if err != nil {
			abortUnauthorized(c, "Invalid or expired token")
			return
		}

		c.Set(AuthUserKey, claims)
		c.Next()
	}
}

// RequireRole returns middleware that restricts access to specific system roles.
func RequireRole(roles ...types.SystemRole) gin.HandlerFunc {
	roleSet := make(map[types.SystemRole]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(c *gin.Context) {
		claims, exists := c.Get(AuthUserKey)
		if !exists {
			abortUnauthorized(c, "Authentication required")
			return
		}

		userClaims := claims.(*jwt.Claims)
		if !roleSet[userClaims.SystemRole] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Insufficient permissions",
				},
			})
			return
		}

		c.Next()
	}
}

// GetClaims extracts the JWT claims from the Gin context.
// Returns nil if not authenticated.
func GetClaims(c *gin.Context) *jwt.Claims {
	claims, exists := c.Get(AuthUserKey)
	if !exists {
		return nil
	}
	return claims.(*jwt.Claims)
}

// abortUnauthorized sends a 401 and aborts the chain.
func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "UNAUTHORIZED",
			"message": message,
		},
	})
}
