package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VehicleType string

const (
	VehicleMatatu VehicleType = "MATATU"
	VehicleBoda   VehicleType = "BODA"
	VehicleTukTuk VehicleType = "TUK_TUK"
)

// Vehicle represents a transport vehicle registered to a SACCO.
type Vehicle struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SaccoID        uuid.UUID      `json:"sacco_id" gorm:"type:uuid;not null;index"`
	RegistrationNo string         `json:"registration_no" gorm:"uniqueIndex;not null" validate:"required"`
	VehicleType    VehicleType    `json:"vehicle_type" gorm:"not null" validate:"required,oneof=MATATU BODA TUK_TUK"`
	RouteID        *uuid.UUID     `json:"route_id,omitempty" gorm:"type:uuid"`
	Capacity       int            `json:"capacity"`
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Sacco SACCO  `json:"-" gorm:"foreignKey:SaccoID"`
	Route *Route `json:"-" gorm:"foreignKey:RouteID"`
}

func (Vehicle) TableName() string { return "vehicles" }
