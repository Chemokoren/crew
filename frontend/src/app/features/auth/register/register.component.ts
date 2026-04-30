import { Component, inject, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { SystemRole } from '../../../core/models';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="auth-page">
      <div class="auth-bg-pattern"></div>
      <div class="auth-container animate-slide-up">
        <div class="auth-card glass-card">
          <div class="auth-header">
            <div class="auth-logo">
              <span class="logo-icon">⚡</span>
              <h1 class="auth-title">AMY<span class="accent">MIS</span></h1>
            </div>
            <p class="auth-subtitle">Create your account</p>
          </div>

          <form class="auth-form" (ngSubmit)="onSubmit()" id="register-form">
            <div class="form-row">
              <div class="form-group">
                <label class="form-label" for="reg-first-name">First Name</label>
                <input class="form-input" id="reg-first-name" type="text" placeholder="John" [(ngModel)]="firstName" name="firstName" />
              </div>
              <div class="form-group">
                <label class="form-label" for="reg-last-name">Last Name</label>
                <input class="form-input" id="reg-last-name" type="text" placeholder="Doe" [(ngModel)]="lastName" name="lastName" />
              </div>
            </div>

            <div class="form-group">
              <label class="form-label" for="reg-phone">Phone Number</label>
              <input class="form-input" id="reg-phone" type="tel" placeholder="+254712345678" [(ngModel)]="phone" name="phone" required />
            </div>

            <div class="form-group">
              <label class="form-label" for="reg-email">Email (optional)</label>
              <input class="form-input" id="reg-email" type="email" placeholder="john@example.com" [(ngModel)]="email" name="email" />
            </div>

            <div class="form-group">
              <label class="form-label" for="reg-role">Role</label>
              <select class="form-select" id="reg-role" [(ngModel)]="role" name="role" required>
                <option value="CREW">Crew Member</option>
                <option value="SACCO_ADMIN">SACCO Admin</option>
              </select>
            </div>

            @if (role === 'CREW') {
              <div class="form-group">
                <label class="form-label" for="reg-national-id">National ID</label>
                <input class="form-input" id="reg-national-id" type="text" placeholder="12345678" [(ngModel)]="nationalId" name="nationalId" />
              </div>

              <div class="form-group">
                <label class="form-label" for="reg-crew-role">Crew Role</label>
                <select class="form-select" id="reg-crew-role" [(ngModel)]="crewRole" name="crewRole">
                  <option value="DRIVER">Driver</option>
                  <option value="CONDUCTOR">Conductor</option>
                  <option value="RIDER">Rider</option>
                  <option value="OTHER">Other</option>
                </select>
              </div>
            }

            <div class="form-group">
              <label class="form-label" for="reg-password">Password</label>
              <input class="form-input" id="reg-password" type="password" placeholder="Min 8 characters" [(ngModel)]="password" name="password" required minlength="8" autocomplete="new-password" />
            </div>

            <button class="btn btn-primary btn-lg btn-full" type="submit" [disabled]="loading()" id="register-submit">
              @if (loading()) {
                <span class="spinner"></span> Creating Account...
              } @else {
                <span class="material-icons-round">person_add</span> Create Account
              }
            </button>
          </form>

          <div class="auth-footer">
            <p>Already have an account? <a routerLink="/auth/login">Sign In</a></p>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .auth-page {
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--color-bg-primary);
      position: relative;
      overflow: hidden;
    }
    .auth-bg-pattern {
      position: absolute;
      inset: 0;
      background:
        radial-gradient(ellipse at 20% 50%, rgba(0, 210, 255, 0.06) 0%, transparent 50%),
        radial-gradient(ellipse at 80% 20%, rgba(123, 97, 255, 0.04) 0%, transparent 50%);
    }
    .auth-container { position: relative; z-index: 1; width: 100%; max-width: 480px; padding: var(--space-lg); }
    .auth-card { padding: var(--space-xl) !important; }
    .auth-header { text-align: center; margin-bottom: var(--space-xl); }
    .auth-logo { display: flex; align-items: center; justify-content: center; gap: var(--space-sm); margin-bottom: var(--space-sm); }
    .logo-icon { font-size: 28px; width: 44px; height: 44px; display: flex; align-items: center; justify-content: center; background: var(--gradient-accent); border-radius: var(--radius-md); }
    .auth-title { font-family: var(--font-heading); font-size: 1.75rem; font-weight: 800; }
    .accent { background: var(--gradient-accent); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
    .auth-subtitle { color: var(--color-text-muted); font-size: 0.875rem; }
    .auth-form { display: flex; flex-direction: column; gap: var(--space-sm); }
    .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-sm); }
    .btn-full { width: 100%; margin-top: var(--space-sm); }
    .spinner { width: 16px; height: 16px; border: 2px solid rgba(0,0,0,0.2); border-top-color: currentColor; border-radius: 50%; animation: spin 600ms linear infinite; }
    @keyframes spin { to { transform: rotate(360deg); } }
    .auth-footer { text-align: center; margin-top: var(--space-lg); font-size: 0.875rem; color: var(--color-text-muted); a { color: var(--color-accent); font-weight: 500; } }
  `]
})
export class RegisterComponent {
  private auth = inject(AuthService);
  private router = inject(Router);
  private toast = inject(ToastService);

  phone = '';
  email = '';
  password = '';
  firstName = '';
  lastName = '';
  nationalId = '';
  role: SystemRole = 'CREW';
  crewRole = 'DRIVER';
  loading = signal(false);

  onSubmit(): void {
    if (!this.phone || !this.password) {
      this.toast.warning('Phone and password are required');
      return;
    }
    this.loading.set(true);
    this.auth.register({
      phone: this.phone,
      email: this.email || undefined,
      password: this.password,
      role: this.role,
      first_name: this.firstName || undefined,
      last_name: this.lastName || undefined,
      national_id: this.role === 'CREW' ? this.nationalId : undefined,
      crew_role: this.role === 'CREW' ? this.crewRole : undefined,
    }).subscribe({
      next: () => {
        this.toast.success('Account created successfully!');
        this.router.navigate(['/dashboard']);
      },
      error: () => this.loading.set(false),
    });
  }
}
