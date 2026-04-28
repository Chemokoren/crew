package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"gorm.io/gorm"
)

type creditScoreHistoryRepo struct {
	db *gorm.DB
}

func NewCreditScoreHistoryRepo(db *gorm.DB) *creditScoreHistoryRepo {
	return &creditScoreHistoryRepo{db: db}
}

func (r *creditScoreHistoryRepo) Create(ctx context.Context, entry *models.CreditScoreHistory) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *creditScoreHistoryRepo) GetHistory(ctx context.Context, crewMemberID uuid.UUID, limit int) ([]models.CreditScoreHistory, error) {
	var history []models.CreditScoreHistory
	err := r.db.WithContext(ctx).
		Where("crew_member_id = ?", crewMemberID).
		Order("computed_at DESC").
		Limit(limit).
		Find(&history).Error
	if err != nil {
		return nil, fmt.Errorf("get score history: %w", err)
	}
	return history, nil
}
