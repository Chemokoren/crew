package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SACCOFloatRepo is the GORM implementation of repository.SACCOFloatRepository.
type SACCOFloatRepo struct {
	db *gorm.DB
}

func NewSACCOFloatRepo(db *gorm.DB) *SACCOFloatRepo {
	return &SACCOFloatRepo{db: db}
}

func (r *SACCOFloatRepo) GetOrCreate(ctx context.Context, saccoID uuid.UUID) (*models.SACCOFloat, error) {
	var sf models.SACCOFloat
	err := r.db.WithContext(ctx).Where("sacco_id = ?", saccoID).First(&sf).Error
	if err == nil {
		return &sf, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get sacco float: %w", err)
	}
	sf = models.SACCOFloat{
		SaccoID:  saccoID,
		Currency: "KES",
	}
	if err := r.db.WithContext(ctx).Create(&sf).Error; err != nil {
		return nil, fmt.Errorf("create sacco float: %w", err)
	}
	return &sf, nil
}

func (r *SACCOFloatRepo) CreditFloat(ctx context.Context, floatID uuid.UUID, version int, amountCents int64,
	idempotencyKey, reference string) (*models.SACCOFloatTransaction, error) {

	// Check idempotency
	if idempotencyKey != "" {
		var existing models.SACCOFloatTransaction
		if err := r.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
			return &existing, nil
		}
	}

	return r.executeFloatOp(ctx, floatID, version, amountCents, "CREDIT", idempotencyKey, reference)
}

func (r *SACCOFloatRepo) DebitFloat(ctx context.Context, floatID uuid.UUID, version int, amountCents int64,
	idempotencyKey, reference string) (*models.SACCOFloatTransaction, error) {

	if idempotencyKey != "" {
		var existing models.SACCOFloatTransaction
		if err := r.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
			return &existing, nil
		}
	}

	return r.executeFloatOp(ctx, floatID, version, -amountCents, "DEBIT", idempotencyKey, reference)
}

func (r *SACCOFloatRepo) executeFloatOp(ctx context.Context, floatID uuid.UUID, version int, delta int64,
	txType, idempotencyKey, reference string) (*models.SACCOFloatTransaction, error) {

	var tx models.SACCOFloatTransaction

	err := r.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Lock and verify version
		var sf models.SACCOFloat
		if err := dbTx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", floatID).First(&sf).Error; err != nil {
			return fmt.Errorf("lock float: %w", err)
		}
		if sf.Version != version {
			return errs.ErrOptimisticLock
		}

		newBalance := sf.BalanceCents + delta
		if newBalance < 0 {
			return errs.ErrInsufficientBalance
		}

		sf.BalanceCents = newBalance
		sf.Version++
		if delta > 0 {
			now := sf.UpdatedAt
			sf.LastFundedAt = &now
		}

		if err := dbTx.Save(&sf).Error; err != nil {
			return fmt.Errorf("update float balance: %w", err)
		}

		tx = models.SACCOFloatTransaction{
			SACCOFloatID:      floatID,
			IdempotencyKey:    idempotencyKey,
			TransactionType:   txType,
			AmountCents:       abs64(delta),
			BalanceAfterCents: newBalance,
			Currency:          sf.Currency,
			Reference:         reference,
			Status:            models.TxCompleted,
		}
		return dbTx.Create(&tx).Error
	})

	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *SACCOFloatRepo) GetTransactions(ctx context.Context, floatID uuid.UUID, filter repository.SACCOFloatFilter, page, perPage int) ([]models.SACCOFloatTransaction, int64, error) {
	var txs []models.SACCOFloatTransaction
	var total int64

	query := r.db.WithContext(ctx).Model(&models.SACCOFloatTransaction{}).Where("sacco_float_id = ?", floatID)
	if filter.TransactionType != "" {
		query = query.Where("transaction_type = ?", filter.TransactionType)
	}
	if filter.DateFrom != nil {
		query = query.Where("created_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("created_at <= ?", *filter.DateTo)
	}
	query.Count(&total)

	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Order("created_at DESC").Find(&txs).Error; err != nil {
		return nil, 0, fmt.Errorf("list float transactions: %w", err)
	}
	return txs, total, nil
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
