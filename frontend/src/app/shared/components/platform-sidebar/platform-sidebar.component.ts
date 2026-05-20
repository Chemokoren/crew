import { Component, inject, signal, model, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive, Router } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';
import { NotificationStateService } from '../../../core/services/notification-state.service';

interface PlatformNavItem {
  label: string;
  icon: string;
  route: string;
  section: string;
  /** Roles that can see this item. Empty = all platform roles. */
  roles?: string[];
  badge?: string;
}

@Component({
  selector: 'app-platform-sidebar',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive],
  template: `
    <!-- Mobile backdrop -->
    @if (mobileOpen()) {
      <div class="sidebar-backdrop" (click)="closeMobile()"></div>
    }

    <aside class="sidebar platform-sidebar" [class.collapsed]="collapsed()" [class.mobile-open]="mobileOpen()">
      <div class="sidebar-brand">
        <div class="brand-logo">
          <div class="platform-brand-icon">
            <img src="/logo.png" alt="AMY Logo" class="platform-brand-img" />
          </div>
          @if (!collapsed()) {
            <div class="platform-brand-text">
              <span class="brand-name">AMY</span>
              <span class="brand-tag">Platform</span>
            </div>
          }
        </div>
        <button class="collapse-btn desktop-only" (click)="toggle()" id="platform-sidebar-toggle">
          <span class="material-icons-round">{{ collapsed() ? 'chevron_right' : 'chevron_left' }}</span>
        </button>
        <button class="collapse-btn mobile-only" (click)="closeMobile()" id="platform-sidebar-close">
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
             [routerLinkActiveOptions]="{ exact: item.route === '/platform/command-center' }"
             [attr.id]="'pnav-' + item.label.toLowerCase().replace(' ', '-')"
             (click)="onNavClick()">
            <span class="material-icons-round nav-icon">{{ item.icon }}</span>
            @if (!collapsed()) {
              <span class="nav-label">{{ item.label }}</span>
              @if (item.badge) {
                <span class="nav-badge">{{ item.badge }}</span>
              }
            }
          </a>
        }
      </nav>

      <!-- Switch to org view -->
      @if (!collapsed()) {
        <div class="sidebar-switch">
          <a routerLink="/dashboard" class="switch-btn" id="platform-switch-org">
            <span class="material-icons-round">storefront</span>
            <span>Organization View</span>
            <span class="material-icons-round switch-arrow">arrow_forward</span>
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
        <button class="nav-item logout-btn" (click)="logout()" id="platform-sidebar-logout">
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

    .platform-sidebar {
      position: fixed;
      left: 0;
      top: 0;
      bottom: 0;
      width: var(--sidebar-width);
      background: var(--platform-bg-sidebar, #0c0518);
      border-right: 1px solid var(--platform-border, rgba(139, 92, 246, 0.1));
      display: flex;
      flex-direction: column;
      z-index: var(--z-sidebar);
      transition: width var(--transition-base), transform var(--transition-base);
      overflow: hidden;
      color: var(--platform-text, #e9d5ff);
    }

    .platform-sidebar.collapsed {
      width: var(--sidebar-collapsed-width);
    }

    .sidebar-brand {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: var(--space-md);
      border-bottom: 1px solid var(--platform-border, rgba(139, 92, 246, 0.1));
      height: var(--topbar-height);
      flex-shrink: 0;
    }

    .brand-logo {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
    }

    .platform-brand-icon {
      width: 36px;
      height: 36px;
      display: flex;
      align-items: center;
      justify-content: center;
      border: 2px solid transparent;
      background: linear-gradient(var(--platform-bg-sidebar, #0c0518), var(--platform-bg-sidebar, #0c0518)) padding-box,
                  linear-gradient(135deg, #8b5cf6 0%, #ec4899 100%) border-box;
      border-radius: var(--radius-md);
      flex-shrink: 0;
    }

    .platform-brand-img {
      width: 100%;
      height: 100%;
      object-fit: contain;
      padding: 2px;
      box-sizing: border-box;
      flex-shrink: 0;
    }

    .platform-brand-text {
      display: flex;
      flex-direction: column;
      line-height: 1.2;
    }

    .brand-name {
      font-family: var(--font-heading);
      font-size: 1.125rem;
      font-weight: 800;
      background: linear-gradient(135deg, #c084fc 0%, #f472b6 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }

    .brand-tag {
      font-size: 0.625rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.15em;
      color: rgba(196, 181, 253, 0.6);
    }

    .collapse-btn {
      width: 28px;
      height: 28px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: none;
      border: 1px solid rgba(139, 92, 246, 0.2);
      border-radius: var(--radius-sm);
      color: rgba(196, 181, 253, 0.5);
      cursor: pointer;
      transition: all var(--transition-fast);

      &:hover {
        color: #c4b5fd;
        border-color: rgba(139, 92, 246, 0.4);
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
      color: var(--platform-text-muted, rgba(196, 181, 253, 0.4));
      padding: 16px 12px 6px;
      white-space: nowrap;
    }

    .nav-divider {
      height: 1px;
      background: var(--platform-border, rgba(139, 92, 246, 0.1));
      margin: 8px 12px;
    }

    .nav-item {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      padding: 10px 12px;
      border-radius: var(--radius-md);
      color: var(--platform-text-muted, rgba(196, 181, 253, 0.7));
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
        background: rgba(139, 92, 246, 0.08);
        color: var(--platform-text, #e9d5ff);
      }

      &.active {
        background: linear-gradient(135deg, rgba(139, 92, 246, 0.18) 0%, rgba(236, 72, 153, 0.12) 100%);
        color: var(--platform-accent, #c084fc);

        .nav-icon { color: var(--platform-accent, #c084fc); }
      }
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
      background: linear-gradient(135deg, #8b5cf6, #ec4899);
      color: #fff;
      font-size: 0.625rem;
      font-weight: 700;
      display: flex;
      align-items: center;
      justify-content: center;
      line-height: 1;
    }

    /* Switch to org view */
    .sidebar-switch {
      padding: var(--space-sm);
      flex-shrink: 0;
    }

    .switch-btn {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      padding: 10px 12px;
      border-radius: var(--radius-md);
      border: 1px dashed rgba(139, 92, 246, 0.2);
      background: rgba(139, 92, 246, 0.04);
      color: rgba(196, 181, 253, 0.6);
      font-size: 0.8125rem;
      font-weight: 500;
      text-decoration: none;
      cursor: pointer;
      transition: all var(--transition-fast);
      width: 100%;

      .material-icons-round { font-size: 18px; }

      &:hover {
        background: rgba(139, 92, 246, 0.1);
        border-color: rgba(139, 92, 246, 0.35);
        color: #c4b5fd;
      }
    }

    .switch-arrow {
      margin-left: auto;
      font-size: 16px !important;
      opacity: 0.4;
    }

    .sidebar-footer {
      padding: var(--space-sm);
      border-top: 1px solid var(--platform-border, rgba(139, 92, 246, 0.1));
      flex-shrink: 0;
    }

    .sidebar-user-badge {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      padding: 8px 12px;
      margin-bottom: 4px;
      border-radius: var(--radius-md);
      background: rgba(139, 92, 246, 0.05);
    }

    .sidebar-user-avatar {
      width: 32px;
      height: 32px;
      border-radius: var(--radius-md);
      background: linear-gradient(135deg, #8b5cf6 0%, #ec4899 100%);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.75rem;
      font-weight: 700;
      color: #fff;
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
      color: var(--platform-text, #e9d5ff);
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .sidebar-user-role {
      font-size: 0.6875rem;
      color: var(--platform-text-muted, rgba(196, 181, 253, 0.5));
      text-transform: capitalize;
    }

    .logout-btn {
      color: #ef4444 !important;

      &:hover {
        background: rgba(239, 68, 68, 0.1) !important;
      }
    }

    @media (max-width: 768px) {
      .sidebar-backdrop { display: block; }
      .desktop-only { display: none; }
      .mobile-only { display: flex; }

      .platform-sidebar {
        transform: translateX(-100%);
        width: 280px;
        box-shadow: none;
      }

      .platform-sidebar.mobile-open {
        transform: translateX(0);
        box-shadow: 4px 0 24px rgba(0, 0, 0, 0.4);
      }

      .platform-sidebar.collapsed {
        transform: translateX(-100%);
        width: 280px;
      }

      .platform-sidebar.collapsed.mobile-open {
        transform: translateX(0);
      }
    }
  `]
})
export class PlatformSidebarComponent {
  auth = inject(AuthService);
  notifState = inject(NotificationStateService);
  private router = inject(Router);

  collapsed = signal(false);
  mobileOpen = model(false);

  private navItems: PlatformNavItem[] = [
    // OVERVIEW
    { label: 'Command Center', icon: 'space_dashboard', route: '/platform/command-center', section: 'Overview' },

    // MANAGEMENT
    { label: 'Organizations', icon: 'business', route: '/platform/organizations', section: 'Management' },
    { label: 'Users', icon: 'people', route: '/platform/users', section: 'Management' },
    { label: 'Support Center', icon: 'support_agent', route: '/platform/support', section: 'Management',
      roles: ['SYSTEM_ADMIN', 'PLATFORM_ADMIN', 'PLATFORM_SUPPORT', 'PLATFORM_ASSISTANT'] },

    // CONFIGURATION
    { label: 'Roles & Permissions', icon: 'admin_panel_settings', route: '/platform/roles', section: 'Configuration',
      roles: ['SYSTEM_ADMIN', 'PLATFORM_ADMIN'] },
    { label: 'System Settings', icon: 'settings', route: '/platform/settings', section: 'Configuration',
      roles: ['SYSTEM_ADMIN', 'PLATFORM_ADMIN', 'PLATFORM_ASSISTANT'] },
    { label: 'Notifications', icon: 'notifications', route: '/platform/notifications', section: 'Configuration',
      roles: ['SYSTEM_ADMIN', 'PLATFORM_ADMIN', 'PLATFORM_ASSISTANT'] },
    { label: 'Integrations', icon: 'extension', route: '/platform/integrations', section: 'Configuration',
      roles: ['SYSTEM_ADMIN', 'PLATFORM_ADMIN'] },

    // FINANCE
    { label: 'Float Oversight', icon: 'account_balance', route: '/platform/finance', section: 'Finance',
      roles: ['SYSTEM_ADMIN', 'PLATFORM_ADMIN', 'PLATFORM_FINANCE'] },
    { label: 'Reports', icon: 'insights', route: '/platform/reports', section: 'Finance',
      roles: ['SYSTEM_ADMIN', 'PLATFORM_ADMIN', 'PLATFORM_FINANCE', 'PLATFORM_AUDITOR'] },

    // COMPLIANCE
    { label: 'Audit Trail', icon: 'history', route: '/platform/compliance', section: 'Compliance' },
    { label: 'Documents', icon: 'folder', route: '/platform/documents', section: 'Compliance',
      roles: ['SYSTEM_ADMIN', 'PLATFORM_ADMIN', 'PLATFORM_SUPPORT', 'PLATFORM_AUDITOR'] },

    // TEAM
    { label: 'Platform Team', icon: 'group_work', route: '/platform/team', section: 'Team',
      roles: ['SYSTEM_ADMIN', 'PLATFORM_ADMIN'] },
  ];

  /** Computed signal — only recalculates when user role changes, not every change detection cycle. */
  filteredNavItems = computed(() => {
    const role = this.auth.userRole();
    if (!role) return [] as PlatformNavItem[];
    return this.navItems.filter(item => {
      if (item.roles && !item.roles.includes(role)) return false;
      return true;
    });
  });

  toggle(): void {
    this.collapsed.update(v => !v);
  }

  closeMobile(): void {
    this.mobileOpen.set(false);
  }

  onNavClick(): void {
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

