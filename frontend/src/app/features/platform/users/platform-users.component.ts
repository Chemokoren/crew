import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { AdminUser } from '../../../core/models';

@Component({
  selector: 'app-platform-users',
  standalone: true,
  imports: [CommonModule, FormsModule, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title platform-title">User Management</h1>
          <p class="page-subtitle">Search, manage, and assist all platform users across organizations</p>
        </div>
        <div class="page-actions">
          <span class="user-count-badge">{{ totalUsers() }} users</span>
        </div>
      </div>

      <!-- Quick Lookup -->
      <div class="lookup-card glass-card">
        <div class="lookup-header">
          <span class="material-icons-round" style="color:#8b5cf6;">person_search</span>
          <span style="font-weight:600;">Quick User Lookup</span>
        </div>
        <div class="lookup-form">
          <div class="search-input-wrap" style="flex:1;">
            <span class="material-icons-round search-icon">search</span>
            <input class="form-input search-input" type="text"
                   placeholder="Search by phone number..."
                   [(ngModel)]="searchQuery" (ngModelChange)="filterUsers()" id="user-search" />
          </div>
          <select class="form-select filter-select" [(ngModel)]="roleFilter" (ngModelChange)="filterUsers()" id="user-role-filter">
            <option value="">All Roles</option>
            <option value="SYSTEM_ADMIN">System Admin</option>
            <option value="PLATFORM_ADMIN">Platform Admin</option>
            <option value="PLATFORM_SUPPORT">Platform Support</option>
            <option value="PLATFORM_FINANCE">Platform Finance</option>
            <option value="PLATFORM_AUDITOR">Platform Auditor</option>
            <option value="EMPLOYER">Employer</option>
            <option value="EMPLOYEE">Employee</option>
            <option value="LENDER">Lender</option>
            <option value="INSURER">Insurer</option>
          </select>
          <select class="form-select filter-select" [(ngModel)]="statusFilter" (ngModelChange)="filterUsers()" id="user-status-filter">
            <option value="">All Status</option>
            <option value="active">Active</option>
            <option value="disabled">Disabled</option>
          </select>
        </div>
      </div>

      <!-- Loading -->
      @if (loading()) {
        @for (i of [1,2,3,4,5]; track i) {
          <div class="skeleton" style="height: 52px; margin-bottom: 6px;"></div>
        }
      }

      <!-- Empty state -->
      @else if (filteredUsers().length === 0) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">people</span>
          <div class="empty-title">No users found</div>
          <div class="empty-description">Try a different search term or filter</div>
        </div>
      }

      <!-- User table -->
      @else {
        <div class="data-table-wrapper">
          <table class="data-table">
            <thead>
              <tr>
                <th>User</th>
                <th>Role</th>
                <th>Status</th>
                <th>Last Login</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              @for (u of filteredUsers(); track u.id) {
                <tr>
                  <td>
                    <div class="user-cell">
                      <div class="user-cell-avatar" [ngClass]="roleColorClass(u.system_role)">
                        {{ u.phone.slice(-2) }}
                      </div>
                      <div class="user-cell-info">
                        <span class="user-cell-phone">{{ u.phone }}</span>
                        @if (u.email) {
                          <span class="user-cell-email">{{ u.email }}</span>
                        }
                      </div>
                    </div>
                  </td>
                  <td>
                    <span class="badge" [ngClass]="roleBadgeClass(u.system_role)">
                      {{ formatRole(u.system_role) }}
                    </span>
                  </td>
                  <td>
                    <span class="badge" [ngClass]="u.is_active ? 'badge-success' : 'badge-danger'">
                      {{ u.is_active ? 'Active' : 'Disabled' }}
                    </span>
                  </td>
                  <td style="font-size:0.8125rem; color:var(--color-text-muted);">
                    {{ u.last_login_at ? (u.last_login_at | relativeTime) : 'Never' }}
                  </td>
                  <td style="font-size:0.8125rem; color:var(--color-text-muted);">
                    {{ u.created_at | relativeTime }}
                  </td>
                  <td>
                    <div class="action-btns">
                      @if (u.is_active) {
                        <button class="btn btn-sm btn-ghost" style="color: var(--color-danger);"
                                (click)="toggleAccount(u)" title="Disable Account">
                          <span class="material-icons-round" style="font-size:16px;">block</span>
                        </button>
                      } @else {
                        <button class="btn btn-sm btn-ghost" style="color: var(--color-success);"
                                (click)="toggleAccount(u)" title="Enable Account">
                          <span class="material-icons-round" style="font-size:16px;">check_circle</span>
                        </button>
                      }
                      <button class="btn btn-sm btn-ghost" style="color:#8b5cf6;"
                              (click)="openResetModal(u)" title="Reset Password">
                        <span class="material-icons-round" style="font-size:16px;">lock_reset</span>
                      </button>
                      <button class="btn btn-sm btn-ghost" style="color:var(--color-text-muted);"
                              (click)="openDetailPanel(u)" title="View Details">
                        <span class="material-icons-round" style="font-size:16px;">info</span>
                      </button>
                    </div>
                  </td>
                </tr>
              }
            </tbody>
          </table>
        </div>
      }

      <!-- Reset Password Modal -->
      @if (showResetModal()) {
        <div class="modal-backdrop" (click)="showResetModal.set(false)">
          <div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>Reset Password</h3>
              <button class="btn btn-ghost btn-icon" (click)="showResetModal.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <div class="modal-body">
              <p style="font-size:0.8125rem; color:var(--color-text-muted); margin-bottom:var(--space-md);">
                Resetting password for <strong>{{ selectedUser()?.phone }}</strong>
              </p>
              <div class="form-group">
                <label class="form-label">New Password (min 8 chars)</label>
                <input class="form-input" type="password" [(ngModel)]="newPassword" minlength="8"
                       placeholder="Enter new password" id="reset-password-input" />
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="showResetModal.set(false)">Cancel</button>
              <button class="btn btn-primary" (click)="submitReset()"
                      [disabled]="resetting() || newPassword.length < 8">
                {{ resetting() ? 'Resetting...' : 'Reset Password' }}
              </button>
            </div>
          </div>
        </div>
      }

      <!-- User Detail Panel -->
      @if (showDetailPanel()) {
        <div class="modal-backdrop" (click)="showDetailPanel.set(false)">
          <div class="modal-content" style="max-width:560px;" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>User Details</h3>
              <button class="btn btn-ghost btn-icon" (click)="showDetailPanel.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            @if (selectedUser(); as u) {
              <div class="modal-body">
                <div class="detail-header">
                  <div class="detail-avatar" [ngClass]="roleColorClass(u.system_role)">
                    {{ u.phone.slice(-2) }}
                  </div>
                  <div>
                    <div style="font-size:1.125rem;font-weight:700;color:var(--color-text-primary);">{{ u.phone }}</div>
                    @if (u.email) {
                      <div style="font-size:0.8125rem;color:var(--color-text-muted);">{{ u.email }}</div>
                    }
                  </div>
                </div>

                <div class="detail-grid">
                  <div class="detail-item">
                    <span class="detail-label">Role</span>
                    <span class="badge" [ngClass]="roleBadgeClass(u.system_role)">{{ formatRole(u.system_role) }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">Status</span>
                    <span class="badge" [ngClass]="u.is_active ? 'badge-success' : 'badge-danger'">
                      {{ u.is_active ? 'Active' : 'Disabled' }}
                    </span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">User ID</span>
                    <code class="detail-code">{{ u.id }}</code>
                  </div>
                  @if (u.organization_id) {
                    <div class="detail-item">
                      <span class="detail-label">Organization ID</span>
                      <code class="detail-code">{{ u.organization_id }}</code>
                    </div>
                  }
                  @if (u.crew_member_id) {
                    <div class="detail-item">
                      <span class="detail-label">Crew Member ID</span>
                      <code class="detail-code">{{ u.crew_member_id }}</code>
                    </div>
                  }
                  <div class="detail-item">
                    <span class="detail-label">Last Login</span>
                    <span class="detail-value">{{ u.last_login_at ? (u.last_login_at | relativeTime) : 'Never' }}</span>
                  </div>
                  <div class="detail-item">
                    <span class="detail-label">Registered</span>
                    <span class="detail-value">{{ u.created_at | relativeTime }}</span>
                  </div>
                </div>

                <div class="detail-actions">
                  <button class="btn btn-sm btn-secondary" (click)="openResetModal(u); showDetailPanel.set(false)">
                    <span class="material-icons-round">lock_reset</span> Reset Password
                  </button>
                  @if (u.is_active) {
                    <button class="btn btn-sm btn-danger" (click)="toggleAccount(u); showDetailPanel.set(false)">
                      <span class="material-icons-round">block</span> Disable
                    </button>
                  } @else {
                    <button class="btn btn-sm btn-secondary" style="color: var(--color-success);"
                            (click)="toggleAccount(u); showDetailPanel.set(false)">
                      <span class="material-icons-round">check_circle</span> Enable
                    </button>
                  }
                </div>
              </div>
            }
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .platform-title {
      background: linear-gradient(135deg, #c084fc 0%, #f472b6 100%) !important;
      -webkit-background-clip: text !important;
      -webkit-text-fill-color: transparent !important;
      background-clip: text !important;
    }

    .user-count-badge {
      padding: 4px 14px; border-radius: var(--radius-full);
      background: rgba(139,92,246,0.12); color: #8b5cf6;
      font-size: 0.8125rem; font-weight: 600;
    }

    .lookup-card {
      margin-bottom: var(--space-lg); padding: var(--space-md) var(--space-lg) !important;
    }
    .lookup-header {
      display: flex; align-items: center; gap: var(--space-sm); margin-bottom: var(--space-md);
      font-size: 0.875rem; color: var(--color-text-primary);
    }
    .lookup-form {
      display: flex; gap: var(--space-sm); flex-wrap: wrap; align-items: center;
    }

    .search-input-wrap {
      position: relative; min-width: 240px;
    }
    .search-icon {
      position: absolute; left: 12px; top: 50%; transform: translateY(-50%);
      font-size: 18px; color: var(--color-text-muted);
    }
    .search-input { padding-left: 38px; }
    .filter-select { max-width: 180px; }

    .user-cell { display: flex; align-items: center; gap: var(--space-sm); }
    .user-cell-avatar {
      width: 34px; height: 34px; border-radius: var(--radius-md);
      display: flex; align-items: center; justify-content: center;
      font-size: 0.75rem; font-weight: 700; color: #fff; flex-shrink: 0;
    }
    .user-cell-info { display: flex; flex-direction: column; }
    .user-cell-phone { font-size: 0.875rem; font-weight: 600; color: var(--color-text-primary); }
    .user-cell-email { font-size: 0.6875rem; color: var(--color-text-muted); }

    .action-btns { display: flex; gap: 2px; }

    /* Role color classes */
    .role-platform { background: linear-gradient(135deg, #8b5cf6, #ec4899); }
    .role-employer { background: linear-gradient(135deg, #6366f1, #8b5cf6); }
    .role-employee { background: linear-gradient(135deg, #10b981, #22d3ee); }
    .role-lender { background: linear-gradient(135deg, #f59e0b, #ef4444); }
    .role-insurer { background: linear-gradient(135deg, #22d3ee, #6366f1); }
    .role-default { background: linear-gradient(135deg, #6b7280, #9ca3af); }

    /* Detail panel */
    .detail-header {
      display: flex; align-items: center; gap: var(--space-md); margin-bottom: var(--space-lg);
    }
    .detail-avatar {
      width: 48px; height: 48px; border-radius: var(--radius-md);
      display: flex; align-items: center; justify-content: center;
      font-size: 1.125rem; font-weight: 700; color: #fff;
    }
    .detail-grid {
      display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md);
    }
    .detail-item { display: flex; flex-direction: column; gap: 4px; }
    .detail-label {
      font-size: 0.6875rem; font-weight: 600; text-transform: uppercase;
      letter-spacing: 0.06em; color: var(--color-text-muted);
    }
    .detail-value { font-size: 0.875rem; color: var(--color-text-primary); }
    .detail-code {
      font-size: 0.6875rem; font-family: monospace; color: var(--color-text-muted);
      background: var(--color-bg-input); padding: 2px 6px; border-radius: 4px;
      word-break: break-all;
    }
    .detail-actions {
      display: flex; gap: var(--space-sm); margin-top: var(--space-lg);
      padding-top: var(--space-md); border-top: 1px solid var(--color-border);
    }

    @media (max-width: 640px) {
      .lookup-form { flex-direction: column; }
      .search-input-wrap { min-width: 100%; }
      .filter-select { max-width: 100%; width: 100%; }
      .detail-grid { grid-template-columns: 1fr; }
    }
  `]
})
export class PlatformUsersComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  users = signal<AdminUser[]>([]);
  loading = signal(true);
  totalUsers = signal(0);

  searchQuery = '';
  roleFilter = '';
  statusFilter = '';

  showResetModal = signal(false);
  showDetailPanel = signal(false);
  selectedUser = signal<AdminUser | null>(null);
  resetting = signal(false);
  newPassword = '';

  filteredUsers = computed(() => {
    let list = this.users();
    const q = this.searchQuery.toLowerCase().trim();
    if (q) {
      list = list.filter(u => u.phone.includes(q) || (u.email && u.email.toLowerCase().includes(q)));
    }
    if (this.roleFilter) {
      list = list.filter(u => u.system_role === this.roleFilter);
    }
    if (this.statusFilter === 'active') {
      list = list.filter(u => u.is_active);
    } else if (this.statusFilter === 'disabled') {
      list = list.filter(u => !u.is_active);
    }
    return list;
  });

  ngOnInit(): void {
    this.loadUsers();
  }

  loadUsers(): void {
    this.loading.set(true);
    this.api.getUsers({ per_page: '200' }).subscribe({
      next: r => {
        this.users.set(r.data || []);
        this.totalUsers.set(r.meta?.total || r.data?.length || 0);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  filterUsers(): void {
    // Triggers computed signal re-evaluation
  }

  toggleAccount(u: AdminUser): void {
    const action = u.is_active ? 'disable' : 'enable';
    if (!confirm(`${action.charAt(0).toUpperCase() + action.slice(1)} account for ${u.phone}?`)) return;
    const obs = u.is_active ? this.api.disableAccount(u.id) : this.api.enableAccount(u.id);
    obs.subscribe({
      next: () => { this.toast.success(`Account ${action}d`); this.loadUsers(); },
    });
  }

  openResetModal(u: AdminUser): void {
    this.selectedUser.set(u);
    this.newPassword = '';
    this.showResetModal.set(true);
  }

  openDetailPanel(u: AdminUser): void {
    this.selectedUser.set(u);
    this.showDetailPanel.set(true);
  }

  submitReset(): void {
    const u = this.selectedUser();
    if (!u || this.newPassword.length < 8) return;
    this.resetting.set(true);
    this.api.resetPassword(u.id, this.newPassword).subscribe({
      next: () => {
        this.toast.success('Password reset successfully');
        this.showResetModal.set(false);
        this.resetting.set(false);
      },
      error: () => this.resetting.set(false),
    });
  }

  formatRole(role: string): string {
    return role.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, c => c.toUpperCase());
  }

  roleColorClass(role: string): string {
    if (role.startsWith('PLATFORM') || role === 'SYSTEM_ADMIN') return 'role-platform';
    if (role === 'EMPLOYER') return 'role-employer';
    if (role === 'EMPLOYEE') return 'role-employee';
    if (role === 'LENDER') return 'role-lender';
    if (role === 'INSURER') return 'role-insurer';
    return 'role-default';
  }

  roleBadgeClass(role: string): string {
    if (role.startsWith('PLATFORM') || role === 'SYSTEM_ADMIN') return 'badge-accent';
    if (role === 'EMPLOYER') return 'badge-info';
    if (role === 'EMPLOYEE') return 'badge-success';
    if (role === 'LENDER') return 'badge-warning';
    if (role === 'INSURER') return 'badge-neutral';
    return 'badge-neutral';
  }
}
