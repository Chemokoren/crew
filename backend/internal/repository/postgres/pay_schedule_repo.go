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

// PayScheduleRepo is the GORM implementation of repository.PayScheduleRepository.
type PayScheduleRepo struct {
	db *gorm.DB
}

func NewPayScheduleRepo(db *gorm.DB) *PayScheduleRepo {
	return &PayScheduleRepo{db: db}
}

func (r *PayScheduleRepo) Create(ctx context.Context, schedule *models.PaySchedule) error {
	if err := r.db.WithContext(ctx).Create(schedule).Error; err != nil {
		return fmt.Errorf("create pay schedule: %w", err)
	}
	return nil
}

func (r *PayScheduleRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PaySchedule, error) {
	var ps models.PaySchedule
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&ps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get pay schedule by id: %w", err)
	}
	return &ps, nil
}

func (r *PayScheduleRepo) Update(ctx context.Context, schedule *models.PaySchedule) error {
	if err := r.db.WithContext(ctx).Save(schedule).Error; err != nil {
		return fmt.Errorf("update pay schedule: %w", err)
	}
	return nil
}

func (r *PayScheduleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.PaySchedule{}).Error; err != nil {
		return fmt.Errorf("delete pay schedule: %w", err)
	}
	return nil
}

func (r *PayScheduleRepo) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]models.PaySchedule, error) {
	var schedules []models.PaySchedule
	if err := r.db.WithContext(ctx).
		Where("sacco_id = ? AND is_active = true", orgID).
		Order("is_default DESC, name ASC").
		Find(&schedules).Error; err != nil {
		return nil, fmt.Errorf("list pay schedules: %w", err)
	}
	return schedules, nil
}

func (r *PayScheduleRepo) GetDefault(ctx context.Context, orgID uuid.UUID) (*models.PaySchedule, error) {
	var ps models.PaySchedule
	if err := r.db.WithContext(ctx).
		Where("sacco_id = ? AND is_default = true AND is_active = true", orgID).
		First(&ps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get default pay schedule: %w", err)
	}
	return &ps, nil
}
