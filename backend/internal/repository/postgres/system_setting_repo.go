package postgres

import (
	"context"

	"github.com/kibsoft/amy-mis/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SystemSettingRepo is the GORM implementation of repository.SystemSettingRepository.
type SystemSettingRepo struct {
	db *gorm.DB
}

func NewSystemSettingRepo(db *gorm.DB) *SystemSettingRepo {
	return &SystemSettingRepo{db: db}
}

func (r *SystemSettingRepo) Get(ctx context.Context, key string) (*models.SystemSetting, error) {
	var s models.SystemSetting
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SystemSettingRepo) Set(ctx context.Context, setting *models.SystemSetting) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "category", "label", "updated_at"}),
	}).Create(setting).Error
}

func (r *SystemSettingRepo) GetAll(ctx context.Context) ([]models.SystemSetting, error) {
	var settings []models.SystemSetting
	err := r.db.WithContext(ctx).Order("category, key").Find(&settings).Error
	return settings, err
}

func (r *SystemSettingRepo) GetByPrefix(ctx context.Context, prefix string) ([]models.SystemSetting, error) {
	var settings []models.SystemSetting
	err := r.db.WithContext(ctx).Where("key LIKE ?", prefix+"%").Order("key").Find(&settings).Error
	return settings, err
}

func (r *SystemSettingRepo) Delete(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Where("key = ?", key).Delete(&models.SystemSetting{}).Error
}

func (r *SystemSettingRepo) BulkSet(ctx context.Context, settings []models.SystemSetting) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range settings {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "category", "label", "updated_at"}),
			}).Create(&settings[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
