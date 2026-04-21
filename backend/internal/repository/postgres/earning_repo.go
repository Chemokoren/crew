package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EarningRepo is the GORM implementation of repository.EarningRepository.
type EarningRepo struct {
	db *gorm.DB
}

func NewEarningRepo(db *gorm.DB) *EarningRepo {
	return &EarningRepo{db: db}
}

// getDB returns the transaction from context if present, otherwise the default DB.
func (r *EarningRepo) getDB(ctx context.Context) *gorm.DB {
	if tx := database.ExtractTx(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *EarningRepo) Create(ctx context.Context, earning *models.Earning) error {
	if err := r.getDB(ctx).Create(earning).Error; err != nil {
		return fmt.Errorf("create earning: %w", err)
	}
	return nil
}

func (r *EarningRepo) BulkCreate(ctx context.Context, earnings []models.Earning) (int, []repository.BulkError, error) {
	var bulkErrors []repository.BulkError
	created := 0

	err := r.getDB(ctx).Transaction(func(tx *gorm.DB) error {
		for i, e := range earnings {
			if err := tx.Create(&e).Error; err != nil {
				bulkErrors = append(bulkErrors, repository.BulkError{
					Index: i,
					Error: err.Error(),
				})
				continue
			}
			earnings[i] = e
			created++
		}
		return nil
	})

	if err != nil {
		return 0, nil, fmt.Errorf("bulk create earnings: %w", err)
	}

	return created, bulkErrors, nil
}

func (r *EarningRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Earning, error) {
	var earning models.Earning
	if err := r.getDB(ctx).
		Preload("Assignment").
		Preload("CrewMember").
		Where("id = ?", id).First(&earning).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get earning by id: %w", err)
	}
	return &earning, nil
}

func (r *EarningRepo) Update(ctx context.Context, earning *models.Earning) error {
	if err := r.getDB(ctx).Save(earning).Error; err != nil {
		return fmt.Errorf("update earning: %w", err)
	}
	return nil
}

func (r *EarningRepo) List(ctx context.Context, filter repository.EarningFilter, page, perPage int) ([]models.Earning, int64, error) {
	var earnings []models.Earning
	var total int64

	query := r.getDB(ctx).Model(&models.Earning{})

	if filter.CrewMemberID != nil {
		query = query.Where("crew_member_id = ?", *filter.CrewMemberID)
	}
	if filter.AssignmentID != nil {
		query = query.Where("assignment_id = ?", *filter.AssignmentID)
	}
	if filter.EarningType != "" {
		query = query.Where("earning_type = ?", filter.EarningType)
	}
	if filter.DateFrom != nil {
		query = query.Where("earned_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("earned_at <= ?", *filter.DateTo)
	}
	if filter.IsVerified != nil {
		query = query.Where("is_verified = ?", *filter.IsVerified)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("earned_at DESC").Find(&earnings).Error; err != nil {
		return nil, 0, fmt.Errorf("list earnings: %w", err)
	}

	return earnings, total, nil
}

func (r *EarningRepo) GetDailySummary(ctx context.Context, crewMemberID uuid.UUID, date time.Time) (*models.DailyEarningsSummary, error) {
	var summary models.DailyEarningsSummary
	if err := r.getDB(ctx).
		Where("crew_member_id = ? AND date = ?", crewMemberID, date).
		First(&summary).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get daily summary: %w", err)
	}
	return &summary, nil
}

// UpsertDailySummary creates or updates a daily earnings summary.
// Uses PostgreSQL ON CONFLICT for atomic upsert.
func (r *EarningRepo) UpsertDailySummary(ctx context.Context, summary *models.DailyEarningsSummary) error {
	if err := r.getDB(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "crew_member_id"}, {Name: "date"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"total_earned_cents", "total_deductions_cents", "net_amount_cents",
				"assignment_count", "is_processed", "updated_at",
			}),
		}).Create(summary).Error; err != nil {
		return fmt.Errorf("upsert daily summary: %w", err)
	}
	return nil
}
