package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/credit"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// CreditService delegates to the credit.Engine for score computation.
type CreditService interface {
	CalculateScore(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error)
	GetScore(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error)
	GetDetailedScore(ctx context.Context, crewMemberID uuid.UUID) (*credit.ScoreResult, error)
	GetScoreHistory(ctx context.Context, crewMemberID uuid.UUID, limit int) ([]models.CreditScoreHistory, error)
}

type creditService struct {
	engine      *credit.Engine
	creditRepo  repository.CreditScoreRepository
	historyRepo repository.CreditScoreHistoryRepository
}

func NewCreditService(
	engine *credit.Engine,
	creditRepo repository.CreditScoreRepository,
	historyRepo repository.CreditScoreHistoryRepository,
) CreditService {
	return &creditService{
		engine:      engine,
		creditRepo:  creditRepo,
		historyRepo: historyRepo,
	}
}

func (s *creditService) GetScore(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error) {
	// Use the engine's smart caching (24h TTL)
	result, err := s.engine.GetScore(ctx, crewMemberID)
	if err != nil {
		return nil, err
	}
	return s.resultToModel(crewMemberID, result)
}

func (s *creditService) CalculateScore(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error) {
	result, err := s.engine.CalculateScore(ctx, crewMemberID)
	if err != nil {
		return nil, err
	}
	return s.resultToModel(crewMemberID, result)
}

// GetDetailedScore returns the full ScoreResult with factor breakdown, suggestions, and features.
func (s *creditService) GetDetailedScore(ctx context.Context, crewMemberID uuid.UUID) (*credit.ScoreResult, error) {
	return s.engine.GetScore(ctx, crewMemberID)
}

func (s *creditService) GetScoreHistory(ctx context.Context, crewMemberID uuid.UUID, limit int) ([]models.CreditScoreHistory, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	return s.historyRepo.GetHistory(ctx, crewMemberID, limit)
}

func (s *creditService) resultToModel(crewMemberID uuid.UUID, result *credit.ScoreResult) (*models.CreditScore, error) {
	factorsJSON, _ := json.Marshal(result)
	return &models.CreditScore{
		CrewMemberID:     crewMemberID,
		Score:            result.Score,
		Factors:          factorsJSON,
		LastCalculatedAt: result.ComputedAt,
	}, nil
}
