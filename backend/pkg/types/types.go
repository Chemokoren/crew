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
