package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type creditScoreRepo struct {
	db *gorm.DB
}

// NewCreditScoreRepo creates a new Postgres-backed CreditScoreRepository.
func NewCreditScoreRepo(db *gorm.DB) repository.CreditScoreRepository {
	return &creditScoreRepo{db: db}
}

func (r *creditScoreRepo) GetByCrewMemberID(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error) {
	var score models.CreditScore
	err := r.db.WithContext(ctx).Where("crew_member_id = ?", crewMemberID).First(&score).Error
	if err != nil {
		return nil, err
	}
	return &score, nil
}

func (r *creditScoreRepo) Upsert(ctx context.Context, score *models.CreditScore) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "crew_member_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"score", "factors", "last_calculated_at", "updated_at"}),
	}).Create(score).Error
}
