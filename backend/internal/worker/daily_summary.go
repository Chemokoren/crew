package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

// DailySummaryJob aggregates earnings for the previous day into
// daily_earnings_summaries rows. Designed to run once per day (e.g., at 01:00 AM).
type DailySummaryJob struct {
	earningRepo    repository.EarningRepository
	assignmentRepo repository.AssignmentRepository
	logger         *slog.Logger
}

// NewDailySummaryJob creates a new DailySummaryJob.
func NewDailySummaryJob(
	earningRepo repository.EarningRepository,
	assignmentRepo repository.AssignmentRepository,
	logger *slog.Logger,
) *DailySummaryJob {
	return &DailySummaryJob{
		earningRepo:    earningRepo,
		assignmentRepo: assignmentRepo,
		logger:         logger,
	}
}

// AsJob returns a Job configuration suitable for the Scheduler.
// Runs every 24 hours by default.
func (j *DailySummaryJob) AsJob() Job {
	return Job{
		Name:     "daily_earnings_summary",
		Interval: 24 * time.Hour,
		RunFunc:  j.Run,
	}
}

// Run processes earnings for the previous day and upserts daily summaries.
// It fetches all completed assignments for yesterday, groups earnings by crew member,
// and creates/updates daily summary records.
func (j *DailySummaryJob) Run(ctx context.Context) error {
	yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)

	j.logger.Info("processing daily summaries",
		slog.String("date", yesterday.Format("2006-01-02")),
	)

	// Fetch all completed assignments for yesterday
	filter := repository.AssignmentFilter{
		ShiftDate: &yesterday,
		Status:    string(models.AssignmentCompleted),
	}

	assignments, _, err := j.assignmentRepo.List(ctx, filter, 1, 10000) // Large page to get all
	if err != nil {
		return fmt.Errorf("list yesterday's assignments: %w", err)
	}

	if len(assignments) == 0 {
		j.logger.Info("no completed assignments for date",
			slog.String("date", yesterday.Format("2006-01-02")),
		)
		return nil
	}

	// Group assignments by crew member and aggregate earnings
	type crewSummary struct {
		totalEarned   int64
		assignCount   int
	}
	summaries := make(map[string]*crewSummary) // key: crew_member_id

	for _, a := range assignments {
		key := a.CrewMemberID.String()
		if _, ok := summaries[key]; !ok {
			summaries[key] = &crewSummary{}
		}
		summaries[key].assignCount++

		// Fetch earnings for this assignment
		earningFilter := repository.EarningFilter{
			AssignmentID: &a.ID,
		}
		earnings, _, err := j.earningRepo.List(ctx, earningFilter, 1, 100)
		if err != nil {
			j.logger.Error("failed to fetch earnings for assignment",
				slog.String("assignment_id", a.ID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}

		for _, e := range earnings {
			summaries[key].totalEarned += e.AmountCents
		}
	}

	// Upsert daily summaries
	var processed, failed int
	for crewIDStr, summary := range summaries {
		// Find the crew member ID from assignments
		var crewMemberID = findCrewMemberID(assignments, crewIDStr)
		if crewMemberID == nil {
			continue
		}

		// Check if summary already exists and is processed
		existing, err := j.earningRepo.GetDailySummary(ctx, *crewMemberID, yesterday)
		if err == nil && existing.IsProcessed {
			j.logger.Debug("daily summary already processed, skipping",
				slog.String("crew_member_id", crewIDStr),
				slog.String("date", yesterday.Format("2006-01-02")),
			)
			continue
		}
		if err != nil && !errors.Is(err, errs.ErrNotFound) {
			j.logger.Error("failed to check existing summary",
				slog.String("crew_member_id", crewIDStr),
				slog.String("error", err.Error()),
			)
			failed++
			continue
		}

		dailySummary := &models.DailyEarningsSummary{
			CrewMemberID:        *crewMemberID,
			Date:                yesterday,
			TotalEarnedCents:    summary.totalEarned,
			TotalDeductionsCents: 0, // Deductions calculated during payroll
			NetAmountCents:      summary.totalEarned,
			Currency:            "KES",
			AssignmentCount:     summary.assignCount,
			IsProcessed:         true,
		}

		if err := j.earningRepo.UpsertDailySummary(ctx, dailySummary); err != nil {
			j.logger.Error("failed to upsert daily summary",
				slog.String("crew_member_id", crewIDStr),
				slog.String("error", err.Error()),
			)
			failed++
			continue
		}

		processed++
	}

	j.logger.Info("daily summary processing complete",
		slog.String("date", yesterday.Format("2006-01-02")),
		slog.Int("total_crew", len(summaries)),
		slog.Int("processed", processed),
		slog.Int("failed", failed),
	)

	return nil
}

// findCrewMemberID finds the UUID for a crew member from the assignment list.
func findCrewMemberID(assignments []models.Assignment, crewIDStr string) *uuid.UUID {
	for _, a := range assignments {
		if a.CrewMemberID.String() == crewIDStr {
			id := a.CrewMemberID
			return &id
		}
	}
	return nil
}
