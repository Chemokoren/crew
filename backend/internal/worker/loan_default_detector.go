package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/kibsoft/amy-mis/internal/service"
)

// LoanDefaultDetectorJob scans for overdue loans and marks them as DEFAULTED
// after a configurable grace period.
//
// Design decisions:
//   - Grace period of 7 days after due_at before marking as defaulted.
//     This gives borrowers a buffer for delayed M-Pesa transfers.
//   - Runs every 6 hours (same cadence as balance snapshots).
//   - Only transitions DISBURSED/REPAYING → DEFAULTED, never re-defaults.
type LoanDefaultDetectorJob struct {
	loanSvc     service.LoanService
	gracePeriod time.Duration
	logger      *slog.Logger
}

// NewLoanDefaultDetectorJob creates a new default detection job.
func NewLoanDefaultDetectorJob(
	loanSvc service.LoanService,
	logger *slog.Logger,
) *LoanDefaultDetectorJob {
	return &LoanDefaultDetectorJob{
		loanSvc:     loanSvc,
		gracePeriod: 7 * 24 * time.Hour, // 7-day grace period
		logger:      logger,
	}
}

func (j *LoanDefaultDetectorJob) AsJob() Job {
	return Job{
		Name:     "loan_default_detector",
		Interval: 6 * time.Hour,
		RunFunc:  j.Run,
	}
}

func (j *LoanDefaultDetectorJob) Run(ctx context.Context) error {
	overdue, err := j.loanSvc.GetOverdueLoans(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	var defaulted int

	for _, loan := range overdue {
		if loan.DueAt == nil {
			continue
		}

		// Only default if past the grace period
		graceDeadline := loan.DueAt.Add(j.gracePeriod)
		if now.Before(graceDeadline) {
			continue // Still within grace period
		}

		if _, err := j.loanSvc.MarkDefaulted(ctx, loan.ID); err != nil {
			j.logger.Error("failed to mark loan as defaulted",
				slog.String("loan_id", loan.ID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}
		defaulted++

		j.logger.Warn("loan marked as defaulted",
			slog.String("loan_id", loan.ID.String()),
			slog.String("crew_member_id", loan.CrewMemberID.String()),
			slog.Int("days_past_due", int(now.Sub(*loan.DueAt).Hours()/24)),
		)
	}

	j.logger.Info("loan default detection complete",
		slog.Int("overdue_found", len(overdue)),
		slog.Int("defaulted", defaulted),
	)

	return nil
}
