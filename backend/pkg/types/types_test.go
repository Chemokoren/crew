package types

import "testing"

func TestSystemRoleIsValid(t *testing.T) {
	tests := []struct {
		role  SystemRole
		valid bool
	}{
		{RoleSystemAdmin, true},
		{RoleSaccoAdmin, true},
		{RoleCrewUser, true},
		{RoleLender, true},
		{RoleInsurer, true},
		{"UNKNOWN", false},
		{"", false},
		{"admin", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.valid {
				t.Errorf("SystemRole(%q).IsValid() = %v, want %v", tt.role, got, tt.valid)
			}
		})
	}
}

func TestValidRoles(t *testing.T) {
	roles := ValidRoles()
	if len(roles) != 5 {
		t.Errorf("ValidRoles() returned %d roles, want 5", len(roles))
	}
}
