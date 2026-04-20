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

// WalletRepo is the GORM implementation of repository.WalletRepository.
// It uses belt-and-suspenders concurrency control:
//   - Pessimistic: SELECT ... FOR UPDATE on the wallet row
//   - Optimistic: version check before committing
type WalletRepo struct {
	db *gorm.DB
}

func NewWalletRepo(db *gorm.DB) *WalletRepo {
	return &WalletRepo{db: db}
}

func (r *WalletRepo) Create(ctx context.Context, wallet *models.Wallet) error {
	if err := r.db.WithContext(ctx).Create(wallet).Error; err != nil {
		return fmt.Errorf("create wallet: %w", err)
	}
	return nil
}

func (r *WalletRepo) GetByCrewMemberID(ctx context.Context, crewMemberID uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := r.db.WithContext(ctx).Where("crew_member_id = ?", crewMemberID).First(&wallet).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get wallet by crew_member_id: %w", err)
	}
	return &wallet, nil
}

// CreditWallet atomically credits a wallet using SELECT FOR UPDATE + version check.
func (r *WalletRepo) CreditWallet(
	ctx context.Context,
	walletID uuid.UUID,
	version int,
	amountCents int64,
	category models.TransactionCategory,
	idempotencyKey, reference, description string,
) (*models.WalletTransaction, error) {
	var tx *models.WalletTransaction

	err := r.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// 1. Check idempotency (fast path)
		if idempotencyKey != "" {
			var existing models.WalletTransaction
			if err := dbTx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
				tx = &existing
				return nil // Idempotent replay — return existing tx
			}
		}

		// 2. Pessimistic lock: SELECT ... FOR UPDATE
		var wallet models.Wallet
		if err := dbTx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", walletID).First(&wallet).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return fmt.Errorf("lock wallet: %w", err)
		}

		// 3. Optimistic lock: version check
		if wallet.Version != version {
			return errs.ErrOptimisticLock
		}

		// 4. Apply credit
		newBalance := wallet.BalanceCents + amountCents
		wallet.BalanceCents = newBalance
		wallet.TotalCreditedCents += amountCents
		wallet.Version++

		if err := dbTx.Save(&wallet).Error; err != nil {
			return fmt.Errorf("update wallet: %w", err)
		}

		// 5. Create transaction record
		tx = &models.WalletTransaction{
			WalletID:          walletID,
			IdempotencyKey:    idempotencyKey,
			TransactionType:   models.TxCredit,
			Category:          category,
			AmountCents:       amountCents,
			BalanceAfterCents: newBalance,
			Currency:          wallet.Currency,
			Reference:         reference,
			Description:       description,
			Status:            models.TxCompleted,
		}

		if err := dbTx.Create(tx).Error; err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return tx, nil
}

// DebitWallet atomically debits a wallet, checking balance sufficiency.
func (r *WalletRepo) DebitWallet(
	ctx context.Context,
	walletID uuid.UUID,
	version int,
	amountCents int64,
	category models.TransactionCategory,
	idempotencyKey, reference, description string,
) (*models.WalletTransaction, error) {
	var tx *models.WalletTransaction

	err := r.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		// 1. Check idempotency (fast path)
		if idempotencyKey != "" {
			var existing models.WalletTransaction
			if err := dbTx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
				tx = &existing
				return nil
			}
		}

		// 2. Pessimistic lock
		var wallet models.Wallet
		if err := dbTx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", walletID).First(&wallet).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return fmt.Errorf("lock wallet: %w", err)
		}

		// 3. Optimistic lock
		if wallet.Version != version {
			return errs.ErrOptimisticLock
		}

		// 4. Balance check — CRITICAL: prevents overdraw
		if wallet.BalanceCents < amountCents {
			return errs.ErrInsufficientBalance
		}

		// 5. Apply debit
		newBalance := wallet.BalanceCents - amountCents
		wallet.BalanceCents = newBalance
		wallet.TotalDebitedCents += amountCents
		wallet.Version++

		if err := dbTx.Save(&wallet).Error; err != nil {
			return fmt.Errorf("update wallet: %w", err)
		}

		// 6. Create transaction record
		tx = &models.WalletTransaction{
			WalletID:          walletID,
			IdempotencyKey:    idempotencyKey,
			TransactionType:   models.TxDebit,
			Category:          category,
			AmountCents:       amountCents,
			BalanceAfterCents: newBalance,
			Currency:          wallet.Currency,
			Reference:         reference,
			Description:       description,
			Status:            models.TxCompleted,
		}

		if err := dbTx.Create(tx).Error; err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *WalletRepo) GetTransactions(ctx context.Context, walletID uuid.UUID, filter repository.TxFilter, page, perPage int) ([]models.WalletTransaction, int64, error) {
	var txs []models.WalletTransaction
	var total int64

	query := r.db.WithContext(ctx).Model(&models.WalletTransaction{}).Where("wallet_id = ?", walletID)

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.TransactionType != "" {
		query = query.Where("transaction_type = ?", filter.TransactionType)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
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
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}

	return txs, total, nil
}

func (r *WalletRepo) GetByIdempotencyKey(ctx context.Context, key string) (*models.WalletTransaction, error) {
	var tx models.WalletTransaction
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&tx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("get by idempotency key: %w", err)
	}
	return &tx, nil
}
