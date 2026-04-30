import { Component, inject, signal, model, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';

interface NavItem {
  label: string;
  icon: string;
  route: string;
  roles?: string[];
  section?: string;
}

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
          <span class="logo-icon">⚡</span>
          @if (!collapsed()) {
            <span class="logo-text">AMY<span class="logo-accent">MIS</span></span>
          }
        </div>
        <button class="collapse-btn desktop-only" (click)="toggle()" id="sidebar-toggle">
          <span class="material-icons-round">{{ collapsed() ? 'chevron_right' : 'chevron_left' }}</span>
        </button>
        <button class="collapse-btn mobile-only" (click)="closeMobile()" id="sidebar-close-mobile">
          <span class="material-icons-round">close</span>
        </button>
      </div>

      <nav class="sidebar-nav">
        @for (item of filteredNavItems(); track item.route; let i = $index) {
          @if (item.section && (i === 0 || filteredNavItems()[i - 1]?.section !== item.section)) {
            @if (!collapsed()) {
              <div class="nav-section-label">{{ item.section }}</div>
            } @else {
              <div class="nav-divider"></div>
            }
          }
          <a class="nav-item" [routerLink]="item.route" routerLinkActive="active"
             [attr.id]="'nav-' + item.label.toLowerCase().replace(' ', '-')"
             (click)="onNavClick()">
            <span class="material-icons-round nav-icon">{{ item.icon }}</span>
            @if (!collapsed()) {
              <span class="nav-label">{{ item.label }}</span>
            }
          </a>
        }
      </nav>

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

    .nav-icon {
      font-size: 20px;
      width: 24px;
      flex-shrink: 0;
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

  collapsed = signal(false);
  mobileOpen = model(false);
  mobileClose = output<void>();

  private navItems: NavItem[] = [
    { label: 'Dashboard', icon: 'dashboard', route: '/dashboard', section: 'Overview' },
    { label: 'Crew', icon: 'groups', route: '/crew', roles: ['SYSTEM_ADMIN', 'SACCO_ADMIN'], section: 'Operations' },
    { label: 'Assignments', icon: 'assignment', route: '/assignments', roles: ['SYSTEM_ADMIN', 'SACCO_ADMIN'], section: 'Operations' },
    { label: 'Earnings', icon: 'trending_up', route: '/earnings', section: 'Operations' },
    { label: 'Wallets', icon: 'account_balance_wallet', route: '/wallets', section: 'Finance' },
    { label: 'SACCOs', icon: 'business', route: '/saccos', roles: ['SYSTEM_ADMIN', 'SACCO_ADMIN'], section: 'Organization' },
    { label: 'Vehicles', icon: 'directions_bus', route: '/vehicles', roles: ['SYSTEM_ADMIN', 'SACCO_ADMIN'], section: 'Organization' },
    { label: 'Routes', icon: 'route', route: '/routes', roles: ['SYSTEM_ADMIN', 'SACCO_ADMIN'], section: 'Organization' },
    { label: 'Payroll', icon: 'receipt_long', route: '/payroll', roles: ['SYSTEM_ADMIN', 'SACCO_ADMIN'], section: 'Finance' },
    { label: 'Loans', icon: 'savings', route: '/loans', section: 'Finance' },
    { label: 'Insurance', icon: 'health_and_safety', route: '/insurance', section: 'Finance' },
    { label: 'Notifications', icon: 'notifications', route: '/notifications', section: 'System' },
    { label: 'Admin', icon: 'admin_panel_settings', route: '/admin', roles: ['SYSTEM_ADMIN'], section: 'System' },
  ];

  filteredNavItems(): NavItem[] {
    const role = this.auth.userRole();
    if (!role) return [];
    return this.navItems.filter(item => {
      if (!item.roles) return true;
      return item.roles.includes(role);
    });
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
