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

// WorkSiteRepo is the GORM implementation of repository.WorkSiteRepository.
type WorkSiteRepo struct {
	db *gorm.DB
}

func NewWorkSiteRepo(db *gorm.DB) *WorkSiteRepo {
	return &WorkSiteRepo{db: db}
}

func (r *WorkSiteRepo) Create(ctx context.Context, site *models.WorkSite) error {
	if err := r.db.WithContext(ctx).Create(site).Error; err != nil {
		return fmt.Errorf("create work site: %w", err)
	}
	return nil
}

func (r *WorkSiteRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkSite, error) {
	var site models.WorkSite
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&site).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get work site by id: %w", err)
	}
	return &site, nil
}

func (r *WorkSiteRepo) Update(ctx context.Context, site *models.WorkSite) error {
	if err := r.db.WithContext(ctx).Save(site).Error; err != nil {
		return fmt.Errorf("update work site: %w", err)
	}
	return nil
}

func (r *WorkSiteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.WorkSite{}).Error; err != nil {
		return fmt.Errorf("delete work site: %w", err)
	}
	return nil
}

func (r *WorkSiteRepo) List(ctx context.Context, orgID *uuid.UUID, page, perPage int, search string) ([]models.WorkSite, int64, error) {
	var sites []models.WorkSite
	var total int64

	query := r.db.WithContext(ctx).Model(&models.WorkSite{})
	if orgID != nil {
		query = query.Where("organization_id = ?", *orgID)
	}
	if search != "" {
		s := "%" + search + "%"
		query = query.Where("(name ILIKE ? OR project_ref ILIKE ? OR address ILIKE ?)", s, s, s)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("name ASC").Find(&sites).Error; err != nil {
		return nil, 0, fmt.Errorf("list work sites: %w", err)
	}

	return sites, total, nil
}
