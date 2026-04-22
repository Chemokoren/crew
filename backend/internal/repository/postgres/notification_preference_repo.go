package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationPreferenceRepo struct {
	db *gorm.DB
}

func NewNotificationPreferenceRepo(db *gorm.DB) *NotificationPreferenceRepo {
	return &NotificationPreferenceRepo{db: db}
}

func (r *NotificationPreferenceRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.NotificationPreference, error) {
	var p models.NotificationPreference
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get notification preferences: %w", err)
	}
	return &p, nil
}

func (r *NotificationPreferenceRepo) Upsert(ctx context.Context, p *models.NotificationPreference) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"sms_opt_in", "push_opt_in", "in_app_opt_in", "marketing_opt_in", "updated_at"}),
	}).Create(p).Error
}
