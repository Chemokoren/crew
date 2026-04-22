package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// WalletReconciliationJob checks for pending wallet transactions and flags stale ones.
type WalletReconciliationJob struct {
	walletRepo repository.WalletRepository
	logger     *slog.Logger
}

func NewWalletReconciliationJob(repo repository.WalletRepository, logger *slog.Logger) *WalletReconciliationJob {
	return &WalletReconciliationJob{walletRepo: repo, logger: logger}
}

func (j *WalletReconciliationJob) AsJob() Job {
	return Job{
		Name:     "wallet_reconciliation",
		Interval: 6 * time.Hour,
		RunFunc:  j.Run,
	}
}

func (j *WalletReconciliationJob) Run(ctx context.Context) error {
	// Get all wallets and check for pending transactions older than 24h
	wallets, _, err := j.walletRepo.List(ctx, 1, 10000)
	if err != nil {
		return err
	}

	staleThreshold := time.Now().Add(-24 * time.Hour)
	var flagged int

	for _, w := range wallets {
		filter := repository.TxFilter{Status: string(models.TxPending)}
		txns, _, err := j.walletRepo.GetTransactions(ctx, w.ID, filter, 1, 100)
		if err != nil {
			continue
		}
		for _, tx := range txns {
			if tx.CreatedAt.Before(staleThreshold) {
				tx.Status = models.TxFailed
				tx.Description = tx.Description + " [AUTO-FLAGGED: stale pending >24h]"
				if err := j.walletRepo.UpdateTransaction(ctx, &tx); err != nil {
					j.logger.Error("failed to flag stale transaction",
						slog.String("tx_id", tx.ID.String()),
						slog.String("error", err.Error()),
					)
					continue
				}
				flagged++
			}
		}
	}

	j.logger.Info("wallet reconciliation complete",
		slog.Int("wallets_checked", len(wallets)),
		slog.Int("stale_flagged", flagged),
	)
	return nil
}
