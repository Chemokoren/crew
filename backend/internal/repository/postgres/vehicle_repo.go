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

// VehicleRepo is the GORM implementation of repository.VehicleRepository.
type VehicleRepo struct {
	db *gorm.DB
}

func NewVehicleRepo(db *gorm.DB) *VehicleRepo {
	return &VehicleRepo{db: db}
}

func (r *VehicleRepo) Create(ctx context.Context, vehicle *models.Vehicle) error {
	if err := r.db.WithContext(ctx).Create(vehicle).Error; err != nil {
		return fmt.Errorf("create vehicle: %w", err)
	}
	return nil
}

func (r *VehicleRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Vehicle, error) {
	var vehicle models.Vehicle
	if err := r.db.WithContext(ctx).Preload("Sacco").Preload("Route").Where("id = ?", id).First(&vehicle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get vehicle by id: %w", err)
	}
	return &vehicle, nil
}

func (r *VehicleRepo) Update(ctx context.Context, vehicle *models.Vehicle) error {
	if err := r.db.WithContext(ctx).Save(vehicle).Error; err != nil {
		return fmt.Errorf("update vehicle: %w", err)
	}
	return nil
}

func (r *VehicleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Vehicle{}).Error; err != nil {
		return fmt.Errorf("delete vehicle: %w", err)
	}
	return nil
}

func (r *VehicleRepo) List(ctx context.Context, saccoID *uuid.UUID, page, perPage int) ([]models.Vehicle, int64, error) {
	var vehicles []models.Vehicle
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Vehicle{})
	if saccoID != nil {
		query = query.Where("sacco_id = ?", *saccoID)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Preload("Sacco").Preload("Route").Offset(offset).Limit(perPage).Order("registration_no ASC").Find(&vehicles).Error; err != nil {
		return nil, 0, fmt.Errorf("list vehicles: %w", err)
	}

	return vehicles, total, nil
}
