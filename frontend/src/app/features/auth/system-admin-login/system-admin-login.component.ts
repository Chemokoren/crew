import { Component, inject, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';

@Component({
  selector: 'app-system-admin-login',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="sa-page">
      <div class="sa-bg"></div>
      <div class="sa-container">
        <div class="sa-card">
          <div class="sa-header">
            <div class="shield-icon">
              <span class="material-icons-round">admin_panel_settings</span>
            </div>
            <h1>System Administrator</h1>
            <p>Restricted access — authorized personnel only</p>
          </div>

          <form (ngSubmit)="onSubmit()" class="sa-form">
            <div class="sa-field">
              <label for="sa-phone">Phone Number</label>
              <div class="input-wrapper">
                <span class="material-icons-round input-icon">phone</span>
                <input id="sa-phone" type="tel" [(ngModel)]="phone" name="phone"
                       placeholder="+254..." autocomplete="tel" autofocus />
              </div>
            </div>

            <div class="sa-field">
              <label for="sa-password">Password</label>
              <div class="input-wrapper">
                <span class="material-icons-round input-icon">lock</span>
                <input id="sa-password" [type]="showPw() ? 'text' : 'password'"
                       [(ngModel)]="password" name="password"
                       placeholder="••••••••" autocomplete="current-password" />
                <button type="button" class="pw-toggle" (click)="showPw.update(v => !v)">
                  <span class="material-icons-round">{{ showPw() ? 'visibility_off' : 'visibility' }}</span>
                </button>
              </div>
            </div>

            <button type="submit" class="sa-btn" [disabled]="loading()">
              @if (loading()) {
                <span class="spinner"></span> Authenticating...
              } @else {
                <span class="material-icons-round">lock_open</span> Sign In
              }
            </button>
          </form>

          <div class="sa-footer">
            <a routerLink="/maintenance" class="back-link" id="sa-back">
              <span class="material-icons-round">arrow_back</span> Back to maintenance page
            </a>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .sa-page {
      min-height: 100vh;
      display: flex; align-items: center; justify-content: center;
      background: #0a0e1a;
      position: relative; overflow: hidden;
      font-family: var(--font-body, 'Inter', sans-serif);
    }
    .sa-bg {
      position: absolute; inset: 0;
      background:
        radial-gradient(ellipse at 30% 20%, rgba(245, 158, 11, 0.06) 0%, transparent 50%),
        radial-gradient(ellipse at 70% 80%, rgba(245, 158, 11, 0.04) 0%, transparent 50%);
    }

    .sa-container {
      position: relative; z-index: 1;
      width: 100%; max-width: 420px;
      padding: 24px;
      animation: slideUp 0.4s ease-out;
    }
    @keyframes slideUp {
      from { opacity: 0; transform: translateY(20px); }
      to { opacity: 1; transform: translateY(0); }
    }

    .sa-card {
      background: rgba(15, 23, 42, 0.8);
      backdrop-filter: blur(20px);
      border: 1px solid rgba(245, 158, 11, 0.15);
      border-radius: 20px;
      padding: 40px 32px;
    }

    .sa-header {
      text-align: center; margin-bottom: 32px;
      h1 {
        font-size: 1.375rem; font-weight: 700;
        color: #f1f5f9; margin: 16px 0 6px;
      }
      p {
        font-size: 0.8125rem; color: #64748b; margin: 0;
      }
    }
    .shield-icon {
      width: 64px; height: 64px;
      margin: 0 auto;
      border-radius: 16px;
      background: linear-gradient(135deg, rgba(245, 158, 11, 0.15), rgba(217, 119, 6, 0.08));
      border: 1px solid rgba(245, 158, 11, 0.2);
      display: flex; align-items: center; justify-content: center;
      .material-icons-round { font-size: 32px; color: #f59e0b; }
    }

    .sa-form {
      display: flex; flex-direction: column; gap: 18px;
    }
    .sa-field {
      display: flex; flex-direction: column; gap: 6px;
      label {
        font-size: 0.8125rem; font-weight: 500; color: #94a3b8;
      }
    }
    .input-wrapper {
      position: relative; display: flex; align-items: center;
    }
    .input-icon {
      position: absolute; left: 14px;
      font-size: 18px; color: #475569;
      pointer-events: none;
    }
    .sa-field input {
      width: 100%;
      background: rgba(255, 255, 255, 0.04);
      border: 1px solid rgba(255, 255, 255, 0.1);
      border-radius: 12px;
      padding: 13px 14px 13px 44px;
      font-size: 0.9375rem;
      color: #e2e8f0;
      outline: none;
      transition: border-color 0.2s, box-shadow 0.2s;
      &:focus {
        border-color: rgba(245, 158, 11, 0.4);
        box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.08);
      }
      &::placeholder { color: #475569; }
    }
    .pw-toggle {
      position: absolute; right: 12px;
      background: none; border: none;
      color: #475569; cursor: pointer; padding: 4px;
      display: flex;
      .material-icons-round { font-size: 18px; }
      &:hover { color: #94a3b8; }
    }

    .sa-btn {
      display: flex; align-items: center; justify-content: center; gap: 8px;
      width: 100%;
      padding: 14px;
      border: none; border-radius: 12px;
      background: linear-gradient(135deg, #f59e0b, #d97706);
      color: #0a0e1a;
      font-size: 0.9375rem; font-weight: 700;
      cursor: pointer;
      margin-top: 6px;
      transition: all 0.2s;
      .material-icons-round { font-size: 18px; }
      &:hover:not(:disabled) { filter: brightness(1.1); transform: translateY(-1px); }
      &:disabled { opacity: 0.6; cursor: not-allowed; }
    }

    .spinner {
      display: inline-block; width: 16px; height: 16px;
      border: 2px solid rgba(0,0,0,0.2); border-top-color: currentColor;
      border-radius: 50%; animation: spin 600ms linear infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }

    .sa-footer {
      text-align: center; margin-top: 24px;
    }
    .back-link {
      display: inline-flex; align-items: center; gap: 4px;
      font-size: 0.8125rem; color: #64748b;
      text-decoration: none;
      transition: color 0.2s;
      .material-icons-round { font-size: 16px; }
      &:hover { color: #94a3b8; }
    }

    @media (max-width: 480px) {
      .sa-card { padding: 32px 20px; }
    }
  `]
})
export class SystemAdminLoginComponent {
  private auth = inject(AuthService);
  private router = inject(Router);
  private toast = inject(ToastService);

  phone = '';
  password = '';
  loading = signal(false);
  showPw = signal(false);

  onSubmit(): void {
    if (!this.phone || !this.password) {
      this.toast.warning('Please enter phone and password');
      return;
    }

    this.loading.set(true);
    this.auth.login(this.phone, this.password).subscribe({
      next: () => {
        const user = this.auth.currentUser();
        if (user?.system_role === 'SYSTEM_ADMIN') {
          this.toast.success('Welcome, Administrator');
          this.router.navigate(['/platform/command-center']);
        } else {
          this.toast.error('Access denied. Only System Administrators can log in during maintenance.');
          this.auth.logout();
        }
        this.loading.set(false);
      },
      error: (err) => {
        this.loading.set(false);
        const msg = err.error?.error?.message || err.error?.message || 'Invalid credentials';
        this.toast.error(msg);
      },
    });
  }
}
