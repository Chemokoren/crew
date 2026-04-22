package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type NotificationChannel string

const (
	ChannelSMS   NotificationChannel = "SMS"
	ChannelPush  NotificationChannel = "PUSH"
	ChannelInApp NotificationChannel = "IN_APP"
)

type NotificationStatus string

const (
	NotifPending NotificationStatus = "PENDING"
	NotifSent    NotificationStatus = "SENT"
	NotifFailed  NotificationStatus = "FAILED"
	NotifRead    NotificationStatus = "READ"
)

// Notification is a message sent to a user via SMS, push, or in-app.
type Notification struct {
	ID        uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID           `json:"user_id" gorm:"type:uuid;not null;index"`
	Channel   NotificationChannel `json:"channel" gorm:"not null"`
	Title     string              `json:"title" gorm:"not null"`
	Body      string              `json:"body" gorm:"not null"`
	Data      json.RawMessage     `json:"data,omitempty" gorm:"type:jsonb"`
	Status    NotificationStatus  `json:"status" gorm:"default:'PENDING'"`
	SentAt    *time.Time          `json:"sent_at,omitempty"`
	ReadAt    *time.Time          `json:"read_at,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }

// NotificationTemplate defines reusable templates for system events.
type NotificationTemplate struct {
	ID            uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventName     string              `json:"event_name" gorm:"uniqueIndex;not null"`
	Channel       NotificationChannel `json:"channel" gorm:"not null"`
	TitleTemplate string              `json:"title_template" gorm:"not null"`
	BodyTemplate  string              `json:"body_template" gorm:"not null"`
	IsActive      bool                `json:"is_active" gorm:"default:true"`
}

func (NotificationTemplate) TableName() string { return "notification_templates" }

// NotificationPreference stores user-specific opt-in/opt-out settings for channels.
type NotificationPreference struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           uuid.UUID `json:"user_id" gorm:"type:uuid;uniqueIndex;not null"`
	SMSOptIn         bool      `json:"sms_opt_in" gorm:"default:true"`
	PushOptIn        bool      `json:"push_opt_in" gorm:"default:true"`
	InAppOptIn       bool      `json:"in_app_opt_in" gorm:"default:true"`
	MarketingOptIn   bool      `json:"marketing_opt_in" gorm:"default:false"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (NotificationPreference) TableName() string { return "notification_preferences" }
