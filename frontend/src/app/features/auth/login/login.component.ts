import { Component, inject, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { environment } from '../../../../environments/environment';

@Component({
  selector: 'app-login',
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
            <p class="auth-subtitle">Workforce Financial Operating System</p>
          </div>

          <form class="auth-form" (ngSubmit)="onSubmit()" id="login-form">
            <div class="form-group">
              <label class="form-label" for="login-phone">Phone Number</label>
              <input
                class="form-input"
                id="login-phone"
                type="tel"
                placeholder="+254712345678"
                [(ngModel)]="phone"
                name="phone"
                required
                autocomplete="tel"
              />
            </div>

            <div class="form-group">
              <div class="label-row">
                <label class="form-label" for="login-password">Password</label>
                <button type="button" class="forgot-link" (click)="openForgotModal()" id="forgot-password-link">
                  Forgot password?
                </button>
              </div>
              <div class="password-wrapper">
                <input
                  class="form-input"
                  id="login-password"
                  [type]="showPassword() ? 'text' : 'password'"
                  placeholder="Enter your password"
                  [(ngModel)]="password"
                  name="password"
                  required
                  autocomplete="current-password"
                />
                <button type="button" class="password-toggle" (click)="showPassword.update(v => !v)">
                  <span class="material-icons-round">{{ showPassword() ? 'visibility_off' : 'visibility' }}</span>
                </button>
              </div>
            </div>

            <button class="btn btn-primary btn-lg btn-full" type="submit" [disabled]="loading()" id="login-submit">
              @if (loading()) {
                <span class="spinner"></span> Signing in...
              } @else {
                <span class="material-icons-round">login</span> Sign In
              }
            </button>
          </form>

          <div class="auth-footer">
            <p>Don't have an account? <a routerLink="/auth/register">Register</a></p>
          </div>
        </div>
      </div>
    </div>

    <!-- Forgot Password Modal — OTP-based self-service reset -->
    @if (showForgotModal()) {
      <div class="modal-backdrop" (click)="onBackdropClick()">
        <div class="modal-content forgot-modal" (click)="$event.stopPropagation()">
          <div class="modal-header">
            <h3>Reset Password</h3>
            <button class="btn btn-ghost btn-icon" (click)="closeForgotModal()">
              <span class="material-icons-round">close</span>
            </button>
          </div>

          <!-- Step indicator -->
          <div class="step-indicator">
            <div class="step-dot" [class.active]="forgotStep() === 'phone'" [class.done]="stepIndex() > 0">
              <span>1</span>
            </div>
            <div class="step-line" [class.done]="stepIndex() > 0"></div>
            <div class="step-dot" [class.active]="forgotStep() === 'otp'" [class.done]="stepIndex() > 1">
              <span>2</span>
            </div>
            <div class="step-line" [class.done]="stepIndex() > 1"></div>
            <div class="step-dot" [class.active]="forgotStep() === 'new_password'" [class.done]="stepIndex() > 2">
              <span>3</span>
            </div>
          </div>

          @switch (forgotStep()) {
            <!-- Step 1: Enter phone number -->
            @case ('phone') {
              <div class="modal-body">
                <p class="forgot-desc">Enter your registered phone number. We'll send a 6-digit verification code to reset your password.</p>
                <div class="form-group">
                  <label class="form-label" for="forgot-phone">Phone Number</label>
                  <input
                    class="form-input"
                    id="forgot-phone"
                    type="tel"
                    placeholder="0712345678"
                    [(ngModel)]="forgotPhone"
                    name="forgotPhone"
                    autocomplete="tel"
                  />
                </div>

                <div class="form-group">
                  <label class="form-label">Send code via</label>
                  <div class="channel-selector">
                    <button
                      type="button"
                      class="channel-btn"
                      [class.active]="selectedChannel === 'email'"
                      (click)="selectedChannel = 'email'"
                    >
                      <span class="material-icons-round">email</span>
                      <span>Email</span>
                    </button>
                    <button
                      type="button"
                      class="channel-btn"
                      [class.active]="selectedChannel === 'sms'"
                      (click)="selectedChannel = 'sms'"
                    >
                      <span class="material-icons-round">sms</span>
                      <span>SMS</span>
                    </button>
                    <button
                      type="button"
                      class="channel-btn"
                      [class.active]="selectedChannel === 'whatsapp'"
                      (click)="selectedChannel = 'whatsapp'"
                    >
                      <span class="material-icons-round">chat</span>
                      <span>WhatsApp</span>
                    </button>
                  </div>
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn btn-secondary" (click)="closeForgotModal()">Cancel</button>
                <button class="btn btn-primary" (click)="requestOTP()" [disabled]="sendingOTP()" id="forgot-send-otp">
                  @if (sendingOTP()) {
                    <span class="spinner-sm"></span> Sending...
                  } @else {
                    <span class="material-icons-round" style="font-size:16px;">send</span> Send OTP
                  }
                </button>
              </div>
            }

            <!-- Step 2: Enter OTP -->
            @case ('otp') {
              <div class="modal-body">
                <div class="otp-sent-card">
                  <div class="otp-icon-wrapper">
                    <span class="material-icons-round">mark_email_read</span>
                  </div>
                  <h4>Verification Code Sent</h4>
                  <p class="otp-desc">A 6-digit code has been sent via <strong>{{ channelLabel() }}</strong> to <strong>{{ forgotPhone }}</strong>. Enter it below to verify your identity.</p>
                </div>

                <div class="form-group">
                  <label class="form-label" for="otp-code">Verification Code</label>
                  <input
                    class="form-input otp-input"
                    id="otp-code"
                    type="text"
                    placeholder="000000"
                    [(ngModel)]="otpCode"
                    name="otpCode"
                    maxlength="6"
                    autocomplete="one-time-code"
                    inputmode="numeric"
                    pattern="[0-9]*"
                  />
                </div>

                <div class="otp-meta">
                  <span class="otp-timer">Code expires in 10 minutes</span>
                  <button type="button" class="resend-link" (click)="requestOTP()" [disabled]="sendingOTP()">
                    @if (sendingOTP()) { Sending... } @else { Resend Code }
                  </button>
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn btn-secondary" (click)="forgotStep.set('phone')">
                  <span class="material-icons-round" style="font-size:16px;">arrow_back</span> Back
                </button>
                <button class="btn btn-primary" (click)="verifyOTP()" [disabled]="verifyingOTP()" id="forgot-verify-otp">
                  @if (verifyingOTP()) {
                    <span class="spinner-sm"></span> Verifying...
                  } @else {
                    <span class="material-icons-round" style="font-size:16px;">verified</span> Verify Code
                  }
                </button>
              </div>
            }

            <!-- Step 3: Set new password -->
            @case ('new_password') {
              <div class="modal-body">
                <div class="otp-sent-card success-card">
                  <div class="success-icon-wrapper">
                    <span class="material-icons-round">check_circle</span>
                  </div>
                  <h4>Identity Verified</h4>
                  <p class="otp-desc">Set your new password below.</p>
                </div>

                <div class="form-group">
                  <label class="form-label" for="new-pw">New Password</label>
                  <input
                    class="form-input"
                    id="new-pw"
                    type="password"
                    placeholder="Min 8 characters"
                    [(ngModel)]="newPassword"
                    name="newPassword"
                    minlength="8"
                    autocomplete="new-password"
                  />
                </div>
                <div class="form-group">
                  <label class="form-label" for="confirm-pw">Confirm Password</label>
                  <input
                    class="form-input"
                    id="confirm-pw"
                    type="password"
                    placeholder="Re-enter new password"
                    [(ngModel)]="confirmNewPassword"
                    name="confirmNewPassword"
                    autocomplete="new-password"
                  />
                  @if (confirmNewPassword && newPassword !== confirmNewPassword) {
                    <span class="form-error">Passwords do not match</span>
                  }
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn btn-primary btn-full" (click)="resetPassword()" [disabled]="resettingPassword()" id="forgot-reset-pw">
                  @if (resettingPassword()) {
                    <span class="spinner-sm"></span> Resetting...
                  } @else {
                    <span class="material-icons-round" style="font-size:16px;">lock_reset</span> Reset Password
                  }
                </button>
              </div>
            }

            <!-- Step 4: Success -->
            @case ('success') {
              <div class="modal-body">
                <div class="otp-sent-card success-card">
                  <div class="success-icon-wrapper big">
                    <span class="material-icons-round">task_alt</span>
                  </div>
                  <h4>Password Reset Complete!</h4>
                  <p class="otp-desc">You can now sign in with your new password.</p>
                </div>
              </div>
              <div class="modal-footer">
                <button class="btn btn-primary btn-full" (click)="closeForgotModal()" id="forgot-success-done">
                  <span class="material-icons-round" style="font-size:16px;">login</span> Sign In
                </button>
              </div>
            }
          }
        </div>
      </div>
    }
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
        radial-gradient(ellipse at 80% 20%, rgba(123, 97, 255, 0.04) 0%, transparent 50%),
        radial-gradient(ellipse at 50% 80%, rgba(0, 210, 255, 0.03) 0%, transparent 50%);
    }

    .auth-container {
      position: relative;
      z-index: 1;
      width: 100%;
      max-width: 420px;
      padding: var(--space-lg);
    }

    .auth-card { padding: var(--space-xl) !important; }
    .auth-header { text-align: center; margin-bottom: var(--space-xl); }

    .auth-logo {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: var(--space-sm);
      margin-bottom: var(--space-sm);
    }

    .logo-icon {
      font-size: 28px;
      width: 44px;
      height: 44px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--gradient-accent);
      border-radius: var(--radius-md);
    }

    .auth-title { font-family: var(--font-heading); font-size: 1.75rem; font-weight: 800; }
    .accent { background: var(--gradient-accent); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
    .auth-subtitle { color: var(--color-text-muted); font-size: 0.875rem; }
    .auth-form { display: flex; flex-direction: column; gap: var(--space-md); }

    .label-row { display: flex; align-items: center; justify-content: space-between; }

    .forgot-link {
      background: none; border: none; color: var(--color-accent);
      font-size: 0.8125rem; font-weight: 500; cursor: pointer; padding: 0;
      transition: opacity var(--transition-fast);
      &:hover { opacity: 0.8; text-decoration: underline; }
    }

    .password-wrapper { position: relative; }
    .password-toggle {
      position: absolute; right: 12px; top: 50%; transform: translateY(-50%);
      background: none; border: none; color: var(--color-text-muted);
      cursor: pointer; padding: 0; display: flex;
      .material-icons-round { font-size: 18px; }
      &:hover { color: var(--color-text-secondary); }
    }

    .btn-full { width: 100%; margin-top: var(--space-sm); }

    .spinner {
      width: 16px; height: 16px;
      border: 2px solid rgba(0, 0, 0, 0.2); border-top-color: currentColor;
      border-radius: 50%; animation: spin 600ms linear infinite;
    }

    .spinner-sm {
      display: inline-block; width: 14px; height: 14px;
      border: 2px solid rgba(255, 255, 255, 0.2); border-top-color: currentColor;
      border-radius: 50%; animation: spin 600ms linear infinite;
    }

    @keyframes spin { to { transform: rotate(360deg); } }

    .auth-footer {
      text-align: center; margin-top: var(--space-lg);
      font-size: 0.875rem; color: var(--color-text-muted);
      a { color: var(--color-accent); font-weight: 500; }
    }

    /* Forgot Password Modal */
    .forgot-modal { max-width: 440px; }

    .forgot-desc {
      font-size: 0.875rem; color: var(--color-text-secondary);
      margin-bottom: var(--space-md); line-height: 1.5;
    }

    /* Step indicator */
    .step-indicator {
      display: flex; align-items: center; justify-content: center;
      gap: 0; padding: var(--space-md) var(--space-lg) 0;
    }

    .step-dot {
      width: 28px; height: 28px; border-radius: 50%;
      background: rgba(255, 255, 255, 0.06); border: 2px solid var(--color-border);
      display: flex; align-items: center; justify-content: center;
      font-size: 0.75rem; font-weight: 600; color: var(--color-text-muted);
      transition: all 300ms ease;
      flex-shrink: 0;

      &.active {
        background: var(--gradient-accent); border-color: transparent;
        color: var(--color-text-inverse); box-shadow: 0 0 12px rgba(0, 210, 255, 0.3);
      }

      &.done {
        background: var(--color-success); border-color: var(--color-success);
        color: white;
      }
    }

    .step-line {
      width: 40px; height: 2px; background: var(--color-border);
      transition: background 300ms ease;
      &.done { background: var(--color-success); }
    }

    /* OTP card */
    .otp-sent-card {
      text-align: center; padding: var(--space-md);
      border-radius: var(--radius-md);
      background: rgba(0, 210, 255, 0.04); border: 1px solid rgba(0, 210, 255, 0.1);
      margin-bottom: var(--space-lg);

      h4 { font-size: 1rem; font-weight: 600; margin-bottom: 4px; }
    }

    .otp-icon-wrapper {
      width: 48px; height: 48px; margin: 0 auto var(--space-sm); border-radius: 50%;
      background: rgba(0, 210, 255, 0.1);
      display: flex; align-items: center; justify-content: center;
      .material-icons-round { font-size: 28px; color: var(--color-accent); }
    }

    .success-card {
      background: rgba(0, 211, 130, 0.04); border-color: rgba(0, 211, 130, 0.1);
    }

    .success-icon-wrapper {
      width: 48px; height: 48px; margin: 0 auto var(--space-sm); border-radius: 50%;
      background: rgba(0, 211, 130, 0.12);
      display: flex; align-items: center; justify-content: center;
      .material-icons-round { font-size: 28px; color: var(--color-success); }

      &.big {
        width: 64px; height: 64px;
        .material-icons-round { font-size: 36px; }
      }
    }

    .otp-desc {
      font-size: 0.8125rem; color: var(--color-text-muted); line-height: 1.4;
      strong { color: var(--color-text-primary); }
    }

    .otp-input {
      text-align: center; font-size: 1.5rem; font-weight: 700;
      letter-spacing: 0.5em; font-family: var(--font-mono, monospace);
    }

    .otp-meta {
      display: flex; align-items: center; justify-content: space-between;
      margin-top: var(--space-xs);
    }

    .otp-timer { font-size: 0.75rem; color: var(--color-text-muted); }

    .resend-link {
      background: none; border: none; color: var(--color-accent);
      font-size: 0.8125rem; font-weight: 500; cursor: pointer; padding: 0;
      &:hover { text-decoration: underline; }
      &:disabled { opacity: 0.4; cursor: not-allowed; }
    }

    /* Channel selector */
    .channel-selector {
      display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px;
    }

    .channel-btn {
      display: flex; flex-direction: column; align-items: center; gap: 4px;
      padding: 10px 8px; border-radius: var(--radius-md);
      background: rgba(255, 255, 255, 0.03); border: 1px solid var(--color-border);
      color: var(--color-text-muted); cursor: pointer;
      transition: all 200ms ease; font-size: 0.75rem;

      .material-icons-round { font-size: 20px; }

      &:hover {
        background: rgba(0, 210, 255, 0.04); border-color: rgba(0, 210, 255, 0.2);
        color: var(--color-text-secondary);
      }

      &.active {
        background: rgba(0, 210, 255, 0.08); border-color: var(--color-accent);
        color: var(--color-accent); font-weight: 600;
        box-shadow: 0 0 8px rgba(0, 210, 255, 0.15);
      }
    }
  `]
})
export class LoginComponent {
  private auth = inject(AuthService);
  private http = inject(HttpClient);
  private router = inject(Router);
  private toast = inject(ToastService);

  phone = '';
  password = '';
  loading = signal(false);
  showPassword = signal(false);

  // Forgot password state
  showForgotModal = signal(false);
  forgotStep = signal<'phone' | 'otp' | 'new_password' | 'success'>('phone');
  forgotPhone = '';
  otpCode = '';
  resetToken = '';
  newPassword = '';
  confirmNewPassword = '';
  selectedChannel: 'email' | 'sms' | 'whatsapp' = 'email';

  sendingOTP = signal(false);
  verifyingOTP = signal(false);
  resettingPassword = signal(false);

  channelLabel(): string {
    const labels: Record<string, string> = { email: 'Email', sms: 'SMS', whatsapp: 'WhatsApp' };
    return labels[this.selectedChannel] || this.selectedChannel;
  }

  stepIndex(): number {
    const steps = ['phone', 'otp', 'new_password', 'success'];
    return steps.indexOf(this.forgotStep());
  }

  /** Only allow backdrop-close on the first step; mid-flow clicks are ignored. */
  onBackdropClick(): void {
    if (this.forgotStep() === 'phone') {
      this.closeForgotModal();
    }
    // On otp/new_password/success steps, clicking outside does nothing
  }

  onSubmit(): void {
    if (!this.phone || !this.password) {
      this.toast.warning('Please enter phone and password');
      return;
    }

    this.loading.set(true);
    this.auth.login(this.phone, this.password).subscribe({
      next: () => {
        this.toast.success('Welcome back!');
        this.router.navigate(['/dashboard']);
      },
      error: (err) => {
        this.loading.set(false);
        const msg = err.error?.error?.message || err.error?.message || 'Invalid phone number or password';
        this.toast.error(msg);
      },
    });
  }

  openForgotModal(): void {
    this.showForgotModal.set(true);
    this.forgotStep.set('phone');
    this.forgotPhone = this.phone; // Pre-fill from login form if available
  }

  closeForgotModal(): void {
    this.showForgotModal.set(false);
    this.forgotStep.set('phone');
    this.forgotPhone = '';
    this.otpCode = '';
    this.resetToken = '';
    this.newPassword = '';
    this.confirmNewPassword = '';
    this.selectedChannel = 'email';
  }

  // Step 1: Request OTP via selected channel
  requestOTP(): void {
    if (!this.forgotPhone) {
      this.toast.warning('Please enter your phone number');
      return;
    }

    this.sendingOTP.set(true);
    this.http.post<any>(`${environment.apiUrl}/auth/forgot-password`, {
      phone: this.forgotPhone,
      channel: this.selectedChannel
    }, {
      headers: { 'X-Skip-Error-Toast': 'true' }
    }).subscribe({
      next: () => {
        this.toast.success('Verification code sent to your phone');
        this.forgotStep.set('otp');
        this.sendingOTP.set(false);
      },
      error: (err) => {
        this.sendingOTP.set(false);
        const msg = err.error?.message || err.error?.error || 'Failed to send OTP';
        this.toast.error(msg);
      },
    });
  }

  // Step 2: Verify OTP
  verifyOTP(): void {
    if (!this.otpCode || this.otpCode.length !== 6) {
      this.toast.warning('Please enter the 6-digit code');
      return;
    }

    this.verifyingOTP.set(true);
    this.http.post<any>(`${environment.apiUrl}/auth/verify-otp`, {
      phone: this.forgotPhone,
      otp: this.otpCode
    }, {
      headers: { 'X-Skip-Error-Toast': 'true' }
    }).subscribe({
      next: (res) => {
        this.resetToken = res.data?.reset_token || '';
        this.forgotStep.set('new_password');
        this.verifyingOTP.set(false);
      },
      error: (err) => {
        this.verifyingOTP.set(false);
        const msg = err.error?.message || err.error?.error || 'Invalid verification code';
        this.toast.error(msg);
      },
    });
  }

  // Step 3: Reset password
  resetPassword(): void {
    if (!this.newPassword || this.newPassword.length < 8) {
      this.toast.warning('Password must be at least 8 characters');
      return;
    }
    if (this.newPassword !== this.confirmNewPassword) {
      this.toast.error('Passwords do not match');
      return;
    }

    this.resettingPassword.set(true);
    this.http.post<any>(`${environment.apiUrl}/auth/reset-password`, {
      phone: this.forgotPhone,
      reset_token: this.resetToken,
      new_password: this.newPassword
    }, {
      headers: { 'X-Skip-Error-Toast': 'true' }
    }).subscribe({
      next: () => {
        this.forgotStep.set('success');
        this.resettingPassword.set(false);
      },
      error: (err) => {
        this.resettingPassword.set(false);
        const msg = err.error?.message || err.error?.error || 'Failed to reset password';
        this.toast.error(msg);
      },
    });
  }
}
