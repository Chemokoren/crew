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

// RouteRepo is the GORM implementation of repository.RouteRepository.
type RouteRepo struct {
	db *gorm.DB
}

func NewRouteRepo(db *gorm.DB) *RouteRepo {
	return &RouteRepo{db: db}
}

func (r *RouteRepo) Create(ctx context.Context, route *models.Route) error {
	if err := r.db.WithContext(ctx).Create(route).Error; err != nil {
		return fmt.Errorf("create route: %w", err)
	}
	return nil
}

func (r *RouteRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Route, error) {
	var route models.Route
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&route).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get route by id: %w", err)
	}
	return &route, nil
}

func (r *RouteRepo) Update(ctx context.Context, route *models.Route) error {
	if err := r.db.WithContext(ctx).Save(route).Error; err != nil {
		return fmt.Errorf("update route: %w", err)
	}
	return nil
}

func (r *RouteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Route{}).Error; err != nil {
		return fmt.Errorf("delete route: %w", err)
	}
	return nil
}

func (r *RouteRepo) List(ctx context.Context, page, perPage int, search string) ([]models.Route, int64, error) {
	var routes []models.Route
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Route{})
	if search != "" {
		s := "%" + search + "%"
		query = query.Where("(name ILIKE ? OR start_point ILIKE ? OR end_point ILIKE ?)", s, s, s)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("name ASC").Find(&routes).Error; err != nil {
		return nil, 0, fmt.Errorf("list routes: %w", err)
	}

	return routes, total, nil
}
