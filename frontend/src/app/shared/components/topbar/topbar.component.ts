import { Component, inject, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';

@Component({
  selector: 'app-topbar',
  standalone: true,
  imports: [CommonModule, RouterLink],
  template: `
    <header class="topbar">
      <button class="mobile-menu-btn" (click)="menuToggle.emit()" id="mobile-menu-toggle">
        <span class="material-icons-round">menu</span>
      </button>

      <div class="topbar-spacer"></div>

      <div class="topbar-actions">
        <a routerLink="/notifications" class="topbar-btn" id="topbar-notifications">
          <span class="material-icons-round">notifications_none</span>
          <span class="notification-dot"></span>
        </a>

        <a routerLink="/profile" class="user-menu" id="topbar-profile-link">
          <div class="user-avatar">
            {{ userInitials() }}
          </div>
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
    .topbar {
      position: fixed;
      top: 0;
      left: var(--sidebar-width);
      right: 0;
      height: var(--topbar-height);
      background: var(--color-bg-topbar);
      backdrop-filter: blur(12px);
      -webkit-backdrop-filter: blur(12px);
      border-bottom: 1px solid var(--color-border);
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
      border: 1px solid var(--color-border);
      border-radius: var(--radius-md);
      color: var(--color-text-secondary);
      cursor: pointer;
      transition: all var(--transition-fast);

      &:hover {
        background: rgba(255, 255, 255, 0.05);
        color: var(--color-text-primary);
        border-color: var(--color-border-hover);
      }
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
      border: 1px solid var(--color-border);
      border-radius: var(--radius-md);
      color: var(--color-text-secondary);
      cursor: pointer;
      transition: all var(--transition-fast);
      text-decoration: none;

      &:hover {
        color: var(--color-text-primary);
        border-color: var(--color-border-hover);
        background: rgba(255, 255, 255, 0.03);
      }
    }

    .notification-dot {
      position: absolute;
      top: 8px;
      right: 8px;
      width: 8px;
      height: 8px;
      background: var(--color-accent);
      border-radius: 50%;
      border: 2px solid var(--color-bg-topbar);
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
        background: rgba(255, 255, 255, 0.04);
      }
    }

    .user-avatar {
      width: 36px;
      height: 36px;
      border-radius: var(--radius-md);
      background: var(--gradient-accent);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.8125rem;
      font-weight: 700;
      color: var(--color-text-inverse);
      flex-shrink: 0;
    }

    .user-info {
      display: flex;
      flex-direction: column;
    }

    .user-name {
      font-size: 0.8125rem;
      font-weight: 600;
      color: var(--color-text-primary);
    }

    .user-role {
      font-size: 0.6875rem;
      color: var(--color-text-muted);
      text-transform: capitalize;
    }

    @media (max-width: 768px) {
      .topbar {
        left: 0;
      }

      .mobile-menu-btn {
        display: flex;
      }

      .user-info {
        display: none;
      }
    }
  `]
})
export class TopbarComponent {
  auth = inject(AuthService);
  menuToggle = output<void>();

  userInitials(): string {
    const user = this.auth.currentUser();
    if (!user) return '?';
    return user.phone.slice(-2);
  }

  formatRole(role: string): string {
    return role.replace(/_/g, ' ').toLowerCase();
  }
}
