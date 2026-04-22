package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"gorm.io/gorm"
)

type loanApplicationRepo struct {
	db *gorm.DB
}

// NewLoanApplicationRepo creates a new Postgres-backed LoanApplicationRepository.
func NewLoanApplicationRepo(db *gorm.DB) repository.LoanApplicationRepository {
	return &loanApplicationRepo{db: db}
}

func (r *loanApplicationRepo) Create(ctx context.Context, loan *models.LoanApplication) error {
	return r.db.WithContext(ctx).Create(loan).Error
}

func (r *loanApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.LoanApplication, error) {
	var loan models.LoanApplication
	err := r.db.WithContext(ctx).First(&loan, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &loan, nil
}

func (r *loanApplicationRepo) Update(ctx context.Context, loan *models.LoanApplication) error {
	return r.db.WithContext(ctx).Save(loan).Error
}

func (r *loanApplicationRepo) List(ctx context.Context, filter repository.LoanApplicationFilter, page, perPage int) ([]models.LoanApplication, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.LoanApplication{})

	if filter.CrewMemberID != nil {
		query = query.Where("crew_member_id = ?", *filter.CrewMemberID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.LenderID != nil {
		query = query.Where("lender_id = ?", *filter.LenderID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var loans []models.LoanApplication
	err := query.Order("created_at DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&loans).Error

	return loans, total, err
}
