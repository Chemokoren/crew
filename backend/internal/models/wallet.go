package models

import (
	"time"

	"github.com/google/uuid"
)

// Wallet holds a crew member's balance. One wallet per crew member.
// Uses optimistic locking via Version field to prevent overdraw races.
type Wallet struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID       uuid.UUID  `json:"crew_member_id" gorm:"type:uuid;uniqueIndex;not null"`
	BalanceCents       int64      `json:"balance_cents" gorm:"type:bigint;default:0"`
	TotalCreditedCents int64      `json:"total_credited_cents" gorm:"type:bigint;default:0"`
	TotalDebitedCents  int64      `json:"total_debited_cents" gorm:"type:bigint;default:0"`
	Currency           string     `json:"currency" gorm:"default:'KES';not null"`
	Version            int        `json:"-" gorm:"not null;default:0"`
	IsActive           bool       `json:"is_active" gorm:"default:true"`
	LastPayoutAt       *time.Time `json:"last_payout_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	CrewMember CrewMember `json:"-" gorm:"foreignKey:CrewMemberID"`
}

func (Wallet) TableName() string { return "wallets" }

// --- Transaction types ---

type TransactionType string

const (
	TxCredit TransactionType = "CREDIT"
	TxDebit  TransactionType = "DEBIT"
)

type TransactionCategory string

const (
	TxCatEarning    TransactionCategory = "EARNING"
	TxCatWithdrawal TransactionCategory = "WITHDRAWAL"
	TxCatDeduction  TransactionCategory = "DEDUCTION"
	TxCatTopUp      TransactionCategory = "TOP_UP"
	TxCatReversal   TransactionCategory = "REVERSAL"
	TxCatLoan       TransactionCategory = "LOAN"
)

type TransactionStatus string

const (
	TxPending   TransactionStatus = "PENDING"
	TxCompleted TransactionStatus = "COMPLETED"
	TxFailed    TransactionStatus = "FAILED"
	TxReversed  TransactionStatus = "REVERSED"
)

// WalletTransaction is an immutable ledger entry.
type WalletTransaction struct {
	ID                uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WalletID          uuid.UUID           `json:"wallet_id" gorm:"type:uuid;not null;index"`
	IdempotencyKey    string              `json:"idempotency_key,omitempty" gorm:"uniqueIndex"`
	TransactionType   TransactionType     `json:"transaction_type" gorm:"not null"`
	Category          TransactionCategory `json:"category" gorm:"not null"`
	AmountCents       int64               `json:"amount_cents" gorm:"type:bigint;not null"`
	BalanceAfterCents int64               `json:"balance_after_cents" gorm:"type:bigint"`
	Currency          string              `json:"currency" gorm:"default:'KES';not null"`
	Reference         string              `json:"reference,omitempty"`
	Description       string              `json:"description,omitempty"`
	Status            TransactionStatus   `json:"status" gorm:"default:'PENDING'"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

func (WalletTransaction) TableName() string { return "wallet_transactions" }

// --- SACCO Float ---

// SACCOFloat tracks a SACCO's available funds for crew payouts.
type SACCOFloat struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SaccoID      uuid.UUID  `json:"sacco_id" gorm:"type:uuid;uniqueIndex;not null"`
	BalanceCents int64      `json:"balance_cents" gorm:"type:bigint;default:0"`
	Currency     string     `json:"currency" gorm:"default:'KES';not null"`
	Version      int        `json:"-" gorm:"not null;default:0"`
	LastFundedAt *time.Time `json:"last_funded_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	Sacco SACCO `json:"-" gorm:"foreignKey:SaccoID"`
}

func (SACCOFloat) TableName() string { return "sacco_floats" }

// SACCOFloatTransaction records SACCO float funding and payout events.
type SACCOFloatTransaction struct {
	ID                uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SACCOFloatID      uuid.UUID         `json:"sacco_float_id" gorm:"type:uuid;not null;index"`
	IdempotencyKey    string            `json:"idempotency_key,omitempty" gorm:"uniqueIndex"`
	TransactionType   string            `json:"transaction_type" gorm:"not null"`
	AmountCents       int64             `json:"amount_cents" gorm:"type:bigint;not null"`
	BalanceAfterCents int64             `json:"balance_after_cents" gorm:"type:bigint"`
	Currency          string            `json:"currency" gorm:"default:'KES';not null"`
	Reference         string            `json:"reference,omitempty"`
	Status            TransactionStatus `json:"status" gorm:"default:'PENDING'"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

func (SACCOFloatTransaction) TableName() string { return "sacco_float_transactions" }
