package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WalletSnapshotRepo is the GORM implementation of repository.WalletSnapshotRepository.
type WalletSnapshotRepo struct {
	db *gorm.DB
}

func NewWalletSnapshotRepo(db *gorm.DB) *WalletSnapshotRepo {
	return &WalletSnapshotRepo{db: db}
}

// Upsert creates or updates a daily snapshot for a wallet.
// ON CONFLICT (wallet_id, snapshot_date) → update balance_cents.
// This makes the job idempotent — safe to re-run multiple times per day.
func (r *WalletSnapshotRepo) Upsert(ctx context.Context, snapshot *models.WalletDailySnapshot) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "wallet_id"}, {Name: "snapshot_date"}},
			DoUpdates: clause.AssignmentColumns([]string{"balance_cents"}),
		}).
		Create(snapshot).Error
}

// BatchUpsert efficiently upserts multiple snapshots in a single query.
// Uses GORM's CreateInBatches with ON CONFLICT for idempotency.
func (r *WalletSnapshotRepo) BatchUpsert(ctx context.Context, snapshots []models.WalletDailySnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "wallet_id"}, {Name: "snapshot_date"}},
			DoUpdates: clause.AssignmentColumns([]string{"balance_cents"}),
		}).
		CreateInBatches(snapshots, 500).Error
}

// GetAvgBalance computes the average daily balance for a crew member over a date range.
// Returns the average in cents. If no snapshots exist, returns 0.
func (r *WalletSnapshotRepo) GetAvgBalance(ctx context.Context, crewMemberID uuid.UUID, from, to time.Time) (int64, error) {
	var result struct {
		AvgBalance *float64
	}
	err := r.db.WithContext(ctx).
		Model(&models.WalletDailySnapshot{}).
		Select("AVG(balance_cents) as avg_balance").
		Where("crew_member_id = ? AND snapshot_date >= ? AND snapshot_date <= ?",
			crewMemberID, from, to).
		Scan(&result).Error
	if err != nil {
		return 0, fmt.Errorf("get avg balance: %w", err)
	}
	if result.AvgBalance == nil {
		return 0, nil
	}
	return int64(*result.AvgBalance), nil
}

// GetSnapshots retrieves daily snapshots for a crew member within a date range.
// Results are ordered by date ascending for time-series analysis.
func (r *WalletSnapshotRepo) GetSnapshots(ctx context.Context, crewMemberID uuid.UUID, from, to time.Time) ([]models.WalletDailySnapshot, error) {
	var snapshots []models.WalletDailySnapshot
	err := r.db.WithContext(ctx).
		Where("crew_member_id = ? AND snapshot_date >= ? AND snapshot_date <= ?",
			crewMemberID, from, to).
		Order("snapshot_date ASC").
		Find(&snapshots).Error
	if err != nil {
		return nil, fmt.Errorf("get snapshots: %w", err)
	}
	return snapshots, nil
}

// GetLatest retrieves the most recent snapshot for a crew member.
func (r *WalletSnapshotRepo) GetLatest(ctx context.Context, crewMemberID uuid.UUID) (*models.WalletDailySnapshot, error) {
	var snapshot models.WalletDailySnapshot
	err := r.db.WithContext(ctx).
		Where("crew_member_id = ?", crewMemberID).
		Order("snapshot_date DESC").
		First(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No snapshots yet — not an error
		}
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	return &snapshot, nil
}
