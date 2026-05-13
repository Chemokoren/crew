package models

// ---------------------------------------------------------------------------
// Permission Keys — Centralized permission key constants for all AMY modules.
//
// Naming convention:  module.action  or  module.subresource.action
// All keys are lowercase with dots as separators.
//
// These constants are the source-of-truth used by:
//   - Permission registry (internal/rbac/registry.go)
//   - Middleware checks (internal/middleware/permission.go)
//   - Frontend permission guards / directives
// ---------------------------------------------------------------------------

// ── Workers Module ──────────────────────────────────────────────────────────

const (
	PermWorkersView       = "workers.view"
	PermWorkersCreate     = "workers.create"
	PermWorkersUpdate     = "workers.update"
	PermWorkersDelete     = "workers.delete"
	PermWorkersExport     = "workers.export"
	PermWorkersImport     = "workers.import"
	PermWorkersAssign     = "workers.assign"
	PermWorkersArchive    = "workers.archive"
	PermWorkersRestore    = "workers.restore"
	PermWorkersVerifyKYC  = "workers.verify_kyc"
	PermWorkersBulkImport = "workers.bulk_import"
	PermWorkersViewFinancialProfile = "workers.view_financial_profile"
)

// ── Assignments Module ──────────────────────────────────────────────────────

const (
	PermAssignmentsView           = "assignments.view"
	PermAssignmentsCreate         = "assignments.create"
	PermAssignmentsUpdate         = "assignments.update"
	PermAssignmentsDelete         = "assignments.delete"
	PermAssignmentsExport         = "assignments.export"
	PermAssignmentsApprove        = "assignments.approve"
	PermAssignmentsReject         = "assignments.reject"
	PermAssignmentsClockIn        = "assignments.clock_in"
	PermAssignmentsClockOut       = "assignments.clock_out"
	PermAssignmentsBulkAssign     = "assignments.bulk_assign"
	PermAssignmentsBulkAllocate   = "assignments.bulk_allocate"
	PermAssignmentsVerifyGPS      = "assignments.verify_gps"
	PermAssignmentsApproveOvertime = "assignments.approve_overtime"
	PermAssignmentsArchive        = "assignments.archive"
)

// ── Earnings Module ─────────────────────────────────────────────────────────

const (
	PermEarningsView    = "earnings.view"
	PermEarningsCreate  = "earnings.create"
	PermEarningsUpdate  = "earnings.update"
	PermEarningsAdjust  = "earnings.adjust"
	PermEarningsApprove = "earnings.approve"
	PermEarningsReject  = "earnings.reject"
	PermEarningsExport  = "earnings.export"
	PermEarningsArchive = "earnings.archive"
)

// ── Wallet Module ───────────────────────────────────────────────────────────

const (
	PermWalletView               = "wallet.view"
	PermWalletWithdraw           = "wallet.withdraw"
	PermWalletTransfer           = "wallet.transfer"
	PermWalletFundFloat          = "wallet.fund_float"
	PermWalletApprovePayout      = "wallet.approve_payout"
	PermWalletReconcile          = "wallet.reconcile"
	PermWalletReverseTransaction = "wallet.reverse_transaction"
	PermWalletViewTransactions   = "wallet.view_transactions"
	PermWalletExport             = "wallet.export"
	PermWalletApproveWithdrawal  = "wallet.approve_withdrawal"
)

// ── Payroll Module ──────────────────────────────────────────────────────────

const (
	PermPayrollView            = "payroll.view"
	PermPayrollCreate          = "payroll.create"
	PermPayrollRun             = "payroll.run"
	PermPayrollApprove         = "payroll.approve"
	PermPayrollReject          = "payroll.reject"
	PermPayrollProcess         = "payroll.process"
	PermPayrollExport          = "payroll.export"
	PermPayrollManageSchedules = "payroll.manage_schedules"
	PermPayrollManagePeriods   = "payroll.manage_periods"
	PermPayrollViewEntries     = "payroll.view_entries"
)

// ── Compliance Module ───────────────────────────────────────────────────────

const (
	PermComplianceView              = "compliance.view"
	PermComplianceGenerateReports   = "compliance.generate_reports"
	PermComplianceManageDeductions  = "compliance.manage_deductions"
	PermComplianceSubmitStatutory   = "compliance.submit_statutory"
	PermComplianceManageRates       = "compliance.manage_rates"
	PermComplianceExport            = "compliance.export"
)

// ── Loans Module ────────────────────────────────────────────────────────────

const (
	PermLoansView     = "loans.view"
	PermLoansApply    = "loans.apply"
	PermLoansApprove  = "loans.approve"
	PermLoansReject   = "loans.reject"
	PermLoansDisburse = "loans.disburse"
	PermLoansExport   = "loans.export"
	PermLoansManage   = "loans.manage"
)

// ── Insurance Module ────────────────────────────────────────────────────────

const (
	PermInsuranceView           = "insurance.view"
	PermInsuranceEnroll         = "insurance.enroll"
	PermInsuranceCancel         = "insurance.cancel"
	PermInsuranceManagePolicies = "insurance.manage_policies"
	PermInsuranceExport         = "insurance.export"
)

// ── Documents Module ────────────────────────────────────────────────────────

const (
	PermDocumentsView   = "documents.view"
	PermDocumentsUpload = "documents.upload"
	PermDocumentsDelete = "documents.delete"
	PermDocumentsVerify = "documents.verify"
	PermDocumentsExport = "documents.export"
)

// ── Settings Module ─────────────────────────────────────────────────────────

const (
	PermSettingsView          = "settings.view"
	PermSettingsUpdate        = "settings.update"
	PermSettingsManageTenant  = "settings.manage_tenant"
	PermSettingsManageBilling = "settings.manage_billing"
	PermSettingsManageConfig  = "settings.manage_config"
)

// ── Reports Module ──────────────────────────────────────────────────────────

const (
	PermReportsView         = "reports.view"
	PermReportsExport       = "reports.export"
	PermReportsCreateCustom = "reports.create_custom"
	PermReportsSchedule     = "reports.schedule"
)

// ── Roles & Permissions Module ──────────────────────────────────────────────

const (
	PermRolesView              = "roles.view"
	PermRolesCreate            = "roles.create"
	PermRolesUpdate            = "roles.update"
	PermRolesDelete            = "roles.delete"
	PermRolesAssign            = "roles.assign"
	PermRolesManagePermissions = "roles.manage_permissions"
	PermRolesViewTemplates     = "roles.view_templates"
	PermRolesApplyTemplates    = "roles.apply_templates"
)

// ── Users Module ────────────────────────────────────────────────────────────

const (
	PermUsersView       = "users.view"
	PermUsersCreate     = "users.create"
	PermUsersUpdate     = "users.update"
	PermUsersDeactivate = "users.deactivate"
	PermUsersManageRoles = "users.manage_roles"
	PermUsersExport     = "users.export"
)

// ── Audit Module ────────────────────────────────────────────────────────────

const (
	PermAuditView   = "audit.view"
	PermAuditExport = "audit.export"
)

// ── Organizations Module ────────────────────────────────────────────────────

const (
	PermOrganizationsView   = "organizations.view"
	PermOrganizationsCreate = "organizations.create"
	PermOrganizationsUpdate = "organizations.update"
	PermOrganizationsDelete = "organizations.delete"
	PermOrganizationsExport = "organizations.export"
	PermOrganizationsManageConfig = "organizations.manage_config"
)

// ── Vehicles Module ─────────────────────────────────────────────────────────

const (
	PermVehiclesView   = "vehicles.view"
	PermVehiclesCreate = "vehicles.create"
	PermVehiclesUpdate = "vehicles.update"
	PermVehiclesDelete = "vehicles.delete"
	PermVehiclesExport = "vehicles.export"
)

// ── Routes Module ───────────────────────────────────────────────────────────

const (
	PermRoutesView   = "routes.view"
	PermRoutesCreate = "routes.create"
	PermRoutesUpdate = "routes.update"
	PermRoutesDelete = "routes.delete"
)

// ── Work Sites Module ───────────────────────────────────────────────────────

const (
	PermWorkSitesView   = "work_sites.view"
	PermWorkSitesCreate = "work_sites.create"
	PermWorkSitesUpdate = "work_sites.update"
	PermWorkSitesDelete = "work_sites.delete"
)

// ── Notifications Module ────────────────────────────────────────────────────

const (
	PermNotificationsView            = "notifications.view"
	PermNotificationsManageTemplates = "notifications.manage_templates"
	PermNotificationsSend            = "notifications.send"
)

// ── Platform Module (super-admin only) ──────────────────────────────────────

const (
	PermPlatformManageOrganizations = "platform.manage_organizations"
	PermPlatformManageUsers         = "platform.manage_users"
	PermPlatformManageFinance       = "platform.manage_finance"
	PermPlatformViewAnalytics       = "platform.view_analytics"
	PermPlatformManageCompliance    = "platform.manage_compliance"
	PermPlatformManageIntegrations  = "platform.manage_integrations"
	PermPlatformManageRoles         = "platform.manage_roles"
	PermPlatformViewAudit           = "platform.view_audit"
	PermPlatformManageSupport       = "platform.manage_support"
	PermPlatformManageSettings      = "platform.manage_settings"
)

// ── Credit Module ───────────────────────────────────────────────────────────

const (
	PermCreditView         = "credit.view"
	PermCreditScoreCompute = "credit.score_compute"
	PermCreditExport       = "credit.export"
)

// ── Facilitators Module ─────────────────────────────────────────────────────

const (
	PermFacilitatorsView   = "facilitators.view"
	PermFacilitatorsCreate = "facilitators.create"
	PermFacilitatorsUpdate = "facilitators.update"
	PermFacilitatorsDelete = "facilitators.delete"
)
