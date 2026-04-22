package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// EarningService provides business logic for earning queries and summaries.
type EarningService struct {
	earningRepo repository.EarningRepository
	logger      *slog.Logger
}

// NewEarningService creates a new EarningService.
func NewEarningService(earningRepo repository.EarningRepository, logger *slog.Logger) *EarningService {
	return &EarningService{earningRepo: earningRepo, logger: logger}
}

// ListEarnings returns filtered, paginated earnings.
func (s *EarningService) ListEarnings(ctx context.Context, filter repository.EarningFilter, page, perPage int) ([]models.Earning, int64, error) {
	return s.earningRepo.List(ctx, filter, page, perPage)
}

// GetDailySummary retrieves the daily earnings summary for a crew member.
func (s *EarningService) GetDailySummary(ctx context.Context, crewMemberID uuid.UUID, date time.Time) (*models.DailyEarningsSummary, error) {
	return s.earningRepo.GetDailySummary(ctx, crewMemberID, date)
}

// GetEarningsByAssignment retrieves all earnings for a specific assignment.
func (s *EarningService) GetEarningsByAssignment(ctx context.Context, assignmentID uuid.UUID) ([]models.Earning, error) {
	filter := repository.EarningFilter{AssignmentID: &assignmentID}
	earnings, _, err := s.earningRepo.List(ctx, filter, 1, 100)
	return earnings, err
}
