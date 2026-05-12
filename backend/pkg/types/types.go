// Package types defines shared domain types used across the application.
// This avoids circular imports between packages like models, jwt, and handlers.
package types

// SystemRole represents a user's role in the AMY MIS system.
// Used in JWT claims, authentication middleware, and RBAC checks.
type SystemRole string

const (
	RoleSystemAdmin SystemRole = "SYSTEM_ADMIN"
	RoleEmployer    SystemRole = "EMPLOYER"
	RoleEmployee    SystemRole = "EMPLOYEE"
	RoleLender      SystemRole = "LENDER"
	RoleInsurer     SystemRole = "INSURER"

	// Platform staff roles — separated admin experience
	RolePlatformAdmin     SystemRole = "PLATFORM_ADMIN"
	RolePlatformSupport   SystemRole = "PLATFORM_SUPPORT"
	RolePlatformFinance   SystemRole = "PLATFORM_FINANCE"
	RolePlatformAuditor   SystemRole = "PLATFORM_AUDITOR"
	RolePlatformAssistant SystemRole = "PLATFORM_ASSISTANT"
)

// Backward compatibility aliases — use new names in new code.
var (
	RoleSaccoAdmin = RoleEmployer
	RoleCrewUser   = RoleEmployee
)

// ValidRoles returns all valid system roles.
func ValidRoles() []SystemRole {
	return []SystemRole{
		RoleSystemAdmin,
		RoleEmployer,
		RoleEmployee,
		RoleLender,
		RoleInsurer,
		RolePlatformAdmin,
		RolePlatformSupport,
		RolePlatformFinance,
		RolePlatformAuditor,
		RolePlatformAssistant,
	}
}

// IsPlatformRole returns true if the role is a platform-level staff role.
func (r SystemRole) IsPlatformRole() bool {
	switch r {
	case RoleSystemAdmin, RolePlatformAdmin, RolePlatformSupport,
		RolePlatformFinance, RolePlatformAuditor, RolePlatformAssistant:
		return true
	}
	return false
}

// IsValid checks if the role is a recognized system role.
func (r SystemRole) IsValid() bool {
	for _, valid := range ValidRoles() {
		if r == valid {
			return true
		}
	}
	return false
}
