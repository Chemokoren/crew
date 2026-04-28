package mock

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

type CreditScoreRepo struct {
	mu     sync.Mutex
	scores map[uuid.UUID]*models.CreditScore
}

func NewCreditScoreRepo() *CreditScoreRepo {
	return &CreditScoreRepo{scores: make(map[uuid.UUID]*models.CreditScore)}
}

func (r *CreditScoreRepo) GetByCrewMemberID(_ context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.scores {
		if s.CrewMemberID == crewMemberID {
			return s, nil
		}
	}
	return nil, nil // Return nil, err instead? Wait, gorm returns ErrRecordNotFound. For mock, we'll return nil, err if needed, or simply nil without error if standard allows.
}

func (r *CreditScoreRepo) Upsert(_ context.Context, score *models.CreditScore) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if score.ID == uuid.Nil {
		score.ID = uuid.New()
	}
	r.scores[score.ID] = score
	return nil
}

type LoanApplicationRepo struct {
	mu    sync.Mutex
	loans map[uuid.UUID]*models.LoanApplication
}

func NewLoanApplicationRepo() *LoanApplicationRepo {
	return &LoanApplicationRepo{loans: make(map[uuid.UUID]*models.LoanApplication)}
}

func (r *LoanApplicationRepo) Create(_ context.Context, loan *models.LoanApplication) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if loan.ID == uuid.Nil {
		loan.ID = uuid.New()
	}
	r.loans[loan.ID] = loan
	return nil
}

func (r *LoanApplicationRepo) GetByID(_ context.Context, id uuid.UUID) (*models.LoanApplication, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.loans[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return l, nil
}

func (r *LoanApplicationRepo) Update(_ context.Context, loan *models.LoanApplication) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loans[loan.ID] = loan
	return nil
}

func (r *LoanApplicationRepo) List(_ context.Context, _ repository.LoanApplicationFilter, _, _ int) ([]models.LoanApplication, int64, error) {
	return nil, 0, nil
}

type InsurancePolicyRepo struct {
	mu       sync.Mutex
	policies map[uuid.UUID]*models.InsurancePolicy
}

func NewInsurancePolicyRepo() *InsurancePolicyRepo {
	return &InsurancePolicyRepo{policies: make(map[uuid.UUID]*models.InsurancePolicy)}
}

func (r *InsurancePolicyRepo) Create(_ context.Context, policy *models.InsurancePolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	r.policies[policy.ID] = policy
	return nil
}

func (r *InsurancePolicyRepo) GetByID(_ context.Context, id uuid.UUID) (*models.InsurancePolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.policies[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return p, nil
}

func (r *InsurancePolicyRepo) Update(_ context.Context, policy *models.InsurancePolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.policies[policy.ID] = policy
	return nil
}

func (r *InsurancePolicyRepo) List(_ context.Context, _ repository.InsurancePolicyFilter, _, _ int) ([]models.InsurancePolicy, int64, error) {
	return nil, 0, nil
}

// --- Credit Score History Mock ---

type CreditScoreHistoryRepo struct {
	mu      sync.Mutex
	history []models.CreditScoreHistory
}

func NewCreditScoreHistoryRepo() *CreditScoreHistoryRepo {
	return &CreditScoreHistoryRepo{}
}

func (r *CreditScoreHistoryRepo) Create(_ context.Context, h *models.CreditScoreHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	r.history = append(r.history, *h)
	return nil
}

func (r *CreditScoreHistoryRepo) GetHistory(_ context.Context, crewMemberID uuid.UUID, limit int) ([]models.CreditScoreHistory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []models.CreditScoreHistory
	for _, h := range r.history {
		if h.CrewMemberID == crewMemberID {
			result = append(result, h)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
