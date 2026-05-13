import { Component, inject, signal, model, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive, Router } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';
import { OrgContextService } from '../../../core/services/org-context.service';
import { NotificationStateService } from '../../../core/services/notification-state.service';

interface NavItem {
  label: string;
  icon: string;
  route: string;
  roles?: string[];
  section?: string;
  /** Feature key for industry-based visibility filtering. */
  feature?: string;
}

/** Routes exempt from KYC blocking */
const KYC_EXEMPT_ROUTES = ['/profile', '/notifications'];

@Component({
  selector: 'app-sidebar',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive],
  template: `
    <!-- Mobile backdrop -->
    @if (mobileOpen()) {
      <div class="sidebar-backdrop" (click)="closeMobile()"></div>
    }

    <aside class="sidebar" [class.collapsed]="collapsed()" [class.mobile-open]="mobileOpen()">
      <div class="sidebar-brand">
        <div class="brand-logo">
          <img src="logo.png" alt="AMY MIS Logo" class="dynamic-logo">
        </div>
        <button class="collapse-btn desktop-only" (click)="toggle()" id="sidebar-toggle">
          <span class="material-icons-round">{{ collapsed() ? 'chevron_right' : 'chevron_left' }}</span>
        </button>
        <button class="collapse-btn mobile-only" (click)="closeMobile()" id="sidebar-close-mobile">
          <span class="material-icons-round">close</span>
        </button>
      </div>

      <!-- KYC blocked banner -->
      @if (auth.isKycBlocked() && !collapsed()) {
        <div class="kyc-banner" (click)="navigateTo('/profile')">
          <span class="material-icons-round kyc-banner-icon">gpp_bad</span>
          <div class="kyc-banner-text">
            <strong>KYC Not Verified</strong>
            <span>Complete verification to unlock all features</span>
          </div>
        </div>
      }

      <nav class="sidebar-nav">
        @for (item of filteredNavItems(); track item.route; let i = $index) {
          @if (item.section && (i === 0 || filteredNavItems()[i - 1]?.section !== item.section)) {
            @if (!collapsed()) {
              <div class="nav-section-label">{{ item.section }}</div>
            } @else {
              <div class="nav-divider"></div>
            }
          }
          @if (isItemLocked(item.route)) {
            <div class="nav-item nav-item-locked" (click)="onLockedClick()" [attr.id]="'nav-' + item.label.toLowerCase().replace(' ', '-')">
              <span class="material-icons-round nav-icon">{{ item.icon }}</span>
              @if (!collapsed()) {
                <span class="nav-label">{{ item.label }}</span>
                <span class="material-icons-round nav-lock-icon">lock</span>
              }
            </div>
          } @else {
            <a class="nav-item" [routerLink]="item.route" routerLinkActive="active"
               [attr.id]="'nav-' + item.label.toLowerCase().replace(' ', '-')"
               (click)="onNavClick()">
              <span class="material-icons-round nav-icon">{{ item.icon }}</span>
              @if (!collapsed()) {
                <span class="nav-label">{{ item.label }}</span>
                @if (item.label === 'Notifications' && notifState.unreadCount() > 0) {
                  <span class="nav-badge">{{ notifState.unreadCount() > 99 ? '99+' : notifState.unreadCount() }}</span>
                }
              }
            </a>
          }
        }
      </nav>

      <!-- Platform switch for admin users -->
      @if (!collapsed() && auth.isPlatformUser()) {
        <div class="sidebar-platform-switch">
          <a routerLink="/platform" class="platform-switch-link" id="sidebar-platform-switch">
            <span class="material-icons-round" style="font-size:18px;">hub</span>
            <span>Platform Admin</span>
            <span class="material-icons-round" style="font-size:14px;margin-left:auto;opacity:0.4;">arrow_forward</span>
          </a>
        </div>
      }

      <div class="sidebar-footer">
        @if (!collapsed()) {
          <div class="sidebar-user-badge">
            <div class="sidebar-user-avatar">{{ userInitials() }}</div>
            <div class="sidebar-user-info">
              <span class="sidebar-user-phone">{{ auth.currentUser()?.phone }}</span>
              <span class="sidebar-user-role">{{ formatRole(auth.currentUser()?.system_role) }}</span>
            </div>
          </div>
        }
        <button class="nav-item logout-btn" (click)="logout()" id="sidebar-logout">
          <span class="material-icons-round nav-icon">logout</span>
          @if (!collapsed()) {
            <span class="nav-label">Logout</span>
          }
        </button>
      </div>
    </aside>
  `,
  styles: [`
    .sidebar-backdrop {
      display: none;
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.5);
      backdrop-filter: blur(4px);
      z-index: calc(var(--z-sidebar) - 1);
      animation: fadeIn 200ms ease-out;
    }

    .sidebar {
      position: fixed;
      left: 0;
      top: 0;
      bottom: 0;
      width: var(--sidebar-width);
      background: var(--color-bg-sidebar);
      border-right: 1px solid var(--color-border);
      display: flex;
      flex-direction: column;
      z-index: var(--z-sidebar);
      transition: width var(--transition-base), transform var(--transition-base);
      overflow: hidden;
    }

    .sidebar.collapsed {
      width: var(--sidebar-collapsed-width);
    }

    .sidebar-brand {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: var(--space-md);
      border-bottom: 1px solid var(--color-border);
      height: var(--topbar-height);
      flex-shrink: 0;
    }

    .brand-logo {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
    }

    .logo-icon {
      font-size: 24px;
      width: 36px;
      height: 36px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--gradient-accent);
      border-radius: var(--radius-md);
      flex-shrink: 0;
    }

    .logo-text {
      font-family: var(--font-heading);
      font-size: 1.25rem;
      font-weight: 800;
      color: var(--color-text-primary);
      white-space: nowrap;
    }

    .logo-accent {
      background: var(--gradient-accent);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }

    .collapse-btn {
      width: 28px;
      height: 28px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: none;
      border: 1px solid var(--color-border);
      border-radius: var(--radius-sm);
      color: var(--color-text-muted);
      cursor: pointer;
      transition: all var(--transition-fast);

      &:hover {
        color: var(--color-text-primary);
        border-color: var(--color-border-hover);
      }

      .material-icons-round { font-size: 16px; }
    }

    .mobile-only { display: none; }

    /* ─── KYC Warning Banner ─── */
    .kyc-banner {
      display: flex;
      align-items: center;
      gap: 10px;
      margin: var(--space-sm);
      padding: 10px 12px;
      border-radius: var(--radius-md);
      background: rgba(245, 158, 11, 0.1);
      border: 1px solid rgba(245, 158, 11, 0.25);
      cursor: pointer;
      transition: all var(--transition-fast);
      flex-shrink: 0;
    }
    .kyc-banner:hover {
      background: rgba(245, 158, 11, 0.16);
      border-color: rgba(245, 158, 11, 0.4);
    }
    .kyc-banner-icon {
      font-size: 22px;
      color: #f59e0b;
      flex-shrink: 0;
    }
    .kyc-banner-text {
      display: flex;
      flex-direction: column;
      gap: 1px;
      overflow: hidden;
    }
    .kyc-banner-text strong {
      font-size: 0.75rem;
      font-weight: 700;
      color: #f59e0b;
    }
    .kyc-banner-text span {
      font-size: 0.6875rem;
      color: var(--color-text-muted);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .sidebar-nav {
      flex: 1;
      padding: var(--space-sm);
      overflow-y: auto;
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .nav-section-label {
      font-size: 0.6875rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--color-text-muted);
      padding: 16px 12px 6px;
      white-space: nowrap;
    }

    .nav-divider {
      height: 1px;
      background: var(--color-border);
      margin: 8px 12px;
    }

    .nav-item {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      padding: 10px 12px;
      border-radius: var(--radius-md);
      color: var(--color-text-secondary);
      text-decoration: none;
      font-size: 0.875rem;
      font-weight: 500;
      cursor: pointer;
      transition: all var(--transition-fast);
      white-space: nowrap;
      border: none;
      background: none;
      width: 100%;
      text-align: left;

      &:hover {
        background: rgba(255, 255, 255, 0.04);
        color: var(--color-text-primary);
      }

      &.active {
        background: var(--gradient-accent-soft);
        color: var(--color-accent);

        .nav-icon { color: var(--color-accent); }
      }
    }

    /* ─── Locked nav item (KYC blocked) ─── */
    .nav-item-locked {
      opacity: 0.38;
      cursor: not-allowed;
      user-select: none;

      &:hover {
        background: none;
        color: var(--color-text-secondary);
      }
    }
    .nav-lock-icon {
      margin-left: auto;
      font-size: 14px;
      color: var(--color-text-muted);
      opacity: 0.6;
    }

    .nav-icon {
      font-size: 20px;
      width: 24px;
      flex-shrink: 0;
    }

    .nav-badge {
      margin-left: auto;
      min-width: 18px;
      height: 18px;
      padding: 0 5px;
      border-radius: 9px;
      background: var(--color-accent);
      color: #fff;
      font-size: 0.625rem;
      font-weight: 700;
      display: flex;
      align-items: center;
      justify-content: center;
      line-height: 1;
    }

    .sidebar-footer {
      padding: var(--space-sm);
      border-top: 1px solid var(--color-border);
      flex-shrink: 0;
    }

    .sidebar-user-badge {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      padding: 8px 12px;
      margin-bottom: 4px;
      border-radius: var(--radius-md);
      background: rgba(255, 255, 255, 0.02);
    }

    .sidebar-user-avatar {
      width: 32px;
      height: 32px;
      border-radius: var(--radius-md);
      background: var(--gradient-accent);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.75rem;
      font-weight: 700;
      color: var(--color-text-inverse);
      flex-shrink: 0;
    }

    .sidebar-user-info {
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }

    .sidebar-user-phone {
      font-size: 0.8125rem;
      font-weight: 600;
      color: var(--color-text-primary);
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .sidebar-user-role {
      font-size: 0.6875rem;
      color: var(--color-text-muted);
      text-transform: capitalize;
    }

    .logout-btn {
      color: var(--color-danger) !important;

      &:hover {
        background: var(--color-danger-light) !important;
      }
    }

    .sidebar-platform-switch {
      padding: var(--space-sm);
      flex-shrink: 0;
    }

    .platform-switch-link {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      padding: 10px 12px;
      border-radius: var(--radius-md);
      border: 1px dashed rgba(139, 92, 246, 0.25);
      background: rgba(139, 92, 246, 0.05);
      color: rgba(139, 92, 246, 0.8);
      font-size: 0.8125rem;
      font-weight: 600;
      text-decoration: none;
      cursor: pointer;
      transition: all var(--transition-fast);

      &:hover {
        background: rgba(139, 92, 246, 0.1);
        border-color: rgba(139, 92, 246, 0.4);
        color: #8b5cf6;
      }
    }

    @media (max-width: 768px) {
      .sidebar-backdrop { display: block; }
      .desktop-only { display: none; }
      .mobile-only { display: flex; }

      .sidebar {
        transform: translateX(-100%);
        width: 280px;
        box-shadow: none;
      }

      .sidebar.mobile-open {
        transform: translateX(0);
        box-shadow: 4px 0 24px rgba(0, 0, 0, 0.4);
      }

      .sidebar.collapsed {
        transform: translateX(-100%);
        width: 280px;
      }

      .sidebar.collapsed.mobile-open {
        transform: translateX(0);
      }
    }
  `]
})
export class SidebarComponent {
  auth = inject(AuthService);
  orgCtx = inject(OrgContextService);
  notifState = inject(NotificationStateService);
  private router = inject(Router);

  collapsed = signal(false);
  mobileOpen = model(false);
  mobileClose = output<void>();

  private baseNavItems: Omit<NavItem, 'label'>[] = [
    { icon: 'dashboard', route: '/dashboard', section: 'Overview' },
    { icon: 'groups', route: '/crew', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'Operations' },
    { icon: 'assignment', route: '/assignments', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'Operations' },
    { icon: 'playlist_add', route: '/assignments-bulk', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'Operations' },
    { icon: 'trending_up', route: '/earnings', section: 'Operations' },
    { icon: 'account_balance_wallet', route: '/wallets', section: 'Finance' },
    { icon: 'business', route: '/employers', roles: ['SYSTEM_ADMIN'], section: 'Organization' },
    { icon: 'business', route: '/settings/tenant', roles: ['EMPLOYER'], section: 'Organization' },
    { icon: 'directions_bus', route: '/vehicles', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'Organization', feature: 'vehicles' },
    { icon: 'route', route: '/routes', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'Organization', feature: 'routes' },
    { icon: 'location_on', route: '/work-sites', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'Organization', feature: 'work-sites' },
    { icon: 'support_agent', route: '/facilitators', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'Operations', feature: 'facilitators' },
    { icon: 'receipt_long', route: '/payroll', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'Finance' },
    { icon: 'event_repeat', route: '/pay-schedules', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'Finance' },
    { icon: 'savings', route: '/loans', section: 'Finance' },
    { icon: 'credit_score', route: '/credit', section: 'Finance' },
    { icon: 'health_and_safety', route: '/insurance', section: 'Finance' },
    { icon: 'folder', route: '/documents', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'System' },
    { icon: 'notifications', route: '/notifications', section: 'System' },
    { icon: 'admin_panel_settings', route: '/admin', roles: ['SYSTEM_ADMIN'], section: 'System' },
    { icon: 'tune', route: '/settings/tenant', roles: ['SYSTEM_ADMIN', 'EMPLOYER'], section: 'System' },
  ];

  /** Dynamic nav label mapping — adapts based on industry context */
  private navLabel(route: string): string {
    const labels = this.orgCtx.template().ui_labels;
    const worker = this.orgCtx.workersLabel();
    const assignment = labels['assignment'] || 'Assignment';
    switch (route) {
      case '/dashboard':        return 'Dashboard';
      case '/crew':             return worker;
      case '/assignments':      return assignment + 's';
      case '/assignments-bulk': return 'Bulk ' + assignment;
      case '/earnings':         return 'Earnings';
      case '/wallets':          return 'Wallets';
      case '/employers':        return 'Employers';
      case '/settings/tenant':  return this.auth.isEmployer() ? ('My ' + (labels['organization'] || 'Organization')) : 'Tenant Settings';
      case '/vehicles':         return labels['vehicle'] ? labels['vehicle'] + 's' : 'Vehicles';
      case '/routes':           return 'Routes';
      case '/work-sites':       return labels['work_site'] ? labels['work_site'] + 's' : 'Work Sites';
      case '/facilitators':     return 'Facilitators';
      case '/payroll':          return 'Payroll';
      case '/pay-schedules':    return 'Pay Schedules';
      case '/loans':            return 'Loans';
      case '/credit':           return 'Credit Score';
      case '/insurance':        return 'Insurance';
      case '/documents':        return 'Documents';
      case '/notifications':    return 'Notifications';
      case '/admin':            return 'Admin';
      default:                  return route;
    }
  }

  filteredNavItems(): NavItem[] {
    const role = this.auth.userRole();
    if (!role) return [];
    return this.baseNavItems
      .filter(item => {
        // Role check
        if (item.roles && !item.roles.includes(role)) return false;
        // Industry-adaptive visibility
        if (item.feature && !this.orgCtx.isFeatureVisible(item.feature)) return false;
        return true;
      })
      .map(item => ({ ...item, label: this.navLabel(item.route) }));
  }

  /** Check if a nav item should appear locked due to KYC restrictions */
  isItemLocked(route: string): boolean {
    if (!this.auth.isKycBlocked()) return false;
    return !KYC_EXEMPT_ROUTES.some(r => route.startsWith(r));
  }

  /** Handle click on a locked nav item */
  onLockedClick(): void {
    this.navigateTo('/profile');
  }

  navigateTo(route: string): void {
    this.router.navigate([route]);
    if (window.innerWidth <= 768) {
      this.closeMobile();
    }
  }

  toggle(): void {
    this.collapsed.update(v => !v);
  }

  closeMobile(): void {
    this.mobileOpen.set(false);
  }

  onNavClick(): void {
    // Auto-close on mobile when navigating
    if (window.innerWidth <= 768) {
      this.closeMobile();
    }
  }

  userInitials(): string {
    const user = this.auth.currentUser();
    if (!user) return '?';
    return user.phone.slice(-2);
  }

  formatRole(role: string | null | undefined): string {
    if (!role) return '';
    return role.replace(/_/g, ' ').toLowerCase();
  }

  logout(): void {
    this.auth.logout();
  }
}
