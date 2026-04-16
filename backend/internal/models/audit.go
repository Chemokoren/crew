package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLog records all create/update/delete actions for compliance.
type AuditLog struct {
	ID         uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID       `json:"user_id" gorm:"type:uuid;not null;index"`
	Action     string          `json:"action" gorm:"not null"`
	Resource   string          `json:"resource" gorm:"not null;index"`
	ResourceID *uuid.UUID      `json:"resource_id,omitempty" gorm:"type:uuid;index"`
	OldValue   json.RawMessage `json:"old_value,omitempty" gorm:"type:jsonb"`
	NewValue   json.RawMessage `json:"new_value,omitempty" gorm:"type:jsonb"`
	IPAddress  string          `json:"ip_address"`
	UserAgent  string          `json:"user_agent"`
	CreatedAt  time.Time       `json:"created_at" gorm:"index"`
}

func (AuditLog) TableName() string { return "audit_logs" }
