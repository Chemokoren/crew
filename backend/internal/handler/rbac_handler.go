package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/handler/dto"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/service"
)

// RBACHandler handles all RBAC REST API endpoints.
type RBACHandler struct {
	svc *service.RBACService
}

// NewRBACHandler creates a new RBAC handler.
func NewRBACHandler(svc *service.RBACService) *RBACHandler {
	return &RBACHandler{svc: svc}
}

// RegisterRoutes registers all RBAC routes under the given router group.
func (h *RBACHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rbac := rg.Group("/rbac")
	{
		// Roles
		rbac.GET("/roles", h.ListRoles)
		rbac.POST("/roles", h.CreateRole)
		rbac.GET("/roles/:id", h.GetRole)
		rbac.PUT("/roles/:id", h.UpdateRole)
		rbac.DELETE("/roles/:id", h.DeleteRole)
		rbac.POST("/roles/:id/clone", h.CloneRole)
		rbac.POST("/roles/:id/activate", h.ToggleRoleActive)
		rbac.GET("/roles/:id/permissions", h.GetRolePermissions)
		rbac.PUT("/roles/:id/permissions", h.SetRolePermissions)
		rbac.POST("/roles/compare", h.CompareRoles)

		// Permissions
		rbac.GET("/permissions", h.ListPermissions)
		rbac.GET("/permissions/modules", h.ListPermissionModules)

		// User roles
		rbac.GET("/users/:id/roles", h.GetUserRoles)
		rbac.POST("/users/:id/roles", h.AssignRoleToUser)
		rbac.DELETE("/users/:id/roles/:roleId", h.RevokeRoleFromUser)
		rbac.GET("/users/:id/permissions", h.GetUserPermissions)

		// Templates
		rbac.GET("/templates", h.ListTemplates)
		rbac.POST("/templates/:id/apply", h.ApplyTemplate)

		// Policies
		rbac.GET("/policies", h.ListPolicies)
		rbac.POST("/policies", h.CreatePolicy)

		// Matrix
		rbac.GET("/matrix", h.GetPermissionMatrix)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Roles
// ═══════════════════════════════════════════════════════════════════════════

func (h *RBACHandler) ListRoles(c *gin.Context) {
	page, perPage := parsePagination(c)
	filter := repository.RoleFilter{
		IndustryType: c.Query("industry_type"),
		Search:       c.Query("search"),
	}
	if tid := c.Query("tenant_id"); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			filter.TenantID = &id
		}
	}
	if v := c.Query("is_system"); v != "" {
		b := v == "true"
		filter.IsSystem = &b
	}
	if v := c.Query("is_template"); v != "" {
		b := v == "true"
		filter.IsTemplate = &b
	}
	if v := c.Query("is_active"); v != "" {
		b := v == "true"
		filter.IsActive = &b
	}

	roles, total, err := h.svc.ListRoles(c.Request.Context(), filter, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to list roles", err))
		return
	}

	// Enrich with permission counts
	type roleWithCount struct {
		models.Role
		PermissionCount int   `json:"permission_count"`
		UserCount       int64 `json:"user_count"`
	}
	enriched := make([]roleWithCount, len(roles))
	for i, r := range roles {
		perms, _ := h.svc.GetRolePermissions(c.Request.Context(), r.ID)
		users, _ := h.svc.CountUsersWithRole(c.Request.Context(), r.ID)
		enriched[i] = roleWithCount{Role: r, PermissionCount: len(perms), UserCount: users}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    enriched,
		"meta":    paginationMeta(page, perPage, total),
	})
}

func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request", err))
		return
	}

	claims := middleware.GetClaims(c)
	role := &models.Role{
		Name:         req.Name,
		Description:  req.Description,
		TenantID:     req.TenantID,
		IndustryType: req.IndustryType,
		ParentRoleID: req.ParentRoleID,
		IsTemplate:   req.IsTemplate,
		IsActive:     true,
	}
	if claims != nil {
		role.CreatedBy = &claims.UserID
		role.UpdatedBy = &claims.UserID
	}

	if err := h.svc.CreateRole(c.Request.Context(), role); err != nil {
		c.JSON(http.StatusConflict, errorResponse("Failed to create role", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": role})
}

func (h *RBACHandler) GetRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid role ID", err))
		return
	}

	role, err := h.svc.GetRole(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("Role not found", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": role})
}

func (h *RBACHandler) UpdateRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid role ID", err))
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request", err))
		return
	}

	role, err := h.svc.GetRole(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("Role not found", err))
		return
	}

	role.Name = req.Name
	role.Description = req.Description
	if req.IsActive != nil {
		role.IsActive = *req.IsActive
	}
	claims := middleware.GetClaims(c)
	if claims != nil {
		role.UpdatedBy = &claims.UserID
	}

	if err := h.svc.UpdateRole(c.Request.Context(), role); err != nil {
		c.JSON(http.StatusConflict, errorResponse("Failed to update role", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": role})
}

func (h *RBACHandler) DeleteRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid role ID", err))
		return
	}

	claims := middleware.GetClaims(c)
	var deletedBy *uuid.UUID
	if claims != nil {
		deletedBy = &claims.UserID
	}

	if err := h.svc.DeleteRole(c.Request.Context(), id, deletedBy); err != nil {
		c.JSON(http.StatusConflict, errorResponse("Failed to delete role", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Role archived"})
}

func (h *RBACHandler) CloneRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid role ID", err))
		return
	}

	var req dto.CloneRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request", err))
		return
	}

	claims := middleware.GetClaims(c)
	var createdBy *uuid.UUID
	if claims != nil {
		createdBy = &claims.UserID
	}

	cloned, err := h.svc.CloneRole(c.Request.Context(), id, req.Name, req.TenantID, createdBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to clone role", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": cloned})
}

func (h *RBACHandler) ToggleRoleActive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid role ID", err))
		return
	}

	var req dto.ToggleActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request", err))
		return
	}

	claims := middleware.GetClaims(c)
	var updatedBy *uuid.UUID
	if claims != nil {
		updatedBy = &claims.UserID
	}

	if err := h.svc.ToggleRoleActive(c.Request.Context(), id, req.Active, updatedBy); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to toggle role", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Role status updated"})
}

func (h *RBACHandler) CompareRoles(c *gin.Context) {
	var req dto.CompareRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request", err))
		return
	}

	onlyA, onlyB, shared, err := h.svc.CompareRoles(c.Request.Context(), req.RoleAID, req.RoleBID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to compare roles", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": dto.RoleComparisonResponse{
			OnlyInA: nonNil(onlyA), OnlyInB: nonNil(onlyB), Shared: nonNil(shared),
		},
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Permissions
// ═══════════════════════════════════════════════════════════════════════════

func (h *RBACHandler) GetRolePermissions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid role ID", err))
		return
	}

	perms, err := h.svc.GetRolePermissions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to get permissions", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": perms})
}

func (h *RBACHandler) SetRolePermissions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid role ID", err))
		return
	}

	var req dto.SetPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request", err))
		return
	}

	claims := middleware.GetClaims(c)
	var grantedBy *uuid.UUID
	if claims != nil {
		grantedBy = &claims.UserID
	}

	if err := h.svc.SetRolePermissions(c.Request.Context(), id, req.PermissionKeys, grantedBy); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to set permissions", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Permissions updated"})
}

func (h *RBACHandler) ListPermissions(c *gin.Context) {
	filter := repository.PermissionFilter{
		Module:    c.Query("module"),
		Category:  c.Query("category"),
		RiskLevel: c.Query("risk_level"),
		Search:    c.Query("search"),
	}

	perms, err := h.svc.ListPermissions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to list permissions", err))
		return
	}

	// Group by module for UI consumption
	grouped := make(map[string][]models.PermissionDef)
	for _, p := range perms {
		grouped[p.Module] = append(grouped[p.Module], p)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": perms, "grouped": grouped})
}

func (h *RBACHandler) ListPermissionModules(c *gin.Context) {
	modules := h.svc.GetPermissionModules()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": modules})
}

// ═══════════════════════════════════════════════════════════════════════════
// User Roles
// ═══════════════════════════════════════════════════════════════════════════

func (h *RBACHandler) GetUserRoles(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid user ID", err))
		return
	}

	var tenantID *uuid.UUID
	if tid := c.Query("tenant_id"); tid != "" {
		if id, e := uuid.Parse(tid); e == nil {
			tenantID = &id
		}
	}

	roles, err := h.svc.GetUserRoles(c.Request.Context(), userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to get user roles", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": roles})
}

func (h *RBACHandler) AssignRoleToUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid user ID", err))
		return
	}

	var req dto.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request", err))
		return
	}

	claims := middleware.GetClaims(c)
	var assignedBy *uuid.UUID
	if claims != nil {
		assignedBy = &claims.UserID
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	if err := h.svc.AssignRole(c.Request.Context(), userID, req.RoleID, req.TenantID, assignedBy, expiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to assign role", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Role assigned"})
}

func (h *RBACHandler) RevokeRoleFromUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid user ID", err))
		return
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid role ID", err))
		return
	}

	var tenantID *uuid.UUID
	if tid := c.Query("tenant_id"); tid != "" {
		if id, e := uuid.Parse(tid); e == nil {
			tenantID = &id
		}
	}

	claims := middleware.GetClaims(c)
	var revokedBy *uuid.UUID
	if claims != nil {
		revokedBy = &claims.UserID
	}

	if err := h.svc.RevokeRole(c.Request.Context(), userID, roleID, tenantID, revokedBy); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to revoke role", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Role revoked"})
}

func (h *RBACHandler) GetUserPermissions(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid user ID", err))
		return
	}

	var tenantID *uuid.UUID
	if tid := c.Query("tenant_id"); tid != "" {
		if id, e := uuid.Parse(tid); e == nil {
			tenantID = &id
		}
	}

	keys, err := h.svc.GetUserPermissions(c.Request.Context(), userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to get permissions", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": keys})
}

// ═══════════════════════════════════════════════════════════════════════════
// Templates
// ═══════════════════════════════════════════════════════════════════════════

func (h *RBACHandler) ListTemplates(c *gin.Context) {
	industry := c.Query("industry_type")
	templates, err := h.svc.ListTemplates(c.Request.Context(), industry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to list templates", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": templates})
}

func (h *RBACHandler) ApplyTemplate(c *gin.Context) {
	tmplID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid template ID", err))
		return
	}

	var req dto.ApplyTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request", err))
		return
	}

	claims := middleware.GetClaims(c)
	var appliedBy *uuid.UUID
	if claims != nil {
		appliedBy = &claims.UserID
	}

	role, err := h.svc.ApplyTemplate(c.Request.Context(), tmplID, req.TenantID, appliedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to apply template", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": role})
}

// ═══════════════════════════════════════════════════════════════════════════
// Policies
// ═══════════════════════════════════════════════════════════════════════════

func (h *RBACHandler) ListPolicies(c *gin.Context) {
	page, perPage := parsePagination(c)
	var tenantID *uuid.UUID
	if tid := c.Query("tenant_id"); tid != "" {
		if id, e := uuid.Parse(tid); e == nil {
			tenantID = &id
		}
	}

	policies, total, err := h.svc.ListPolicies(c.Request.Context(), tenantID, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to list policies", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": policies, "meta": paginationMeta(page, perPage, total)})
}

func (h *RBACHandler) CreatePolicy(c *gin.Context) {
	var req dto.CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("Invalid request", err))
		return
	}

	conditions, _ := json.Marshal(req.Conditions)
	claims := middleware.GetClaims(c)
	var createdBy *uuid.UUID
	if claims != nil {
		createdBy = &claims.UserID
	}

	policy := &models.Policy{
		Name:          req.Name,
		Description:   req.Description,
		PermissionKey: req.PermissionKey,
		Conditions:    conditions,
		Effect:        models.PolicyEffect(req.Effect),
		IsActive:      true,
		Priority:      req.Priority,
		TenantID:      req.TenantID,
		CreatedBy:     createdBy,
	}

	if err := h.svc.CreatePolicy(c.Request.Context(), policy); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to create policy", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": policy})
}

// ═══════════════════════════════════════════════════════════════════════════
// Matrix
// ═══════════════════════════════════════════════════════════════════════════

func (h *RBACHandler) GetPermissionMatrix(c *gin.Context) {
	filter := repository.RoleFilter{
		IndustryType: c.Query("industry_type"),
	}
	if tid := c.Query("tenant_id"); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			filter.TenantID = &id
		}
	}
	active := true
	filter.IsActive = &active

	roles, modules, grants, err := h.svc.GetPermissionMatrix(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("Failed to build matrix", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": dto.PermissionMatrixResponse{
			Roles: roles, Modules: modules, Grants: grants,
		},
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

func (h *RBACHandler) CountUsersWithRole(ctx gin.Context, roleID uuid.UUID) (int64, error) {
	return h.svc.CountUsersWithRole(ctx.Request.Context(), roleID)
}

func errorResponse(msg string, err error) gin.H {
	return gin.H{
		"success": false,
		"error": gin.H{
			"code":    "ERROR",
			"message": msg + ": " + err.Error(),
		},
	}
}

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}

func paginationMeta(page, perPage int, total int64) gin.H {
	totalPages := (total + int64(perPage) - 1) / int64(perPage)
	return gin.H{
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
	}
}

func nonNil(ss []string) []string {
	if ss == nil {
		return []string{}
	}
	return ss
}
