package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationType classifies the kind of organization (tenant).
type OrganizationType string

const (
	OrgTypeSacco           OrganizationType = "SACCO"
	OrgTypeConstructionFirm OrganizationType = "CONSTRUCTION_FIRM"
	OrgTypeLogisticsCompany OrganizationType = "LOGISTICS_COMPANY"
	OrgTypeHealthNGO        OrganizationType = "HEALTH_NGO"
	OrgTypeAgricultureCoop  OrganizationType = "AGRICULTURE_COOP"
	OrgTypeHospitalityGroup OrganizationType = "HOSPITALITY_GROUP"
	OrgTypeGeneral          OrganizationType = "GENERAL"
)

// Organization represents a tenant organization (SACCO, contractor, NGO, etc.).
// Previously named "SACCO" — renamed for industry-agnostic support (Decision D1).
type Organization struct {
	ID                 uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name               string           `json:"name" gorm:"not null" validate:"required"`
	RegistrationNumber string           `json:"registration_number" gorm:"uniqueIndex;not null"`
	County             string           `json:"county" gorm:"not null"`
	SubCounty          string           `json:"sub_county"`
	ContactPhone       string           `json:"contact_phone" gorm:"not null"`
	ContactEmail       string           `json:"contact_email"`
	Currency           string           `json:"currency" gorm:"default:'KES';not null"`
	IsActive           bool             `json:"is_active" gorm:"default:true"`
	OrganizationType   OrganizationType `json:"organization_type" gorm:"type:varchar(50);not null;default:'SACCO'"`
	IndustryType       IndustryType     `json:"industry_type" gorm:"type:varchar(30);not null;default:'TRANSPORT'"`
	DefaultLanguage    string           `json:"default_language" gorm:"type:varchar(10);not null;default:'sw'"`
	TenantConfig       json.RawMessage  `json:"tenant_config" gorm:"type:jsonb;default:'{}'"`
	DisplayName        string           `json:"display_name,omitempty" gorm:"type:varchar(255)"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	DeletedAt          gorm.DeletedAt   `json:"-" gorm:"index"`

	// Relations
	Vehicles     []Vehicle                  `json:"-" gorm:"foreignKey:OrganizationID"`
	Memberships  []CrewOrganizationMembership `json:"-" gorm:"foreignKey:OrganizationID"`
	JobTypes     []TenantJobType            `json:"-" gorm:"foreignKey:OrganizationID"`
	PaySchedules []PaySchedule              `json:"-" gorm:"foreignKey:OrganizationID"`
}

// TableName returns the actual database table name.
// The DB table is still "saccos" — will be migrated to "organizations" in Phase H.
func (Organization) TableName() string { return "saccos" }

// GetTenantConfig deserializes the JSONB tenant_config into a typed struct.
func (o *Organization) GetTenantConfig() (*TenantConfig, error) {
	var cfg TenantConfig
	if len(o.TenantConfig) == 0 {
		return &cfg, nil
	}
	if err := json.Unmarshal(o.TenantConfig, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SetTenantConfig serializes a typed TenantConfig into the JSONB field.
func (o *Organization) SetTenantConfig(cfg *TenantConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	o.TenantConfig = data
	return nil
}

// IsSACCO returns true if this is a SACCO-type organization (transport).
func (o *Organization) IsSACCO() bool {
	return o.OrganizationType == OrgTypeSacco
}

// DisplayLabel returns a user-friendly label based on organization type.
func (o *Organization) DisplayLabel() string {
	labels := map[OrganizationType]string{
		OrgTypeSacco:            "SACCO",
		OrgTypeConstructionFirm: "Contractor",
		OrgTypeLogisticsCompany: "Logistics Co.",
		OrgTypeHealthNGO:        "Health Organization",
		OrgTypeAgricultureCoop:  "Cooperative",
		OrgTypeHospitalityGroup: "Hospitality Group",
		OrgTypeGeneral:          "Organization",
	}
	if label, ok := labels[o.OrganizationType]; ok {
		return label
	}
	return "Organization"
}

// SACCO is a type alias for backward compatibility during migration.
// New code should use Organization directly.
type SACCO = Organization
