package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CreditScore tracks a crew member's internal creditworthiness rating.
type CreditScore struct {
	ID               uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID     uuid.UUID       `json:"crew_member_id" gorm:"type:uuid;uniqueIndex;not null"`
	Score            int             `json:"score"`
	Factors          json.RawMessage `json:"factors" gorm:"type:jsonb"`
	LastCalculatedAt time.Time       `json:"last_calculated_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func (CreditScore) TableName() string { return "credit_scores" }

type LoanStatus string

const (
	LoanApplied   LoanStatus = "APPLIED"
	LoanApproved  LoanStatus = "APPROVED"
	LoanDisbursed LoanStatus = "DISBURSED"
	LoanRepaying  LoanStatus = "REPAYING"
	LoanCompleted LoanStatus = "COMPLETED"
	LoanDefaulted LoanStatus = "DEFAULTED"
	LoanRejected  LoanStatus = "REJECTED"
)

// LoanApplication represents a crew member's loan request.
type LoanApplication struct {
	ID                   uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID         uuid.UUID    `json:"crew_member_id" gorm:"type:uuid;not null;index"`
	Category             LoanCategory `json:"category" gorm:"type:varchar(30);not null;default:'PERSONAL';index:idx_loan_crew_cat"`
	Purpose              string       `json:"purpose,omitempty" gorm:"type:varchar(255)"`
	AmountRequestedCents int64        `json:"amount_requested_cents" gorm:"type:bigint"`
	AmountApprovedCents  int64        `json:"amount_approved_cents" gorm:"type:bigint"`
	InterestRate         float64      `json:"interest_rate" gorm:"type:decimal(5,4)"`
	TenureDays           int          `json:"tenure_days"`
	Currency             string       `json:"currency" gorm:"default:'KES';not null"`
	Status               LoanStatus   `json:"status" gorm:"default:'APPLIED'"`
	LenderID             *uuid.UUID   `json:"lender_id,omitempty" gorm:"type:uuid"`
	DisbursedAt          *time.Time   `json:"disbursed_at,omitempty"`
	DueAt                *time.Time   `json:"due_at,omitempty"`
	RepaidAt             *time.Time   `json:"repaid_at,omitempty"`
	TotalRepaidCents     int64        `json:"total_repaid_cents" gorm:"type:bigint;default:0"`
	DaysPastDue          int          `json:"days_past_due" gorm:"default:0"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

func (LoanApplication) TableName() string { return "loan_applications" }

// IsOverdue returns true if the loan is past its due date.
func (l *LoanApplication) IsOverdue() bool {
	if l.DueAt == nil {
		return false
	}
	return time.Now().After(*l.DueAt) &&
		(l.Status == LoanDisbursed || l.Status == LoanRepaying)
}

// WasRepaidOnTime returns true if the loan was fully repaid before or on the due date.
func (l *LoanApplication) WasRepaidOnTime() bool {
	if l.RepaidAt == nil || l.DueAt == nil {
		return false
	}
	return !l.RepaidAt.After(*l.DueAt)
}

type PolicyStatus string

const (
	PolicyActive  PolicyStatus = "ACTIVE"
	PolicyLapsed  PolicyStatus = "LAPSED"
	PolicyClaimed PolicyStatus = "CLAIMED"
)

// InsurancePolicy represents a crew member's insurance coverage.
type InsurancePolicy struct {
	ID                 uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID       uuid.UUID    `json:"crew_member_id" gorm:"type:uuid;not null;index"`
	Provider           string       `json:"provider" gorm:"not null"`
	PolicyType         string       `json:"policy_type" gorm:"not null"`
	PremiumAmountCents int64        `json:"premium_amount_cents" gorm:"type:bigint"`
	PremiumFrequency   string       `json:"premium_frequency" gorm:"not null"`
	Currency           string       `json:"currency" gorm:"default:'KES';not null"`
	Status             PolicyStatus `json:"status" gorm:"default:'ACTIVE'"`
	StartDate          time.Time    `json:"start_date" gorm:"type:date"`
	EndDate            time.Time    `json:"end_date" gorm:"type:date"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

func (InsurancePolicy) TableName() string { return "insurance_policies" }

// WalletDailySnapshot records a wallet's closing balance for a specific day.
// Used by the credit scoring engine to compute accurate 30-day average balances.
type WalletDailySnapshot struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WalletID     uuid.UUID `json:"wallet_id" gorm:"type:uuid;not null;index:idx_wds_wallet_date"`
	CrewMemberID uuid.UUID `json:"crew_member_id" gorm:"type:uuid;not null;index:idx_wds_crew_date"`
	BalanceCents int64     `json:"balance_cents" gorm:"type:bigint;not null"`
	Currency     string    `json:"currency" gorm:"default:'KES';not null"`
	SnapshotDate time.Time `json:"snapshot_date" gorm:"type:date;not null;uniqueIndex:uq_wallet_snapshot_date"`
	CreatedAt    time.Time `json:"created_at"`
}

func (WalletDailySnapshot) TableName() string { return "wallet_daily_snapshots" }

// CreditScoreHistory records every score computation for trajectory analysis.
type CreditScoreHistory struct {
	ID            uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID  uuid.UUID       `json:"crew_member_id" gorm:"type:uuid;not null;index:idx_csh_crew_date"`
	Score         int             `json:"score" gorm:"index:idx_csh_score"`
	Grade         string          `json:"grade" gorm:"type:varchar(20);not null"`
	ModelVersion  string          `json:"model_version" gorm:"type:varchar(50);not null"`
	Factors       json.RawMessage `json:"factors" gorm:"type:jsonb"`
	Suggestions   json.RawMessage `json:"suggestions" gorm:"type:jsonb"`
	ComputedAt    time.Time       `json:"computed_at" gorm:"not null;default:now()"`
}

func (CreditScoreHistory) TableName() string { return "credit_score_history" }

// NegativeEventType constants
const (
	EventFraudFlag   = "FRAUD_FLAG"
	EventDispute     = "DISPUTE"
	EventAccountLock = "ACCOUNT_LOCK"
	EventKYCFailure  = "KYC_FAILURE"
	EventReversedTx  = "REVERSED_TX"
)

// CreditNegativeEvent records risk signals that impact credit scoring.
type CreditNegativeEvent struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID  uuid.UUID  `json:"crew_member_id" gorm:"type:uuid;not null;index:idx_cne_crew"`
	EventType     string     `json:"event_type" gorm:"type:varchar(50);not null;index:idx_cne_type"`
	Severity      string     `json:"severity" gorm:"type:varchar(20);not null;default:'MEDIUM'"`
	Description   string     `json:"description" gorm:"type:text"`
	Resolved      bool       `json:"resolved" gorm:"default:false"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (CreditNegativeEvent) TableName() string { return "credit_negative_events" }
