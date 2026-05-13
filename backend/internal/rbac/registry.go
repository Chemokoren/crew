package rbac

import (
	"sync"

	"github.com/kibsoft/amy-mis/internal/models"
)

// PermissionDefinition describes a single permission for the registry.
type PermissionDefinition struct {
	Key         string
	Module      string
	Description string
	RiskLevel   string // low, medium, high, critical
	Category    string // crud, workflow, financial, compliance, admin, reporting
	DependsOn   []string
}

// Module groups permissions under a named module for registration.
type Module struct {
	Name        string
	Permissions []PermissionDefinition
}

// Registry is a thread-safe in-memory registry of all permission definitions.
type Registry struct {
	mu          sync.RWMutex
	permissions map[string]PermissionDefinition
	modules     map[string][]PermissionDefinition
	order       []string // insertion order for stable iteration
}

var (
	globalRegistry *Registry
	once           sync.Once
)

// Global returns the singleton permission registry.
func Global() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			permissions: make(map[string]PermissionDefinition),
			modules:     make(map[string][]PermissionDefinition),
		}
		globalRegistry.seedAll()
	})
	return globalRegistry
}

// Register adds a module's permissions to the registry.
func (r *Registry) Register(mod Module) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range mod.Permissions {
		p.Module = mod.Name
		if _, exists := r.permissions[p.Key]; !exists {
			r.order = append(r.order, p.Key)
		}
		r.permissions[p.Key] = p
		r.modules[mod.Name] = append(r.modules[mod.Name], p)
	}
}

// GetAll returns all registered permissions in insertion order.
func (r *Registry) GetAll() []PermissionDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]PermissionDefinition, 0, len(r.order))
	for _, key := range r.order {
		result = append(result, r.permissions[key])
	}
	return result
}

// GetByModule returns permissions for a specific module.
func (r *Registry) GetByModule(module string) []PermissionDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modules[module]
}

// GetByKey returns a single permission definition.
func (r *Registry) GetByKey(key string) (PermissionDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.permissions[key]
	return p, ok
}

// ModuleNames returns all registered module names.
func (r *Registry) ModuleNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.modules))
	seen := map[string]bool{}
	for _, key := range r.order {
		mod := r.permissions[key].Module
		if !seen[mod] {
			seen[mod] = true
			names = append(names, mod)
		}
	}
	return names
}

// Count returns total registered permissions.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.permissions)
}

// seedAll registers all AMY permission modules.
func (r *Registry) seedAll() {
	r.Register(Module{Name: "Workers", Permissions: []PermissionDefinition{
		{Key: models.PermWorkersView, Description: "View worker profiles", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermWorkersCreate, Description: "Create new workers", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermWorkersUpdate, Description: "Update worker details", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermWorkersDelete, Description: "Delete workers", RiskLevel: models.RiskHigh, Category: models.CategoryCRUD},
		{Key: models.PermWorkersExport, Description: "Export worker data", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
		{Key: models.PermWorkersImport, Description: "Import workers in bulk", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermWorkersAssign, Description: "Assign workers to tasks", RiskLevel: models.RiskLow, Category: models.CategoryWorkflow},
		{Key: models.PermWorkersArchive, Description: "Archive inactive workers", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermWorkersRestore, Description: "Restore archived workers", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermWorkersVerifyKYC, Description: "Verify worker KYC documents", RiskLevel: models.RiskHigh, Category: models.CategoryCompliance},
		{Key: models.PermWorkersBulkImport, Description: "Bulk import workers from CSV", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermWorkersViewFinancialProfile, Description: "View worker financial profiles", RiskLevel: models.RiskMedium, Category: models.CategoryFinancial},
	}})

	r.Register(Module{Name: "Assignments", Permissions: []PermissionDefinition{
		{Key: models.PermAssignmentsView, Description: "View assignments", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermAssignmentsCreate, Description: "Create assignments", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermAssignmentsUpdate, Description: "Update assignments", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermAssignmentsDelete, Description: "Delete assignments", RiskLevel: models.RiskHigh, Category: models.CategoryCRUD},
		{Key: models.PermAssignmentsExport, Description: "Export assignment data", RiskLevel: models.RiskLow, Category: models.CategoryReporting},
		{Key: models.PermAssignmentsApprove, Description: "Approve assignments", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
		{Key: models.PermAssignmentsReject, Description: "Reject assignments", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
		{Key: models.PermAssignmentsClockIn, Description: "Clock in to assignments", RiskLevel: models.RiskLow, Category: models.CategoryWorkflow},
		{Key: models.PermAssignmentsClockOut, Description: "Clock out from assignments", RiskLevel: models.RiskLow, Category: models.CategoryWorkflow},
		{Key: models.PermAssignmentsBulkAssign, Description: "Bulk assign workers", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
		{Key: models.PermAssignmentsBulkAllocate, Description: "Bulk allocate resources", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
		{Key: models.PermAssignmentsVerifyGPS, Description: "Verify GPS location for assignments", RiskLevel: models.RiskLow, Category: models.CategoryCompliance},
		{Key: models.PermAssignmentsApproveOvertime, Description: "Approve overtime hours", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
		{Key: models.PermAssignmentsArchive, Description: "Archive old assignments", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
	}})

	r.Register(Module{Name: "Earnings", Permissions: []PermissionDefinition{
		{Key: models.PermEarningsView, Description: "View earnings records", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermEarningsCreate, Description: "Create earning entries", RiskLevel: models.RiskMedium, Category: models.CategoryFinancial},
		{Key: models.PermEarningsUpdate, Description: "Update earning records", RiskLevel: models.RiskMedium, Category: models.CategoryFinancial},
		{Key: models.PermEarningsAdjust, Description: "Adjust earning amounts", RiskLevel: models.RiskHigh, Category: models.CategoryFinancial},
		{Key: models.PermEarningsApprove, Description: "Approve earnings for payout", RiskLevel: models.RiskHigh, Category: models.CategoryWorkflow},
		{Key: models.PermEarningsReject, Description: "Reject earnings entries", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
		{Key: models.PermEarningsExport, Description: "Export earnings data", RiskLevel: models.RiskLow, Category: models.CategoryReporting},
		{Key: models.PermEarningsArchive, Description: "Archive earnings records", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
	}})

	r.Register(Module{Name: "Wallet", Permissions: []PermissionDefinition{
		{Key: models.PermWalletView, Description: "View wallet balances", RiskLevel: models.RiskLow, Category: models.CategoryFinancial},
		{Key: models.PermWalletWithdraw, Description: "Withdraw from wallet", RiskLevel: models.RiskCritical, Category: models.CategoryFinancial},
		{Key: models.PermWalletTransfer, Description: "Transfer between wallets", RiskLevel: models.RiskCritical, Category: models.CategoryFinancial},
		{Key: models.PermWalletFundFloat, Description: "Fund organization float", RiskLevel: models.RiskCritical, Category: models.CategoryFinancial},
		{Key: models.PermWalletApprovePayout, Description: "Approve payout requests", RiskLevel: models.RiskCritical, Category: models.CategoryWorkflow},
		{Key: models.PermWalletReconcile, Description: "Reconcile wallet transactions", RiskLevel: models.RiskHigh, Category: models.CategoryFinancial},
		{Key: models.PermWalletReverseTransaction, Description: "Reverse wallet transactions", RiskLevel: models.RiskCritical, Category: models.CategoryFinancial},
		{Key: models.PermWalletViewTransactions, Description: "View transaction history", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermWalletExport, Description: "Export wallet data", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
		{Key: models.PermWalletApproveWithdrawal, Description: "Approve withdrawal requests", RiskLevel: models.RiskCritical, Category: models.CategoryWorkflow},
	}})

	r.Register(Module{Name: "Payroll", Permissions: []PermissionDefinition{
		{Key: models.PermPayrollView, Description: "View payroll runs", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermPayrollCreate, Description: "Create payroll runs", RiskLevel: models.RiskHigh, Category: models.CategoryFinancial},
		{Key: models.PermPayrollRun, Description: "Execute payroll processing", RiskLevel: models.RiskCritical, Category: models.CategoryFinancial},
		{Key: models.PermPayrollApprove, Description: "Approve payroll for disbursement", RiskLevel: models.RiskCritical, Category: models.CategoryWorkflow},
		{Key: models.PermPayrollReject, Description: "Reject payroll runs", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
		{Key: models.PermPayrollProcess, Description: "Process payroll disbursements", RiskLevel: models.RiskCritical, Category: models.CategoryFinancial},
		{Key: models.PermPayrollExport, Description: "Export payroll data", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
		{Key: models.PermPayrollManageSchedules, Description: "Manage pay schedules", RiskLevel: models.RiskMedium, Category: models.CategoryAdmin},
		{Key: models.PermPayrollManagePeriods, Description: "Manage pay periods", RiskLevel: models.RiskMedium, Category: models.CategoryAdmin},
		{Key: models.PermPayrollViewEntries, Description: "View payroll entries", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
	}})

	r.Register(Module{Name: "Compliance", Permissions: []PermissionDefinition{
		{Key: models.PermComplianceView, Description: "View compliance dashboard", RiskLevel: models.RiskLow, Category: models.CategoryCompliance},
		{Key: models.PermComplianceGenerateReports, Description: "Generate compliance reports", RiskLevel: models.RiskMedium, Category: models.CategoryCompliance},
		{Key: models.PermComplianceManageDeductions, Description: "Manage statutory deductions", RiskLevel: models.RiskHigh, Category: models.CategoryCompliance},
		{Key: models.PermComplianceSubmitStatutory, Description: "Submit statutory returns", RiskLevel: models.RiskCritical, Category: models.CategoryCompliance},
		{Key: models.PermComplianceManageRates, Description: "Manage statutory rates", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermComplianceExport, Description: "Export compliance data", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
	}})

	r.Register(Module{Name: "Loans", Permissions: []PermissionDefinition{
		{Key: models.PermLoansView, Description: "View loan applications", RiskLevel: models.RiskLow, Category: models.CategoryFinancial},
		{Key: models.PermLoansApply, Description: "Apply for loans", RiskLevel: models.RiskMedium, Category: models.CategoryFinancial},
		{Key: models.PermLoansApprove, Description: "Approve loan applications", RiskLevel: models.RiskCritical, Category: models.CategoryWorkflow},
		{Key: models.PermLoansReject, Description: "Reject loan applications", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
		{Key: models.PermLoansDisburse, Description: "Disburse approved loans", RiskLevel: models.RiskCritical, Category: models.CategoryFinancial},
		{Key: models.PermLoansExport, Description: "Export loan data", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
		{Key: models.PermLoansManage, Description: "Manage loan configurations", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
	}})

	r.Register(Module{Name: "Insurance", Permissions: []PermissionDefinition{
		{Key: models.PermInsuranceView, Description: "View insurance policies", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermInsuranceEnroll, Description: "Enroll in insurance", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
		{Key: models.PermInsuranceCancel, Description: "Cancel insurance policies", RiskLevel: models.RiskHigh, Category: models.CategoryWorkflow},
		{Key: models.PermInsuranceManagePolicies, Description: "Manage insurance configurations", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermInsuranceExport, Description: "Export insurance data", RiskLevel: models.RiskLow, Category: models.CategoryReporting},
	}})

	r.Register(Module{Name: "Documents", Permissions: []PermissionDefinition{
		{Key: models.PermDocumentsView, Description: "View documents", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermDocumentsUpload, Description: "Upload documents", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermDocumentsDelete, Description: "Delete documents", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermDocumentsVerify, Description: "Verify document authenticity", RiskLevel: models.RiskHigh, Category: models.CategoryCompliance},
		{Key: models.PermDocumentsExport, Description: "Export documents", RiskLevel: models.RiskLow, Category: models.CategoryReporting},
	}})

	r.Register(Module{Name: "Settings", Permissions: []PermissionDefinition{
		{Key: models.PermSettingsView, Description: "View system settings", RiskLevel: models.RiskLow, Category: models.CategoryAdmin},
		{Key: models.PermSettingsUpdate, Description: "Update settings", RiskLevel: models.RiskMedium, Category: models.CategoryAdmin},
		{Key: models.PermSettingsManageTenant, Description: "Manage tenant configuration", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermSettingsManageBilling, Description: "Manage billing settings", RiskLevel: models.RiskCritical, Category: models.CategoryAdmin},
		{Key: models.PermSettingsManageConfig, Description: "Manage advanced config", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
	}})

	r.Register(Module{Name: "Reports", Permissions: []PermissionDefinition{
		{Key: models.PermReportsView, Description: "View reports", RiskLevel: models.RiskLow, Category: models.CategoryReporting},
		{Key: models.PermReportsExport, Description: "Export reports", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
		{Key: models.PermReportsCreateCustom, Description: "Create custom reports", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
		{Key: models.PermReportsSchedule, Description: "Schedule automated reports", RiskLevel: models.RiskMedium, Category: models.CategoryAdmin},
	}})

	r.Register(Module{Name: "Roles", Permissions: []PermissionDefinition{
		{Key: models.PermRolesView, Description: "View roles and permissions", RiskLevel: models.RiskLow, Category: models.CategoryAdmin},
		{Key: models.PermRolesCreate, Description: "Create new roles", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermRolesUpdate, Description: "Update role configurations", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermRolesDelete, Description: "Delete custom roles", RiskLevel: models.RiskCritical, Category: models.CategoryAdmin},
		{Key: models.PermRolesAssign, Description: "Assign roles to users", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermRolesManagePermissions, Description: "Manage role permissions", RiskLevel: models.RiskCritical, Category: models.CategoryAdmin},
		{Key: models.PermRolesViewTemplates, Description: "View role templates", RiskLevel: models.RiskLow, Category: models.CategoryAdmin},
		{Key: models.PermRolesApplyTemplates, Description: "Apply role templates to tenants", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
	}})

	r.Register(Module{Name: "Users", Permissions: []PermissionDefinition{
		{Key: models.PermUsersView, Description: "View user accounts", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermUsersCreate, Description: "Create user accounts", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermUsersUpdate, Description: "Update user profiles", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermUsersDeactivate, Description: "Deactivate user accounts", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermUsersManageRoles, Description: "Manage user role assignments", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermUsersExport, Description: "Export user data", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
	}})

	r.Register(Module{Name: "Audit", Permissions: []PermissionDefinition{
		{Key: models.PermAuditView, Description: "View audit logs", RiskLevel: models.RiskLow, Category: models.CategoryCompliance},
		{Key: models.PermAuditExport, Description: "Export audit logs", RiskLevel: models.RiskMedium, Category: models.CategoryCompliance},
	}})

	r.Register(Module{Name: "Organizations", Permissions: []PermissionDefinition{
		{Key: models.PermOrganizationsView, Description: "View organizations", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermOrganizationsCreate, Description: "Create organizations", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermOrganizationsUpdate, Description: "Update organizations", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermOrganizationsDelete, Description: "Delete organizations", RiskLevel: models.RiskCritical, Category: models.CategoryAdmin},
		{Key: models.PermOrganizationsExport, Description: "Export organization data", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
		{Key: models.PermOrganizationsManageConfig, Description: "Manage organization config", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
	}})

	r.Register(Module{Name: "Vehicles", Permissions: []PermissionDefinition{
		{Key: models.PermVehiclesView, Description: "View vehicles", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermVehiclesCreate, Description: "Register vehicles", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermVehiclesUpdate, Description: "Update vehicle details", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermVehiclesDelete, Description: "Remove vehicles", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermVehiclesExport, Description: "Export vehicle data", RiskLevel: models.RiskLow, Category: models.CategoryReporting},
	}})

	r.Register(Module{Name: "Routes", Permissions: []PermissionDefinition{
		{Key: models.PermRoutesView, Description: "View routes", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermRoutesCreate, Description: "Create routes", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermRoutesUpdate, Description: "Update routes", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermRoutesDelete, Description: "Delete routes", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
	}})

	r.Register(Module{Name: "WorkSites", Permissions: []PermissionDefinition{
		{Key: models.PermWorkSitesView, Description: "View work sites", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermWorkSitesCreate, Description: "Create work sites", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermWorkSitesUpdate, Description: "Update work sites", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermWorkSitesDelete, Description: "Delete work sites", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
	}})

	r.Register(Module{Name: "Notifications", Permissions: []PermissionDefinition{
		{Key: models.PermNotificationsView, Description: "View notifications", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermNotificationsManageTemplates, Description: "Manage notification templates", RiskLevel: models.RiskMedium, Category: models.CategoryAdmin},
		{Key: models.PermNotificationsSend, Description: "Send notifications", RiskLevel: models.RiskMedium, Category: models.CategoryWorkflow},
	}})

	r.Register(Module{Name: "Platform", Permissions: []PermissionDefinition{
		{Key: models.PermPlatformManageOrganizations, Description: "Manage all organizations", RiskLevel: models.RiskCritical, Category: models.CategoryAdmin},
		{Key: models.PermPlatformManageUsers, Description: "Manage all platform users", RiskLevel: models.RiskCritical, Category: models.CategoryAdmin},
		{Key: models.PermPlatformManageFinance, Description: "Manage platform finances", RiskLevel: models.RiskCritical, Category: models.CategoryFinancial},
		{Key: models.PermPlatformViewAnalytics, Description: "View platform analytics", RiskLevel: models.RiskLow, Category: models.CategoryReporting},
		{Key: models.PermPlatformManageCompliance, Description: "Manage platform compliance", RiskLevel: models.RiskCritical, Category: models.CategoryCompliance},
		{Key: models.PermPlatformManageIntegrations, Description: "Manage external integrations", RiskLevel: models.RiskHigh, Category: models.CategoryAdmin},
		{Key: models.PermPlatformManageRoles, Description: "Manage platform-level roles", RiskLevel: models.RiskCritical, Category: models.CategoryAdmin},
		{Key: models.PermPlatformViewAudit, Description: "View platform audit logs", RiskLevel: models.RiskLow, Category: models.CategoryCompliance},
		{Key: models.PermPlatformManageSupport, Description: "Manage support tickets", RiskLevel: models.RiskMedium, Category: models.CategoryAdmin},
		{Key: models.PermPlatformManageSettings, Description: "Manage platform settings", RiskLevel: models.RiskCritical, Category: models.CategoryAdmin},
	}})

	r.Register(Module{Name: "Credit", Permissions: []PermissionDefinition{
		{Key: models.PermCreditView, Description: "View credit scores", RiskLevel: models.RiskLow, Category: models.CategoryFinancial},
		{Key: models.PermCreditScoreCompute, Description: "Trigger credit score computation", RiskLevel: models.RiskMedium, Category: models.CategoryFinancial},
		{Key: models.PermCreditExport, Description: "Export credit data", RiskLevel: models.RiskMedium, Category: models.CategoryReporting},
	}})

	r.Register(Module{Name: "Facilitators", Permissions: []PermissionDefinition{
		{Key: models.PermFacilitatorsView, Description: "View facilitators", RiskLevel: models.RiskLow, Category: models.CategoryCRUD},
		{Key: models.PermFacilitatorsCreate, Description: "Create facilitators", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermFacilitatorsUpdate, Description: "Update facilitators", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
		{Key: models.PermFacilitatorsDelete, Description: "Delete facilitators", RiskLevel: models.RiskMedium, Category: models.CategoryCRUD},
	}})
}
