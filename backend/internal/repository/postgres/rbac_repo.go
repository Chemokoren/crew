package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RBACRepo implements repository.RBACRepository using PostgreSQL via GORM.
type RBACRepo struct {
	db *gorm.DB
}

// NewRBACRepo creates a new RBAC repository.
func NewRBACRepo(db *gorm.DB) *RBACRepo {
	return &RBACRepo{db: db}
}

// ═══════════════════════════════════════════════════════════════════════════
// Roles
// ═══════════════════════════════════════════════════════════════════════════

func (r *RBACRepo) CreateRole(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *RBACRepo) GetRoleByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).First(&role, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RBACRepo) GetRoleBySlug(ctx context.Context, slug string, tenantID *uuid.UUID) (*models.Role, error) {
	var role models.Role
	q := r.db.WithContext(ctx).Where("slug = ?", slug)
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	} else {
		q = q.Where("tenant_id IS NULL")
	}
	if err := q.First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RBACRepo) UpdateRole(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *RBACRepo) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Role{}, "id = ?", id).Error
}

func (r *RBACRepo) ListRoles(ctx context.Context, filter repository.RoleFilter, page, perPage int) ([]models.Role, int64, error) {
	var roles []models.Role
	var total int64

	q := r.db.WithContext(ctx).Model(&models.Role{})

	if filter.TenantID != nil {
		q = q.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.IndustryType != "" {
		q = q.Where("industry_type = ?", filter.IndustryType)
	}
	if filter.IsSystem != nil {
		q = q.Where("is_system = ?", *filter.IsSystem)
	}
	if filter.IsTemplate != nil {
		q = q.Where("is_template = ?", *filter.IsTemplate)
	}
	if filter.IsActive != nil {
		q = q.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		q = q.Where("name ILIKE ? OR description ILIKE ?", search, search)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := q.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&roles).Error; err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

func (r *RBACRepo) CloneRole(ctx context.Context, sourceID uuid.UUID, newName, newSlug string, tenantID *uuid.UUID, createdBy *uuid.UUID) (*models.Role, error) {
	var cloned models.Role

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Load source role
		var source models.Role
		if err := tx.First(&source, "id = ?", sourceID).Error; err != nil {
			return fmt.Errorf("source role not found: %w", err)
		}

		// Create cloned role
		cloned = models.Role{
			Name:         newName,
			Slug:         newSlug,
			Description:  source.Description,
			TenantID:     tenantID,
			IndustryType: source.IndustryType,
			IsSystem:     false,
			IsTemplate:   false,
			IsActive:     true,
			ParentRoleID: &sourceID,
			Metadata:     source.Metadata,
			CreatedBy:    createdBy,
			UpdatedBy:    createdBy,
		}
		if err := tx.Create(&cloned).Error; err != nil {
			return err
		}

		// Copy permissions
		var perms []models.RolePermission
		if err := tx.Where("role_id = ?", sourceID).Find(&perms).Error; err != nil {
			return err
		}
		for _, p := range perms {
			newRP := models.RolePermission{
				RoleID:       cloned.ID,
				PermissionID: p.PermissionID,
				GrantedBy:    createdBy,
				GrantedAt:    time.Now(),
			}
			if err := tx.Create(&newRP).Error; err != nil {
				return err
			}
		}
		return nil
	})

	return &cloned, err
}

// ═══════════════════════════════════════════════════════════════════════════
// Permissions
// ═══════════════════════════════════════════════════════════════════════════

func (r *RBACRepo) ListPermissions(ctx context.Context, filter repository.PermissionFilter) ([]models.PermissionDef, error) {
	var perms []models.PermissionDef
	q := r.db.WithContext(ctx).Model(&models.PermissionDef{})

	if filter.Module != "" {
		q = q.Where("module = ?", filter.Module)
	}
	if filter.Category != "" {
		q = q.Where("category = ?", filter.Category)
	}
	if filter.RiskLevel != "" {
		q = q.Where("risk_level = ?", filter.RiskLevel)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		q = q.Where("key ILIKE ? OR description ILIKE ?", search, search)
	}

	err := q.Order("module, key").Find(&perms).Error
	return perms, err
}

func (r *RBACRepo) GetPermissionByKey(ctx context.Context, key string) (*models.PermissionDef, error) {
	var perm models.PermissionDef
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&perm).Error
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

func (r *RBACRepo) GetPermissionsByKeys(ctx context.Context, keys []string) ([]models.PermissionDef, error) {
	var perms []models.PermissionDef
	if len(keys) == 0 {
		return perms, nil
	}
	err := r.db.WithContext(ctx).Where("key IN ?", keys).Find(&perms).Error
	return perms, err
}

func (r *RBACRepo) SyncPermissions(ctx context.Context, defs []models.PermissionDef) error {
	if len(defs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"module", "description", "risk_level", "category", "depends_on", "metadata", "updated_at"}),
	}).CreateInBatches(defs, 50).Error
}

// ═══════════════════════════════════════════════════════════════════════════
// Role-Permission Mapping
// ═══════════════════════════════════════════════════════════════════════════

func (r *RBACRepo) AssignPermissions(ctx context.Context, roleID uuid.UUID, permIDs []uuid.UUID, grantedBy *uuid.UUID) error {
	if len(permIDs) == 0 {
		return nil
	}
	rps := make([]models.RolePermission, len(permIDs))
	now := time.Now()
	for i, pid := range permIDs {
		rps[i] = models.RolePermission{
			RoleID: roleID, PermissionID: pid,
			GrantedBy: grantedBy, GrantedAt: now,
		}
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rps).Error
}

func (r *RBACRepo) RevokePermissions(ctx context.Context, roleID uuid.UUID, permIDs []uuid.UUID) error {
	if len(permIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id IN ?", roleID, permIDs).
		Delete(&models.RolePermission{}).Error
}

func (r *RBACRepo) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.PermissionDef, error) {
	var perms []models.PermissionDef
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
		Where("rp.role_id = ?", roleID).
		Order("permissions.module, permissions.key").
		Find(&perms).Error
	return perms, err
}

func (r *RBACRepo) GetRolePermissionKeys(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	var keys []string
	err := r.db.WithContext(ctx).
		Model(&models.PermissionDef{}).
		Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
		Where("rp.role_id = ?", roleID).
		Pluck("permissions.key", &keys).Error
	return keys, err
}

func (r *RBACRepo) BulkSetPermissions(ctx context.Context, roleID uuid.UUID, permIDs []uuid.UUID, grantedBy *uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete all existing
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		// Insert new set
		if len(permIDs) == 0 {
			return nil
		}
		rps := make([]models.RolePermission, len(permIDs))
		now := time.Now()
		for i, pid := range permIDs {
			rps[i] = models.RolePermission{
				RoleID: roleID, PermissionID: pid,
				GrantedBy: grantedBy, GrantedAt: now,
			}
		}
		return tx.CreateInBatches(rps, 100).Error
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// User-Role Mapping
// ═══════════════════════════════════════════════════════════════════════════

func (r *RBACRepo) AssignRoleToUser(ctx context.Context, userRole *models.UserRole) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "role_id"}, {Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_active", "assigned_by", "assigned_at", "expires_at"}),
	}).Create(userRole).Error
}

func (r *RBACRepo) RevokeRoleFromUser(ctx context.Context, userID, roleID uuid.UUID, tenantID *uuid.UUID) error {
	q := r.db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, roleID)
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	} else {
		q = q.Where("tenant_id IS NULL")
	}
	return q.Update("is_active", false).Error
}

func (r *RBACRepo) GetUserRoles(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID) ([]models.UserRole, error) {
	var userRoles []models.UserRole
	q := r.db.WithContext(ctx).Preload("Role").
		Where("user_id = ? AND is_active = true", userID).
		Where("expires_at IS NULL OR expires_at > ?", time.Now())

	if tenantID != nil {
		q = q.Where("(tenant_id = ? OR tenant_id IS NULL)", *tenantID)
	}

	err := q.Find(&userRoles).Error
	return userRoles, err
}

func (r *RBACRepo) GetUserPermissionKeys(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID) ([]string, error) {
	var keys []string

	q := r.db.WithContext(ctx).
		Table("permissions p").
		Select("DISTINCT p.key").
		Joins("JOIN role_permissions rp ON rp.permission_id = p.id").
		Joins("JOIN user_roles ur ON ur.role_id = rp.role_id").
		Joins("JOIN roles ro ON ro.id = ur.role_id AND ro.is_active = true AND ro.deleted_at IS NULL").
		Where("ur.user_id = ? AND ur.is_active = true", userID).
		Where("ur.expires_at IS NULL OR ur.expires_at > ?", time.Now())

	if tenantID != nil {
		q = q.Where("(ur.tenant_id = ? OR ur.tenant_id IS NULL)", *tenantID)
	}

	err := q.Pluck("p.key", &keys).Error
	return keys, err
}

func (r *RBACRepo) CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserRole{}).
		Where("role_id = ? AND is_active = true", roleID).
		Count(&count).Error
	return count, err
}

// ═══════════════════════════════════════════════════════════════════════════
// Policies
// ═══════════════════════════════════════════════════════════════════════════

func (r *RBACRepo) CreatePolicy(ctx context.Context, policy *models.Policy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}

func (r *RBACRepo) UpdatePolicy(ctx context.Context, policy *models.Policy) error {
	return r.db.WithContext(ctx).Save(policy).Error
}

func (r *RBACRepo) GetActivePolicies(ctx context.Context, permKey string, tenantID *uuid.UUID) ([]models.Policy, error) {
	var policies []models.Policy
	q := r.db.WithContext(ctx).
		Where("permission_key = ? AND is_active = true", permKey).
		Order("priority DESC")

	if tenantID != nil {
		q = q.Where("(tenant_id = ? OR tenant_id IS NULL)", *tenantID)
	}

	err := q.Find(&policies).Error
	return policies, err
}

func (r *RBACRepo) ListPolicies(ctx context.Context, tenantID *uuid.UUID, page, perPage int) ([]models.Policy, int64, error) {
	var policies []models.Policy
	var total int64

	q := r.db.WithContext(ctx).Model(&models.Policy{})
	if tenantID != nil {
		q = q.Where("tenant_id = ? OR tenant_id IS NULL", *tenantID)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	err := q.Order("priority DESC, created_at DESC").Offset(offset).Limit(perPage).Find(&policies).Error
	return policies, total, err
}

// ═══════════════════════════════════════════════════════════════════════════
// Templates
// ═══════════════════════════════════════════════════════════════════════════

func (r *RBACRepo) UpsertTemplate(ctx context.Context, tmpl *models.RoleTemplate) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "role_slug"}, {Name: "industry_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"role_name", "description", "permissions", "is_default", "sort_order", "updated_at"}),
	}).Create(tmpl).Error
}

func (r *RBACRepo) ListTemplates(ctx context.Context, industryType string) ([]models.RoleTemplate, error) {
	var templates []models.RoleTemplate
	q := r.db.WithContext(ctx)
	if industryType != "" {
		q = q.Where("industry_type = ?", industryType)
	}
	err := q.Order("industry_type, sort_order").Find(&templates).Error
	return templates, err
}

func (r *RBACRepo) GetTemplateByID(ctx context.Context, id uuid.UUID) (*models.RoleTemplate, error) {
	var tmpl models.RoleTemplate
	err := r.db.WithContext(ctx).First(&tmpl, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &tmpl, nil
}

// Compile-time interface check.
var _ repository.RBACRepository = (*RBACRepo)(nil)
