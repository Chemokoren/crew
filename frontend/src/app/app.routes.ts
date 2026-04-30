import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { guestGuard } from './core/guards/guest.guard';
import { roleGuard } from './core/guards/role.guard';

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

  // --- Authenticated routes ---
  {
    path: '',
    canActivate: [authGuard],
    children: [
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
      {
        path: 'dashboard',
        loadComponent: () => import('./features/dashboard/dashboard.component').then(m => m.DashboardComponent),
      },
      {
        path: 'profile',
        loadComponent: () => import('./features/auth/profile/profile.component').then(m => m.ProfileComponent),
      },
      {
        path: 'crew',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/crew/crew-list/crew-list.component').then(m => m.CrewListComponent),
      },
      {
        path: 'crew/:id',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/crew/crew-detail/crew-detail.component').then(m => m.CrewDetailComponent),
      },
      {
        path: 'assignments',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/assignments/assignment-list/assignment-list.component').then(m => m.AssignmentListComponent),
      },
      {
        path: 'assignments/:id',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/assignments/assignment-detail/assignment-detail.component').then(m => m.AssignmentDetailComponent),
      },
      {
        path: 'earnings',
        loadComponent: () => import('./features/earnings/earnings.component').then(m => m.EarningsComponent),
      },
      {
        path: 'wallets',
        loadComponent: () => import('./features/wallets/wallet-dashboard/wallet-dashboard.component').then(m => m.WalletDashboardComponent),
      },
      {
        path: 'saccos',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/saccos/sacco-list/sacco-list.component').then(m => m.SaccoListComponent),
      },
      {
        path: 'saccos/:id',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/saccos/sacco-detail/sacco-detail.component').then(m => m.SaccoDetailComponent),
      },
      {
        path: 'vehicles',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/vehicles/vehicle-list/vehicle-list.component').then(m => m.VehicleListComponent),
      },
      {
        path: 'vehicles/:id',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/vehicles/vehicle-detail/vehicle-detail.component').then(m => m.VehicleDetailComponent),
      },
      {
        path: 'routes',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/routes/route-list/route-list.component').then(m => m.RouteListComponent),
      },
      {
        path: 'routes/:id',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/routes/route-detail/route-detail.component').then(m => m.RouteDetailComponent),
      },
      {
        path: 'payroll',
        canActivate: [roleGuard('SYSTEM_ADMIN', 'SACCO_ADMIN')],
        loadComponent: () => import('./features/payroll/payroll-list/payroll-list.component').then(m => m.PayrollListComponent),
      },
      {
        path: 'loans',
        loadComponent: () => import('./features/loans/loan-list/loan-list.component').then(m => m.LoanListComponent),
      },
      {
        path: 'insurance',
        loadComponent: () => import('./features/insurance/insurance-list/insurance-list.component').then(m => m.InsuranceListComponent),
      },
      {
        path: 'notifications',
        loadComponent: () => import('./features/notifications/notification-list/notification-list.component').then(m => m.NotificationListComponent),
      },
      {
        path: 'admin',
        canActivate: [roleGuard('SYSTEM_ADMIN')],
        loadComponent: () => import('./features/admin/admin-dashboard/admin-dashboard.component').then(m => m.AdminDashboardComponent),
      },
    ],
  },
  { path: '**', redirectTo: 'dashboard' },
];
