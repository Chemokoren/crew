package models

// Permission represents a system-level operation permission.
type Permission string

const (
	PermCreateAssignment      Permission = "create_assignment"
	PermApproveEarnings       Permission = "approve_earnings"
	PermManagePayroll         Permission = "manage_payroll"
	PermViewFinancialProfiles Permission = "view_financial_profiles"
	PermManageCrew            Permission = "manage_crew"
	PermManageSettings        Permission = "manage_settings"
	PermViewReports           Permission = "view_reports"
	PermManageDocuments       Permission = "manage_documents"
)

// RolePermissions defines default permissions by job type category.
// These are the baseline — organizations can override via RoleConfig (AD-7).
var RolePermissions = map[JobTypeCategory][]Permission{
	JobCategoryPrimary: {
		// Primary workers (drivers, masons, CHVs) — view own data only
	},
	JobCategoryFacilitator: {
		PermCreateAssignment,
		PermViewReports,
	},
	JobCategorySupervisor: {
		PermCreateAssignment,
		PermApproveEarnings,
		PermViewFinancialProfiles,
		PermManageCrew,
		PermViewReports,
	},
	JobCategorySupport: {
		PermManagePayroll,
		PermViewFinancialProfiles,
		PermManageCrew,
		PermManageSettings,
		PermManageDocuments,
		PermViewReports,
	},
}

// HasPermission checks if a job type category has a specific permission.
func HasPermission(category JobTypeCategory, perm Permission) bool {
	perms, ok := RolePermissions[category]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// GetPermissions returns all permissions for a job type category.
func GetPermissions(category JobTypeCategory) []Permission {
	perms, ok := RolePermissions[category]
	if !ok {
		return nil
	}
	return perms
}
