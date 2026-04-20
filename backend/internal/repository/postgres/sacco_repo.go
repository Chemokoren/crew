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

// SACCORepo is the GORM implementation of repository.SACCORepository.
type SACCORepo struct {
	db *gorm.DB
}

func NewSACCORepo(db *gorm.DB) *SACCORepo {
	return &SACCORepo{db: db}
}

func (r *SACCORepo) Create(ctx context.Context, sacco *models.SACCO) error {
	if err := r.db.WithContext(ctx).Create(sacco).Error; err != nil {
		return fmt.Errorf("create sacco: %w", err)
	}
	return nil
}

func (r *SACCORepo) GetByID(ctx context.Context, id uuid.UUID) (*models.SACCO, error) {
	var sacco models.SACCO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&sacco).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get sacco by id: %w", err)
	}
	return &sacco, nil
}

func (r *SACCORepo) Update(ctx context.Context, sacco *models.SACCO) error {
	if err := r.db.WithContext(ctx).Save(sacco).Error; err != nil {
		return fmt.Errorf("update sacco: %w", err)
	}
	return nil
}

func (r *SACCORepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.SACCO{}).Error; err != nil {
		return fmt.Errorf("delete sacco: %w", err)
	}
	return nil
}

func (r *SACCORepo) List(ctx context.Context, page, perPage int, search string) ([]models.SACCO, int64, error) {
	var saccos []models.SACCO
	var total int64

	query := r.db.WithContext(ctx).Model(&models.SACCO{})
	if search != "" {
		s := "%" + search + "%"
		query = query.Where("(name ILIKE ? OR registration_number ILIKE ? OR county ILIKE ?)", s, s, s)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("name ASC").Find(&saccos).Error; err != nil {
		return nil, 0, fmt.Errorf("list saccos: %w", err)
	}

	return saccos, total, nil
}
