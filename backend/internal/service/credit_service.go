package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

type CreditService interface {
	CalculateScore(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error)
	GetScore(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error)
}

type creditService struct {
	creditRepo     repository.CreditScoreRepository
	earningRepo    repository.EarningRepository
	assignmentRepo repository.AssignmentRepository
}

func NewCreditService(
	creditRepo repository.CreditScoreRepository,
	earningRepo repository.EarningRepository,
	assignmentRepo repository.AssignmentRepository,
) CreditService {
	return &creditService{
		creditRepo:     creditRepo,
		earningRepo:    earningRepo,
		assignmentRepo: assignmentRepo,
	}
}

func (s *creditService) GetScore(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error) {
	return s.creditRepo.GetByCrewMemberID(ctx, crewMemberID)
}

func (s *creditService) CalculateScore(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error) {
	// A simple credit scoring algorithm based on recent earnings and assignments.
	// 1. Fetch recent assignments (e.g., last 30 days)
	// 2. Fetch total verified earnings
	// 3. Compute score

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	
	// Fetch assignments
	assignments, _, err := s.assignmentRepo.List(ctx, repository.AssignmentFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &thirtyDaysAgo,
	}, 1, 1000)
	if err != nil {
		return nil, err
	}

	completedAssignments := 0
	for _, a := range assignments {
		if a.Status == models.AssignmentCompleted {
			completedAssignments++
		}
	}

	// Fetch earnings
	isVerified := true
	earnings, _, err := s.earningRepo.List(ctx, repository.EarningFilter{
		CrewMemberID: &crewMemberID,
		DateFrom:     &thirtyDaysAgo,
		IsVerified:   &isVerified,
	}, 1, 1000)
	if err != nil {
		return nil, err
	}

	var totalEarningsCents int64
	for _, e := range earnings {
		totalEarningsCents += e.AmountCents
	}

	// Calculate points
	// Base score: 300
	// 10 points per completed assignment (max 300)
	// 1 point per 1000 KES earned (max 250)
	
	score := 300
	
	assignmentPoints := completedAssignments * 10
	if assignmentPoints > 300 {
		assignmentPoints = 300
	}
	score += assignmentPoints

	earningPoints := int(totalEarningsCents / 100000) // 1000 KES = 100,000 cents
	if earningPoints > 250 {
		earningPoints = 250
	}
	score += earningPoints

	factors := map[string]interface{}{
		"completed_assignments": completedAssignments,
		"assignment_points":     assignmentPoints,
		"total_earnings_kes":    float64(totalEarningsCents) / 100,
		"earning_points":        earningPoints,
		"base_score":            300,
	}
	
	factorsBytes, _ := json.Marshal(factors)

	creditScore := &models.CreditScore{
		CrewMemberID:     crewMemberID,
		Score:            score,
		Factors:          factorsBytes,
		LastCalculatedAt: time.Now(),
	}

	if err := s.creditRepo.Upsert(ctx, creditScore); err != nil {
		return nil, err
	}

	return creditScore, nil
}
