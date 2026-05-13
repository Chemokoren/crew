// Package mock provides in-memory RBAC repository for testing.
package mock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

// RBACRepo is an in-memory implementation of repository.RBACRepository.
type RBACRepo struct {
	mu              sync.RWMutex
	roles           map[uuid.UUID]*models.Role
	permissions     map[uuid.UUID]*models.PermissionDef
	rolePermissions map[uuid.UUID]map[uuid.UUID]bool // roleID -> set of permIDs
	userRoles       []models.UserRole
	policies        []models.Policy
	templates       map[uuid.UUID]*models.RoleTemplate
}

func NewRBACRepo() *RBACRepo {
	return &RBACRepo{
		roles:           make(map[uuid.UUID]*models.Role),
		permissions:     make(map[uuid.UUID]*models.PermissionDef),
		rolePermissions: make(map[uuid.UUID]map[uuid.UUID]bool),
		templates:       make(map[uuid.UUID]*models.RoleTemplate),
	}
}

// --- Roles ---

func (r *RBACRepo) CreateRole(_ context.Context, role *models.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now
	r.roles[role.ID] = role
	return nil
}

func (r *RBACRepo) GetRoleByID(_ context.Context, id uuid.UUID) (*models.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if role, ok := r.roles[id]; ok {
		return role, nil
	}
	return nil, errs.ErrNotFound
}

func (r *RBACRepo) GetRoleBySlug(_ context.Context, slug string, tenantID *uuid.UUID) (*models.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, role := range r.roles {
		if role.Slug == slug {
			if tenantID == nil || role.TenantID == nil || *role.TenantID == *tenantID {
				return role, nil
			}
		}
	}
	return nil, errs.ErrNotFound
}

func (r *RBACRepo) UpdateRole(_ context.Context, role *models.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	role.UpdatedAt = time.Now()
	r.roles[role.ID] = role
	return nil
}

func (r *RBACRepo) DeleteRole(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.roles[id]; !ok {
		return errs.ErrNotFound
	}
	delete(r.roles, id)
	delete(r.rolePermissions, id)
	return nil
}

func (r *RBACRepo) ListRoles(_ context.Context, filter repository.RoleFilter, page, perPage int) ([]models.Role, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []models.Role
	for _, role := range r.roles {
		if filter.IndustryType != "" && role.IndustryType != filter.IndustryType {
			continue
		}
		if filter.IsSystem != nil && role.IsSystem != *filter.IsSystem {
			continue
		}
		if filter.IsActive != nil && role.IsActive != *filter.IsActive {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(role.Name), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, *role)
	}
	total := int64(len(result))
	start := (page - 1) * perPage
	if start >= len(result) {
		return nil, total, nil
	}
	end := start + perPage
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], total, nil
}

func (r *RBACRepo) CloneRole(_ context.Context, sourceID uuid.UUID, newName, newSlug string, tenantID *uuid.UUID, createdBy *uuid.UUID) (*models.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	src, ok := r.roles[sourceID]
	if !ok {
		return nil, errs.ErrNotFound
	}
	clone := *src
	clone.ID = uuid.New()
	clone.Name = newName
	clone.Slug = newSlug
	clone.TenantID = tenantID
	clone.CreatedBy = createdBy
	clone.IsSystem = false
	clone.IsTemplate = false
	now := time.Now()
	clone.CreatedAt = now
	clone.UpdatedAt = now
	r.roles[clone.ID] = &clone

	// Copy permissions
	if perms, ok := r.rolePermissions[sourceID]; ok {
		r.rolePermissions[clone.ID] = make(map[uuid.UUID]bool)
		for pid := range perms {
			r.rolePermissions[clone.ID][pid] = true
		}
	}
	return &clone, nil
}

// --- Permissions ---

func (r *RBACRepo) ListPermissions(_ context.Context, filter repository.PermissionFilter) ([]models.PermissionDef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []models.PermissionDef
	for _, p := range r.permissions {
		if filter.Module != "" && p.Module != filter.Module {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(p.Key), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, *p)
	}
	return result, nil
}

func (r *RBACRepo) GetPermissionByKey(_ context.Context, key string) (*models.PermissionDef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.permissions {
		if p.Key == key {
			return p, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (r *RBACRepo) GetPermissionsByKeys(_ context.Context, keys []string) ([]models.PermissionDef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	var result []models.PermissionDef
	for _, p := range r.permissions {
		if keySet[p.Key] {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (r *RBACRepo) SyncPermissions(_ context.Context, defs []models.PermissionDef) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range defs {
		d := defs[i]
		found := false
		for _, existing := range r.permissions {
			if existing.Key == d.Key {
				existing.Module = d.Module
				existing.Description = d.Description
				existing.RiskLevel = d.RiskLevel
				existing.Category = d.Category
				found = true
				break
			}
		}
		if !found {
			if d.ID == uuid.Nil {
				d.ID = uuid.New()
			}
			r.permissions[d.ID] = &d
		}
	}
	return nil
}

// --- Role-Permission mapping ---

func (r *RBACRepo) AssignPermissions(_ context.Context, roleID uuid.UUID, permIDs []uuid.UUID, _ *uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rolePermissions[roleID]; !ok {
		r.rolePermissions[roleID] = make(map[uuid.UUID]bool)
	}
	for _, pid := range permIDs {
		r.rolePermissions[roleID][pid] = true
	}
	return nil
}

func (r *RBACRepo) RevokePermissions(_ context.Context, roleID uuid.UUID, permIDs []uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if perms, ok := r.rolePermissions[roleID]; ok {
		for _, pid := range permIDs {
			delete(perms, pid)
		}
	}
	return nil
}

func (r *RBACRepo) GetRolePermissions(_ context.Context, roleID uuid.UUID) ([]models.PermissionDef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []models.PermissionDef
	if permIDs, ok := r.rolePermissions[roleID]; ok {
		for pid := range permIDs {
			if p, ok := r.permissions[pid]; ok {
				result = append(result, *p)
			}
		}
	}
	return result, nil
}

func (r *RBACRepo) GetRolePermissionKeys(_ context.Context, roleID uuid.UUID) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []string
	if permIDs, ok := r.rolePermissions[roleID]; ok {
		for pid := range permIDs {
			if p, ok := r.permissions[pid]; ok {
				result = append(result, p.Key)
			}
		}
	}
	return result, nil
}

func (r *RBACRepo) BulkSetPermissions(_ context.Context, roleID uuid.UUID, permIDs []uuid.UUID, _ *uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rolePermissions[roleID] = make(map[uuid.UUID]bool)
	for _, pid := range permIDs {
		r.rolePermissions[roleID][pid] = true
	}
	return nil
}

// --- User-Role mapping ---

func (r *RBACRepo) AssignRoleToUser(_ context.Context, ur *models.UserRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.userRoles {
		if existing.UserID == ur.UserID && existing.RoleID == ur.RoleID {
			return fmt.Errorf("duplicate assignment")
		}
	}
	r.userRoles = append(r.userRoles, *ur)
	return nil
}

func (r *RBACRepo) RevokeRoleFromUser(_ context.Context, userID, roleID uuid.UUID, _ *uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, ur := range r.userRoles {
		if ur.UserID == userID && ur.RoleID == roleID {
			r.userRoles = append(r.userRoles[:i], r.userRoles[i+1:]...)
			return nil
		}
	}
	return errs.ErrNotFound
}

func (r *RBACRepo) GetUserRoles(_ context.Context, userID uuid.UUID, _ *uuid.UUID) ([]models.UserRole, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []models.UserRole
	for _, ur := range r.userRoles {
		if ur.UserID == userID {
			if role, ok := r.roles[ur.RoleID]; ok {
				ur.Role = *role
			}
			result = append(result, ur)
		}
	}
	return result, nil
}

func (r *RBACRepo) GetUserPermissionKeys(_ context.Context, userID uuid.UUID, _ *uuid.UUID) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keySet := make(map[string]bool)
	for _, ur := range r.userRoles {
		if ur.UserID == userID {
			if permIDs, ok := r.rolePermissions[ur.RoleID]; ok {
				for pid := range permIDs {
					if p, ok := r.permissions[pid]; ok {
						keySet[p.Key] = true
					}
				}
			}
		}
	}
	var result []string
	for k := range keySet {
		result = append(result, k)
	}
	return result, nil
}

func (r *RBACRepo) CountUsersWithRole(_ context.Context, roleID uuid.UUID) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var count int64
	for _, ur := range r.userRoles {
		if ur.RoleID == roleID {
			count++
		}
	}
	return count, nil
}

// --- Policies ---

func (r *RBACRepo) CreatePolicy(_ context.Context, policy *models.Policy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	r.policies = append(r.policies, *policy)
	return nil
}

func (r *RBACRepo) UpdatePolicy(_ context.Context, policy *models.Policy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.policies {
		if p.ID == policy.ID {
			r.policies[i] = *policy
			return nil
		}
	}
	return errs.ErrNotFound
}

func (r *RBACRepo) GetActivePolicies(_ context.Context, permKey string, _ *uuid.UUID) ([]models.Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []models.Policy
	for _, p := range r.policies {
		if p.PermissionKey == permKey && p.IsActive {
			result = append(result, p)
		}
	}
	return result, nil
}

func (r *RBACRepo) ListPolicies(_ context.Context, _ *uuid.UUID, page, perPage int) ([]models.Policy, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	total := int64(len(r.policies))
	start := (page - 1) * perPage
	if start >= len(r.policies) {
		return nil, total, nil
	}
	end := start + perPage
	if end > len(r.policies) {
		end = len(r.policies)
	}
	return r.policies[start:end], total, nil
}

// --- Templates ---

func (r *RBACRepo) UpsertTemplate(_ context.Context, tmpl *models.RoleTemplate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tmpl.ID == uuid.Nil {
		tmpl.ID = uuid.New()
	}
	r.templates[tmpl.ID] = tmpl
	return nil
}

func (r *RBACRepo) ListTemplates(_ context.Context, industryType string) ([]models.RoleTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []models.RoleTemplate
	for _, t := range r.templates {
		if industryType == "" || t.IndustryType == industryType {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (r *RBACRepo) GetTemplateByID(_ context.Context, id uuid.UUID) (*models.RoleTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.templates[id]; ok {
		return t, nil
	}
	return nil, errs.ErrNotFound
}

// Helper: Seed adds a permission directly for test setup.
func (r *RBACRepo) Seed(perms ...models.PermissionDef) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range perms {
		p := perms[i]
		if p.ID == uuid.Nil {
			p.ID = uuid.New()
		}
		r.permissions[p.ID] = &p
	}
}
