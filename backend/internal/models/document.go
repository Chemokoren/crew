package models

import (
	"time"

	"github.com/google/uuid"
)

type DocumentType string

const (
	DocKYCFront   DocumentType = "KYC_ID_FRONT"
	DocKYCBack    DocumentType = "KYC_ID_BACK"
	DocKYCSelfie  DocumentType = "KYC_SELFIE"
	DocSACCOReg   DocumentType = "SACCO_REGISTRATION"
	DocVehicleLog DocumentType = "VEHICLE_LOGBOOK"
	DocOther      DocumentType = "OTHER"
)

// Document represents an uploaded file (KYC docs, vehicle logs, etc.).
// Files are stored in MinIO; this model tracks metadata.
type Document struct {
	ID           uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CrewMemberID *uuid.UUID   `json:"crew_member_id,omitempty" gorm:"column:crew_member_id;type:uuid;index"`
	OrganizationID      *uuid.UUID   `json:"organization_id,omitempty" gorm:"column:sacco_id;type:uuid;index"`
	VehicleID    *uuid.UUID   `json:"vehicle_id,omitempty" gorm:"column:vehicle_id;type:uuid;index"`
	DocumentType DocumentType `json:"document_type" gorm:"not null"`
	FileName     string       `json:"file_name" gorm:"not null"`
	FileSize     int64        `json:"file_size"`
	MimeType     string       `json:"mime_type"`
	StoragePath  string       `json:"-" gorm:"not null"` // MinIO object key — never exposed
	UploadedByID uuid.UUID    `json:"uploaded_by_id" gorm:"type:uuid;not null"`
	Status       string       `json:"status" gorm:"default:'PENDING'"`
	VerifiedByID *uuid.UUID   `json:"verified_by_id,omitempty" gorm:"type:uuid"`
	VerifiedAt   *time.Time   `json:"verified_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

func (Document) TableName() string { return "documents" }
