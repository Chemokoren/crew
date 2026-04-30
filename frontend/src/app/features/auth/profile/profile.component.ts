import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { User } from '../../../core/models';

@Component({
  selector: 'app-profile',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">My Profile</h1>
          <p class="page-subtitle">Manage your account information and security</p>
        </div>
      </div>

      @if (loading()) {
        <div class="profile-grid">
          <div class="skeleton" style="height: 320px;"></div>
          <div class="skeleton" style="height: 320px;"></div>
        </div>
      } @else if (user()) {
        <div class="profile-grid">
          <!-- Profile Card -->
          <div class="glass-card profile-card">
            <div class="profile-header">
              <div class="profile-avatar">{{ userInitials() }}</div>
              <div class="profile-identity">
                <span class="profile-phone">{{ user()!.phone }}</span>
                <span class="badge badge-accent">{{ formatRole(user()!.system_role) }}</span>
              </div>
            </div>

            <div class="detail-rows">
              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">badge</span> User ID
                </span>
                <code class="detail-value text-accent">{{ user()!.id | slice:0:8 }}…</code>
              </div>

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">phone</span> Phone
                </span>
                <span class="detail-value">{{ user()!.phone }}</span>
              </div>

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">mail</span> Email
                </span>
                @if (editingEmail()) {
                  <div class="inline-edit">
                    <input class="form-input" [(ngModel)]="emailValue" placeholder="john@example.com" style="max-width: 220px;" id="profile-email-input" />
                    <button class="btn btn-primary btn-sm" (click)="saveEmail()" [disabled]="savingEmail()" id="profile-email-save">
                      @if (savingEmail()) { <span class="spinner-sm"></span> } @else { Save }
                    </button>
                    <button class="btn btn-ghost btn-sm" (click)="cancelEmailEdit()">Cancel</button>
                  </div>
                } @else {
                  <div class="inline-edit">
                    <span class="detail-value">{{ user()!.email || '—' }}</span>
                    <button class="btn btn-ghost btn-sm" (click)="startEmailEdit()" id="profile-email-edit">
                      <span class="material-icons-round" style="font-size:16px;">edit</span>
                    </button>
                  </div>
                }
              </div>

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">security</span> Role
                </span>
                <span class="detail-value">{{ formatRole(user()!.system_role) }}</span>
              </div>

              @if (user()!.crew_member_id) {
                <div class="detail-row">
                  <span class="detail-label">
                    <span class="material-icons-round detail-icon">groups</span> Crew Member ID
                  </span>
                  <code class="detail-value text-accent">{{ user()!.crew_member_id! | slice:0:8 }}…</code>
                </div>
              }

              @if (user()!.sacco_id) {
                <div class="detail-row">
                  <span class="detail-label">
                    <span class="material-icons-round detail-icon">business</span> SACCO ID
                  </span>
                  <code class="detail-value text-accent">{{ user()!.sacco_id! | slice:0:8 }}…</code>
                </div>
              }

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">calendar_today</span> Joined
                </span>
                <span class="detail-value">{{ user()!.created_at | date:'mediumDate' }}</span>
              </div>

              @if (user()!.last_login_at) {
                <div class="detail-row">
                  <span class="detail-label">
                    <span class="material-icons-round detail-icon">login</span> Last Login
                  </span>
                  <span class="detail-value">{{ user()!.last_login_at | date:'medium' }}</span>
                </div>
              }

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">circle</span> Status
                </span>
                <span class="badge" [ngClass]="user()!.is_active ? 'badge-success' : 'badge-danger'">
                  {{ user()!.is_active ? 'Active' : 'Inactive' }}
                </span>
              </div>
            </div>
          </div>

          <!-- Security Card -->
          <div class="glass-card security-card">
            <h3 class="card-title">
              <span class="material-icons-round" style="font-size: 20px; color: var(--color-accent);">shield</span>
              Security
            </h3>

            <div class="security-section">
              <div class="security-item">
                <div class="security-info">
                  <span class="security-label">Password</span>
                  <span class="security-description">Change your account password to keep your account secure.</span>
                </div>
                <button class="btn btn-secondary btn-sm" (click)="showPasswordModal.set(true)" id="btn-change-password">
                  <span class="material-icons-round" style="font-size:16px;">lock</span> Change Password
                </button>
              </div>

              <div class="security-divider"></div>

              <div class="security-item">
                <div class="security-info">
                  <span class="security-label">Session</span>
                  <span class="security-description">Sign out of your account on this device.</span>
                </div>
                <button class="btn btn-danger btn-sm" (click)="auth.logout()" id="btn-logout-profile">
                  <span class="material-icons-round" style="font-size:16px;">logout</span> Sign Out
                </button>
              </div>
            </div>
          </div>
        </div>
      }

      <!-- Change Password Modal -->
      @if (showPasswordModal()) {
        <div class="modal-backdrop" (click)="showPasswordModal.set(false)">
          <div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>Change Password</h3>
              <button class="btn btn-ghost btn-icon" (click)="showPasswordModal.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <form class="modal-body" (ngSubmit)="changePassword()" id="change-password-form">
              <div class="form-group">
                <label class="form-label" for="cp-current">Current Password</label>
                <input class="form-input" id="cp-current" type="password" placeholder="Enter current password"
                       [(ngModel)]="currentPassword" name="currentPassword" required autocomplete="current-password" />
              </div>
              <div class="form-group">
                <label class="form-label" for="cp-new">New Password</label>
                <input class="form-input" id="cp-new" type="password" placeholder="Min 8 characters"
                       [(ngModel)]="newPassword" name="newPassword" required minlength="8" autocomplete="new-password" />
              </div>
              <div class="form-group">
                <label class="form-label" for="cp-confirm">Confirm New Password</label>
                <input class="form-input" id="cp-confirm" type="password" placeholder="Re-enter new password"
                       [(ngModel)]="confirmPassword" name="confirmPassword" required autocomplete="new-password" />
                @if (confirmPassword && newPassword !== confirmPassword) {
                  <span class="form-error">Passwords do not match</span>
                }
              </div>
            </form>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="showPasswordModal.set(false)">Cancel</button>
              <button class="btn btn-primary" (click)="changePassword()" [disabled]="changingPassword()" id="submit-change-password">
                @if (changingPassword()) {
                  <span class="spinner-sm"></span> Changing...
                } @else {
                  <span class="material-icons-round" style="font-size:16px;">lock</span> Update Password
                }
              </button>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .profile-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: var(--space-lg);
    }

    @media (max-width: 900px) {
      .profile-grid { grid-template-columns: 1fr; }
    }

    .profile-card, .security-card {
      padding: var(--space-lg) !important;
    }

    .profile-header {
      display: flex;
      align-items: center;
      gap: var(--space-lg);
      padding-bottom: var(--space-lg);
      margin-bottom: var(--space-md);
      border-bottom: 1px solid var(--color-border);
    }

    .profile-avatar {
      width: 64px;
      height: 64px;
      border-radius: var(--radius-lg);
      background: var(--gradient-accent);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 1.25rem;
      font-weight: 800;
      color: var(--color-text-inverse);
      flex-shrink: 0;
    }

    .profile-identity {
      display: flex;
      flex-direction: column;
      gap: 4px;
    }

    .profile-phone {
      font-family: var(--font-heading);
      font-size: 1.25rem;
      font-weight: 700;
      color: var(--color-text-primary);
    }

    .detail-rows {
      display: flex;
      flex-direction: column;
      gap: 0;
    }

    .detail-row {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 10px 0;
      border-bottom: 1px solid var(--color-border);
      gap: var(--space-md);

      &:last-child { border-bottom: none; }
    }

    .detail-label {
      font-size: 0.8125rem;
      color: var(--color-text-muted);
      display: flex;
      align-items: center;
      gap: 6px;
      white-space: nowrap;
    }

    .detail-icon {
      font-size: 16px;
    }

    .detail-value {
      font-size: 0.875rem;
      color: var(--color-text-primary);
      font-weight: 500;
      text-align: right;
    }

    .inline-edit {
      display: flex;
      align-items: center;
      gap: var(--space-xs);
    }

    .card-title {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      font-size: 1rem;
      font-weight: 600;
      margin-bottom: var(--space-lg);
      color: var(--color-text-primary);
    }

    .security-section {
      display: flex;
      flex-direction: column;
    }

    .security-item {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: var(--space-md);
      padding: var(--space-md) 0;
    }

    .security-info {
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .security-label {
      font-size: 0.875rem;
      font-weight: 600;
      color: var(--color-text-primary);
    }

    .security-description {
      font-size: 0.75rem;
      color: var(--color-text-muted);
    }

    .security-divider {
      height: 1px;
      background: var(--color-border);
    }

    .spinner-sm {
      display: inline-block;
      width: 14px;
      height: 14px;
      border: 2px solid rgba(255, 255, 255, 0.2);
      border-top-color: currentColor;
      border-radius: 50%;
      animation: spin 600ms linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    @media (max-width: 600px) {
      .security-item {
        flex-direction: column;
        align-items: flex-start;
        gap: var(--space-sm);
      }

      .detail-row {
        flex-direction: column;
        align-items: flex-start;
        gap: 4px;
      }

      .detail-value {
        text-align: left;
      }
    }
  `]
})
export class ProfileComponent implements OnInit {
  auth = inject(AuthService);
  private toast = inject(ToastService);

  user = signal<User | null>(null);
  loading = signal(true);

  // Email edit state
  editingEmail = signal(false);
  savingEmail = signal(false);
  emailValue = '';

  // Password change state
  showPasswordModal = signal(false);
  changingPassword = signal(false);
  currentPassword = '';
  newPassword = '';
  confirmPassword = '';

  ngOnInit(): void {
    this.auth.fetchProfile().subscribe({
      next: (res) => {
        this.user.set(res.data);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  userInitials(): string {
    const u = this.user();
    if (!u) return '?';
    return u.phone.slice(-2);
  }

  formatRole(role: string): string {
    return role.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, l => l.toUpperCase());
  }

  startEmailEdit(): void {
    this.emailValue = this.user()?.email ?? '';
    this.editingEmail.set(true);
  }

  cancelEmailEdit(): void {
    this.editingEmail.set(false);
    this.emailValue = '';
  }

  saveEmail(): void {
    this.savingEmail.set(true);
    // Re-fetch profile after email update to reflect server-side changes.
    // The backend may not have a dedicated email update endpoint, so we
    // trigger fetchProfile to sync. If there is one, wire it here.
    this.auth.fetchProfile().subscribe({
      next: (res) => {
        this.user.set(res.data);
        this.savingEmail.set(false);
        this.editingEmail.set(false);
        this.toast.success('Profile refreshed');
      },
      error: () => this.savingEmail.set(false),
    });
  }

  changePassword(): void {
    if (!this.currentPassword || !this.newPassword || !this.confirmPassword) {
      this.toast.warning('All password fields are required');
      return;
    }

    if (this.newPassword.length < 8) {
      this.toast.warning('New password must be at least 8 characters');
      return;
    }

    if (this.newPassword !== this.confirmPassword) {
      this.toast.error('Passwords do not match');
      return;
    }

    this.changingPassword.set(true);
    this.auth.changePassword(this.currentPassword, this.newPassword).subscribe({
      next: () => {
        this.toast.success('Password changed successfully');
        this.showPasswordModal.set(false);
        this.changingPassword.set(false);
        this.currentPassword = '';
        this.newPassword = '';
        this.confirmPassword = '';
      },
      error: () => this.changingPassword.set(false),
    });
  }
}
