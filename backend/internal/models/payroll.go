package models

import (
	"time"

	"github.com/google/uuid"
)

type PayrollStatus string

const (
	PayrollDraft      PayrollStatus = "DRAFT"
	PayrollProcessing PayrollStatus = "PROCESSING"
	PayrollApproved   PayrollStatus = "APPROVED"
	PayrollSubmitted  PayrollStatus = "SUBMITTED"
	PayrollCompleted  PayrollStatus = "COMPLETED"
)

// PayrollRun represents a payroll processing cycle for a SACCO.
type PayrollRun struct {
	ID                   uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SaccoID              uuid.UUID     `json:"sacco_id" gorm:"type:uuid;not null;index"`
	PeriodStart          time.Time     `json:"period_start" gorm:"type:date;not null"`
	PeriodEnd            time.Time     `json:"period_end" gorm:"type:date;not null"`
	Status               PayrollStatus `json:"status" gorm:"default:'DRAFT'"`
	TotalGrossCents      int64         `json:"total_gross_cents" gorm:"type:bigint"`
	TotalDeductionsCents int64         `json:"total_deductions_cents" gorm:"type:bigint"`
	TotalNetCents        int64         `json:"total_net_cents" gorm:"type:bigint"`
	Currency             string        `json:"currency" gorm:"default:'KES';not null"`
	ProcessedByID        uuid.UUID     `json:"processed_by_id" gorm:"type:uuid"`
	ApprovedByID         *uuid.UUID    `json:"approved_by_id,omitempty" gorm:"type:uuid"`
	SubmittedAt          *time.Time    `json:"submitted_at,omitempty"`
	PerpayReference      string        `json:"perpay_reference,omitempty"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`

	Sacco   SACCO          `json:"-" gorm:"foreignKey:SaccoID"`
	Entries []PayrollEntry `json:"-" gorm:"foreignKey:PayrollRunID"`
}

func (PayrollRun) TableName() string { return "payroll_runs" }

// PayrollEntry is a single crew member's payroll calculation within a run.
type PayrollEntry struct {
	ID                        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PayrollRunID              uuid.UUID `json:"payroll_run_id" gorm:"type:uuid;not null;index"`
	CrewMemberID              uuid.UUID `json:"crew_member_id" gorm:"type:uuid;not null"`
	GrossEarningsCents        int64     `json:"gross_earnings_cents" gorm:"type:bigint"`
	SHADeductionCents         int64     `json:"sha_deduction_cents" gorm:"type:bigint"`
	NSSFDeductionCents        int64     `json:"nssf_deduction_cents" gorm:"type:bigint"`
	HousingLevyDeductionCents int64     `json:"housing_levy_deduction_cents" gorm:"type:bigint"`
	OtherDeductionsCents      int64     `json:"other_deductions_cents" gorm:"type:bigint"`
	NetPayCents               int64     `json:"net_pay_cents" gorm:"type:bigint"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`

	CrewMember CrewMember `json:"-" gorm:"foreignKey:CrewMemberID"`
}

func (PayrollEntry) TableName() string { return "payroll_entries" }

type RateType string

const (
	RatePercentage RateType = "PERCENTAGE"
	RateFixed      RateType = "FIXED"
	RateTiered     RateType = "TIERED"
)

// StatutoryRate holds Kenya statutory deduction rates (SHA, NSSF, Housing Levy).
type StatutoryRate struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name          string    `json:"name" gorm:"not null"`
	Rate          float64   `json:"rate" gorm:"type:decimal(8,4)"`
	RateType      RateType  `json:"rate_type" gorm:"not null"`
	EffectiveFrom time.Time `json:"effective_from" gorm:"type:date;not null"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (StatutoryRate) TableName() string { return "statutory_rates" }
