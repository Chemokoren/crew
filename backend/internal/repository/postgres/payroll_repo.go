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

// PayrollRepo is the GORM implementation of repository.PayrollRepository.
type PayrollRepo struct {
	db *gorm.DB
}

func NewPayrollRepo(db *gorm.DB) *PayrollRepo {
	return &PayrollRepo{db: db}
}

func (r *PayrollRepo) Create(ctx context.Context, run *models.PayrollRun) error {
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		return fmt.Errorf("create payroll run: %w", err)
	}
	return nil
}

func (r *PayrollRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PayrollRun, error) {
	var run models.PayrollRun
	if err := r.db.WithContext(ctx).
		Preload("Entries").
		Preload("Entries.CrewMember").
		Where("id = ?", id).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get payroll run by id: %w", err)
	}
	return &run, nil
}

func (r *PayrollRepo) Update(ctx context.Context, run *models.PayrollRun) error {
	if err := r.db.WithContext(ctx).Save(run).Error; err != nil {
		return fmt.Errorf("update payroll run: %w", err)
	}
	return nil
}

func (r *PayrollRepo) List(ctx context.Context, saccoID *uuid.UUID, page, perPage int) ([]models.PayrollRun, int64, error) {
	var runs []models.PayrollRun
	var total int64

	query := r.db.WithContext(ctx).Model(&models.PayrollRun{})
	if saccoID != nil {
		query = query.Where("sacco_id = ?", *saccoID)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("period_start DESC").Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("list payroll runs: %w", err)
	}

	return runs, total, nil
}

func (r *PayrollRepo) CreateEntries(ctx context.Context, entries []models.PayrollEntry) error {
	if err := r.db.WithContext(ctx).Create(&entries).Error; err != nil {
		return fmt.Errorf("create payroll entries: %w", err)
	}
	return nil
}

func (r *PayrollRepo) GetEntries(ctx context.Context, runID uuid.UUID) ([]models.PayrollEntry, error) {
	var entries []models.PayrollEntry
	if err := r.db.WithContext(ctx).
		Preload("CrewMember").
		Where("payroll_run_id = ?", runID).
		Order("created_at ASC").
		Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("get payroll entries: %w", err)
	}
	return entries, nil
}
