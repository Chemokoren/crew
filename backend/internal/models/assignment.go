package models

import (
	"time"

	"github.com/google/uuid"
)

type AssignmentStatus string

const (
	AssignmentScheduled AssignmentStatus = "SCHEDULED"
	AssignmentActive    AssignmentStatus = "ACTIVE"
	AssignmentCompleted AssignmentStatus = "COMPLETED"
	AssignmentCancelled AssignmentStatus = "CANCELLED"
)

type EarningModel string

const (
	EarningFixed      EarningModel = "FIXED"
	EarningCommission EarningModel = "COMMISSION"
	EarningHybrid     EarningModel = "HYBRID"
	EarningHourly     EarningModel = "HOURLY"
	EarningDailyRate  EarningModel = "DAILY_RATE"
	EarningPerTask    EarningModel = "PER_TASK"
	EarningPerPiece   EarningModel = "PER_PIECE"
	EarningSalary     EarningModel = "SALARY"
)

type CommissionBasis string

const (
	CommissionOnFareTotal CommissionBasis = "FARE_TOTAL"
	CommissionOnTripCount CommissionBasis = "TRIP_COUNT"
	CommissionOnRevenue   CommissionBasis = "REVENUE"
)

// WorkType classifies the kind of assignment for industry-agnostic scheduling.
type WorkType string

const (
	WorkTypeShift   WorkType = "SHIFT"   // Transport: vehicle shift
	WorkTypeDaily   WorkType = "DAILY"   // Construction: full day on-site
	WorkTypeHourly  WorkType = "HOURLY"  // Health: visit-based hours
	WorkTypeTask    WorkType = "TASK"    // Logistics: per-delivery
	WorkTypeProject WorkType = "PROJECT" // Road construction: milestone-based
	WorkTypeBooking WorkType = "BOOKING" // Facilitator: per-booking commission
)

// Assignment represents a crew member's work assignment (shift, daily, hourly, task, etc.).
type Assignment struct {
	ID               uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID     uuid.UUID        `json:"crew_member_id" gorm:"type:uuid;not null;index"`
	VehicleID        *uuid.UUID       `json:"vehicle_id,omitempty" gorm:"type:uuid;index"`
	OrganizationID   uuid.UUID        `json:"organization_id" gorm:"column:sacco_id;type:uuid;not null;index"`
	RouteID          *uuid.UUID       `json:"route_id,omitempty" gorm:"type:uuid"`
	ShiftDate        time.Time        `json:"shift_date" gorm:"type:date;not null;index"`
	ShiftStart       time.Time        `json:"shift_start" gorm:"not null"`
	ShiftEnd         *time.Time       `json:"shift_end,omitempty"`
	Status           AssignmentStatus `json:"status" gorm:"default:'SCHEDULED';index"`
	EarningModel     EarningModel     `json:"earning_model" gorm:"not null"`
	FixedAmountCents int64            `json:"fixed_amount_cents,omitempty" gorm:"type:bigint;default:0"`
	CommissionRate   float64          `json:"commission_rate,omitempty" gorm:"type:decimal(5,4)"`
	HybridBaseCents  int64            `json:"hybrid_base_cents,omitempty" gorm:"type:bigint;default:0"`
	CommissionBasis  CommissionBasis  `json:"commission_basis,omitempty"`
	Notes            string           `json:"notes,omitempty"`
	CreatedByID      uuid.UUID        `json:"created_by_id" gorm:"type:uuid;not null"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`

	// --- Generalized assignment fields (Phase C) ---
	WorkType         WorkType   `json:"work_type" gorm:"type:varchar(20);not null;default:'SHIFT'"`
	WorkSite         string     `json:"work_site,omitempty" gorm:"type:varchar(255)"`
	ProjectRef       string     `json:"project_ref,omitempty" gorm:"type:varchar(100)"`
	HoursWorked      *float64   `json:"hours_worked,omitempty" gorm:"type:decimal(6,2)"`
	UnitsCompleted   *int       `json:"units_completed,omitempty"`
	HourlyRateCents  int64      `json:"hourly_rate_cents,omitempty" gorm:"type:bigint;default:0"`
	DailyRateCents   int64      `json:"daily_rate_cents,omitempty" gorm:"type:bigint;default:0"`
	PerUnitRateCents int64      `json:"per_unit_rate_cents,omitempty" gorm:"type:bigint;default:0"`
	OvertimeHours    *float64   `json:"overtime_hours,omitempty" gorm:"type:decimal(6,2)"`
	OvertimeRateCents int64     `json:"overtime_rate_cents,omitempty" gorm:"type:bigint;default:0"`
	CheckInAt        *time.Time `json:"check_in_at,omitempty"`
	CheckOutAt       *time.Time `json:"check_out_at,omitempty"`
	PayScheduleID    *uuid.UUID `json:"pay_schedule_id,omitempty" gorm:"type:uuid"`

	// Relations
	CrewMember  CrewMember  `json:"-" gorm:"foreignKey:CrewMemberID"`
	Vehicle     Vehicle     `json:"-" gorm:"foreignKey:VehicleID"`
	Organization Organization  `json:"-" gorm:"foreignKey:OrganizationID"`
	Route       *Route      `json:"-" gorm:"foreignKey:RouteID"`
	Earnings    []Earning   `json:"-" gorm:"foreignKey:AssignmentID"`
	PaySchedule *PaySchedule `json:"-" gorm:"foreignKey:PayScheduleID"`
}

func (Assignment) TableName() string { return "assignments" }

// IsTransport returns true if this is a transport-type assignment.
func (a *Assignment) IsTransport() bool {
	return a.WorkType == WorkTypeShift && a.VehicleID != nil
}
