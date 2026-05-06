package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkSite represents a named project location or work site owned by an organization.
type WorkSite struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID      `json:"organization_id" gorm:"column:organization_id;type:uuid;not null;index"`
	Name           string         `json:"name" gorm:"type:varchar(255);not null"`
	ProjectRef     string         `json:"project_ref,omitempty" gorm:"type:varchar(100)"`
	Address        string         `json:"address,omitempty" gorm:"type:varchar(500)"`
	Description    string         `json:"description,omitempty" gorm:"type:text"`
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	CreatedByID    uuid.UUID      `json:"created_by_id" gorm:"type:uuid;not null"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Organization SACCO `json:"-" gorm:"foreignKey:OrganizationID"`
}

func (WorkSite) TableName() string { return "work_sites" }
