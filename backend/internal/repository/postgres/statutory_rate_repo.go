package postgres

import (
	"context"
	"fmt"

	"github.com/kibsoft/amy-mis/internal/models"
	"gorm.io/gorm"
)

// StatutoryRateRepo is the GORM implementation of repository.StatutoryRateRepository.
type StatutoryRateRepo struct {
	db *gorm.DB
}

func NewStatutoryRateRepo(db *gorm.DB) *StatutoryRateRepo {
	return &StatutoryRateRepo{db: db}
}

func (r *StatutoryRateRepo) GetActiveRates(ctx context.Context) ([]models.StatutoryRate, error) {
	var rates []models.StatutoryRate
	if err := r.db.WithContext(ctx).Where("is_active = true").Order("name ASC").Find(&rates).Error; err != nil {
		return nil, fmt.Errorf("get active rates: %w", err)
	}
	return rates, nil
}

func (r *StatutoryRateRepo) Create(ctx context.Context, rate *models.StatutoryRate) error {
	if err := r.db.WithContext(ctx).Create(rate).Error; err != nil {
		return fmt.Errorf("create statutory rate: %w", err)
	}
	return nil
}

func (r *StatutoryRateRepo) Update(ctx context.Context, rate *models.StatutoryRate) error {
	if err := r.db.WithContext(ctx).Save(rate).Error; err != nil {
		return fmt.Errorf("update statutory rate: %w", err)
	}
	return nil
}
