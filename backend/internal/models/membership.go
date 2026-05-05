package models

import (
	"time"

	"github.com/google/uuid"
)

// OrganizationRole defines the role a member holds within an organization.
type OrganizationRole string

const (
	OrgRoleMember   OrganizationRole = "MEMBER"
	OrgRoleAdmin    OrganizationRole = "ADMIN"
	OrgRoleChairman OrganizationRole = "CHAIRMAN"
)

// Backward compatibility aliases
type SACCORole = OrganizationRole

const (
	SACCORoleMember   = OrgRoleMember
	SACCORoleAdmin    = OrgRoleAdmin
	SACCORoleChairman = OrgRoleChairman
)

// CrewOrganizationMembership links a crew member to an organization with a role.
type CrewOrganizationMembership struct {
	ID             uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID   uuid.UUID        `json:"crew_member_id" gorm:"type:uuid;not null;index"`
	OrganizationID uuid.UUID        `json:"organization_id" gorm:"column:sacco_id;type:uuid;not null;index"`
	RoleInOrg      OrganizationRole `json:"role_in_org" gorm:"column:role_in_sacco;default:'MEMBER'"`
	JoinedAt       time.Time        `json:"joined_at" gorm:"not null"`
	LeftAt         *time.Time       `json:"left_at,omitempty"`
	IsActive       bool             `json:"is_active" gorm:"default:true"`
	PayScheduleID  *uuid.UUID       `json:"pay_schedule_id,omitempty" gorm:"type:uuid"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`

	// Relations
	CrewMember   CrewMember    `json:"-" gorm:"foreignKey:CrewMemberID"`
	Organization Organization  `json:"-" gorm:"foreignKey:OrganizationID"`
	PaySchedule  *PaySchedule  `json:"-" gorm:"foreignKey:PayScheduleID"`
}

func (CrewOrganizationMembership) TableName() string { return "crew_sacco_memberships" }

// Backward compatibility alias
type CrewSACCOMembership = CrewOrganizationMembership
