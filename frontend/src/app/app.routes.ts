import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { guestGuard } from './core/guards/guest.guard';
import { roleGuard } from './core/guards/role.guard';
import { kycGuard } from './core/guards/kyc.guard';
import { platformGuard } from './core/guards/platform.guard';
import { anyPermissionGuard } from './core/guards/permission.guard';

export const routes: Routes = [
  // --- Auth routes (guest-only) ---
  {
    path: 'auth',
    canActivate: [guestGuard],
    loadComponent: () => import('./features/auth/login/login.component').then(m => m.LoginComponent),
  },
  {
    path: 'auth/login',
    canActivate: [guestGuard],
    loadComponent: () => import('./features/auth/login/login.component').then(m => m.LoginComponent),
  },
  {
    path: 'auth/register',
    canActivate: [guestGuard],
    loadComponent: () => import('./features/auth/register/register.component').then(m => m.RegisterComponent),
  },

  // --- Platform Admin routes (separate shell) ---
  {
    path: 'platform',
    canActivate: [authGuard, platformGuard],
    children: [
      { path: '', redirectTo: 'command-center', pathMatch: 'full' },
      {
        path: 'command-center',
        loadComponent: () => import('./features/platform/command-center/command-center.component').then(m => m.CommandCenterComponent),
      },
      {
        path: 'organizations',
        loadComponent: () => import('./features/platform/organizations/platform-organizations.component').then(m => m.PlatformOrganizationsComponent),
      },
      {
        path: 'users',
        loadComponent: () => import('./features/platform/users/platform-users.component').then(m => m.PlatformUsersComponent),
      },
      {
        path: 'support',
        loadComponent: () => import('./features/platform/support/platform-support.component').then(m => m.PlatformSupportComponent),
      },
      {
        path: 'roles',
        canActivate: [anyPermissionGuard('roles.view', 'platform.manage_roles')],
        loadComponent: () => import('./features/platform/roles/platform-roles.component').then(m => m.PlatformRolesComponent),
      },
      {
        path: 'settings',
        loadComponent: () => import('./features/platform/settings/platform-settings.component').then(m => m.PlatformSettingsComponent),
      },
      {
        path: 'notifications',
        loadComponent: () => import('./features/platform/notifications/platform-notifications.component').then(m => m.PlatformNotificationsComponent),
      },
      {
        path: 'integrations',
        loadComponent: () => import('./features/platform/integrations/platform-integrations.component').then(m => m.PlatformIntegrationsComponent),
      },
      {
        path: 'finance',
        loadComponent: () => import('./features/platform/finance/platform-finance.component').then(m => m.PlatformFinanceComponent),
      },
      {
        path: 'reports',
        loadComponent: () => import('./features/platform/reports/platform-reports.component').then(m => m.PlatformReportsComponent),
      },
      {
        path: 'compliance',
        loadComponent: () => import('./features/platform/compliance/platform-compliance.component').then(m => m.PlatformComplianceComponent),
      },
      {
        path: 'documents',
        loadComponent: () => import('./features/platform/documents/platform-documents.component').then(m => m.PlatformDocumentsComponent),
      },
      {
        path: 'team',
        loadComponent: () => import('./features/platform/team/platform-team.component').then(m => m.PlatformTeamComponent),
      },
    ],
  },

  // --- Authenticated routes (Organization shell) ---
  {
    path: '',
    canActivate: [authGuard],
    children: [
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },

      // ── KYC-exempt routes (always accessible) ──
      {
        path: 'profile',
        loadComponent: () => import('./features/auth/profile/profile.component').then(m => m.ProfileComponent),
      },
      {
        path: 'notifications',
        loadComponent: () => import('./features/notifications/notification-list/notification-list.component').then(m => m.NotificationListComponent),
      },
      {
        path: 'notifications/preferences',
        loadComponent: () => import('./features/notifications/notification-preferences/notification-preferences.component').then(m => m.NotificationPreferencesComponent),
      },

      // ── KYC-protected routes (blocked for unverified employees) ──
      {
        path: 'dashboard',
        canActivate: [kycGuard],
        loadComponent: () => import('./features/dashboard/dashboard.component').then(m => m.DashboardComponent),
      },
      {
        path: 'crew',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/crew/crew-list/crew-list.component').then(m => m.CrewListComponent),
      },
      {
        path: 'crew/:id',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/crew/crew-detail/crew-detail.component').then(m => m.CrewDetailComponent),
      },
      {
        path: 'crew/:id/financial-profile',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER', 'LENDER', 'INSURER')],
        loadComponent: () => import('./features/crew/financial-profile/financial-profile.component').then(m => m.FinancialProfileComponent),
      },
      {
        path: 'assignments',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/assignments/assignment-list/assignment-list.component').then(m => m.AssignmentListComponent),
      },
      {
        // MUST be before assignments/:id to prevent 'bulk' being treated as an ID
        path: 'assignments/bulk',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/assignments/bulk-assignment/bulk-assignment.component').then(m => m.BulkAssignmentComponent),
      },
      {
        path: 'assignments/:id',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/assignments/assignment-detail/assignment-detail.component').then(m => m.AssignmentDetailComponent),
      },
      {
        // Legacy alias kept for backward compatibility
        path: 'assignments-bulk',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/assignments/bulk-assignment/bulk-assignment.component').then(m => m.BulkAssignmentComponent),
      },
      {
        path: 'earnings',
        canActivate: [kycGuard],
        loadComponent: () => import('./features/earnings/earnings.component').then(m => m.EarningsComponent),
      },
      {
        path: 'wallets',
        canActivate: [kycGuard],
        loadComponent: () => import('./features/wallets/wallet-dashboard/wallet-dashboard.component').then(m => m.WalletDashboardComponent),
      },
      {
        path: 'employers',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN')],
        loadComponent: () => import('./features/saccos/sacco-list/sacco-list.component').then(m => m.SaccoListComponent),
      },
      {
        path: 'employers/:id',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN')],
        loadComponent: () => import('./features/saccos/sacco-detail/sacco-detail.component').then(m => m.SaccoDetailComponent),
      },
      {
        path: 'vehicles',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/vehicles/vehicle-list/vehicle-list.component').then(m => m.VehicleListComponent),
      },
      {
        path: 'vehicles/:id',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/vehicles/vehicle-detail/vehicle-detail.component').then(m => m.VehicleDetailComponent),
      },
      {
        path: 'routes',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/routes/route-list/route-list.component').then(m => m.RouteListComponent),
      },
      {
        path: 'routes/:id',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/routes/route-detail/route-detail.component').then(m => m.RouteDetailComponent),
      },
      {
        path: 'payroll',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/payroll/payroll-list/payroll-list.component').then(m => m.PayrollListComponent),
      },
      {
        path: 'payroll/:id',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/payroll/payroll-detail/payroll-detail.component').then(m => m.PayrollDetailComponent),
      },
      {
        path: 'pay-schedules',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/payroll/pay-schedule-dashboard/pay-schedule-dashboard.component').then(m => m.PayScheduleDashboardComponent),
      },
      {
        path: 'statutory-rates',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN')],
        loadComponent: () => import('./features/payroll/statutory-rates/statutory-rates.component').then(m => m.StatutoryRatesComponent),
      },
      {
        path: 'loans',
        canActivate: [kycGuard],
        loadComponent: () => import('./features/loans/loan-list/loan-list.component').then(m => m.LoanListComponent),
      },
      {
        path: 'loans/:id',
        canActivate: [kycGuard],
        loadComponent: () => import('./features/loans/loan-detail/loan-detail.component').then(m => m.LoanDetailComponent),
      },
      {
        path: 'credit',
        canActivate: [kycGuard],
        loadComponent: () => import('./features/credit/credit-score/credit-score.component').then(m => m.CreditScoreComponent),
      },
      {
        path: 'insurance',
        canActivate: [kycGuard],
        loadComponent: () => import('./features/insurance/insurance-list/insurance-list.component').then(m => m.InsuranceListComponent),
      },
      {
        path: 'documents',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/documents/document-list/document-list.component').then(m => m.DocumentListComponent),
      },
      {
        path: 'admin',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN')],
        loadComponent: () => import('./features/admin/admin-dashboard/admin-dashboard.component').then(m => m.AdminDashboardComponent),
      },
      {
        path: 'settings/tenant',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/settings/tenant-settings.component').then(m => m.TenantSettingsComponent),
      },
      {
        path: 'work-sites',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/work-sites/work-sites.component').then(m => m.WorkSitesComponent),
      },
      {
        path: 'facilitators',
        canActivate: [kycGuard, roleGuard('SYSTEM_ADMIN', 'EMPLOYER')],
        loadComponent: () => import('./features/facilitators/facilitators.component').then(m => m.FacilitatorsComponent),
      },
    ],
  },
  {
    path: '**',
    loadComponent: () => import('./features/not-found/not-found.component').then(m => m.NotFoundComponent),
  },
];
