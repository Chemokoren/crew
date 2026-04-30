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
)

// AssignmentRepo is the GORM implementation of repository.AssignmentRepository.
type AssignmentRepo struct {
	db *gorm.DB
}

func NewAssignmentRepo(db *gorm.DB) *AssignmentRepo {
	return &AssignmentRepo{db: db}
}

// getDB returns the transaction from context if present, otherwise the default DB.
func (r *AssignmentRepo) getDB(ctx context.Context) *gorm.DB {
	if tx := database.ExtractTx(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *AssignmentRepo) Create(ctx context.Context, assignment *models.Assignment) error {
	if err := r.getDB(ctx).Create(assignment).Error; err != nil {
		return fmt.Errorf("create assignment: %w", err)
	}
	return nil
}

func (r *AssignmentRepo) BulkCreate(ctx context.Context, assignments []models.Assignment) (int, []repository.BulkError, error) {
	var bulkErrors []repository.BulkError
	created := 0

	err := r.getDB(ctx).Transaction(func(tx *gorm.DB) error {
		for i, a := range assignments {
			if err := tx.Create(&a).Error; err != nil {
				bulkErrors = append(bulkErrors, repository.BulkError{
					Index: i,
					Error: err.Error(),
				})
				continue
			}
			assignments[i] = a
			created++
		}
		return nil
	})

	if err != nil {
		return 0, nil, fmt.Errorf("bulk create assignments: %w", err)
	}

	return created, bulkErrors, nil
}

func (r *AssignmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Assignment, error) {
	var assignment models.Assignment
	if err := r.getDB(ctx).
		Preload("CrewMember").
		Preload("Vehicle").
		Preload("Sacco").
		Preload("Route").
		Where("id = ?", id).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get assignment by id: %w", err)
	}
	return &assignment, nil
}

func (r *AssignmentRepo) Update(ctx context.Context, assignment *models.Assignment) error {
	if err := r.getDB(ctx).Save(assignment).Error; err != nil {
		return fmt.Errorf("update assignment: %w", err)
	}
	return nil
}

func (r *AssignmentRepo) List(ctx context.Context, filter repository.AssignmentFilter, page, perPage int) ([]models.Assignment, int64, error) {
	var assignments []models.Assignment
	var total int64

	query := r.getDB(ctx).Model(&models.Assignment{})

	if filter.SaccoID != nil {
		query = query.Where("sacco_id = ?", *filter.SaccoID)
	}
	if filter.CrewMemberID != nil {
		query = query.Where("crew_member_id = ?", *filter.CrewMemberID)
	}
	if filter.VehicleID != nil {
		query = query.Where("vehicle_id = ?", *filter.VehicleID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ShiftDate != nil {
		query = query.Where("shift_date = ?", *filter.ShiftDate)
	}
	if filter.DateFrom != nil {
		query = query.Where("shift_date >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("shift_date <= ?", *filter.DateTo)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.
		Preload("CrewMember").
		Preload("Vehicle").
		Preload("Sacco").
		Preload("Route").
		Offset(offset).Limit(perPage).
		Order("shift_date DESC, shift_start DESC").
		Find(&assignments).Error; err != nil {
		return nil, 0, fmt.Errorf("list assignments: %w", err)
	}

	return assignments, total, nil
}

func (r *AssignmentRepo) HasActiveAssignment(ctx context.Context, crewMemberID uuid.UUID, date time.Time) (bool, error) {
	var count int64
	err := r.getDB(ctx).Model(&models.Assignment{}).
		Where("crew_member_id = ? AND shift_date = ? AND status IN ?", crewMemberID, date, []string{"SCHEDULED", "ACTIVE"}).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check active assignment: %w", err)
	}
	return count > 0, nil
}
