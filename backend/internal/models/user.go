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
	OrganizationID *uuid.UUID       `json:"organization_id,omitempty" gorm:"column:sacco_id;type:uuid"`
	PreferredLanguage string        `json:"preferred_language" gorm:"type:varchar(10);not null;default:'sw'"`
	IsActive     bool             `json:"is_active" gorm:"default:true"`
	LastLoginAt  *time.Time       `json:"last_login_at,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`

	// Relations (loaded via Preload, never auto-serialized)
	CrewMember   *CrewMember   `json:"-" gorm:"foreignKey:CrewMemberID"`
	Organization *Organization `json:"-" gorm:"foreignKey:OrganizationID"`
}

// TableName overrides the default table name.
func (User) TableName() string { return "users" }
