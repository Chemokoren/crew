package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// txKey is the context key for injecting a GORM transaction.
type txKey struct{}

// TxManager provides transactional execution across multiple repository calls.
// Services inject this to wrap multi-step operations in a single DB transaction.
type TxManager struct {
	db *gorm.DB
}

// NewTxManager creates a new transaction manager.
func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// RunInTx executes fn inside a database transaction.
// The transaction is committed if fn returns nil, rolled back otherwise.
// Repositories called within fn automatically pick up the transaction
// via ExtractTx on the context.
func (m *TxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := InjectTx(ctx, tx)
		if err := fn(txCtx); err != nil {
			return fmt.Errorf("transaction failed: %w", err)
		}
		return nil
	})
}

// InjectTx stores a GORM transaction in the context.
func InjectTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// ExtractTx retrieves a GORM transaction from the context.
// Returns nil if no transaction is active.
func ExtractTx(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if ok {
		return tx
	}
	return nil
}
