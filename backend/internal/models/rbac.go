package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Role — a named set of permissions, scoped to a tenant or global.
// ---------------------------------------------------------------------------

// Role represents an RBAC role in the system.
// Roles can be global (TenantID == nil) or tenant-specific.
// System roles cannot be deleted; template roles can be cloned.
type Role struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string          `json:"name" gorm:"type:varchar(120);not null" binding:"required"`
	Slug         string          `json:"slug" gorm:"type:varchar(120);not null"`
	Description  string          `json:"description" gorm:"type:text;not null;default:''"`
	TenantID     *uuid.UUID      `json:"tenant_id,omitempty" gorm:"type:uuid;index"`
	IndustryType string          `json:"industry_type" gorm:"type:varchar(30);not null;default:''"`
	IsSystem     bool            `json:"is_system" gorm:"not null;default:false"`
	IsTemplate   bool            `json:"is_template" gorm:"not null;default:false"`
	IsActive     bool            `json:"is_active" gorm:"not null;default:true"`
	ParentRoleID *uuid.UUID      `json:"parent_role_id,omitempty" gorm:"type:uuid"`
	Metadata     json.RawMessage `json:"metadata" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedBy    *uuid.UUID      `json:"created_by,omitempty" gorm:"type:uuid"`
	UpdatedBy    *uuid.UUID      `json:"updated_by,omitempty" gorm:"type:uuid"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    gorm.DeletedAt  `json:"-" gorm:"index"`

	// Relations (loaded via Preload)
	Permissions []PermissionDef `json:"permissions,omitempty" gorm:"many2many:role_permissions;joinForeignKey:role_id;joinReferences:permission_id"`
	ParentRole  *Role           `json:"-" gorm:"foreignKey:ParentRoleID"`
}

func (Role) TableName() string { return "roles" }

// ---------------------------------------------------------------------------
// PermissionDef — a single permission registered in the system.
// ---------------------------------------------------------------------------

// PermissionDef represents a permission definition stored in the database.
// Permissions are seeded from the Go permission registry.
type PermissionDef struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Key         string          `json:"key" gorm:"type:varchar(120);uniqueIndex;not null"`
	Module      string          `json:"module" gorm:"type:varchar(60);not null"`
	Description string          `json:"description" gorm:"type:text;not null;default:''"`
	RiskLevel   string          `json:"risk_level" gorm:"type:varchar(20);not null;default:'low'"`
	Category    string          `json:"category" gorm:"type:varchar(60);not null;default:''"`
	IsSystem    bool            `json:"is_system" gorm:"not null;default:true"`
	DependsOn   StringArray     `json:"depends_on" gorm:"type:text[];default:'{}'"`
	Metadata    json.RawMessage `json:"metadata" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (PermissionDef) TableName() string { return "permissions" }

// ---------------------------------------------------------------------------
// RolePermission — join table between roles and permissions.
// ---------------------------------------------------------------------------

// RolePermission maps a permission to a role (many-to-many join).
type RolePermission struct {
	RoleID       uuid.UUID  `json:"role_id" gorm:"type:uuid;primaryKey"`
	PermissionID uuid.UUID  `json:"permission_id" gorm:"type:uuid;primaryKey"`
	GrantedBy    *uuid.UUID `json:"granted_by,omitempty" gorm:"type:uuid"`
	GrantedAt    time.Time  `json:"granted_at" gorm:"not null;default:now()"`
}

func (RolePermission) TableName() string { return "role_permissions" }

// ---------------------------------------------------------------------------
// UserRole — assigns a role to a user within a tenant context.
// ---------------------------------------------------------------------------

// UserRole represents a user-to-role assignment, optionally scoped to a tenant.
// Supports temporary assignments via ExpiresAt.
type UserRole struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	RoleID     uuid.UUID  `json:"role_id" gorm:"type:uuid;not null;index"`
	TenantID   *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid;index"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty" gorm:"type:uuid"`
	AssignedAt time.Time  `json:"assigned_at" gorm:"not null;default:now()"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	IsActive   bool       `json:"is_active" gorm:"not null;default:true"`

	// Relations
	User   User         `json:"-" gorm:"foreignKey:UserID"`
	Role   Role         `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	Tenant *Organization `json:"-" gorm:"foreignKey:TenantID"`
}

func (UserRole) TableName() string { return "user_roles" }

// IsExpired returns true if the assignment has an expiry date that has passed.
func (ur *UserRole) IsExpired() bool {
	if ur.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ur.ExpiresAt)
}

// IsEffective returns true if the assignment is active and not expired.
func (ur *UserRole) IsEffective() bool {
	return ur.IsActive && !ur.IsExpired()
}

// ---------------------------------------------------------------------------
// Policy — dynamic policy-based access control conditions.
// ---------------------------------------------------------------------------

// PolicyEffect determines whether a policy allows or denies access.
type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "ALLOW"
	PolicyEffectDeny  PolicyEffect = "DENY"
)

// Policy represents a dynamic access control policy.
// Policies add conditional constraints on top of RBAC permissions.
// Example: wallet.approve_payout DENY if amount > 100000 AND !mfa_verified.
type Policy struct {
	ID            uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name          string          `json:"name" gorm:"type:varchar(120);not null" binding:"required"`
	Description   string          `json:"description" gorm:"type:text;not null;default:''"`
	PermissionKey string          `json:"permission_key" gorm:"type:varchar(120);not null;index"`
	Conditions    json.RawMessage `json:"conditions" gorm:"type:jsonb;not null;default:'{}'"`
	Effect        PolicyEffect    `json:"effect" gorm:"type:varchar(10);not null;default:'DENY'"`
	IsActive      bool            `json:"is_active" gorm:"not null;default:true"`
	Priority      int             `json:"priority" gorm:"not null;default:0"`
	TenantID      *uuid.UUID      `json:"tenant_id,omitempty" gorm:"type:uuid;index"`
	CreatedBy     *uuid.UUID      `json:"created_by,omitempty" gorm:"type:uuid"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func (Policy) TableName() string { return "policies" }

// PolicyConditions represents the parsed conditions JSON for a policy.
type PolicyConditions struct {
	// TimeRange restricts access to specific hours (24h format, tenant TZ).
	TimeRange *TimeRangeCondition `json:"time_range,omitempty"`
	// MFARequired forces MFA verification before allowing this action.
	MFARequired *bool `json:"mfa_required,omitempty"`
	// MaxAmount sets an upper limit on monetary amounts (in cents).
	MaxAmount *int64 `json:"max_amount,omitempty"`
	// RequiredRiskLevel only triggers when the action risk is >= this level.
	RequiredRiskLevel string `json:"required_risk_level,omitempty"`
	// IPAllowList restricts to specific IP addresses/CIDRs.
	IPAllowList []string `json:"ip_allow_list,omitempty"`
}

// TimeRangeCondition restricts access to a time window.
type TimeRangeCondition struct {
	StartHour int    `json:"start_hour"` // 0-23
	EndHour   int    `json:"end_hour"`   // 0-23
	Timezone  string `json:"timezone"`   // e.g. "Africa/Nairobi"
	DaysOfWeek []int `json:"days_of_week,omitempty"` // 0=Sun, 6=Sat
}

// ---------------------------------------------------------------------------
// RoleTemplate — pre-built industry role templates for seeding.
// ---------------------------------------------------------------------------

// RoleTemplate defines a pre-configured role for an industry type.
// Templates are seeded at startup and can be applied to new tenants.
type RoleTemplate struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	IndustryType string          `json:"industry_type" gorm:"type:varchar(30);not null;index"`
	RoleName     string          `json:"role_name" gorm:"type:varchar(120);not null"`
	RoleSlug     string          `json:"role_slug" gorm:"type:varchar(120);not null"`
	Description  string          `json:"description" gorm:"type:text;not null;default:''"`
	Permissions  json.RawMessage `json:"permissions" gorm:"type:jsonb;not null;default:'[]'"`
	IsDefault    bool            `json:"is_default" gorm:"not null;default:false"`
	SortOrder    int             `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (RoleTemplate) TableName() string { return "role_templates" }

// GetPermissionKeys parses the Permissions JSON array into a string slice.
func (rt *RoleTemplate) GetPermissionKeys() ([]string, error) {
	var keys []string
	if len(rt.Permissions) == 0 {
		return keys, nil
	}
	if err := json.Unmarshal(rt.Permissions, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

// SetPermissionKeys serializes a string slice into the Permissions JSON field.
func (rt *RoleTemplate) SetPermissionKeys(keys []string) error {
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	rt.Permissions = data
	return nil
}

// ---------------------------------------------------------------------------
// StringArray — PostgreSQL text[] support for GORM.
// ---------------------------------------------------------------------------

// StringArray implements Scanner/Valuer for PostgreSQL text[] columns.
type StringArray []string

// Scan implements sql.Scanner for PostgreSQL text[].
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = StringArray{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			*a = StringArray{}
			return nil
		}
		bytes = []byte(str)
	}
	return json.Unmarshal(bytes, a)
}

// Value implements driver.Valuer for PostgreSQL text[].
func (a StringArray) Value() (interface{}, error) {
	if a == nil {
		return "{}", nil
	}
	return json.Marshal(a)
}

// ---------------------------------------------------------------------------
// RiskLevel constants for permission classification.
// ---------------------------------------------------------------------------

const (
	RiskLow      = "low"
	RiskMedium   = "medium"
	RiskHigh     = "high"
	RiskCritical = "critical"
)

// ---------------------------------------------------------------------------
// Permission category constants for grouping.
// ---------------------------------------------------------------------------

const (
	CategoryCRUD       = "crud"
	CategoryWorkflow   = "workflow"
	CategoryFinancial  = "financial"
	CategoryCompliance = "compliance"
	CategoryAdmin      = "admin"
	CategoryReporting  = "reporting"
)
