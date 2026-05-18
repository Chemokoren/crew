package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/types"
)

// MaintenanceMode checks if maintenance mode is active and blocks non-admin users
// with a 503 Service Unavailable response. System admins can bypass.
func MaintenanceMode(settingsRepo repository.SystemSettingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Always allow health, auth, and system status endpoints
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/health") ||
			strings.HasPrefix(path, "/api/v1/auth/") ||
			strings.HasPrefix(path, "/api/v1/system/") {
			c.Next()
			return
		}

		// Check if maintenance is active
		setting, err := settingsRepo.Get(c.Request.Context(), "maintenance.active")
		if err != nil || setting == nil || setting.Value != "true" {
			c.Next()
			return
		}

		// Only SYSTEM_ADMIN users can bypass maintenance
		claims := GetClaims(c)
		if claims != nil && claims.SystemRole == types.RoleSystemAdmin {
			c.Next()
			return
		}

		// Get maintenance message
		msg := "System is undergoing scheduled maintenance. Please try again later."
		if msgSetting, err := settingsRepo.Get(c.Request.Context(), "maintenance.message"); err == nil && msgSetting != nil && msgSetting.Value != "" {
			msg = msgSetting.Value
		}

		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MAINTENANCE",
				"message": msg,
			},
		})
	}
}
