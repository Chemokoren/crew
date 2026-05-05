import { Component, inject, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { SystemRole, IndustryType } from '../../../core/models';

interface IndustryOption {
  value: IndustryType;
  label: string;
  icon: string;
  description: string;
}

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
              <img src="logo.png" alt="AMY MIS Logo" class="dynamic-logo">
            </div>
            <p class="auth-subtitle">Create your account</p>
          </div>

          <form class="auth-form" (ngSubmit)="onSubmit()" id="register-form">
            <!-- Step indicator for SACCO_ADMIN -->
            @if (role === 'SACCO_ADMIN') {
              <div class="step-indicator">
                <div class="step" [class.active]="step() === 1" [class.completed]="step() > 1">
                  <span class="step-number">1</span>
                  <span class="step-label">Personal</span>
                </div>
                <div class="step-connector" [class.active]="step() > 1"></div>
                <div class="step" [class.active]="step() === 2">
                  <span class="step-number">2</span>
                  <span class="step-label">Organization</span>
                </div>
              </div>
            }

            <!-- STEP 1: Personal info (always shown for CREW, or step 1 for SACCO_ADMIN) -->
            @if (role === 'CREW' || step() === 1) {
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
                  <option value="SACCO_ADMIN">Organization Admin</option>
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

              @if (role === 'SACCO_ADMIN') {
                <button class="btn btn-primary btn-lg btn-full" type="button" (click)="nextStep()" id="register-next">
                  <span class="material-icons-round">arrow_forward</span> Next: Organization Details
                </button>
              } @else {
                <button class="btn btn-primary btn-lg btn-full" type="submit" [disabled]="loading()" id="register-submit">
                  @if (loading()) {
                    <span class="spinner"></span> Creating Account...
                  } @else {
                    <span class="material-icons-round">person_add</span> Create Account
                  }
                </button>
              }
            }

            <!-- STEP 2: Organization details (only for SACCO_ADMIN) -->
            @if (role === 'SACCO_ADMIN' && step() === 2) {
              <div class="form-group">
                <label class="form-label" for="reg-org-name">Organization Name</label>
                <input class="form-input" id="reg-org-name" type="text" placeholder="e.g. Metro Trans SACCO" [(ngModel)]="orgName" name="orgName" required />
              </div>

              <div class="form-group">
                <label class="form-label" for="reg-org-reg-no">Registration Number</label>
                <input class="form-input" id="reg-org-reg-no" type="text" placeholder="e.g. CS/2024/001234" [(ngModel)]="orgRegNo" name="orgRegNo" required />
              </div>

              <div class="form-row">
                <div class="form-group">
                  <label class="form-label" for="reg-org-county">County</label>
                  <input class="form-input" id="reg-org-county" type="text" placeholder="e.g. Nairobi" [(ngModel)]="orgCounty" name="orgCounty" required />
                </div>
                <div class="form-group">
                  <label class="form-label" for="reg-org-phone">Org Contact Phone</label>
                  <input class="form-input" id="reg-org-phone" type="tel" placeholder="+254700000000" [(ngModel)]="orgPhone" name="orgPhone" required />
                </div>
              </div>

              <div class="form-group">
                <label class="form-label" for="reg-industry">Organization Type / Industry</label>
                <div class="industry-grid">
                  @for (ind of industries; track ind.value) {
                    <button
                      type="button"
                      class="industry-card"
                      [class.selected]="industryType === ind.value"
                      (click)="selectIndustry(ind.value)"
                      [id]="'industry-' + ind.value.toLowerCase()">
                      <span class="material-icons-round industry-icon">{{ ind.icon }}</span>
                      <span class="industry-label">{{ ind.label }}</span>
                      <span class="industry-desc">{{ ind.description }}</span>
                    </button>
                  }
                </div>
              </div>

              <div class="form-row step-actions">
                <button class="btn btn-outline btn-lg" type="button" (click)="prevStep()" id="register-back">
                  <span class="material-icons-round">arrow_back</span> Back
                </button>
                <button class="btn btn-primary btn-lg" type="submit" [disabled]="loading()" id="register-submit">
                  @if (loading()) {
                    <span class="spinner"></span> Creating...
                  } @else {
                    <span class="material-icons-round">rocket_launch</span> Create Organization
                  }
                </button>
              </div>
            }
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
    .auth-container { position: relative; z-index: 1; width: 100%; max-width: 520px; padding: var(--space-lg); }
    .auth-card { padding: var(--space-xl) !important; }
    .auth-header { text-align: center; margin-bottom: var(--space-xl); }
    .auth-logo { display: flex; align-items: center; justify-content: center; gap: var(--space-sm); margin-bottom: var(--space-sm); }
    .auth-subtitle { color: var(--color-text-muted); font-size: 0.875rem; }
    .auth-form { display: flex; flex-direction: column; gap: var(--space-sm); }
    .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-sm); }
    .btn-full { width: 100%; margin-top: var(--space-sm); }
    .spinner { width: 16px; height: 16px; border: 2px solid rgba(0,0,0,0.2); border-top-color: currentColor; border-radius: 50%; animation: spin 600ms linear infinite; }
    @keyframes spin { to { transform: rotate(360deg); } }
    .auth-footer { text-align: center; margin-top: var(--space-lg); font-size: 0.875rem; color: var(--color-text-muted); a { color: var(--color-accent); font-weight: 500; } }

    /* Step indicator */
    .step-indicator {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0;
      margin-bottom: var(--space-md);
    }
    .step {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 4px;
      opacity: 0.4;
      transition: all 300ms ease;
    }
    .step.active, .step.completed {
      opacity: 1;
    }
    .step-number {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      background: var(--color-bg-tertiary);
      border: 2px solid var(--color-border);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.8125rem;
      font-weight: 700;
      color: var(--color-text-muted);
      transition: all 300ms ease;
    }
    .step.active .step-number {
      background: var(--gradient-accent);
      border-color: var(--color-accent);
      color: #fff;
    }
    .step.completed .step-number {
      background: rgba(0, 210, 255, 0.15);
      border-color: var(--color-accent);
      color: var(--color-accent);
    }
    .step-label {
      font-size: 0.6875rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--color-text-muted);
    }
    .step-connector {
      width: 60px;
      height: 2px;
      background: var(--color-border);
      margin: 0 var(--space-sm);
      margin-bottom: 18px;
      transition: background 300ms ease;
    }
    .step-connector.active {
      background: var(--color-accent);
    }

    /* Industry grid */
    .industry-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: var(--space-xs);
    }
    .industry-card {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 4px;
      padding: var(--space-sm) var(--space-xs);
      border-radius: var(--radius-md);
      background: var(--color-bg-secondary);
      border: 2px solid var(--color-border);
      cursor: pointer;
      transition: all 200ms ease;
      text-align: center;
    }
    .industry-card:hover {
      border-color: var(--color-border-hover);
      background: rgba(255,255,255,0.04);
      transform: translateY(-1px);
    }
    .industry-card.selected {
      border-color: var(--color-accent);
      background: rgba(0, 210, 255, 0.08);
      box-shadow: 0 0 0 1px var(--color-accent);
    }
    .industry-icon {
      font-size: 24px;
      color: var(--color-text-muted);
      transition: color 200ms ease;
    }
    .industry-card.selected .industry-icon {
      color: var(--color-accent);
    }
    .industry-label {
      font-size: 0.75rem;
      font-weight: 600;
      color: var(--color-text-primary);
    }
    .industry-desc {
      font-size: 0.625rem;
      color: var(--color-text-muted);
      line-height: 1.3;
    }

    /* Step actions */
    .step-actions {
      margin-top: var(--space-sm);
    }
    .btn-outline {
      background: transparent;
      border: 1px solid var(--color-border);
      color: var(--color-text-secondary);
    }
    .btn-outline:hover {
      border-color: var(--color-text-muted);
      color: var(--color-text-primary);
    }

    @media (max-width: 480px) {
      .industry-grid {
        grid-template-columns: repeat(2, 1fr);
      }
    }
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

  // Organization fields
  orgName = '';
  orgRegNo = '';
  orgCounty = '';
  orgPhone = '';
  industryType: IndustryType = 'TRANSPORT';
  step = signal(1);

  industries: IndustryOption[] = [
    { value: 'TRANSPORT', label: 'Transport', icon: 'directions_bus', description: 'SACCO, fleet mgmt' },
    { value: 'CONSTRUCTION', label: 'Construction', icon: 'construction', description: 'Sites & projects' },
    { value: 'HEALTH', label: 'Health', icon: 'health_and_safety', description: 'CHVs, clinics' },
    { value: 'LOGISTICS', label: 'Logistics', icon: 'local_shipping', description: 'Delivery & cargo' },
    { value: 'AGRICULTURE', label: 'Agriculture', icon: 'agriculture', description: 'Farms & coops' },
    { value: 'HOSPITALITY', label: 'Hospitality', icon: 'hotel', description: 'Hotels, restaurants' },
  ];

  selectIndustry(type: IndustryType): void {
    this.industryType = type;
  }

  nextStep(): void {
    if (!this.phone || !this.password) {
      this.toast.warning('Phone and password are required');
      return;
    }
    if (this.password.length < 8) {
      this.toast.warning('Password must be at least 8 characters');
      return;
    }
    this.step.set(2);
  }

  prevStep(): void {
    this.step.set(1);
  }

  onSubmit(): void {
    if (!this.phone || !this.password) {
      this.toast.warning('Phone and password are required');
      return;
    }

    // Validate org fields for SACCO_ADMIN
    if (this.role === 'SACCO_ADMIN') {
      if (!this.orgName) { this.toast.warning('Organization name is required'); return; }
      if (!this.orgRegNo) { this.toast.warning('Registration number is required'); return; }
      if (!this.orgCounty) { this.toast.warning('County is required'); return; }
      if (!this.orgPhone) { this.toast.warning('Organization phone is required'); return; }
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
      // Organization fields
      organization_name: this.role === 'SACCO_ADMIN' ? this.orgName : undefined,
      organization_reg_no: this.role === 'SACCO_ADMIN' ? this.orgRegNo : undefined,
      organization_county: this.role === 'SACCO_ADMIN' ? this.orgCounty : undefined,
      organization_phone: this.role === 'SACCO_ADMIN' ? this.orgPhone : undefined,
      industry_type: this.role === 'SACCO_ADMIN' ? this.industryType : undefined,
    }).subscribe({
      next: () => {
        this.toast.success('Account created successfully!');
        this.router.navigate(['/dashboard']);
      },
      error: () => this.loading.set(false),
    });
  }
}
