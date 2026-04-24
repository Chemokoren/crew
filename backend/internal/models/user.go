package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/pkg/types"
)

// User is the SOLE auth entity. All login/password/phone lives here.
// CrewMember is a profile — it never stores credentials.
type User struct {
	ID           uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Phone        string           `json:"phone" gorm:"uniqueIndex;not null"`
	Email        string           `json:"email,omitempty"`
	PasswordHash string           `json:"-" gorm:"not null"`
	PINHash      string           `json:"-" gorm:"column:pin_hash"`
	SystemRole   types.SystemRole `json:"system_role" gorm:"not null"`
	CrewMemberID *uuid.UUID       `json:"crew_member_id,omitempty" gorm:"type:uuid"`
	SaccoID      *uuid.UUID       `json:"sacco_id,omitempty" gorm:"type:uuid"`
	IsActive     bool             `json:"is_active" gorm:"default:true"`
	LastLoginAt  *time.Time       `json:"last_login_at,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`

	// Relations (loaded via Preload, never auto-serialized)
	CrewMember *CrewMember `json:"-" gorm:"foreignKey:CrewMemberID"`
	Sacco      *SACCO      `json:"-" gorm:"foreignKey:SaccoID"`
}

// TableName overrides the default table name.
func (User) TableName() string { return "users" }
