package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/rbac"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/types"
)

// RBACService provides business logic for roles, permissions, and authorization.
type RBACService struct {
	repo         repository.RBACRepository
	auditSvc     *AuditService
	cache        *PermissionCache
	registry     *rbac.Registry
	policyEngine *rbac.PolicyEngine
}

// NewRBACService creates a new RBAC service.
func NewRBACService(repo repository.RBACRepository, auditSvc *AuditService, cache *PermissionCache) *RBACService {
	return &RBACService{
		repo:         repo,
		auditSvc:     auditSvc,
		cache:        cache,
		registry:     rbac.Global(),
		policyEngine: rbac.NewPolicyEngine(),
	}
}

// systemRoleCtxKey is the context key for passing the user's system role
// from the middleware to the RBAC service without tight coupling.
type systemRoleCtxKey struct{}

// SetSystemRoleInContext stores the user's system_role in the request context
// so HasPermissionWithContext can resolve the system role's RBAC permissions.
func SetSystemRoleInContext(ctx context.Context, role types.SystemRole) context.Context {
	return context.WithValue(ctx, systemRoleCtxKey{}, role)
}

// ═══════════════════════════════════════════════════════════════════════════
// Authorization Checks
// ═══════════════════════════════════════════════════════════════════════════

// HasPermission checks if a user has a specific permission (cache → DB fallback).
func (s *RBACService) HasPermission(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, permKey string) bool {
	return s.HasPermissionWithContext(ctx, userID, tenantID, permKey, rbac.EvaluationContext{
		CurrentTime: time.Now(),
		Timezone:    "Africa/Nairobi",
	})
}

// HasPermissionWithContext checks user RBAC grants and then applies active dynamic policies.
// It also checks permissions from the user's system-role RBAC role (if the middleware
// provides systemRole via the context), ensuring all authorization flows through the DB.
func (s *RBACService) HasPermissionWithContext(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, permKey string, evalCtx rbac.EvaluationContext) bool {
	// Fast path: check cache
	var keys []string
	if s.cache != nil {
		if allowed, cached := s.cache.HasPermission(ctx, userID, tenantID, permKey); cached {
			if !allowed {
				return false
			}
			keys, _ = s.cache.Get(ctx, userID, tenantID)
		}
	}

	// Slow path: load from DB and cache
	if keys == nil {
		var err error
		keys, err = s.repo.GetUserPermissionKeys(ctx, userID, tenantID)
		if err != nil {
			slog.Warn("rbac: failed to load permissions", slog.Any("error", err), slog.String("user_id", userID.String()))
			return false
		}

		if s.cache != nil {
			s.cache.Set(ctx, userID, tenantID, keys)
		}
	}

	granted := false
	for _, k := range keys {
		if k == permKey {
			granted = true
			break
		}
	}

	// If not directly granted, check the user's system-role RBAC permissions.
	// The system role is extracted from JWT claims by the middleware and stored
	// in the context so the RBAC service can resolve it without hardcoded maps.
	if !granted {
		sysRole, _ := ctx.Value(systemRoleCtxKey{}).(types.SystemRole)
		if sysRole != "" {
			if sysKeys := s.getSystemRolePermissions(ctx, sysRole); sysKeys != nil {
				for _, k := range sysKeys {
					if k == permKey {
						granted = true
						break
					}
				}
			}
		}
	}

	if !granted {
		return false
	}

	return s.allowedByPolicies(ctx, tenantID, permKey, evalCtx)
}

// HasAnyPermission checks if a user has at least one of the specified permissions.
func (s *RBACService) HasAnyPermission(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, permKeys ...string) bool {
	for _, key := range permKeys {
		if s.HasPermission(ctx, userID, tenantID, key) {
			return true
		}
	}
	return false
}

// GetUserPermissions returns all effective permission keys for a user.
func (s *RBACService) GetUserPermissions(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID) ([]string, error) {
	return s.GetUserPermissionsWithRole(ctx, userID, tenantID, "")
}

// GetUserPermissionsWithRole returns effective permission keys for a user,
// merging permissions from their explicit RBAC role assignments AND from
// the system role's RBAC role (looked up by slug in the roles table).
func (s *RBACService) GetUserPermissionsWithRole(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, systemRole types.SystemRole) ([]string, error) {
	if s.cache != nil {
		if keys, ok := s.cache.Get(ctx, userID, tenantID); ok {
			return keys, nil
		}
	}
	keys, err := s.repo.GetUserPermissionKeys(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}

	// Merge permissions from the user's system-role RBAC role.
	if systemRole != "" {
		if sysKeys := s.getSystemRolePermissions(ctx, systemRole); sysKeys != nil {
			set := make(map[string]struct{}, len(keys)+len(sysKeys))
			for _, k := range keys {
				set[k] = struct{}{}
			}
			for _, k := range sysKeys {
				set[k] = struct{}{}
			}
			merged := make([]string, 0, len(set))
			for k := range set {
				merged = append(merged, k)
			}
			keys = merged
		}
	}

	if s.cache != nil {
		s.cache.Set(ctx, userID, tenantID, keys)
	}
	return keys, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Role Management
// ═══════════════════════════════════════════════════════════════════════════

// CreateRole creates a new role with validation.
func (s *RBACService) CreateRole(ctx context.Context, role *models.Role) error {
	if role.Slug == "" {
		role.Slug = slugify(role.Name)
	}

	// Check slug uniqueness within tenant
	existing, err := s.repo.GetRoleBySlug(ctx, role.Slug, role.TenantID)
	if err == nil && existing != nil {
		return fmt.Errorf("role slug '%s' already exists in this tenant", role.Slug)
	}

	if err := s.repo.CreateRole(ctx, role); err != nil {
		return err
	}

	s.logAudit(ctx, role.CreatedBy, "role.created", "role", &role.ID, nil, role)
	return nil
}

// UpdateRole updates an existing role with system-role protection.
func (s *RBACService) UpdateRole(ctx context.Context, role *models.Role) error {
	existing, err := s.repo.GetRoleByID(ctx, role.ID)
	if err != nil {
		return err
	}
	if existing.IsSystem && role.Name != existing.Name {
		return fmt.Errorf("cannot rename system role '%s'", existing.Name)
	}

	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return err
	}

	s.logAudit(ctx, role.UpdatedBy, "role.updated", "role", &role.ID, existing, role)
	return nil
}

// DeleteRole soft-deletes a role with system-role protection.
func (s *RBACService) DeleteRole(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID) error {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return fmt.Errorf("cannot delete system role '%s'", role.Name)
	}

	count, _ := s.repo.CountUsersWithRole(ctx, id)
	if count > 0 {
		return fmt.Errorf("cannot delete role '%s' — %d users still assigned", role.Name, count)
	}

	if err := s.repo.DeleteRole(ctx, id); err != nil {
		return err
	}

	s.logAudit(ctx, deletedBy, "role.deleted", "role", &id, role, nil)
	return nil
}

// GetRole returns a role by ID.
func (s *RBACService) GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	return s.repo.GetRoleByID(ctx, id)
}

// ListRoles returns paginated roles with filtering.
func (s *RBACService) ListRoles(ctx context.Context, filter repository.RoleFilter, page, perPage int) ([]models.Role, int64, error) {
	return s.repo.ListRoles(ctx, filter, page, perPage)
}

// CloneRole clones a role with all its permissions.
func (s *RBACService) CloneRole(ctx context.Context, sourceID uuid.UUID, newName string, tenantID *uuid.UUID, createdBy *uuid.UUID) (*models.Role, error) {
	slug := slugify(newName)
	cloned, err := s.repo.CloneRole(ctx, sourceID, newName, slug, tenantID, createdBy)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, createdBy, "role.cloned", "role", &cloned.ID,
		map[string]interface{}{"source_role_id": sourceID.String()}, cloned)
	return cloned, nil
}

// ToggleRoleActive activates or deactivates a role.
func (s *RBACService) ToggleRoleActive(ctx context.Context, id uuid.UUID, active bool, updatedBy *uuid.UUID) error {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}
	role.IsActive = active
	role.UpdatedBy = updatedBy
	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return err
	}

	action := "role.activated"
	if !active {
		action = "role.deactivated"
	}
	s.logAudit(ctx, updatedBy, action, "role", &id, nil, map[string]bool{"is_active": active})

	// Invalidate cache for all users with this role
	if s.cache != nil {
		s.cache.InvalidateAll(ctx)
	}
	return nil
}

// CompareRoles returns the permission diff between two roles.
func (s *RBACService) CompareRoles(ctx context.Context, roleAID, roleBID uuid.UUID) (onlyA, onlyB, shared []string, err error) {
	keysA, err := s.repo.GetRolePermissionKeys(ctx, roleAID)
	if err != nil {
		return nil, nil, nil, err
	}
	keysB, err := s.repo.GetRolePermissionKeys(ctx, roleBID)
	if err != nil {
		return nil, nil, nil, err
	}

	setA := toSet(keysA)
	setB := toSet(keysB)

	for k := range setA {
		if setB[k] {
			shared = append(shared, k)
		} else {
			onlyA = append(onlyA, k)
		}
	}
	for k := range setB {
		if !setA[k] {
			onlyB = append(onlyB, k)
		}
	}
	return
}

// ═══════════════════════════════════════════════════════════════════════════
// Permission Management
// ═══════════════════════════════════════════════════════════════════════════

// SyncRegistryPermissions syncs the in-memory registry to the database.
func (s *RBACService) SyncRegistryPermissions(ctx context.Context) error {
	allDefs := s.registry.GetAll()
	dbDefs := make([]models.PermissionDef, len(allDefs))
	for i, d := range allDefs {
		meta, _ := json.Marshal(map[string]interface{}{})
		dbDefs[i] = models.PermissionDef{
			Key:         d.Key,
			Module:      d.Module,
			Description: d.Description,
			RiskLevel:   d.RiskLevel,
			Category:    d.Category,
			IsSystem:    true,
			DependsOn:   models.StringArray(d.DependsOn),
			Metadata:    meta,
			UpdatedAt:   time.Now(),
		}
	}
	return s.repo.SyncPermissions(ctx, dbDefs)
}

// SyncSystemRoles creates global platform system roles and attaches their permissions.
func (s *RBACService) SyncSystemRoles(ctx context.Context) error {
	// Sync both PLATFORM and SYSTEM role templates so every system_role
	// has a matching RBAC role in the database.
	var templates []rbac.RoleTemplateDefinition
	templates = append(templates, rbac.GetTemplatesForIndustry("PLATFORM")...)
	templates = append(templates, rbac.GetTemplatesForIndustry("SYSTEM")...)

	for _, t := range templates {
		role, _ := s.repo.GetRoleBySlug(ctx, t.RoleSlug, nil)
		if role == nil {
			role = &models.Role{
				Name:         t.RoleName,
				Slug:         t.RoleSlug,
				Description:  t.Description,
				IndustryType: t.IndustryType,
				IsSystem:     true,
				IsTemplate:   false,
				IsActive:     true,
			}
			if err := s.repo.CreateRole(ctx, role); err != nil {
				return fmt.Errorf("create system role %s: %w", t.RoleSlug, err)
			}
		} else {
			role.Name = t.RoleName
			role.Description = t.Description
			role.IndustryType = t.IndustryType
			role.IsSystem = true
			role.IsTemplate = false
			role.IsActive = true
			if err := s.repo.UpdateRole(ctx, role); err != nil {
				return fmt.Errorf("update system role %s: %w", t.RoleSlug, err)
			}
		}

		perms, err := s.resolvePermissions(ctx, t.Permissions)
		if err != nil {
			return fmt.Errorf("resolve permissions for %s: %w", t.RoleSlug, err)
		}
		permIDs := make([]uuid.UUID, len(perms))
		for i, p := range perms {
			permIDs[i] = p.ID
		}
		if err := s.repo.BulkSetPermissions(ctx, role.ID, permIDs, nil); err != nil {
			return fmt.Errorf("set permissions for %s: %w", t.RoleSlug, err)
		}
	}
	return nil
}

// ListPermissions returns all permissions with optional filtering.
func (s *RBACService) ListPermissions(ctx context.Context, filter repository.PermissionFilter) ([]models.PermissionDef, error) {
	return s.repo.ListPermissions(ctx, filter)
}

// GetPermissionModules returns module names from the registry.
func (s *RBACService) GetPermissionModules() []string {
	return s.registry.ModuleNames()
}

// SetRolePermissions replaces all permissions for a role (bulk update).
// System roles CAN be edited — this allows admins to control feature
// visibility (e.g. hide Loans/Insurance from employees until ready).
// Changes persist until the next SyncSystemRoles call on restart.
func (s *RBACService) SetRolePermissions(ctx context.Context, roleID uuid.UUID, permKeys []string, grantedBy *uuid.UUID) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsSystem {
		slog.Info("modifying system role permissions",
			slog.String("role", role.Name),
			slog.String("slug", role.Slug),
			slog.Int("new_count", len(permKeys)),
		)
	}

	// Resolve keys to IDs
	permKeys = uniqueStrings(permKeys)
	perms, err := s.resolvePermissions(ctx, permKeys)
	if err != nil {
		return err
	}
	if err := validatePermissionDependencies(perms, permKeys); err != nil {
		return err
	}
	permIDs := make([]uuid.UUID, len(perms))
	for i, p := range perms {
		permIDs[i] = p.ID
	}

	// Prevent privilege escalation: check if granter has these permissions
	if grantedBy != nil {
		granterPerms, err := s.GetUserPermissions(ctx, *grantedBy, role.TenantID)
		if err != nil {
			return fmt.Errorf("failed to verify granter permissions: %v", err)
		}
		granterPermMap := make(map[string]bool)
		for _, kp := range granterPerms {
			granterPermMap[kp] = true
		}
		for _, reqPerm := range permKeys {
			if !granterPermMap[reqPerm] {
				return fmt.Errorf("privilege escalation prevented: cannot grant permission '%s' which you do not possess", reqPerm)
			}
		}
	}

	oldKeys, _ := s.repo.GetRolePermissionKeys(ctx, roleID)

	if err := s.repo.BulkSetPermissions(ctx, roleID, permIDs, grantedBy); err != nil {
		return err
	}

	s.logAudit(ctx, grantedBy, "role.permissions_updated", "role", &roleID,
		map[string]interface{}{"old_permissions": oldKeys},
		map[string]interface{}{"new_permissions": permKeys})

	// Invalidate cache for affected users
	if s.cache != nil {
		s.cache.InvalidateAll(ctx)
	}
	return nil
}

// GetRolePermissions returns all permission definitions for a role.
func (s *RBACService) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.PermissionDef, error) {
	return s.repo.GetRolePermissions(ctx, roleID)
}

// ═══════════════════════════════════════════════════════════════════════════
// User-Role Assignment
// ═══════════════════════════════════════════════════════════════════════════

// AssignRole assigns a role to a user within a tenant.
func (s *RBACService) AssignRole(ctx context.Context, userID, roleID uuid.UUID, tenantID *uuid.UUID, assignedBy *uuid.UUID, expiresAt *time.Time) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if !role.IsActive {
		return fmt.Errorf("cannot assign inactive role '%s'", role.Name)
	}
	if role.TenantID != nil {
		if tenantID == nil || *tenantID != *role.TenantID {
			return fmt.Errorf("role '%s' belongs to a different tenant", role.Name)
		}
	}

	if assignedBy != nil {
		rolePerms, err := s.repo.GetRolePermissionKeys(ctx, roleID)
		if err != nil {
			return fmt.Errorf("failed to verify role permissions: %w", err)
		}
		assignerPerms, err := s.GetUserPermissions(ctx, *assignedBy, tenantID)
		if err != nil {
			return fmt.Errorf("failed to verify assigner permissions: %w", err)
		}
		assignerPermMap := toSet(assignerPerms)
		for _, rolePerm := range rolePerms {
			if !assignerPermMap[rolePerm] {
				return fmt.Errorf("privilege escalation prevented: cannot assign role with permission '%s' which you do not possess", rolePerm)
			}
		}
	}

	ur := &models.UserRole{
		UserID:     userID,
		RoleID:     roleID,
		TenantID:   tenantID,
		AssignedBy: assignedBy,
		AssignedAt: time.Now(),
		ExpiresAt:  expiresAt,
		IsActive:   true,
	}

	if err := s.repo.AssignRoleToUser(ctx, ur); err != nil {
		return err
	}

	if s.cache != nil {
		s.cache.Invalidate(ctx, userID)
	}
	s.logAudit(ctx, assignedBy, "role.assigned", "user_role", nil, nil,
		map[string]interface{}{"user_id": userID, "role_id": roleID, "tenant_id": tenantID})
	return nil
}

// RevokeRole revokes a role from a user.
func (s *RBACService) RevokeRole(ctx context.Context, userID, roleID uuid.UUID, tenantID *uuid.UUID, revokedBy *uuid.UUID) error {
	if err := s.repo.RevokeRoleFromUser(ctx, userID, roleID, tenantID); err != nil {
		return err
	}

	if s.cache != nil {
		s.cache.Invalidate(ctx, userID)
	}
	s.logAudit(ctx, revokedBy, "role.revoked", "user_role", nil, nil,
		map[string]interface{}{"user_id": userID, "role_id": roleID, "tenant_id": tenantID})
	return nil
}

// GetUserRoles returns all active roles for a user.
func (s *RBACService) GetUserRoles(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID) ([]models.UserRole, error) {
	return s.repo.GetUserRoles(ctx, userID, tenantID)
}

// ═══════════════════════════════════════════════════════════════════════════
// Templates
// ═══════════════════════════════════════════════════════════════════════════

// SyncTemplates syncs industry role templates from code to database.
func (s *RBACService) SyncTemplates(ctx context.Context) error {
	templates := rbac.GetIndustryRoleTemplates()
	for _, t := range templates {
		permJSON, _ := json.Marshal(t.Permissions)
		tmpl := &models.RoleTemplate{
			IndustryType: t.IndustryType,
			RoleName:     t.RoleName,
			RoleSlug:     t.RoleSlug,
			Description:  t.Description,
			Permissions:  permJSON,
			IsDefault:    t.IsDefault,
			SortOrder:    t.SortOrder,
			UpdatedAt:    time.Now(),
		}
		if err := s.repo.UpsertTemplate(ctx, tmpl); err != nil {
			return fmt.Errorf("sync template '%s': %w", t.RoleSlug, err)
		}
	}
	return nil
}

// ListTemplates returns role templates, optionally filtered by industry.
func (s *RBACService) ListTemplates(ctx context.Context, industryType string) ([]models.RoleTemplate, error) {
	return s.repo.ListTemplates(ctx, industryType)
}

// ApplyTemplate creates tenant-specific roles from a template.
func (s *RBACService) ApplyTemplate(ctx context.Context, templateID uuid.UUID, tenantID uuid.UUID, appliedBy *uuid.UUID) (*models.Role, error) {
	tmpl, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	permKeys, err := tmpl.GetPermissionKeys()
	if err != nil {
		return nil, fmt.Errorf("parse template permissions: %w", err)
	}

	// Create the role
	role := &models.Role{
		Name:         tmpl.RoleName,
		Slug:         tmpl.RoleSlug,
		Description:  tmpl.Description,
		TenantID:     &tenantID,
		IndustryType: tmpl.IndustryType,
		IsSystem:     false,
		IsTemplate:   false,
		IsActive:     true,
		CreatedBy:    appliedBy,
		UpdatedBy:    appliedBy,
	}

	// Check if slug exists for this tenant — append suffix if needed
	if existing, _ := s.repo.GetRoleBySlug(ctx, role.Slug, &tenantID); existing != nil {
		role.Slug = fmt.Sprintf("%s-%s", role.Slug, uuid.New().String()[:8])
	}

	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	// Assign permissions
	if len(permKeys) > 0 {
		if err := s.SetRolePermissions(ctx, role.ID, permKeys, appliedBy); err != nil {
			_ = s.DeleteRole(ctx, role.ID, appliedBy)
			return nil, fmt.Errorf("assign template permissions: %w", err)
		}
	}

	return role, nil
}

// GetPermissionMatrix returns a full matrix of roles × permissions for the UI.
func (s *RBACService) GetPermissionMatrix(ctx context.Context, filter repository.RoleFilter) (roles []models.Role, modules []string, grants map[string][]string, err error) {
	roles, _, err = s.repo.ListRoles(ctx, filter, 1, 100)
	if err != nil {
		return
	}

	modules = s.registry.ModuleNames()
	grants = make(map[string][]string)

	for _, role := range roles {
		keys, e := s.repo.GetRolePermissionKeys(ctx, role.ID)
		if e != nil {
			continue
		}
		grants[role.ID.String()] = keys
	}
	return
}

// CountUsersWithRole returns how many users are assigned to a role.
func (s *RBACService) CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int64, error) {
	return s.repo.CountUsersWithRole(ctx, roleID)
}

// ═══════════════════════════════════════════════════════════════════════════
// Policies
// ═══════════════════════════════════════════════════════════════════════════

// CreatePolicy creates a new access control policy.
func (s *RBACService) CreatePolicy(ctx context.Context, policy *models.Policy) error {
	if _, err := s.repo.GetPermissionByKey(ctx, policy.PermissionKey); err != nil {
		return fmt.Errorf("unknown permission key '%s': %w", policy.PermissionKey, err)
	}
	if err := s.repo.CreatePolicy(ctx, policy); err != nil {
		return err
	}
	s.logAudit(ctx, policy.CreatedBy, "policy.created", "policy", &policy.ID, nil, policy)
	return nil
}

// ListPolicies returns paginated policies for a tenant.
func (s *RBACService) ListPolicies(ctx context.Context, tenantID *uuid.UUID, page, perPage int) ([]models.Policy, int64, error) {
	return s.repo.ListPolicies(ctx, tenantID, page, perPage)
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

func (s *RBACService) logAudit(ctx context.Context, actorID *uuid.UUID, action, resource string, resourceID *uuid.UUID, oldVal, newVal interface{}) {
	if s.auditSvc == nil {
		return
	}
	s.auditSvc.Log(ctx, actorID, action, resource, resourceID, oldVal, newVal, "", "")
}

// AuditPermissionDenied records denied permission checks from middleware.
func (s *RBACService) AuditPermissionDenied(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID, permKeys []string, method, path, reason, ipAddress, userAgent string) {
	s.logAudit(ctx, &userID, "permission.denied", "permission", nil, nil, map[string]interface{}{
		"tenant_id":   tenantID,
		"permissions": permKeys,
		"method":      method,
		"path":        path,
		"reason":      reason,
		"ip_address":  ipAddress,
		"user_agent":  userAgent,
	})
}

func (s *RBACService) allowedByPolicies(ctx context.Context, tenantID *uuid.UUID, permKey string, evalCtx rbac.EvaluationContext) bool {
	if s.policyEngine == nil {
		return true
	}

	policies, err := s.repo.GetActivePolicies(ctx, permKey, tenantID)
	if err != nil {
		slog.Warn("rbac: failed to load active policies", slog.Any("error", err), slog.String("permission", permKey))
		return false
	}
	if len(policies) == 0 {
		return true
	}

	if evalCtx.CurrentTime.IsZero() {
		evalCtx.CurrentTime = time.Now()
	}
	if evalCtx.Timezone == "" {
		evalCtx.Timezone = "Africa/Nairobi"
	}
	if evalCtx.RiskLevel == "" {
		if def, ok := s.registry.GetByKey(permKey); ok {
			evalCtx.RiskLevel = def.RiskLevel
		}
	}

	result := s.policyEngine.Evaluate(ctx, policies, evalCtx)
	if !result.Allowed {
		slog.Warn("rbac: policy denied permission",
			slog.String("permission", permKey),
			slog.String("policy", result.DeniedBy),
			slog.String("reason", result.Reason),
		)
	}
	return result.Allowed
}

func (s *RBACService) resolvePermissions(ctx context.Context, permKeys []string) ([]models.PermissionDef, error) {
	perms, err := s.repo.GetPermissionsByKeys(ctx, permKeys)
	if err != nil {
		return nil, err
	}
	if len(perms) != len(permKeys) {
		found := make(map[string]bool, len(perms))
		for _, p := range perms {
			found[p.Key] = true
		}
		var missing []string
		for _, key := range permKeys {
			if !found[key] {
				missing = append(missing, key)
			}
		}
		return nil, fmt.Errorf("unknown permission keys: %s", strings.Join(missing, ", "))
	}
	return perms, nil
}

func validatePermissionDependencies(perms []models.PermissionDef, permKeys []string) error {
	selected := toSet(permKeys)
	for _, p := range perms {
		for _, dep := range p.DependsOn {
			if !selected[dep] {
				return fmt.Errorf("permission '%s' depends on missing permission '%s'", p.Key, dep)
			}
		}
	}
	return nil
}

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Remove non-alphanumeric (except hyphens)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		result = append(result, v)
	}
	return result
}

// getSystemRolePermissions looks up the RBAC system role that matches
// the given system_role (via SystemRoleSlug) and returns its permission
// keys from the database. Returns nil if the role doesn't exist.
func (s *RBACService) getSystemRolePermissions(ctx context.Context, sysRole types.SystemRole) []string {
	slug := sysRole.SystemRoleSlug()
	if slug == "" {
		return nil
	}
	role, err := s.repo.GetRoleBySlug(ctx, slug, nil)
	if err != nil || role == nil {
		return nil
	}
	keys, err := s.repo.GetRolePermissionKeys(ctx, role.ID)
	if err != nil {
		slog.Warn("rbac: failed to load system role permissions",
			slog.String("slug", slug), slog.Any("error", err))
		return nil
	}
	return keys
}
