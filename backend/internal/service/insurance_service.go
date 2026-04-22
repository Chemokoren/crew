package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

type InsuranceService interface {
	CreatePolicy(ctx context.Context, crewMemberID uuid.UUID, provider, policyType, frequency string, premiumCents int64, startDate, endDate time.Time) (*models.InsurancePolicy, error)
	GetPolicy(ctx context.Context, id uuid.UUID) (*models.InsurancePolicy, error)
	ListPolicies(ctx context.Context, filter repository.InsurancePolicyFilter, page, perPage int) ([]models.InsurancePolicy, int64, error)
	MarkPolicyLapsed(ctx context.Context, id uuid.UUID) error
}

type insuranceService struct {
	repo repository.InsurancePolicyRepository
}

func NewInsuranceService(repo repository.InsurancePolicyRepository) InsuranceService {
	return &insuranceService{repo: repo}
}

func (s *insuranceService) CreatePolicy(ctx context.Context, crewMemberID uuid.UUID, provider, policyType, frequency string, premiumCents int64, startDate, endDate time.Time) (*models.InsurancePolicy, error) {
	policy := &models.InsurancePolicy{
		CrewMemberID:       crewMemberID,
		Provider:           provider,
		PolicyType:         policyType,
		PremiumAmountCents: premiumCents,
		PremiumFrequency:   frequency,
		Currency:           "KES",
		Status:             models.PolicyActive,
		StartDate:          startDate,
		EndDate:            endDate,
	}

	if err := s.repo.Create(ctx, policy); err != nil {
		return nil, err
	}
	return policy, nil
}

func (s *insuranceService) GetPolicy(ctx context.Context, id uuid.UUID) (*models.InsurancePolicy, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *insuranceService) ListPolicies(ctx context.Context, filter repository.InsurancePolicyFilter, page, perPage int) ([]models.InsurancePolicy, int64, error) {
	return s.repo.List(ctx, filter, page, perPage)
}

func (s *insuranceService) MarkPolicyLapsed(ctx context.Context, id uuid.UUID) error {
	policy, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	policy.Status = models.PolicyLapsed
	return s.repo.Update(ctx, policy)
}
