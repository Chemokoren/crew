package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"gorm.io/gorm"
)

// TenantJobTypeRepo is the GORM implementation of repository.TenantJobTypeRepository.
type TenantJobTypeRepo struct {
	db *gorm.DB
}

func NewTenantJobTypeRepo(db *gorm.DB) *TenantJobTypeRepo {
	return &TenantJobTypeRepo{db: db}
}

func (r *TenantJobTypeRepo) Create(ctx context.Context, jobType *models.TenantJobType) error {
	if err := r.db.WithContext(ctx).Create(jobType).Error; err != nil {
		return fmt.Errorf("create tenant job type: %w", err)
	}
	return nil
}

func (r *TenantJobTypeRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.TenantJobType, error) {
	var jt models.TenantJobType
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&jt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get tenant job type by id: %w", err)
	}
	return &jt, nil
}

func (r *TenantJobTypeRepo) Update(ctx context.Context, jobType *models.TenantJobType) error {
	if err := r.db.WithContext(ctx).Save(jobType).Error; err != nil {
		return fmt.Errorf("update tenant job type: %w", err)
	}
	return nil
}

func (r *TenantJobTypeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.TenantJobType{}).Error; err != nil {
		return fmt.Errorf("delete tenant job type: %w", err)
	}
	return nil
}

func (r *TenantJobTypeRepo) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]models.TenantJobType, error) {
	var jobTypes []models.TenantJobType
	if err := r.db.WithContext(ctx).
		Where("sacco_id = ? AND is_active = true", orgID).
		Order("sort_order ASC, display_name ASC").
		Find(&jobTypes).Error; err != nil {
		return nil, fmt.Errorf("list tenant job types: %w", err)
	}
	return jobTypes, nil
}

func (r *TenantJobTypeRepo) GetByCode(ctx context.Context, orgID uuid.UUID, code string) (*models.TenantJobType, error) {
	var jt models.TenantJobType
	if err := r.db.WithContext(ctx).
		Where("sacco_id = ? AND code = ?", orgID, code).
		First(&jt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get tenant job type by code: %w", err)
	}
	return &jt, nil
}
