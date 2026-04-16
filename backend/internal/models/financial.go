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
)

// LoanApplication represents a crew member's loan request.
type LoanApplication struct {
	ID                   uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID         uuid.UUID  `json:"crew_member_id" gorm:"type:uuid;not null;index"`
	AmountRequestedCents int64      `json:"amount_requested_cents" gorm:"type:bigint"`
	AmountApprovedCents  int64      `json:"amount_approved_cents" gorm:"type:bigint"`
	InterestRate         float64    `json:"interest_rate" gorm:"type:decimal(5,4)"`
	TenureDays           int        `json:"tenure_days"`
	Currency             string     `json:"currency" gorm:"default:'KES';not null"`
	Status               LoanStatus `json:"status" gorm:"default:'APPLIED'"`
	LenderID             uuid.UUID  `json:"lender_id" gorm:"type:uuid"`
	DisbursedAt          *time.Time `json:"disbursed_at,omitempty"`
	DueAt                *time.Time `json:"due_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (LoanApplication) TableName() string { return "loan_applications" }

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
