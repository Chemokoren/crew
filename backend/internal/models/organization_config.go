package models

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OrganizationConfig holds typed, queryable configuration for a tenant organization.
// Implements Decision D2 (Layer 2: Standard configurable domains).
type OrganizationConfig struct {
	OrganizationID             uuid.UUID `json:"organization_id" gorm:"type:uuid;primaryKey"`
	AssignmentTypes            StringArr `json:"assignment_types" gorm:"type:text[];default:'{SHIFT}'"` // SHIFT, DAILY, HOURLY, TASK, PROJECT, BOOKING
	EarningModels              StringArr `json:"earning_models" gorm:"type:text[];default:'{FIXED}'"` // FIXED, COMMISSION, HOURLY, DAILY_RATE, PER_TASK, SALARY
	PaymentFrequencies         StringArr `json:"payment_frequencies" gorm:"type:text[];default:'{DAILY}'"` // DAILY, WEEKLY, BI_WEEKLY, MONTHLY
	StatutoryBodies            StringArr `json:"statutory_bodies" gorm:"type:text[];default:'{}'"`     // SHA, NSSF, HousingLevy
	RequiresGPS                bool      `json:"requires_gps" gorm:"default:false"`
	RequiresSupervisorApproval bool      `json:"requires_supervisor_approval" gorm:"default:false"`
	MaxHoursPerDay             float64   `json:"max_hours_per_day" gorm:"type:numeric;default:12"`
	OvertimeMultiplier         float64   `json:"overtime_multiplier" gorm:"type:numeric;default:1.5"`
	UpdatedAt                  time.Time `json:"updated_at"`

	// Relations
	Organization Organization `json:"-" gorm:"foreignKey:OrganizationID"`
}

func (OrganizationConfig) TableName() string { return "organization_configs" }

// RoleConfig holds per-role overrides for an organization.
// Implements Decision D2 (Layer 2: role-level configuration).
type RoleConfig struct {
	ID                       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID           uuid.UUID `json:"organization_id" gorm:"type:uuid;not null;index"`
	JobTypeCode              string    `json:"job_type_code" gorm:"type:varchar(50);not null"`
	AllowedEarningModels     StringArr `json:"allowed_earning_models" gorm:"type:text[]"`
	DefaultEarningModel      string    `json:"default_earning_model" gorm:"type:varchar(30)"`
	StatutoryApplicable      StringArr `json:"statutory_applicable" gorm:"type:text[]"`
	PaymentFrequencyOverride string    `json:"payment_frequency_override,omitempty" gorm:"type:varchar(20)"`
	OvertimeMultiplier       float64   `json:"overtime_multiplier" gorm:"type:numeric;default:1.0"`
	CreatedAt                time.Time `json:"created_at"`

	// Relations
	Organization Organization `json:"-" gorm:"foreignKey:OrganizationID"`
}

func (RoleConfig) TableName() string { return "role_configs" }

// OrganizationConfigExtension stores advanced/edge-case configuration in JSONB.
// Implements Decision D2 (Layer 3: Advanced JSONB extensions).
type OrganizationConfigExtension struct {
	OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;primaryKey"`
	CustomConfig   JSON      `json:"custom_config" gorm:"type:jsonb;default:'{}'"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Relations
	Organization Organization `json:"-" gorm:"foreignKey:OrganizationID"`
}

func (OrganizationConfigExtension) TableName() string { return "organization_config_extensions" }

// --- Decision D7: Worker Statutory Opt-In ---

// StatutoryElectionStatus represents the status of a worker's statutory election.
type StatutoryElectionStatus string

const (
	StatutoryOptIn           StatutoryElectionStatus = "OPT_IN"
	StatutoryOptOut          StatutoryElectionStatus = "OPT_OUT"
	StatutoryEmployerMandated StatutoryElectionStatus = "EMPLOYER_MANDATED"
)

// WorkerStatutoryElection records a worker's statutory deduction election per organization.
// Implements Decision D7: Opt-in per role with employer override.
type WorkerStatutoryElection struct {
	ID             uuid.UUID               `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkerID       uuid.UUID               `json:"worker_id" gorm:"type:uuid;not null;index"`
	OrganizationID uuid.UUID               `json:"organization_id" gorm:"type:uuid;not null;index"`
	StatutoryBody  string                  `json:"statutory_body" gorm:"type:varchar(50);not null"` // SHA, NSSF, HousingLevy
	Status         StatutoryElectionStatus `json:"status" gorm:"type:varchar(20);not null;default:'OPT_IN'"`
	ElectedBy      uuid.UUID               `json:"elected_by" gorm:"type:uuid;not null"` // worker_id or admin_id
	EffectiveFrom  time.Time               `json:"effective_from" gorm:"type:date;not null"`
	EffectiveTo    *time.Time              `json:"effective_to,omitempty" gorm:"type:date"`
	CreatedAt      time.Time               `json:"created_at"`

	// Relations
	Worker       CrewMember   `json:"-" gorm:"foreignKey:WorkerID"`
	Organization Organization `json:"-" gorm:"foreignKey:OrganizationID"`
}

func (WorkerStatutoryElection) TableName() string { return "worker_statutory_elections" }

// --- Decision D3: Financial Consent ---

// ConsentTier defines the level of financial data sharing.
type ConsentTier int

const (
	ConsentTier1 ConsentTier = 1 // Aggregated monthly income — implied on application
	ConsentTier2 ConsentTier = 2 // Aggregated + consistency score — explicit consent
	ConsentTier3 ConsentTier = 3 // De-identified transaction history — explicit + separate
	ConsentTier4 ConsentTier = 4 // Raw earnings (worker's own download only)
)

// FinancialConsent records a worker's consent for sharing financial data.
// Implements Decision D3: Consent-gated tiered financial profile API.
type FinancialConsent struct {
	ID        uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkerID  uuid.UUID   `json:"worker_id" gorm:"type:uuid;not null;index"`
	Tier      ConsentTier `json:"tier" gorm:"type:int;not null"`
	PartnerID *uuid.UUID  `json:"partner_id,omitempty" gorm:"type:uuid"` // Lender/insurer who requested
	ConsentedAt time.Time `json:"consented_at" gorm:"not null"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	// Relations
	Worker CrewMember `json:"-" gorm:"foreignKey:WorkerID"`
}

func (FinancialConsent) TableName() string { return "financial_consents" }

// IsActive returns true if the consent is currently active (not expired or revoked).
func (fc *FinancialConsent) IsActive() bool {
	if fc.RevokedAt != nil {
		return false
	}
	if fc.ExpiresAt != nil && fc.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

// --- Helper types for PostgreSQL array columns ---

// StringArr is a helper for PostgreSQL TEXT[] columns.
type StringArr []string

// Scan implements sql.Scanner for PostgreSQL TEXT[] columns.
func (s *StringArr) Scan(value interface{}) error {
	if value == nil {
		*s = StringArr{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return s.scanString(string(v))
	case string:
		return s.scanString(v)
	}
	return nil
}

func (s *StringArr) scanString(v string) error {
	// PostgreSQL returns TEXT[] as {val1,val2,val3}
	v = strings.Trim(v, "{}")
	if v == "" {
		*s = StringArr{}
		return nil
	}
	*s = strings.Split(v, ",")
	return nil
}

// Value implements driver.Valuer for PostgreSQL TEXT[] columns.
func (s StringArr) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "{}", nil
	}
	return "{" + strings.Join(s, ",") + "}", nil
}

// JSON is a helper for JSONB columns using json.RawMessage.
type JSON = json.RawMessage
