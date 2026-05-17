package models

import (
	"time"

	"github.com/google/uuid"
)

// SystemSetting is a key-value store for global platform configuration.
// Keys use dot-notation namespacing: "feature.loans_enabled", "maintenance.active", "defaults.kyc_required", etc.
type SystemSetting struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Key       string    `json:"key" gorm:"type:varchar(255);uniqueIndex;not null"`
	Value     string    `json:"value" gorm:"type:text;not null"`
	ValueType string    `json:"value_type" gorm:"type:varchar(20);not null;default:'string'"` // string, bool, number, json
	Category  string    `json:"category" gorm:"type:varchar(50);not null;default:'general'"`
	Label     string    `json:"label" gorm:"type:varchar(255)"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SystemSetting) TableName() string { return "system_settings" }

// AnnouncementSeverity defines the urgency of an announcement.
type AnnouncementSeverity string

const (
	SeverityInfo     AnnouncementSeverity = "INFO"
	SeverityWarning  AnnouncementSeverity = "WARNING"
	SeverityCritical AnnouncementSeverity = "CRITICAL"
)

// SystemAnnouncement represents a platform-wide announcement.
type SystemAnnouncement struct {
	ID        uuid.UUID            `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title     string               `json:"title" gorm:"type:varchar(255);not null"`
	Body      string               `json:"body" gorm:"type:text;not null"`
	Severity  AnnouncementSeverity `json:"severity" gorm:"type:varchar(20);not null;default:'INFO'"`
	StartAt   *time.Time           `json:"start_at,omitempty" gorm:"type:timestamptz"`
	EndAt     *time.Time           `json:"end_at,omitempty" gorm:"type:timestamptz"`
	IsActive  bool                 `json:"is_active" gorm:"default:true"`
	CreatedBy *uuid.UUID           `json:"created_by,omitempty" gorm:"type:uuid"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

func (SystemAnnouncement) TableName() string { return "system_announcements" }
