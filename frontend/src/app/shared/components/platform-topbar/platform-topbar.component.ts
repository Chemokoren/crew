import { Component, inject, output, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';
import { NotificationStateService } from '../../../core/services/notification-state.service';
import { ThemeService } from '../../../core/services/theme.service';

@Component({
  selector: 'app-platform-topbar',
  standalone: true,
  imports: [CommonModule, RouterLink],
  template: `
    <header class="topbar platform-topbar">
      <button class="mobile-menu-btn" (click)="menuToggle.emit()" id="platform-mobile-menu-toggle"
              aria-label="Toggle sidebar menu">
        <span class="material-icons-round">menu</span>
      </button>

      <div class="platform-context">
        <span class="platform-context-badge">
          <span class="material-icons-round" style="font-size: 14px;">shield</span>
          Platform Admin
        </span>
      </div>

      <div class="topbar-spacer"></div>

      <div class="topbar-actions">
        <a routerLink="/platform/notifications" class="topbar-btn" id="platform-notif-btn"
           aria-label="Notifications">
          <span class="material-icons-round">notifications</span>
          @if (notifState.unreadCount() > 0) {
            <span class="notif-dot"></span>
          }
        </a>

        <button class="topbar-btn" (click)="themeSvc.toggle()" id="platform-theme-toggle"
                [attr.aria-label]="themeSvc.theme()==='dark' ? 'Switch to light mode' : 'Switch to dark mode'">
          <span class="material-icons-round">{{ themeSvc.theme() === 'dark' ? 'light_mode' : 'dark_mode' }}</span>
        </button>

        <a routerLink="/profile" class="user-menu" id="platform-topbar-profile">
          <div class="user-avatar">{{ userInitials() }}</div>
          @if (auth.currentUser(); as user) {
            <div class="user-info">
              <span class="user-name">{{ user.phone }}</span>
              <span class="user-role">{{ formatRole(user.system_role) }}</span>
            </div>
          }
        </a>
      </div>
    </header>
  `,
  styles: [`
    .platform-topbar {
      position: fixed;
      top: 0;
      left: var(--sidebar-width);
      right: 0;
      height: var(--topbar-height);
      background: var(--platform-topbar-bg, rgba(12, 5, 24, 0.92));
      backdrop-filter: blur(12px);
      -webkit-backdrop-filter: blur(12px);
      border-bottom: 1px solid var(--platform-border, rgba(139, 92, 246, 0.1));
      display: flex;
      align-items: center;
      padding: 0 var(--space-lg);
      z-index: var(--z-topbar);
      transition: left var(--transition-base);
    }

    .mobile-menu-btn {
      display: none;
      width: 40px;
      height: 40px;
      align-items: center;
      justify-content: center;
      background: none;
      border: 1px solid rgba(139, 92, 246, 0.2);
      border-radius: var(--radius-md);
      color: rgba(196, 181, 253, 0.7);
      cursor: pointer;
      transition: all var(--transition-fast);
      margin-right: var(--space-md);

      &:hover {
        background: rgba(139, 92, 246, 0.08);
        color: #c4b5fd;
        border-color: rgba(139, 92, 246, 0.3);
      }
    }

    .platform-context {
      display: flex;
      align-items: center;
    }

    .platform-context-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 12px;
      border-radius: var(--radius-full);
      background: linear-gradient(135deg, rgba(139, 92, 246, 0.15) 0%, rgba(236, 72, 153, 0.1) 100%);
      border: 1px solid rgba(139, 92, 246, 0.2);
      color: var(--platform-accent, #c084fc);
      font-size: 0.75rem;
      font-weight: 600;
      letter-spacing: 0.02em;
    }

    .notif-dot {
      position: absolute;
      top: 6px;
      right: 6px;
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: #ec4899;
      border: 2px solid var(--platform-topbar-bg, rgba(12, 5, 24, 0.92));
    }

    .topbar-spacer {
      flex: 1;
    }

    .topbar-actions {
      display: flex;
      align-items: center;
      gap: var(--space-md);
    }

    .topbar-btn {
      position: relative;
      width: 40px;
      height: 40px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: none;
      border: 1px solid var(--platform-border, rgba(139, 92, 246, 0.15));
      border-radius: var(--radius-md);
      color: var(--platform-text-muted, rgba(196, 181, 253, 0.6));
      cursor: pointer;
      transition: all var(--transition-fast);
      text-decoration: none;

      &:hover {
        color: var(--platform-text, #c4b5fd);
        border-color: rgba(139, 92, 246, 0.3);
        background: rgba(139, 92, 246, 0.06);
      }
    }

    .user-menu {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      text-decoration: none;
      cursor: pointer;
      padding: 4px 8px;
      border-radius: var(--radius-md);
      transition: all var(--transition-fast);

      &:hover {
        background: rgba(139, 92, 246, 0.06);
      }
    }

    .user-avatar {
      width: 36px;
      height: 36px;
      border-radius: var(--radius-md);
      background: linear-gradient(135deg, #8b5cf6 0%, #ec4899 100%);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.8125rem;
      font-weight: 700;
      color: #fff;
      flex-shrink: 0;
    }

    .user-info {
      display: flex;
      flex-direction: column;
    }

    .user-name {
      font-size: 0.8125rem;
      font-weight: 600;
      color: var(--platform-text, #e9d5ff);
    }

    .user-role {
      font-size: 0.6875rem;
      color: var(--platform-text-muted, rgba(196, 181, 253, 0.5));
      text-transform: capitalize;
    }

    @media (max-width: 768px) {
      .platform-topbar {
        left: 0;
      }

      .mobile-menu-btn {
        display: flex;
      }

      .user-info {
        display: none;
      }

      .platform-context-badge {
        font-size: 0.6875rem;
        padding: 3px 8px;
      }
    }
  `]
})
export class PlatformTopbarComponent implements OnInit {
  auth = inject(AuthService);
  notifState = inject(NotificationStateService);
  themeSvc = inject(ThemeService);
  menuToggle = output<void>();

  ngOnInit(): void {
    this.notifState.init();
  }

  userInitials(): string {
    const user = this.auth.currentUser();
    if (!user) return '?';
    return user.phone.slice(-2);
  }

  formatRole(role: string): string {
    return role.replace(/_/g, ' ').toLowerCase();
  }
}
