package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type WebhookSource string

const (
	WebhookJamboPay WebhookSource = "JAMBOPAY"
	WebhookPerpay   WebhookSource = "PERPAY"
	WebhookIPRS     WebhookSource = "IPRS"
)

// WebhookEvent stores inbound webhook payloads for reliable processing.
type WebhookEvent struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Source       WebhookSource   `json:"source" gorm:"not null;index"`
	EventType    string          `json:"event_type" gorm:"not null"`
	ExternalRef  string          `json:"external_ref" gorm:"index"`
	Payload      json.RawMessage `json:"payload" gorm:"type:jsonb;not null"`
	IsProcessed  bool            `json:"is_processed" gorm:"default:false;index"`
	ProcessedAt  *time.Time      `json:"processed_at,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
	RetryCount   int             `json:"retry_count" gorm:"default:0"`
	CreatedAt    time.Time       `json:"created_at"`
}

func (WebhookEvent) TableName() string { return "webhook_events" }
