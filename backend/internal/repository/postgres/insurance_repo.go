package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"gorm.io/gorm"
)

type insurancePolicyRepo struct {
	db *gorm.DB
}

// NewInsurancePolicyRepo creates a new Postgres-backed InsurancePolicyRepository.
func NewInsurancePolicyRepo(db *gorm.DB) repository.InsurancePolicyRepository {
	return &insurancePolicyRepo{db: db}
}

func (r *insurancePolicyRepo) Create(ctx context.Context, policy *models.InsurancePolicy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}

func (r *insurancePolicyRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.InsurancePolicy, error) {
	var policy models.InsurancePolicy
	err := r.db.WithContext(ctx).First(&policy, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *insurancePolicyRepo) Update(ctx context.Context, policy *models.InsurancePolicy) error {
	return r.db.WithContext(ctx).Save(policy).Error
}

func (r *insurancePolicyRepo) List(ctx context.Context, filter repository.InsurancePolicyFilter, page, perPage int) ([]models.InsurancePolicy, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.InsurancePolicy{})

	if filter.CrewMemberID != nil {
		query = query.Where("crew_member_id = ?", *filter.CrewMemberID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var policies []models.InsurancePolicy
	err := query.Order("created_at DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&policies).Error

	return policies, total, err
}
