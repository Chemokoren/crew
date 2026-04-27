package credit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// Engine is the top-level credit scoring engine.
// It orchestrates: Feature Engineering → Scoring → Persistence → Explanation.
type Engine struct {
	features   *FeatureComputer
	scorer     Scorer
	creditRepo repository.CreditScoreRepository
	logger     *slog.Logger
}

// NewEngine creates a new credit scoring engine with the given scorer.
func NewEngine(
	features *FeatureComputer,
	scorer Scorer,
	creditRepo repository.CreditScoreRepository,
	logger *slog.Logger,
) *Engine {
	return &Engine{
		features:   features,
		scorer:     scorer,
		creditRepo: creditRepo,
		logger:     logger,
	}
}

// CalculateScore computes and persists a crew member's credit score.
func (e *Engine) CalculateScore(ctx context.Context, crewMemberID uuid.UUID) (*ScoreResult, error) {
	start := time.Now()

	// 1. Feature Engineering
	fv, err := e.features.Compute(ctx, crewMemberID)
	if err != nil {
		return nil, fmt.Errorf("feature computation: %w", err)
	}

	featureLatency := time.Since(start)

	// 2. Score
	result, err := e.scorer.Score(ctx, fv)
	if err != nil {
		return nil, fmt.Errorf("scoring: %w", err)
	}

	scoreLatency := time.Since(start) - featureLatency

	// 3. Persist
	factorsJSON, _ := json.Marshal(result)
	creditScore := &models.CreditScore{
		CrewMemberID:     crewMemberID,
		Score:            result.Score,
		Factors:          factorsJSON,
		LastCalculatedAt: time.Now(),
	}
	if err := e.creditRepo.Upsert(ctx, creditScore); err != nil {
		e.logger.Error("credit: failed to persist score",
			slog.String("crew_member_id", crewMemberID.String()),
			slog.String("error", err.Error()),
		)
		// Non-fatal — still return the computed score
	}

	e.logger.Info("credit score calculated",
		slog.String("crew_member_id", crewMemberID.String()),
		slog.Int("score", result.Score),
		slog.String("grade", result.Grade),
		slog.String("model", result.ModelVersion),
		slog.Duration("feature_latency", featureLatency),
		slog.Duration("score_latency", scoreLatency),
	)

	return result, nil
}

// GetScore retrieves the last calculated score, or calculates a new one if stale/missing.
func (e *Engine) GetScore(ctx context.Context, crewMemberID uuid.UUID) (*ScoreResult, error) {
	existing, err := e.creditRepo.GetByCrewMemberID(ctx, crewMemberID)
	if err == nil && existing != nil {
		// If score is less than 24 hours old, return cached
		if time.Since(existing.LastCalculatedAt) < 24*time.Hour {
			var result ScoreResult
			if err := json.Unmarshal(existing.Factors, &result); err == nil {
				return &result, nil
			}
			// JSON parsing failed — recalculate
		}
	}

	// Score missing or stale — recalculate
	return e.CalculateScore(ctx, crewMemberID)
}
