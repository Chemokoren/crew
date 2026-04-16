package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KYCStatus string

const (
	KYCPending  KYCStatus = "PENDING"
	KYCVerified KYCStatus = "VERIFIED"
	KYCRejected KYCStatus = "REJECTED"
)

type CrewRole string

const (
	RoleDriver    CrewRole = "DRIVER"
	RoleConductor CrewRole = "CONDUCTOR"
	RoleRider     CrewRole = "RIDER"
	RoleOther     CrewRole = "OTHER"
)

// CrewMember is the PROFILE entity. No auth fields here.
// Auth (phone, password) lives in User. Linked via User.CrewMemberID.
type CrewMember struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewID        string         `json:"crew_id" gorm:"uniqueIndex;not null"`
	NationalID    string         `json:"-" gorm:"not null"` // Encrypted at rest
	FirstName     string         `json:"first_name" gorm:"not null" validate:"required"`
	LastName      string         `json:"last_name" gorm:"not null" validate:"required"`
	KYCStatus     KYCStatus      `json:"kyc_status" gorm:"default:'PENDING'"`
	KYCVerifiedAt *time.Time     `json:"kyc_verified_at,omitempty"`
	PhotoURL      string         `json:"photo_url,omitempty"`
	Role          CrewRole       `json:"role" gorm:"not null" validate:"required,oneof=DRIVER CONDUCTOR RIDER OTHER"`
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Memberships []CrewSACCOMembership `json:"-" gorm:"foreignKey:CrewMemberID"`
	Documents   []Document            `json:"-" gorm:"foreignKey:CrewMemberID"`
}

func (CrewMember) TableName() string { return "crew_members" }

// FullName returns the crew member's full name.
func (c CrewMember) FullName() string {
	return c.FirstName + " " + c.LastName
}
