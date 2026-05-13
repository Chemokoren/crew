package dto

import "github.com/google/uuid"

// ── Role DTOs ───────────────────────────────────────────────────────────────

// CreateRoleRequest is the payload for POST /api/v1/rbac/roles.
type CreateRoleRequest struct {
	Name         string     `json:"name" binding:"required,min=2,max=120"`
	Description  string     `json:"description"`
	TenantID     *uuid.UUID `json:"tenant_id"`
	IndustryType string     `json:"industry_type"`
	ParentRoleID *uuid.UUID `json:"parent_role_id"`
	IsTemplate   bool       `json:"is_template"`
}

// UpdateRoleRequest is the payload for PUT /api/v1/rbac/roles/:id.
type UpdateRoleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=120"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

// CloneRoleRequest is the payload for POST /api/v1/rbac/roles/:id/clone.
type CloneRoleRequest struct {
	Name     string     `json:"name" binding:"required,min=2,max=120"`
	TenantID *uuid.UUID `json:"tenant_id"`
}

// SetPermissionsRequest is the payload for PUT /api/v1/rbac/roles/:id/permissions.
type SetPermissionsRequest struct {
	PermissionKeys []string `json:"permission_keys" binding:"required"`
}

// CompareRolesRequest is the payload for POST /api/v1/rbac/roles/compare.
type CompareRolesRequest struct {
	RoleAID uuid.UUID `json:"role_a_id" binding:"required"`
	RoleBID uuid.UUID `json:"role_b_id" binding:"required"`
}

// ToggleActiveRequest is the payload for POST /api/v1/rbac/roles/:id/activate.
type ToggleActiveRequest struct {
	Active bool `json:"active"`
}

// ── User-Role DTOs ──────────────────────────────────────────────────────────

// AssignRoleRequest is the payload for POST /api/v1/rbac/users/:id/roles.
type AssignRoleRequest struct {
	RoleID    uuid.UUID  `json:"role_id" binding:"required"`
	TenantID  *uuid.UUID `json:"tenant_id"`
	ExpiresAt *string    `json:"expires_at"` // RFC3339
}

// ── Template DTOs ───────────────────────────────────────────────────────────

// ApplyTemplateRequest is the payload for POST /api/v1/rbac/templates/:id/apply.
type ApplyTemplateRequest struct {
	TenantID uuid.UUID `json:"tenant_id" binding:"required"`
}

// ── Policy DTOs ─────────────────────────────────────────────────────────────

// CreatePolicyRequest is the payload for POST /api/v1/rbac/policies.
type CreatePolicyRequest struct {
	Name          string                 `json:"name" binding:"required"`
	Description   string                 `json:"description"`
	PermissionKey string                 `json:"permission_key" binding:"required"`
	Conditions    map[string]interface{} `json:"conditions" binding:"required"`
	Effect        string                 `json:"effect" binding:"required,oneof=ALLOW DENY"`
	Priority      int                    `json:"priority"`
	TenantID      *uuid.UUID             `json:"tenant_id"`
}

// ── Response DTOs ───────────────────────────────────────────────────────────

// RoleComparisonResponse is returned by POST /api/v1/rbac/roles/compare.
type RoleComparisonResponse struct {
	OnlyInA []string `json:"only_in_a"`
	OnlyInB []string `json:"only_in_b"`
	Shared  []string `json:"shared"`
}

// PermissionMatrixResponse is returned by GET /api/v1/rbac/matrix.
type PermissionMatrixResponse struct {
	Roles   interface{}         `json:"roles"`
	Modules []string            `json:"modules"`
	Grants  map[string][]string `json:"grants"` // roleID → []permKey
}
