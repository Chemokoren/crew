package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
)

// AdminHandler provides system administration endpoints.
type AdminHandler struct {
	authSvc     *service.AuthService
	notifSvc    *service.NotificationService
	auditRepo   repository.AuditLogRepository
	statuteRepo repository.StatutoryRateRepository
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(authSvc *service.AuthService, notifSvc *service.NotificationService, auditRepo repository.AuditLogRepository, statuteRepo repository.StatutoryRateRepository) *AdminHandler {
	return &AdminHandler{authSvc: authSvc, notifSvc: notifSvc, auditRepo: auditRepo, statuteRepo: statuteRepo}
}

// SystemStats godoc
// @Summary SystemStats
// @Description Returns system-wide statistics dashboard
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/stats [get]
func (h *AdminHandler) SystemStats(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.authSvc.GetSystemStats(ctx)
	if err != nil {
		InternalError(c, "Failed to retrieve stats")
		return
	}
	SuccessResponse(c, http.StatusOK, stats)
}

// DisableAccount godoc
// @Summary Disable a user account
// @Tags Admin
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/disable [post]
func (h *AdminHandler) DisableAccount(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid user ID")
		return
	}
	if err := h.authSvc.DisableAccount(c.Request.Context(), userID); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Account disabled"})
}

// EnableAccount godoc
// @Summary Re-enable a user account
// @Tags Admin
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/enable [post]
func (h *AdminHandler) EnableAccount(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid user ID")
		return
	}
	if err := h.authSvc.EnableAccount(c.Request.Context(), userID); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Account enabled"})
}

// ResetPassword godoc
// @Summary Reset a user's password (admin)
// @Tags Admin
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/users/{id}/reset-password [post]
func (h *AdminHandler) ResetPassword(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid user ID")
		return
	}
	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.authSvc.AdminResetPassword(c.Request.Context(), userID, req.NewPassword); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// ListAuditLogs godoc
// @Summary List audit logs
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/audit-logs [get]
func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	resource := c.Query("resource")

	var resourceID *uuid.UUID
	if rid := c.Query("resource_id"); rid != "" {
		if id, err := uuid.Parse(rid); err == nil {
			resourceID = &id
		}
	}

	logs, total, err := h.auditRepo.List(c.Request.Context(), resource, resourceID, page, perPage)
	if err != nil {
		MapServiceError(c, err)
		return
	}
	ListResponse(c, logs, buildMeta(page, perPage, total))
}

// ListStatutoryRates godoc
// @Summary List active statutory deduction rates
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/statutory-rates [get]
func (h *AdminHandler) ListStatutoryRates(c *gin.Context) {
	rates, err := h.statuteRepo.GetActiveRates(c.Request.Context())
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, rates)
}

// ChangePassword godoc
// @Summary Change own password (authenticated user)
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/change-password [post]
func (h *AdminHandler) ChangePassword(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		Unauthorized(c, "Authentication required")
		return
	}
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.authSvc.ChangePassword(c.Request.Context(), claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// ListTemplates godoc
// @Summary List notification templates
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/notifications/templates [get]
func (h *AdminHandler) ListTemplates(c *gin.Context) {
	templates, err := h.notifSvc.ListTemplates(c.Request.Context())
	if err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, templates)
}

// CreateTemplate godoc
// @Summary Create notification template
// @Tags Admin
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/admin/notifications/templates [post]
func (h *AdminHandler) CreateTemplate(c *gin.Context) {
	var t models.NotificationTemplate
	if err := c.ShouldBindJSON(&t); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.notifSvc.CreateTemplate(c.Request.Context(), &t); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusCreated, t)
}

// UpdateTemplate godoc
// @Summary Update notification template
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/notifications/templates [put]
func (h *AdminHandler) UpdateTemplate(c *gin.Context) {
	var t models.NotificationTemplate
	if err := c.ShouldBindJSON(&t); err != nil {
		BadRequest(c, err.Error())
		return
	}
	if err := h.notifSvc.UpdateTemplate(c.Request.Context(), &t); err != nil {
		MapServiceError(c, err)
		return
	}
	SuccessResponse(c, http.StatusOK, t)
}
