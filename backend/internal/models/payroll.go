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
	PayrollFailed     PayrollStatus = "FAILED"
)

// PeriodStatus tracks the lifecycle of a pay period.
type PeriodStatus string

const (
	PeriodOpen       PeriodStatus = "OPEN"
	PeriodClosed     PeriodStatus = "CLOSED"
	PeriodProcessing PeriodStatus = "PROCESSING"
	PeriodCompleted  PeriodStatus = "COMPLETED"
)

// PayPeriod represents a discrete pay window within a schedule (e.g., Mon-Fri for weekly).
type PayPeriod struct {
	ID            uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PayScheduleID uuid.UUID   `json:"pay_schedule_id" gorm:"column:pay_schedule_id;type:uuid;not null;index"`
	OrganizationID       uuid.UUID   `json:"organization_id" gorm:"column:sacco_id;type:uuid;not null;index"`
	PeriodStart   time.Time   `json:"period_start" gorm:"type:date;not null"`
	PeriodEnd     time.Time   `json:"period_end" gorm:"type:date;not null"`
	Status        PeriodStatus `json:"status" gorm:"default:'OPEN';index"`
	ClosedAt      *time.Time  `json:"closed_at,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`

	// Relations
	PaySchedule PaySchedule `json:"-" gorm:"foreignKey:PayScheduleID"`
	Organization Organization       `json:"-" gorm:"foreignKey:OrganizationID"`
}

func (PayPeriod) TableName() string { return "pay_periods" }

// PayrollRun represents a payroll processing cycle for a SACCO.
type PayrollRun struct {
	ID                   uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID              uuid.UUID     `json:"organization_id" gorm:"column:sacco_id;type:uuid;not null;index"`
	PeriodStart          time.Time     `json:"period_start" gorm:"type:date;not null"`
	PeriodEnd            time.Time     `json:"period_end" gorm:"type:date;not null"`
	Status               PayrollStatus `json:"status" gorm:"default:'DRAFT'"`
	TotalGrossCents      int64         `json:"total_gross_cents" gorm:"type:bigint"`
	TotalDeductionsCents int64         `json:"total_deductions_cents" gorm:"type:bigint"`
	TotalNetCents        int64         `json:"total_net_cents" gorm:"type:bigint"`
	Currency             string        `json:"currency" gorm:"default:'KES';not null"`
	ProcessedByID        *uuid.UUID    `json:"processed_by_id,omitempty" gorm:"type:uuid"`
	ApprovedByID         *uuid.UUID    `json:"approved_by_id,omitempty" gorm:"type:uuid"`
	SubmittedAt          *time.Time    `json:"submitted_at,omitempty"`
	PerpayReference      string        `json:"perpay_reference,omitempty"`
	PayScheduleID        *uuid.UUID    `json:"pay_schedule_id,omitempty" gorm:"type:uuid"`
	PayPeriodID          *uuid.UUID    `json:"pay_period_id,omitempty" gorm:"type:uuid"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`

	Organization Organization          `json:"-" gorm:"foreignKey:OrganizationID"`
	Entries     []PayrollEntry `json:"-" gorm:"foreignKey:PayrollRunID"`
	PaySchedule *PaySchedule   `json:"-" gorm:"foreignKey:PayScheduleID"`
	PayPeriod   *PayPeriod     `json:"-" gorm:"foreignKey:PayPeriodID"`
}

func (PayrollRun) TableName() string { return "payroll_runs" }

// PayrollEntry is a single crew member's payroll calculation within a run.
type PayrollEntry struct {
	ID                        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PayrollRunID              uuid.UUID `json:"payroll_run_id" gorm:"column:payroll_run_id;type:uuid;not null;index"`
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
