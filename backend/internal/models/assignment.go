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
)

type CommissionBasis string

const (
	CommissionOnFareTotal CommissionBasis = "FARE_TOTAL"
	CommissionOnTripCount CommissionBasis = "TRIP_COUNT"
	CommissionOnRevenue   CommissionBasis = "REVENUE"
)

// Assignment represents a crew member's shift on a vehicle.
type Assignment struct {
	ID               uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID     uuid.UUID        `json:"crew_member_id" gorm:"type:uuid;not null;index"`
	VehicleID        uuid.UUID        `json:"vehicle_id" gorm:"type:uuid;not null;index"`
	SaccoID          uuid.UUID        `json:"sacco_id" gorm:"type:uuid;not null;index"`
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

	// Relations
	CrewMember CrewMember `json:"-" gorm:"foreignKey:CrewMemberID"`
	Vehicle    Vehicle    `json:"-" gorm:"foreignKey:VehicleID"`
	Sacco      SACCO      `json:"-" gorm:"foreignKey:SaccoID"`
	Route      *Route     `json:"-" gorm:"foreignKey:RouteID"`
	Earnings   []Earning  `json:"-" gorm:"foreignKey:AssignmentID"`
}

func (Assignment) TableName() string { return "assignments" }
