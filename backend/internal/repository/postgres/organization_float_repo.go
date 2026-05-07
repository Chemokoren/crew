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

// OrganizationFloatRepo is the GORM implementation of repository.OrganizationFloatRepository.
type OrganizationFloatRepo struct {
	db *gorm.DB
}

func NewOrganizationFloatRepo(db *gorm.DB) *OrganizationFloatRepo {
	return &OrganizationFloatRepo{db: db}
}

func (r *OrganizationFloatRepo) GetOrCreate(ctx context.Context, orgID uuid.UUID) (*models.OrganizationFloat, error) {
	var sf models.OrganizationFloat
	err := r.db.WithContext(ctx).Where("sacco_id = ?", orgID).First(&sf).Error
	if err == nil {
		return &sf, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get sacco float: %w", err)
	}
	sf = models.OrganizationFloat{
		OrganizationID:  orgID,
		Currency: "KES",
	}
	if err := r.db.WithContext(ctx).Create(&sf).Error; err != nil {
		return nil, fmt.Errorf("create sacco float: %w", err)
	}
	return &sf, nil
}

func (r *OrganizationFloatRepo) CreditFloat(ctx context.Context, floatID uuid.UUID, version int, amountCents int64,
	idempotencyKey, reference string) (*models.OrganizationFloatTransaction, error) {

	// Check idempotency
	if idempotencyKey != "" {
		var existing models.OrganizationFloatTransaction
		if err := r.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
			return &existing, nil
		}
	}

	return r.executeFloatOp(ctx, floatID, version, amountCents, "FUND", idempotencyKey, reference)
}

func (r *OrganizationFloatRepo) DebitFloat(ctx context.Context, floatID uuid.UUID, version int, amountCents int64,
	idempotencyKey, reference string) (*models.OrganizationFloatTransaction, error) {

	if idempotencyKey != "" {
		var existing models.OrganizationFloatTransaction
		if err := r.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
			return &existing, nil
		}
	}

	return r.executeFloatOp(ctx, floatID, version, -amountCents, "PAYOUT", idempotencyKey, reference)
}

func (r *OrganizationFloatRepo) executeFloatOp(ctx context.Context, floatID uuid.UUID, version int, delta int64,
	txType, idempotencyKey, reference string) (*models.OrganizationFloatTransaction, error) {

	var tx models.OrganizationFloatTransaction

	err := r.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// Lock and verify version
		var sf models.OrganizationFloat
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

		tx = models.OrganizationFloatTransaction{
			OrganizationFloatID:      floatID,
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

func (r *OrganizationFloatRepo) GetTransactions(ctx context.Context, floatID uuid.UUID, filter repository.OrganizationFloatFilter, page, perPage int) ([]models.OrganizationFloatTransaction, int64, error) {
	var txs []models.OrganizationFloatTransaction
	var total int64

	query := r.db.WithContext(ctx).Model(&models.OrganizationFloatTransaction{}).Where("sacco_float_id = ?", floatID)
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

// CreatePendingTransaction inserts a float transaction with status=PENDING but
// does NOT update the float balance. Used for STK push flows where the actual
// credit happens only after the payment provider confirms via callback.
func (r *OrganizationFloatRepo) CreatePendingTransaction(ctx context.Context, floatID uuid.UUID,
	amountCents int64, idempotencyKey, reference string) (*models.OrganizationFloatTransaction, error) {

	// Idempotency check
	if idempotencyKey != "" {
		var existing models.OrganizationFloatTransaction
		if err := r.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
			return &existing, nil
		}
	}

	tx := models.OrganizationFloatTransaction{
		OrganizationFloatID: floatID,
		IdempotencyKey:      idempotencyKey,
		TransactionType:     "FUND",
		AmountCents:         amountCents,
		BalanceAfterCents:   0, // Will be set when confirmed
		Currency:            "KES",
		Reference:           reference,
		Status:              models.TxPending,
	}
	if err := r.db.WithContext(ctx).Create(&tx).Error; err != nil {
		return nil, fmt.Errorf("create pending float transaction: %w", err)
	}
	return &tx, nil
}

// ConfirmPendingTransaction atomically credits the float balance and marks
// the pending transaction as COMPLETED. Called when the payment provider
// confirms a successful STK push payment.
func (r *OrganizationFloatRepo) ConfirmPendingTransaction(ctx context.Context, txID uuid.UUID) (*models.OrganizationFloatTransaction, error) {
	var result models.OrganizationFloatTransaction

	err := r.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// 1. Lock the pending transaction
		var pendingTx models.OrganizationFloatTransaction
		if err := dbTx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", txID, models.TxPending).
			First(&pendingTx).Error; err != nil {
			return fmt.Errorf("lock pending tx: %w", err)
		}

		// 2. Lock the float and update balance
		var sf models.OrganizationFloat
		if err := dbTx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", pendingTx.OrganizationFloatID).
			First(&sf).Error; err != nil {
			return fmt.Errorf("lock float: %w", err)
		}

		sf.BalanceCents += pendingTx.AmountCents
		sf.Version++
		now := sf.UpdatedAt
		sf.LastFundedAt = &now

		if err := dbTx.Save(&sf).Error; err != nil {
			return fmt.Errorf("update float balance: %w", err)
		}

		// 3. Update the transaction status
		pendingTx.BalanceAfterCents = sf.BalanceCents
		pendingTx.Status = models.TxCompleted
		if err := dbTx.Save(&pendingTx).Error; err != nil {
			return fmt.Errorf("confirm pending tx: %w", err)
		}

		result = pendingTx
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// FailPendingTransaction marks a pending transaction as FAILED without
// modifying the float balance.
func (r *OrganizationFloatRepo) FailPendingTransaction(ctx context.Context, txID uuid.UUID, reason string) error {
	result := r.db.WithContext(ctx).Model(&models.OrganizationFloatTransaction{}).
		Where("id = ? AND status = ?", txID, models.TxPending).
		Updates(map[string]interface{}{
			"status":    models.TxFailed,
			"reference": gorm.Expr("reference || ' | fail_reason:' || ?", reason),
		})
	if result.Error != nil {
		return fmt.Errorf("fail pending tx: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no pending transaction found with id %s", txID)
	}
	return nil
}

// GetByIdempotencyKey looks up a float transaction by its idempotency key.
func (r *OrganizationFloatRepo) GetByIdempotencyKey(ctx context.Context, key string) (*models.OrganizationFloatTransaction, error) {
	var tx models.OrganizationFloatTransaction
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&tx).Error; err != nil {
		return nil, fmt.Errorf("float tx by idempotency key: %w", err)
	}
	return &tx, nil
}

// AppendReference appends a suffix to the reference of a float transaction.
func (r *OrganizationFloatRepo) AppendReference(ctx context.Context, txID uuid.UUID, refSuffix string) error {
	return r.db.WithContext(ctx).Model(&models.OrganizationFloatTransaction{}).
		Where("id = ?", txID).
		Update("reference", gorm.Expr("reference || ?", refSuffix)).Error
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
