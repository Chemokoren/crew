package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"gorm.io/gorm"
)

type negativeEventRepo struct {
	db *gorm.DB
}

func NewNegativeEventRepo(db *gorm.DB) *negativeEventRepo {
	return &negativeEventRepo{db: db}
}

func (r *negativeEventRepo) Create(ctx context.Context, event *models.CreditNegativeEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *negativeEventRepo) CountUnresolved(ctx context.Context, crewMemberID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.CreditNegativeEvent{}).
		Where("crew_member_id = ? AND resolved = false", crewMemberID).
		Count(&count).Error
	return count, err
}

func (r *negativeEventRepo) CountByType(ctx context.Context, crewMemberID uuid.UUID, eventType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.CreditNegativeEvent{}).
		Where("crew_member_id = ? AND event_type = ? AND resolved = false", crewMemberID, eventType).
		Count(&count).Error
	return count, err
}
