package models

import (
	"time"

	"github.com/google/uuid"
)

type SACCORole string

const (
	SACCORoleMember   SACCORole = "MEMBER"
	SACCORoleAdmin    SACCORole = "ADMIN"
	SACCORoleChairman SACCORole = "CHAIRMAN"
)

// CrewSACCOMembership links a crew member to a SACCO with a role.
type CrewSACCOMembership struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID uuid.UUID  `json:"crew_member_id" gorm:"type:uuid;not null;index"`
	SaccoID      uuid.UUID  `json:"sacco_id" gorm:"type:uuid;not null;index"`
	RoleInSacco  SACCORole  `json:"role_in_sacco" gorm:"default:'MEMBER'"`
	JoinedAt     time.Time  `json:"joined_at" gorm:"not null"`
	LeftAt       *time.Time `json:"left_at,omitempty"`
	IsActive     bool       `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relations
	CrewMember CrewMember `json:"-" gorm:"foreignKey:CrewMemberID"`
	Sacco      SACCO      `json:"-" gorm:"foreignKey:SaccoID"`
}

func (CrewSACCOMembership) TableName() string { return "crew_sacco_memberships" }
