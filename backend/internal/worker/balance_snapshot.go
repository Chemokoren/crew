package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// BalanceSnapshotJob takes a daily snapshot of every wallet's balance.
// This enables the credit scoring engine to compute accurate 30-day
// average balances instead of using the current balance as a proxy.
//
// Design decisions:
//   - Runs every 6 hours (not just once/day) for resilience — if one run
//     fails, the next run captures the balance. The UPSERT ensures
//     only one row per wallet per day.
//   - Uses BatchUpsert for efficiency — single query per batch of 500.
//   - Snapshot date uses UTC truncated to date — consistent across timezones.
type BalanceSnapshotJob struct {
	walletRepo   repository.WalletRepository
	snapshotRepo repository.WalletSnapshotRepository
	logger       *slog.Logger
}

// NewBalanceSnapshotJob creates a new daily balance snapshot job.
func NewBalanceSnapshotJob(
	walletRepo repository.WalletRepository,
	snapshotRepo repository.WalletSnapshotRepository,
	logger *slog.Logger,
) *BalanceSnapshotJob {
	return &BalanceSnapshotJob{
		walletRepo:   walletRepo,
		snapshotRepo: snapshotRepo,
		logger:       logger,
	}
}

// AsJob returns a scheduler-compatible Job definition.
func (j *BalanceSnapshotJob) AsJob() Job {
	return Job{
		Name:     "balance_snapshot",
		Interval: 6 * time.Hour, // Run 4x/day for resilience; UPSERT deduplicates
		RunFunc:  j.Run,
	}
}

// Run takes a snapshot of every wallet's current balance for today.
func (j *BalanceSnapshotJob) Run(ctx context.Context) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Fetch all wallets (paginated in chunks of 1000)
	page := 1
	perPage := 1000
	totalSnapshotted := 0

	for {
		wallets, total, err := j.walletRepo.List(ctx, page, perPage)
		if err != nil {
			return err
		}

		if len(wallets) == 0 {
			break
		}

		// Build snapshots batch
		snapshots := make([]models.WalletDailySnapshot, 0, len(wallets))
		for _, w := range wallets {
			snapshots = append(snapshots, models.WalletDailySnapshot{
				WalletID:     w.ID,
				CrewMemberID: w.CrewMemberID,
				BalanceCents: w.BalanceCents,
				Currency:     w.Currency,
				SnapshotDate: today,
			})
		}

		// Batch upsert — single query, idempotent
		if err := j.snapshotRepo.BatchUpsert(ctx, snapshots); err != nil {
			j.logger.Error("balance snapshot batch failed",
				slog.Int("page", page),
				slog.String("error", err.Error()),
			)
			// Continue with next batch — don't fail the whole job
		} else {
			totalSnapshotted += len(snapshots)
		}

		// Check if we've processed all wallets
		if int64(page*perPage) >= total {
			break
		}
		page++
	}

	j.logger.Info("balance snapshot complete",
		slog.Int("wallets_snapshotted", totalSnapshotted),
		slog.Time("snapshot_date", today),
	)

	return nil
}
