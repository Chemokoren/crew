package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// IndustryType represents the industry vertical of a tenant/organization.
type IndustryType string

const (
	IndustryTransport    IndustryType = "TRANSPORT"
	IndustryConstruction IndustryType = "CONSTRUCTION"
	IndustryHealth       IndustryType = "HEALTH"
	IndustryLogistics    IndustryType = "LOGISTICS"
	IndustryAgriculture  IndustryType = "AGRICULTURE"
	IndustryHospitality  IndustryType = "HOSPITALITY"
	IndustryGeneral      IndustryType = "GENERAL"
	IndustryCustom       IndustryType = "CUSTOM"
)

// JobTypeCategory classifies a job type within a tenant.
type JobTypeCategory string

const (
	JobCategoryPrimary     JobTypeCategory = "PRIMARY"     // Core workers (driver, mason, CHV)
	JobCategoryFacilitator JobTypeCategory = "FACILITATOR" // Booking agents, touts, recruiters
	JobCategorySupport     JobTypeCategory = "SUPPORT"     // Office staff, clerks, admin
	JobCategorySupervisor  JobTypeCategory = "SUPERVISOR"  // Foremen, team leads, coordinators
)

// PayFrequency defines how often workers are paid.
type PayFrequency string

const (
	PayDaily    PayFrequency = "DAILY"
	PayWeekly   PayFrequency = "WEEKLY"
	PayBiWeekly PayFrequency = "BI_WEEKLY"
	PayMonthly  PayFrequency = "MONTHLY"
)

// TenantJobType defines a configurable worker role within a tenant organization.
// Replaces the hardcoded CrewRole enum (DRIVER/CONDUCTOR/RIDER/OTHER).
type TenantJobType struct {
	ID             uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID       `json:"organization_id" gorm:"column:sacco_id;type:uuid;not null;index"`
	Code           string          `json:"code" gorm:"type:varchar(50);not null"`
	DisplayName    string          `json:"display_name" gorm:"type:varchar(100);not null"`
	Category       JobTypeCategory `json:"category" gorm:"type:varchar(30);not null;default:'PRIMARY'"`
	IsActive       bool            `json:"is_active" gorm:"default:true"`
	SortOrder      int             `json:"sort_order" gorm:"default:0"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`

	// Relations
	Organization Organization `json:"-" gorm:"foreignKey:OrganizationID"`
}

func (TenantJobType) TableName() string { return "tenant_job_types" }

// IsFacilitator returns true if this job type is a facilitator role.
func (j TenantJobType) IsFacilitator() bool {
	return j.Category == JobCategoryFacilitator
}

// PaySchedule defines a pay frequency and timing configuration for a tenant.
type PaySchedule struct {
	ID             uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID    `json:"organization_id" gorm:"column:sacco_id;type:uuid;not null;index"`
	Name           string       `json:"name" gorm:"type:varchar(100);not null"`
	Frequency      PayFrequency `json:"frequency" gorm:"type:varchar(20);not null"`
	PayDay         *int         `json:"pay_day,omitempty"`
	CutoffHour     int          `json:"cutoff_hour" gorm:"default:17"`
	IsDefault      bool         `json:"is_default" gorm:"default:false"`
	IsActive       bool         `json:"is_active" gorm:"default:true"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`

	// Relations
	Organization Organization `json:"-" gorm:"foreignKey:OrganizationID"`
}

func (PaySchedule) TableName() string { return "pay_schedules" }

// TenantConfig holds flexible, tenant-level configuration as a typed wrapper over JSONB.
type TenantConfig struct {
	// CreditScoringWeights overrides default credit scoring factor weights.
	CreditScoringWeights map[string]float64 `json:"credit_scoring_weights,omitempty"`
	// StatutoryExemptions lists job type codes exempt from statutory deductions.
	StatutoryExemptions []string `json:"statutory_exemptions,omitempty"`
	// UILabels overrides default UI labels (e.g., "Vehicle" → "Work Site").
	UILabels map[string]string `json:"ui_labels,omitempty"`
	// Features toggles optional features per tenant.
	Features map[string]bool `json:"features,omitempty"`

	// --- KYC Gating (tenant-configurable) ---

	// KYCRequired controls whether KYC verification is enforced for members.
	KYCRequired bool `json:"kyc_required,omitempty"`
	// KYCRestrictedActions lists action codes that require verified KYC.
	// Examples: "WALLET_WITHDRAW", "WALLET_TRANSFER", "BILL_PAY", "LOAN_APPLY", "PAYOUT"
	KYCRestrictedActions []string `json:"kyc_restricted_actions,omitempty"`
	// KYCDocumentTypes lists acceptable KYC document types.
	// Defaults to ["NATIONAL_ID"] if empty and KYCRequired is true.
	KYCDocumentTypes []string `json:"kyc_document_types,omitempty"`
	// KYCVerificationMode controls how users prove identity.
	//   "UPLOAD" (default) — user uploads front + back photos of National ID.
	//   "MANUAL"           — user enters ID number + serial number for IPRS lookup.
	// The non-default mode is always shown as a fallback option in the UI.
	KYCVerificationMode string `json:"kyc_verification_mode,omitempty"`

	// --- Float Top-Up Verification ---

	// TopUpVerificationMode controls how bank/card top-up references are verified.
	//   "API"    — verify reference via bank API integration (auto-approve on success, reject on failure)
	//   "MANUAL" — all bank/card top-ups require manual admin approval
	//   "HYBRID" (default) — try API verification first; if API unavailable or inconclusive, fall back to manual approval
	TopUpVerificationMode string `json:"topup_verification_mode,omitempty"`

	// --- Float Top-Up Method Availability ---

	// AllowedTopUpMethods controls which top-up method categories are available for this tenant.
	// Valid values: "mobile_money", "bank", "card"
	// When empty or nil, all methods are allowed (backward compatible).
	AllowedTopUpMethods []string `json:"allowed_topup_methods,omitempty"`

	// AllowedTopUpChannels provides fine-grained control over individual payment channels.
	// Valid values: "mpesa", "airtel", "tkash", "kcb", "equity", "coop", "rtgs", "visa", "mastercard"
	// When empty or nil, all channels within allowed methods are available (backward compatible).
	// If populated, only these specific channels appear in the wallet top-up UI.
	AllowedTopUpChannels []string `json:"allowed_topup_channels,omitempty"`
}

// KYC verification mode constants.
const (
	KYCModeUpload = "UPLOAD"
	KYCModeManual = "MANUAL"
)

// Top-up verification mode constants.
const (
	TopUpVerifyAPI    = "API"
	TopUpVerifyManual = "MANUAL"
	TopUpVerifyHybrid = "HYBRID"
)

// ResolvedKYCMode returns the effective verification mode, defaulting to UPLOAD.
func (tc *TenantConfig) ResolvedKYCMode() string {
	if tc.KYCVerificationMode == KYCModeManual {
		return KYCModeManual
	}
	return KYCModeUpload
}

// ResolvedTopUpVerificationMode returns the effective top-up verification mode.
// Defaults to HYBRID if not set.
func (tc *TenantConfig) ResolvedTopUpVerificationMode() string {
	switch tc.TopUpVerificationMode {
	case TopUpVerifyAPI:
		return TopUpVerifyAPI
	case TopUpVerifyManual:
		return TopUpVerifyManual
	default:
		return TopUpVerifyHybrid
	}
}

// AllTopUpMethods is the complete set of supported top-up method categories.
var AllTopUpMethods = []string{"mobile_money", "bank", "card"}

// AllTopUpChannels is the complete set of supported individual payment channels.
var AllTopUpChannels = []string{
	"mpesa", "airtel", "tkash",          // mobile_money
	"kcb", "equity", "coop", "rtgs",     // bank
	"visa", "mastercard",                // card
}

// ChannelToMethod maps each individual channel to its parent method category.
var ChannelToMethod = map[string]string{
	"mpesa": "mobile_money", "airtel": "mobile_money", "tkash": "mobile_money",
	"kcb": "bank", "equity": "bank", "coop": "bank", "rtgs": "bank",
	"visa": "card", "mastercard": "card",
}

// ResolvedAllowedTopUpMethods returns the effective set of allowed top-up methods.
// Defaults to all methods if not configured.
func (tc *TenantConfig) ResolvedAllowedTopUpMethods() []string {
	if len(tc.AllowedTopUpMethods) == 0 {
		return AllTopUpMethods
	}
	return tc.AllowedTopUpMethods
}

// IsTopUpMethodAllowed checks whether a given top-up method category is permitted.
func (tc *TenantConfig) IsTopUpMethodAllowed(method string) bool {
	for _, m := range tc.ResolvedAllowedTopUpMethods() {
		if m == method {
			return true
		}
	}
	return false
}

// IsTopUpChannelAllowed checks whether a specific payment channel (provider) is permitted.
// It first checks the method-level gate, then applies channel-level filtering if configured.
func (tc *TenantConfig) IsTopUpChannelAllowed(channel string) bool {
	// 1. Check the parent method category is allowed
	parentMethod, ok := ChannelToMethod[channel]
	if !ok {
		return false // unknown channel
	}
	if !tc.IsTopUpMethodAllowed(parentMethod) {
		return false
	}
	// 2. If no channel-level config, allow all channels within the method
	if len(tc.AllowedTopUpChannels) == 0 {
		return true
	}
	// 3. Check specific channel
	for _, c := range tc.AllowedTopUpChannels {
		if c == channel {
			return true
		}
	}
	return false
}

// DefaultKYCRestrictedActions returns the default set of actions restricted when KYC is not verified.
// By default everything on the employee side is restricted until KYC is verified.
func DefaultKYCRestrictedActions() []string {
	return []string{
		"WALLET_WITHDRAW",
		"WALLET_TRANSFER",
		"BILL_PAY",
		"LOAN_APPLY",
		"PAYOUT",
		"INSURANCE_ENROLL",
		"ASSIGNMENT_ACCEPT",
		"PROFILE_EDIT",
		"DOCUMENT_UPLOAD",
		"CREDIT_SCORE_VIEW",
	}
}

// IsActionKYCRestricted checks whether a given action is restricted for unverified users under this tenant config.
func (tc *TenantConfig) IsActionKYCRestricted(action string) bool {
	if !tc.KYCRequired {
		return false
	}
	actions := tc.KYCRestrictedActions
	if len(actions) == 0 {
		actions = DefaultKYCRestrictedActions()
	}
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

// Scan implements sql.Scanner for GORM JSONB deserialization.
func (tc *TenantConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(value.(string)), tc)
	}
	return json.Unmarshal(bytes, tc)
}

// Value implements driver.Valuer for GORM JSONB serialization.
func (tc TenantConfig) Value() (interface{}, error) {
	return json.Marshal(tc)
}
