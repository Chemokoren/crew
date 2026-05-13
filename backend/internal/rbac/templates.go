package rbac

import "github.com/kibsoft/amy-mis/internal/models"

// RoleTemplateDefinition defines a role template for an industry type.
type RoleTemplateDefinition struct {
	IndustryType string
	RoleName     string
	RoleSlug     string
	Description  string
	Permissions  []string
	IsDefault    bool
	SortOrder    int
}

// GetIndustryRoleTemplates returns all role templates for all industries.
func GetIndustryRoleTemplates() []RoleTemplateDefinition {
	var all []RoleTemplateDefinition
	all = append(all, transportTemplates()...)
	all = append(all, constructionTemplates()...)
	all = append(all, logisticsTemplates()...)
	all = append(all, healthTemplates()...)
	all = append(all, agricultureTemplates()...)
	all = append(all, hospitalityTemplates()...)
	all = append(all, platformTemplates()...)
	return all
}

// GetTemplatesForIndustry returns role templates for a specific industry.
func GetTemplatesForIndustry(industry string) []RoleTemplateDefinition {
	all := GetIndustryRoleTemplates()
	var filtered []RoleTemplateDefinition
	for _, t := range all {
		if t.IndustryType == industry {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// --- shared permission sets (DRY) ---

var viewOnlyPerms = []string{
	models.PermWorkersView, models.PermAssignmentsView, models.PermEarningsView,
	models.PermWalletView, models.PermDocumentsView, models.PermNotificationsView,
}

var supervisorBasePerms = append([]string{
	models.PermAssignmentsCreate, models.PermAssignmentsUpdate, models.PermAssignmentsApprove,
	models.PermEarningsApprove, models.PermWorkersCreate, models.PermWorkersUpdate,
	models.PermReportsView, models.PermReportsExport,
	models.PermWorkersViewFinancialProfile,
}, viewOnlyPerms...)

var adminBasePerms = append([]string{
	models.PermWorkersDelete, models.PermWorkersExport, models.PermWorkersImport,
	models.PermWorkersVerifyKYC, models.PermWorkersBulkImport,
	models.PermAssignmentsDelete, models.PermAssignmentsExport, models.PermAssignmentsBulkAssign,
	models.PermEarningsCreate, models.PermEarningsAdjust, models.PermEarningsExport,
	models.PermWalletFundFloat, models.PermWalletApprovePayout, models.PermWalletReconcile,
	models.PermWalletViewTransactions, models.PermWalletExport,
	models.PermPayrollView, models.PermPayrollCreate, models.PermPayrollRun,
	models.PermPayrollApprove, models.PermPayrollExport, models.PermPayrollManageSchedules,
	models.PermPayrollManagePeriods, models.PermPayrollViewEntries,
	models.PermComplianceView, models.PermComplianceGenerateReports,
	models.PermComplianceManageDeductions,
	models.PermDocumentsUpload, models.PermDocumentsDelete, models.PermDocumentsVerify,
	models.PermDocumentsExport,
	models.PermSettingsView, models.PermSettingsUpdate, models.PermSettingsManageTenant,
	models.PermRolesView, models.PermRolesCreate, models.PermRolesUpdate,
	models.PermRolesAssign, models.PermRolesManagePermissions,
	models.PermUsersView, models.PermUsersCreate, models.PermUsersUpdate,
	models.PermUsersManageRoles,
	models.PermAuditView,
	models.PermReportsCreateCustom,
	models.PermNotificationsManageTemplates, models.PermNotificationsSend,
}, supervisorBasePerms...)

// ── Transport SACCO ─────────────────────────────────────────────────────────

func transportTemplates() []RoleTemplateDefinition {
	return []RoleTemplateDefinition{
		{
			IndustryType: "TRANSPORT", RoleName: "Driver", RoleSlug: "transport-driver",
			Description: "Primary driver — view own assignments and earnings",
			Permissions: viewOnlyPerms, IsDefault: true, SortOrder: 1,
		},
		{
			IndustryType: "TRANSPORT", RoleName: "Conductor", RoleSlug: "transport-conductor",
			Description: "Conductor — view assignments and clock in/out",
			Permissions: append(viewOnlyPerms, models.PermAssignmentsClockIn, models.PermAssignmentsClockOut),
			IsDefault: true, SortOrder: 2,
		},
		{
			IndustryType: "TRANSPORT", RoleName: "Booking Agent", RoleSlug: "transport-booking-agent",
			Description: "Booking agent — create assignments and view reports",
			Permissions: append(viewOnlyPerms,
				models.PermAssignmentsCreate, models.PermAssignmentsUpdate, models.PermReportsView),
			IsDefault: true, SortOrder: 3,
		},
		{
			IndustryType: "TRANSPORT", RoleName: "Route Supervisor", RoleSlug: "transport-route-supervisor",
			Description: "Route supervisor — manage shifts and approve earnings",
			Permissions: append(supervisorBasePerms,
				models.PermAssignmentsClockIn, models.PermAssignmentsClockOut,
				models.PermRoutesView, models.PermVehiclesView),
			IsDefault: true, SortOrder: 4,
		},
		{
			IndustryType: "TRANSPORT", RoleName: "SACCO Admin", RoleSlug: "transport-sacco-admin",
			Description: "Full SACCO administration — all operations",
			Permissions: append(adminBasePerms,
				models.PermRoutesView, models.PermRoutesCreate, models.PermRoutesUpdate, models.PermRoutesDelete,
				models.PermVehiclesView, models.PermVehiclesCreate, models.PermVehiclesUpdate, models.PermVehiclesDelete, models.PermVehiclesExport),
			IsDefault: true, SortOrder: 5,
		},
	}
}

// ── Construction ────────────────────────────────────────────────────────────

func constructionTemplates() []RoleTemplateDefinition {
	return []RoleTemplateDefinition{
		{
			IndustryType: "CONSTRUCTION", RoleName: "General Laborer", RoleSlug: "construction-laborer",
			Description: "General laborer — view own assignments",
			Permissions: viewOnlyPerms, IsDefault: true, SortOrder: 1,
		},
		{
			IndustryType: "CONSTRUCTION", RoleName: "Equipment Operator", RoleSlug: "construction-operator",
			Description: "Equipment operator — view and clock in/out",
			Permissions: append(viewOnlyPerms, models.PermAssignmentsClockIn, models.PermAssignmentsClockOut),
			IsDefault: true, SortOrder: 2,
		},
		{
			IndustryType: "CONSTRUCTION", RoleName: "Site Foreman", RoleSlug: "construction-foreman",
			Description: "Site foreman — manage site workers and approve time",
			Permissions: append(supervisorBasePerms,
				models.PermAssignmentsApproveOvertime, models.PermAssignmentsVerifyGPS,
				models.PermWorkSitesView),
			IsDefault: true, SortOrder: 3,
		},
		{
			IndustryType: "CONSTRUCTION", RoleName: "Site Supervisor", RoleSlug: "construction-supervisor",
			Description: "Site supervisor — full site operations",
			Permissions: append(adminBasePerms,
				models.PermWorkSitesView, models.PermWorkSitesCreate, models.PermWorkSitesUpdate, models.PermWorkSitesDelete),
			IsDefault: true, SortOrder: 4,
		},
	}
}

// ── Logistics ───────────────────────────────────────────────────────────────

func logisticsTemplates() []RoleTemplateDefinition {
	return []RoleTemplateDefinition{
		{
			IndustryType: "LOGISTICS", RoleName: "Rider", RoleSlug: "logistics-rider",
			Description: "Delivery rider — view and complete deliveries",
			Permissions: append(viewOnlyPerms, models.PermAssignmentsClockIn, models.PermAssignmentsClockOut),
			IsDefault: true, SortOrder: 1,
		},
		{
			IndustryType: "LOGISTICS", RoleName: "Dispatcher", RoleSlug: "logistics-dispatcher",
			Description: "Dispatcher — create and assign deliveries",
			Permissions: append(viewOnlyPerms,
				models.PermAssignmentsCreate, models.PermAssignmentsUpdate,
				models.PermAssignmentsBulkAssign, models.PermReportsView),
			IsDefault: true, SortOrder: 2,
		},
		{
			IndustryType: "LOGISTICS", RoleName: "Sorter", RoleSlug: "logistics-sorter",
			Description: "Warehouse sorter — clock in/out to shifts",
			Permissions: append(viewOnlyPerms, models.PermAssignmentsClockIn, models.PermAssignmentsClockOut),
			IsDefault: true, SortOrder: 3,
		},
		{
			IndustryType: "LOGISTICS", RoleName: "Warehouse Supervisor", RoleSlug: "logistics-warehouse-supervisor",
			Description: "Warehouse supervisor — full logistics admin",
			Permissions: append(adminBasePerms,
				models.PermWorkSitesView, models.PermWorkSitesCreate, models.PermWorkSitesUpdate),
			IsDefault: true, SortOrder: 4,
		},
	}
}

// ── Community Health ────────────────────────────────────────────────────────

func healthTemplates() []RoleTemplateDefinition {
	return []RoleTemplateDefinition{
		{
			IndustryType: "HEALTH", RoleName: "Health Promoter", RoleSlug: "health-promoter",
			Description: "Community health promoter — log visits",
			Permissions: append(viewOnlyPerms,
				models.PermAssignmentsClockIn, models.PermAssignmentsClockOut, models.PermAssignmentsVerifyGPS),
			IsDefault: true, SortOrder: 1,
		},
		{
			IndustryType: "HEALTH", RoleName: "Lead Promoter", RoleSlug: "health-lead-promoter",
			Description: "Lead promoter — supervise and verify visits",
			Permissions: supervisorBasePerms, IsDefault: true, SortOrder: 2,
		},
		{
			IndustryType: "HEALTH", RoleName: "Area Coordinator", RoleSlug: "health-area-coordinator",
			Description: "Area coordinator — full health program admin",
			Permissions: adminBasePerms, IsDefault: true, SortOrder: 3,
		},
	}
}

// ── Agriculture ─────────────────────────────────────────────────────────────

func agricultureTemplates() []RoleTemplateDefinition {
	return []RoleTemplateDefinition{
		{
			IndustryType: "AGRICULTURE", RoleName: "Field Worker", RoleSlug: "agriculture-field-worker",
			Description: "Field worker — view own assignments",
			Permissions: viewOnlyPerms, IsDefault: true, SortOrder: 1,
		},
		{
			IndustryType: "AGRICULTURE", RoleName: "Team Leader", RoleSlug: "agriculture-team-leader",
			Description: "Team leader — manage team assignments",
			Permissions: supervisorBasePerms, IsDefault: true, SortOrder: 2,
		},
		{
			IndustryType: "AGRICULTURE", RoleName: "Estate Manager", RoleSlug: "agriculture-estate-manager",
			Description: "Estate manager — full operations",
			Permissions: adminBasePerms, IsDefault: true, SortOrder: 3,
		},
	}
}

// ── Hospitality ─────────────────────────────────────────────────────────────

func hospitalityTemplates() []RoleTemplateDefinition {
	return []RoleTemplateDefinition{
		{
			IndustryType: "HOSPITALITY", RoleName: "Staff Member", RoleSlug: "hospitality-staff",
			Description: "General staff — view shifts and earnings",
			Permissions: append(viewOnlyPerms, models.PermAssignmentsClockIn, models.PermAssignmentsClockOut),
			IsDefault: true, SortOrder: 1,
		},
		{
			IndustryType: "HOSPITALITY", RoleName: "Shift Lead", RoleSlug: "hospitality-shift-lead",
			Description: "Shift lead — manage shift staff",
			Permissions: supervisorBasePerms, IsDefault: true, SortOrder: 2,
		},
		{
			IndustryType: "HOSPITALITY", RoleName: "Venue Manager", RoleSlug: "hospitality-venue-manager",
			Description: "Venue manager — full operations",
			Permissions: adminBasePerms, IsDefault: true, SortOrder: 3,
		},
	}
}

// ── Platform Roles ──────────────────────────────────────────────────────────

func platformTemplates() []RoleTemplateDefinition {
	allPerms := func() []string {
		reg := Global()
		all := reg.GetAll()
		keys := make([]string, len(all))
		for i, p := range all {
			keys[i] = p.Key
		}
		return keys
	}

	return []RoleTemplateDefinition{
		{
			IndustryType: "PLATFORM", RoleName: "Super Admin", RoleSlug: "platform-super-admin",
			Description: "Full platform access — all permissions",
			Permissions: allPerms(), IsDefault: true, SortOrder: 1,
		},
		{
			IndustryType: "PLATFORM", RoleName: "Platform Auditor", RoleSlug: "platform-auditor",
			Description: "Read-only audit access across all tenants",
			Permissions: []string{
				models.PermAuditView, models.PermAuditExport,
				models.PermPlatformViewAudit, models.PermPlatformViewAnalytics,
				models.PermReportsView, models.PermReportsExport,
				models.PermComplianceView, models.PermComplianceGenerateReports, models.PermComplianceExport,
				models.PermWorkersView, models.PermAssignmentsView, models.PermEarningsView,
				models.PermPayrollView, models.PermPayrollViewEntries,
				models.PermWalletView, models.PermWalletViewTransactions,
				models.PermLoansView, models.PermInsuranceView,
				models.PermOrganizationsView, models.PermUsersView, models.PermRolesView,
			},
			IsDefault: true, SortOrder: 2,
		},
		{
			IndustryType: "PLATFORM", RoleName: "Compliance Officer", RoleSlug: "platform-compliance-officer",
			Description: "Compliance management across tenants",
			Permissions: []string{
				models.PermComplianceView, models.PermComplianceGenerateReports,
				models.PermComplianceManageDeductions, models.PermComplianceSubmitStatutory,
				models.PermComplianceManageRates, models.PermComplianceExport,
				models.PermAuditView, models.PermAuditExport,
				models.PermPlatformManageCompliance, models.PermPlatformViewAudit,
				models.PermWorkersView, models.PermWorkersVerifyKYC,
				models.PermDocumentsView, models.PermDocumentsVerify,
				models.PermReportsView, models.PermReportsExport,
			},
			IsDefault: true, SortOrder: 3,
		},
		{
			IndustryType: "PLATFORM", RoleName: "Support Agent", RoleSlug: "platform-support-agent",
			Description: "Customer support — view and assist users",
			Permissions: []string{
				models.PermWorkersView, models.PermUsersView,
				models.PermAssignmentsView, models.PermEarningsView,
				models.PermWalletView, models.PermWalletViewTransactions,
				models.PermLoansView, models.PermInsuranceView,
				models.PermOrganizationsView,
				models.PermDocumentsView,
				models.PermNotificationsView, models.PermNotificationsSend,
				models.PermPlatformManageSupport,
				models.PermAuditView,
			},
			IsDefault: true, SortOrder: 4,
		},
	}
}
