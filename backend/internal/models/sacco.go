package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SACCO represents a Savings and Credit Cooperative Organization.
type SACCO struct {
	ID                 uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name               string         `json:"name" gorm:"not null" validate:"required"`
	RegistrationNumber string         `json:"registration_number" gorm:"uniqueIndex;not null"`
	County             string         `json:"county" gorm:"not null"`
	SubCounty          string         `json:"sub_county"`
	ContactPhone       string         `json:"contact_phone" gorm:"not null"`
	ContactEmail       string         `json:"contact_email"`
	Currency           string         `json:"currency" gorm:"default:'KES';not null"`
	IsActive           bool           `json:"is_active" gorm:"default:true"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Vehicles    []Vehicle             `json:"-" gorm:"foreignKey:SaccoID"`
	Memberships []CrewSACCOMembership `json:"-" gorm:"foreignKey:SaccoID"`
}

func (SACCO) TableName() string { return "saccos" }
