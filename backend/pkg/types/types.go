// Package types defines shared domain types used across the application.
// This avoids circular imports between packages like models, jwt, and handlers.
package types

// SystemRole represents a user's role in the AMY MIS system.
// Used in JWT claims, authentication middleware, and RBAC checks.
type SystemRole string

const (
	RoleSystemAdmin SystemRole = "SYSTEM_ADMIN"
	RoleSaccoAdmin  SystemRole = "SACCO_ADMIN"
	RoleCrewUser    SystemRole = "CREW"
	RoleLender      SystemRole = "LENDER"
	RoleInsurer     SystemRole = "INSURER"
)

// ValidRoles returns all valid system roles.
func ValidRoles() []SystemRole {
	return []SystemRole{
		RoleSystemAdmin,
		RoleSaccoAdmin,
		RoleCrewUser,
		RoleLender,
		RoleInsurer,
	}
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
