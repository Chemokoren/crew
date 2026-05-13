import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule, KeyValuePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { ToastService } from '../../core/services/toast.service';
import { AuthService } from '../../core/services/auth.service';
import { ConfirmDialogService } from '../../shared/components/confirm-dialog/confirm-dialog.component';
import { Organization, TenantJobType, PaySchedule, IndustryType, PayFrequency, JobTypeCategory, BootstrapResult } from '../../core/models';
import { getIndustryTemplate, INDUSTRY_TEMPLATES, INDUSTRY_ICONS } from '../../core/config/industry-templates';
import { OrgContextService } from '../../core/services/org-context.service';
import { AutocompleteComponent, AutocompleteOption } from '../../shared/components/autocomplete/autocomplete.component';

const INDUSTRIES: { value: IndustryType; label: string; icon: string; desc: string }[] = [
  { value: 'TRANSPORT', label: 'Transport', icon: 'directions_bus', desc: 'SACCOs, matatus, boda-bodas — fleet management with routes, vehicles, and daily shifts' },
  { value: 'CONSTRUCTION', label: 'Construction', icon: 'construction', desc: 'Contractors and builders — project-based work with sites, daily rates, and foreman oversight' },
  { value: 'HEALTH', label: 'Health', icon: 'health_and_safety', desc: 'NGOs and health facilities — community health visits, coverage areas, and stipend tracking' },
  { value: 'LOGISTICS', label: 'Logistics', icon: 'local_shipping', desc: 'Delivery and warehousing — task-based deliveries, riders, and dispatch management' },
  { value: 'AGRICULTURE', label: 'Agriculture', icon: 'agriculture', desc: 'Cooperatives and farms — seasonal workers, piece-rate pay, and harvest tracking' },
  { value: 'HOSPITALITY', label: 'Hospitality', icon: 'hotel', desc: 'Hotels, restaurants, and catering — shift-based staff, locations, and tip pooling' },
  { value: 'GENERAL', label: 'General', icon: 'business', desc: 'Generic workforce management — all features enabled, fully customizable' },
  { value: 'CUSTOM' as IndustryType, label: 'Custom', icon: 'tune', desc: 'Build your own configuration — all assignment types, earning models, and frequencies unlocked' },
];

const PAY_FREQUENCIES: { value: PayFrequency; label: string }[] = [
  { value: 'DAILY', label: 'Daily' },
  { value: 'WEEKLY', label: 'Weekly' },
  { value: 'BI_WEEKLY', label: 'Bi-Weekly' },
  { value: 'MONTHLY', label: 'Monthly' },
];

const JOB_CATEGORIES: { value: JobTypeCategory; label: string; desc: string }[] = [
  { value: 'PRIMARY', label: 'Primary', desc: 'Core workers (drivers, masons, CHVs)' },
  { value: 'FACILITATOR', label: 'Facilitator', desc: 'Booking agents, touts, recruiters' },
  { value: 'SUPPORT', label: 'Support', desc: 'Office staff, clerks, admin' },
  { value: 'SUPERVISOR', label: 'Supervisor', desc: 'Foremen, team leads, coordinators' },
];

@Component({
  selector: 'app-tenant-settings',
  standalone: true,
  imports: [CommonModule, FormsModule, AutocompleteComponent, KeyValuePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Tenant Settings</h1>
          <p class="page-subtitle">Configure industry, job types, and pay schedules for your organization</p>
        </div>
      </div>

      <!-- System Admin: Tenant Selector -->
      @if (isSystemAdmin()) {
        <div class="tenant-selector">
          <div class="tenant-selector-label">
            <span class="material-icons-round" style="font-size:18px;color:var(--color-accent);">business</span>
            <span>Select Organization</span>
          </div>
          <div class="tenant-selector-input">
            <app-autocomplete
              [ngModel]="saccoId"
              (ngModelChange)="onTenantSelected($event)"
              [options]="orgOptions()"
              placeholder="Search organizations by name..."
              inputId="tenant-search"
            ></app-autocomplete>
          </div>
          @if (sacco()) {
            <div class="tenant-selector-badge">
              <span class="material-icons-round" style="font-size:14px;">check_circle</span>
              {{ sacco()!.name }}
            </div>
          }
        </div>
      }

      @if (loading()) {
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));">
          @for (i of [1,2,3]; track i) { <div class="skeleton" style="height:200px;border-radius:var(--radius-lg);"></div> }
        </div>
      } @else if (!sacco()) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">settings</span>
          <div class="empty-title">No organization found</div>
          <div class="empty-description">You need to be assigned to an organization to configure settings.</div>
        </div>
      } @else {

        <!-- Tab Nav -->
        <div class="tab-nav">
          <button class="tab-item" [class.active]="activeTab === 'industry'" (click)="activeTab='industry'">
            <span class="material-icons-round" style="font-size:16px;">factory</span> Industry
          </button>
          <button class="tab-item" [class.active]="activeTab === 'jobs'" (click)="activeTab='jobs'">
            <span class="material-icons-round" style="font-size:16px;">work</span> Job Types
          </button>
          <button class="tab-item" [class.active]="activeTab === 'schedules'" (click)="activeTab='schedules'">
            <span class="material-icons-round" style="font-size:16px;">schedule</span> Pay Schedules
          </button>
          <button class="tab-item" [class.active]="activeTab === 'kyc'" (click)="activeTab='kyc'">
            <span class="material-icons-round" style="font-size:16px;">verified_user</span> KYC Policy
          </button>
          <button class="tab-item" [class.active]="activeTab === 'finance'" (click)="activeTab='finance'">
            <span class="material-icons-round" style="font-size:16px;">account_balance</span> Finance
          </button>
        </div>

        <!-- Industry Tab -->
        @if (activeTab === 'industry') {
          <div class="glass-card ts-section">
            <h3 class="ts-section-title">Industry Type</h3>
            <p class="ts-section-desc">Select the industry vertical for your organization. This controls which UI fields, scoring weights, and product recommendations are shown.</p>
            <div class="ts-industry-grid">
              @for (ind of industries; track ind.value) {
                <button class="ts-industry-card" [class.selected]="sacco()!.industry_type === ind.value"
                        (click)="selectIndustry(ind.value)" [id]="'industry-' + ind.value"
                        [title]="ind.desc">
                  <span class="material-icons-round ts-industry-icon">{{ ind.icon }}</span>
                  <span class="ts-industry-label">{{ ind.label }}</span>
                  <span class="ts-industry-desc">{{ ind.desc }}</span>
                  @if (sacco()!.industry_type === ind.value) {
                    <span class="material-icons-round ts-check">check_circle</span>
                  }
                </button>
              }
            </div>

            <!-- Bootstrap Preview -->
            @if (bootstrapPreview()) {
              <div class="ts-bootstrap-preview">
                <h4 style="margin:0 0 8px; display:flex; align-items:center; gap:8px;">
                  <span class="material-icons-round" style="color:var(--color-accent);">auto_fix_high</span>
                  Template Preview: {{ bootstrapPreview()!.display_label }}
                </h4>
                <div class="ts-preview-grid">
                  <div class="ts-preview-item">
                    <span class="ts-preview-label">Job Types</span>
                    <div class="ts-preview-tags">
                      @for (j of bootstrapPreview()!.default_job_types; track j.code) {
                        <span class="ts-preview-tag" [ngClass]="'cat-' + j.category.toLowerCase()">{{ j.display_name }}</span>
                      }
                    </div>
                  </div>
                  <div class="ts-preview-item">
                    <span class="ts-preview-label">Earning Models</span>
                    <div class="ts-preview-tags">
                      @for (e of bootstrapPreview()!.earning_models; track e) {
                        <span class="ts-preview-tag">{{ e }}</span>
                      }
                    </div>
                  </div>
                  <div class="ts-preview-item">
                    <span class="ts-preview-label">Pay Frequencies</span>
                    <div class="ts-preview-tags">
                      @for (f of bootstrapPreview()!.payment_frequencies; track f) {
                        <span class="ts-preview-tag">{{ f }}</span>
                      }
                    </div>
                  </div>
                  @if (bootstrapPreview()!.statutory_bodies.length > 0) {
                    <div class="ts-preview-item">
                      <span class="ts-preview-label">Statutory Deductions</span>
                      <div class="ts-preview-tags">
                        @for (s of bootstrapPreview()!.statutory_bodies; track s) {
                          <span class="ts-preview-tag cat-support">{{ s }}</span>
                        }
                      </div>
                    </div>
                  }
                  <div class="ts-preview-item">
                    <span class="ts-preview-label">UI Labels</span>
                    <div class="ts-preview-tags">
                      @for (key of labelKeys(bootstrapPreview()!.ui_labels); track key) {
                        <span class="ts-preview-tag">{{ key }}: {{ bootstrapPreview()!.ui_labels[key] }}</span>
                      }
                    </div>
                  </div>
                </div>
                @if (!bootstrapResult()) {
                  <button class="btn btn-primary" (click)="confirmBootstrap()" [disabled]="bootstrapping()" style="margin-top:var(--space-md);">
                    <span class="material-icons-round" style="font-size:16px;">rocket_launch</span>
                    {{ bootstrapping() ? 'Bootstrapping...' : 'Apply Template & Seed Defaults' }}
                  </button>
                } @else {
                  <div class="ts-bootstrap-result">
                    <span class="material-icons-round" style="color:var(--color-success);font-size:20px;">check_circle</span>
                    <div>
                      <strong>Bootstrap Complete</strong>
                       @if (bootstrapResult()!.job_types_seeded.length) {
                        <div>Seeded {{ bootstrapResult()!.job_types_seeded.length }} job types</div>
                      }
                      @if (bootstrapResult()!.job_types_skipped) {
                        <div style="color:var(--color-warning);">Job types skipped (already exist)</div>
                      }
                       @if (bootstrapResult()!.schedules_seeded.length) {
                        <div>Seeded {{ bootstrapResult()!.schedules_seeded.length }} pay schedules</div>
                      }
                      @if (bootstrapResult()!.schedules_skipped) {
                        <div style="color:var(--color-warning);">Pay schedules skipped (already exist)</div>
                      }
                    </div>
                  </div>
                }
              </div>
            }
          </div>
        }

        <!-- Job Types Tab -->
        @if (activeTab === 'jobs') {
          <div class="glass-card ts-section">
            <div class="ts-section-header">
              <div>
                <h3 class="ts-section-title">Job Types</h3>
                <p class="ts-section-desc">Define the {{ orgCtx.label('worker').toLowerCase() }} roles available in your organization. These replace the default role dropdown on the {{ orgCtx.label('worker') }} creation form.</p>
              </div>
              <button class="btn btn-primary btn-sm" (click)="openJobModal()" id="btn-add-job-type">
                <span class="material-icons-round" style="font-size:16px;">add</span> Add Job Type
              </button>
            </div>

            @if (jobTypes().length === 0) {
              <div class="empty-state" style="padding:var(--space-lg);">
                <span class="material-icons-round empty-icon">work_off</span>
                <div class="empty-title">No custom job types</div>
                <div class="empty-description">Add job types like Mason, CHV, Site Foreman, etc.</div>
              </div>
            } @else {
              <div class="ts-job-grid">
                @for (j of jobTypes(); track j.id) {
                  <div class="ts-job-card" [class.inactive]="!j.is_active">
                    <div class="ts-job-header">
                      <code class="ts-job-code">{{ j.code }}</code>
                      <span class="badge" [ngClass]="categoryBadge(j.category)">{{ j.category }}</span>
                    </div>
                    <div class="ts-job-name">{{ j.display_name }}</div>
                    <div class="ts-job-actions">
                      <button class="btn btn-ghost btn-sm" (click)="editJobType(j)" [id]="'edit-job-' + j.id">
                        <span class="material-icons-round" style="font-size:14px;">edit</span>
                      </button>
                      <button class="btn btn-ghost btn-sm" style="color:var(--color-danger);" (click)="deleteJobType(j)" [id]="'del-job-' + j.id">
                        <span class="material-icons-round" style="font-size:14px;">delete</span>
                      </button>
                    </div>
                  </div>
                }
              </div>
            }
          </div>
        }

        <!-- Pay Schedules Tab -->
        @if (activeTab === 'schedules') {
          <div class="glass-card ts-section">
            <div class="ts-section-header">
              <div>
                <h3 class="ts-section-title">Pay Schedules</h3>
                <p class="ts-section-desc">Configure payment frequencies and paydays for workers in your organization.</p>
              </div>
              <button class="btn btn-primary btn-sm" (click)="openScheduleModal()" id="btn-add-schedule">
                <span class="material-icons-round" style="font-size:16px;">add</span> Add Schedule
              </button>
            </div>

            @if (paySchedules().length === 0) {
              <div class="empty-state" style="padding:var(--space-lg);">
                <span class="material-icons-round empty-icon">event_busy</span>
                <div class="empty-title">No pay schedules</div>
                <div class="empty-description">Add schedules like "Daily Cash", "Weekly Friday", "Monthly 28th", etc.</div>
              </div>
            } @else {
              <div class="ts-schedule-grid">
                @for (s of paySchedules(); track s.id) {
                  <div class="ts-schedule-card" [class.is-default]="s.is_default">
                    <div class="ts-schedule-header">
                      <div class="ts-schedule-freq">
                        <span class="material-icons-round" style="font-size:18px;color:var(--color-accent);">{{ scheduleIcon(s.frequency) }}</span>
                        {{ s.frequency }}
                      </div>
                      @if (s.is_default) { <span class="badge badge-success">Default</span> }
                    </div>
                    <div class="ts-schedule-name">{{ s.name }}</div>
                    <div class="ts-schedule-meta">
                      @if (s.pay_day) { <span>Pay day: {{ s.pay_day }}</span> }
                      <span>Cutoff: {{ s.cutoff_hour }}:00</span>
                    </div>
                    <div class="ts-job-actions">
                      <button class="btn btn-ghost btn-sm" (click)="editSchedule(s)" [id]="'edit-sched-' + s.id">
                        <span class="material-icons-round" style="font-size:14px;">edit</span>
                      </button>
                      <button class="btn btn-ghost btn-sm" style="color:var(--color-danger);" (click)="deleteSchedule(s)" [id]="'del-sched-' + s.id">
                        <span class="material-icons-round" style="font-size:14px;">delete</span>
                      </button>
                    </div>
                  </div>
                }
              </div>
            }
          </div>
        }

        <!-- KYC Policy Tab -->
        @if (activeTab === 'kyc') {
          <div class="glass-card ts-section">
            <h3 class="ts-section-title">KYC Verification Policy</h3>
            <p class="ts-section-desc">Control identity verification requirements for your employees. When KYC is required, unverified employees will be restricted from performing the actions you select below.</p>

            <div class="kyc-policy-grid">
              <!-- KYC Required Toggle -->
              <div class="kyc-policy-card">
                <div class="kyc-policy-header">
                  <div>
                    <div class="kyc-policy-title">Require KYC Verification</div>
                    <div class="kyc-policy-desc">When enabled, employees must verify their identity before accessing restricted features.</div>
                  </div>
                  <label class="toggle-switch">
                    <input type="checkbox" [(ngModel)]="kycForm.kyc_required" (ngModelChange)="saveKYCConfig()" id="kyc-required-toggle" />
                    <span class="toggle-slider"></span>
                  </label>
                </div>
              </div>

              <!-- Verification Mode -->
              <div class="kyc-policy-card">
                <div class="kyc-policy-title">Verification Method</div>
                <div class="kyc-policy-desc" style="margin-bottom:var(--space-sm);">How employees prove their identity.</div>
                <div class="kyc-mode-options">
                  <label class="kyc-mode-option" [class.selected]="kycForm.kyc_verification_mode === 'UPLOAD'">
                    <input type="radio" name="kycMode" value="UPLOAD" [(ngModel)]="kycForm.kyc_verification_mode" (ngModelChange)="saveKYCConfig()" />
                    <span class="material-icons-round" style="font-size:20px;">cloud_upload</span>
                    <div>
                      <strong>Upload ID Photos</strong>
                      <span>Employee uploads front &amp; back of their National ID</span>
                    </div>
                  </label>
                  <label class="kyc-mode-option" [class.selected]="kycForm.kyc_verification_mode === 'MANUAL'">
                    <input type="radio" name="kycMode" value="MANUAL" [(ngModel)]="kycForm.kyc_verification_mode" (ngModelChange)="saveKYCConfig()" />
                    <span class="material-icons-round" style="font-size:20px;">edit_note</span>
                    <div>
                      <strong>Enter ID Details</strong>
                      <span>Employee enters ID number &amp; serial for IPRS lookup</span>
                    </div>
                  </label>
                </div>
              </div>

              <!-- Restricted Actions -->
              <div class="kyc-policy-card" style="grid-column: 1 / -1;">
                <div class="kyc-policy-title">Restricted Actions</div>
                <div class="kyc-policy-desc" style="margin-bottom:var(--space-md);">Select which features are blocked for unverified employees. By default, all employee features are restricted until KYC is verified.</div>
                <div class="kyc-actions-grid">
                  @for (action of allRestrictableActions; track action.code) {
                    <label class="kyc-action-item" [class.checked]="isActionRestricted(action.code)">
                      <input type="checkbox" [checked]="isActionRestricted(action.code)" (change)="toggleAction(action.code)" />
                      <span class="material-icons-round kyc-action-icon">{{ action.icon }}</span>
                      <div class="kyc-action-info">
                        <span class="kyc-action-name">{{ action.label }}</span>
                        <span class="kyc-action-desc">{{ action.desc }}</span>
                      </div>
                    </label>
                  }
                </div>
              </div>
            </div>
          </div>
        }

        <!-- Finance Tab -->
        @if (activeTab === 'finance') {
          <div class="glass-card ts-section">
            <h3 class="ts-section-title">Float Top-Up Verification</h3>
            <p class="ts-section-desc">Control how bank and card top-up references are verified before crediting the organization's float balance. This protects against unauthorized float inflation.</p>

            <div class="kyc-policy-grid" style="grid-template-columns: 1fr;">
              <div class="kyc-policy-card">
                <div class="kyc-policy-title" style="margin-bottom:var(--space-sm);">Verification Method</div>
                <div class="kyc-mode-options">
                  <label class="kyc-mode-option" [class.selected]="financeForm.topup_verification_mode === 'HYBRID'">
                    <input type="radio" name="topupVerifyMode" value="HYBRID" [(ngModel)]="financeForm.topup_verification_mode" (ngModelChange)="saveFinanceConfig()" />
                    <span class="material-icons-round" style="font-size:20px;color:#8b5cf6;">sync_alt</span>
                    <div>
                      <strong>Hybrid <span style='font-size:0.65rem;background:rgba(139,92,246,0.15);color:#8b5cf6;padding:2px 6px;border-radius:4px;margin-left:4px;'>Recommended</span></strong>
                      <span>Try bank API verification first. If the API is unavailable, fall back to manual admin approval. Best of both worlds.</span>
                    </div>
                  </label>
                  <label class="kyc-mode-option" [class.selected]="financeForm.topup_verification_mode === 'API'">
                    <input type="radio" name="topupVerifyMode" value="API" [(ngModel)]="financeForm.topup_verification_mode" (ngModelChange)="saveFinanceConfig()" />
                    <span class="material-icons-round" style="font-size:20px;color:#0ea5e9;">api</span>
                    <div>
                      <strong>API Only</strong>
                      <span>Strictly verify via bank API integration. Top-ups are rejected if the API is unavailable or the reference is invalid. Highest security.</span>
                    </div>
                  </label>
                  <label class="kyc-mode-option" [class.selected]="financeForm.topup_verification_mode === 'MANUAL'">
                    <input type="radio" name="topupVerifyMode" value="MANUAL" [(ngModel)]="financeForm.topup_verification_mode" (ngModelChange)="saveFinanceConfig()" />
                    <span class="material-icons-round" style="font-size:20px;color:#f59e0b;">person_search</span>
                    <div>
                      <strong>Manual Only</strong>
                      <span>All bank/card top-ups require manual admin approval. No API calls are made. Use when no bank integration is available.</span>
                    </div>
                  </label>
                </div>

                <!-- Mode explanation card -->
                <div style="margin-top:var(--space-md);padding:var(--space-md);border-radius:var(--radius-md);background:rgba(99,102,241,0.04);border:1px solid rgba(99,102,241,0.15);">
                  <div style="display:flex;align-items:center;gap:6px;margin-bottom:6px;">
                    <span class="material-icons-round" style="font-size:16px;color:var(--color-accent);">info</span>
                    <span style="font-size:0.78rem;font-weight:600;">How it works</span>
                  </div>
                  @if (financeForm.topup_verification_mode === 'API') {
                    <p style="font-size:0.75rem;color:var(--color-text-muted);margin:0;line-height:1.5;">
                      When an admin submits a bank top-up, the system calls the bank API to verify the transaction reference and amount match.
                      If verified, the float is credited immediately. If not found or amounts mismatch, the top-up is <strong>rejected</strong>.
                      If the API is down, the top-up is <strong>blocked</strong> until the API is available.
                    </p>
                  } @else if (financeForm.topup_verification_mode === 'MANUAL') {
                    <p style="font-size:0.75rem;color:var(--color-text-muted);margin:0;line-height:1.5;">
                      All bank/card top-ups are created as <strong>pending</strong> transactions. An admin must manually verify the bank reference
                      and then click "Confirm" in the Wallet page to credit the float. Unverified top-ups can be rejected.
                    </p>
                  } @else {
                    <p style="font-size:0.75rem;color:var(--color-text-muted);margin:0;line-height:1.5;">
                      The system first attempts to verify via bank API. If the API confirms the reference, the float is credited automatically.
                      If the API is unavailable or inconclusive, the top-up is created as <strong>pending</strong> for manual admin review.
                      This provides automation when possible with a safety net when it's not.
                    </p>
                  }
                </div>
              </div>
            </div>
          </div>

          <!-- Allowed Top-Up Methods -->
          <div class="glass-card ts-section" style="margin-top:var(--space-md);">
            <h3 class="ts-section-title">Allowed Top-Up Methods</h3>
            <p class="ts-section-desc">Enable or disable specific top-up methods and channels for this organization. Disabled methods and channels will not appear in the wallet dashboard.</p>

            <div class="topup-methods-grid">
              @for (method of allTopUpMethods; track method.id) {
                <div class="topup-method-card" [class.topup-method-card--disabled]="!isTopUpMethodEnabled(method.id)">
                  <!-- Method Header -->
                  <div class="topup-method-header">
                    <div class="topup-method-info">
                      <span class="material-icons-round" [style.color]="method.color" style="font-size:24px;">{{ method.icon }}</span>
                      <div>
                        <div class="topup-method-label">{{ method.label }}</div>
                        <div class="topup-method-desc">{{ method.desc }}</div>
                      </div>
                    </div>
                    <label class="toggle-switch" (click)="$event.stopPropagation()">
                      <input type="checkbox" [checked]="isTopUpMethodEnabled(method.id)"
                        (change)="toggleTopUpMethod(method.id)" [id]="'toggle-method-' + method.id" />
                      <span class="toggle-slider"></span>
                    </label>
                  </div>

                  <!-- Channel List (only shown when method is enabled) -->
                  @if (isTopUpMethodEnabled(method.id)) {
                    <div class="topup-channels">
                      <div class="topup-channels-label">
                        <span class="material-icons-round" style="font-size:14px;color:var(--color-text-muted);">tune</span>
                        Enabled Channels
                      </div>
                      @for (ch of method.channels; track ch.id) {
                        <label class="topup-channel-item" [class.topup-channel--enabled]="isChannelEnabled(ch.id)">
                          <div class="topup-channel-info">
                            <span class="topup-channel-emoji">{{ ch.emoji }}</span>
                            <span class="topup-channel-name">{{ ch.label }}</span>
                          </div>
                          <label class="toggle-switch toggle-switch--sm" (click)="$event.stopPropagation()">
                            <input type="checkbox" [checked]="isChannelEnabled(ch.id)"
                              (change)="toggleChannel(ch.id, method.id)" [id]="'toggle-ch-' + ch.id" />
                            <span class="toggle-slider"></span>
                          </label>
                        </label>
                      }
                    </div>
                  }
                </div>
              }
            </div>

            @if (financeForm.allowed_topup_methods.length === 0) {
              <div style="margin-top:var(--space-sm);padding:var(--space-sm) var(--space-md);border-radius:var(--radius-md);background:rgba(99,102,241,0.04);border:1px solid rgba(99,102,241,0.15);">
                <div style="display:flex;align-items:center;gap:6px;">
                  <span class="material-icons-round" style="font-size:16px;color:var(--color-accent);">info</span>
                  <span style="font-size:0.75rem;color:var(--color-text-muted);font-weight:500;">All methods and channels are enabled by default when none are explicitly selected. Toggle a method to start customizing.</span>
                </div>
              </div>
            }
          </div>

          <!-- Payroll / Statutory Deductions -->
          <div class="glass-card ts-section" style="margin-top:var(--space-md);">
            <h3 class="ts-section-title">Payroll Configuration</h3>
            <p class="ts-section-desc">Control statutory remittance and which deduction types appear in the Pay Employee modal. All deductions are <strong>off by default</strong> — suitable for informal workers who manage their own contributions.</p>

            <!-- Statutory toggle -->
            <div class="kyc-policy-card" style="margin-bottom:var(--space-md);">
              <div class="kyc-policy-header">
                <div>
                  <div class="kyc-policy-title">Handle Statutory Deductions</div>
                  <div class="kyc-policy-desc">
                    When <strong>enabled</strong>, NSSF, SHA (NHIF), and Housing Levy fields appear in the Pay Employee modal.
                    Enable for <strong>formal sector employers</strong> required to remit statutory contributions.
                    <br><span style="color:var(--color-accent);font-size:0.72rem;font-weight:600;">⚠ Default: Off — informal worker mode.</span>
                  </div>
                </div>
                <label class="toggle-switch">
                  <input type="checkbox" [(ngModel)]="financeForm.handle_statutory_deductions" (ngModelChange)="saveFinanceConfig()" id="toggle-statutory-deductions" />
                  <span class="toggle-slider"></span>
                </label>
              </div>
              @if (financeForm.handle_statutory_deductions) {
                <div style="margin-top:var(--space-sm);padding:8px 12px;border-radius:var(--radius-sm);background:rgba(34,197,94,0.06);border:1px solid rgba(34,197,94,0.2);display:flex;align-items:center;gap:8px;">
                  <span class="material-icons-round" style="color:var(--color-success);font-size:16px;">check_circle</span>
                  <span style="font-size:0.75rem;color:var(--color-success);">Formal sector mode — NSSF, SHA, and Housing Levy active in Pay Employee modal.</span>
                </div>
              }
            </div>

            <!-- Deduction Types -->
            <div style="border-top:1px solid var(--color-border);padding-top:var(--space-md);">
              <div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:var(--space-md);">
                <div>
                  <div style="font-size:0.9rem;font-weight:700;color:var(--color-text-primary);margin-bottom:3px;">Deduction Types</div>
                  <div style="font-size:0.78rem;color:var(--color-text-muted);">Enable the deduction categories that apply to your employees. Disabled types are hidden from the payout modal.</div>
                </div>
              </div>

              <!-- Standard deductions -->
              <div style="display:flex;flex-direction:column;gap:var(--space-sm);margin-bottom:var(--space-md);">
                @for (ded of standardDeductionDefs; track ded.code) {
                  <div class="kyc-policy-card" style="padding:var(--space-sm) var(--space-md);">
                    <div class="kyc-policy-header">
                      <div>
                        <div class="kyc-policy-title" style="font-size:0.85rem;">{{ ded.label }}</div>
                        <div class="kyc-policy-desc" style="font-size:0.72rem;">{{ ded.desc }}</div>
                      </div>
                      <label class="toggle-switch">
                        <input type="checkbox"
                               [checked]="isDeductionEnabled(ded.code)"
                               (change)="toggleDeduction(ded.code)"
                               [id]="'toggle-ded-' + ded.code" />
                        <span class="toggle-slider"></span>
                      </label>
                    </div>
                  </div>
                }
              </div>

              <!-- Custom deductions -->
              @if (financeForm.custom_deduction_labels | keyvalue) {
                @if ((financeForm.custom_deduction_labels | keyvalue).length > 0) {
                  <div style="margin-bottom:var(--space-sm);">
                    <div style="font-size:0.72rem;font-weight:700;text-transform:uppercase;letter-spacing:0.05em;color:var(--color-text-muted);margin-bottom:6px;">Custom Deductions</div>
                    <div style="display:flex;flex-direction:column;gap:6px;">
                      @for (entry of financeForm.custom_deduction_labels | keyvalue; track entry.key) {
                        <div class="kyc-policy-card" style="padding:var(--space-sm) var(--space-md);">
                          <div class="kyc-policy-header">
                            <div>
                              <div class="kyc-policy-title" style="font-size:0.85rem;">{{ entry.value }}</div>
                              <div class="kyc-policy-desc" style="font-size:0.7rem;font-family:monospace;">{{ entry.key }}</div>
                            </div>
                            <div style="display:flex;align-items:center;gap:var(--space-sm);">
                              <label class="toggle-switch">
                                <input type="checkbox"
                                       [checked]="isDeductionEnabled(entry.key)"
                                       (change)="toggleDeduction(entry.key)"
                                       [id]="'toggle-custom-' + entry.key" />
                                <span class="toggle-slider"></span>
                              </label>
                              <button class="btn btn-ghost btn-sm" style="color:var(--color-danger);padding:4px;"
                                      (click)="removeCustomDeduction(entry.key)" [id]="'remove-ded-' + entry.key">
                                <span class="material-icons-round" style="font-size:16px;">delete</span>
                              </button>
                            </div>
                          </div>
                        </div>
                      }
                    </div>
                  </div>
                }
              }

              <!-- Add custom deduction -->
              <div style="padding:var(--space-md);border:1px dashed var(--color-border);border-radius:var(--radius-md);background:var(--color-bg-secondary);">
                <div style="font-size:0.78rem;font-weight:600;color:var(--color-text-primary);margin-bottom:var(--space-sm);">
                  <span class="material-icons-round" style="font-size:15px;vertical-align:middle;color:var(--color-accent);">add_circle</span>
                  Add Custom Deduction
                </div>
                <div style="display:flex;gap:var(--space-sm);align-items:flex-end;">
                  <div style="flex:1;">
                    <label class="form-label" style="font-size:0.72rem;margin-bottom:4px;">Deduction Name</label>
                    <input class="form-input" [(ngModel)]="newDeductionLabel"
                           placeholder="e.g. Motor Vehicle Loan, SACCO Savings"
                           id="input-custom-deduction" style="font-size:0.82rem;">
                  </div>
                  <button class="btn btn-primary btn-sm" (click)="addCustomDeduction()"
                          [disabled]="!newDeductionLabel.trim()" id="btn-add-custom-deduction">
                    <span class="material-icons-round" style="font-size:15px;">add</span> Add
                  </button>
                </div>
                <div style="font-size:0.68rem;color:var(--color-text-muted);margin-top:6px;">
                  A unique code is auto-generated from the name. The deduction will be enabled immediately after adding.
                </div>
              </div>
            </div>
          </div>
        }

        <!-- Job Type Modal -->
        @if (showModal() === 'job') {
          <div class="modal-backdrop" (click)="closeModal()"><div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header"><h3>{{ editingJobId ? 'Edit' : 'Add' }} Job Type</h3><button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button></div>
            <div class="modal-body">
              <label class="form-label">Code (unique identifier)</label>
              <input class="form-input" [(ngModel)]="jobForm.code" placeholder="e.g. MASON, CHV, FOREMAN" id="job-code" style="text-transform:uppercase;">
              <label class="form-label" style="margin-top:var(--space-sm);">Display Name</label>
              <input class="form-input" [(ngModel)]="jobForm.display_name" placeholder="e.g. Mason, Community Health Volunteer" id="job-name">
              <label class="form-label" style="margin-top:var(--space-sm);">Category</label>
              <select class="form-select" [(ngModel)]="jobForm.category" id="job-category">
                @for (c of jobCategories; track c.value) { <option [value]="c.value">{{ c.label }} — {{ c.desc }}</option> }
              </select>
            </div>
            <div class="modal-footer"><button class="btn btn-ghost" (click)="closeModal()">Cancel</button><button class="btn btn-primary" (click)="submitJob()" [disabled]="submitting()" id="btn-submit-job">{{ submitting() ? 'Saving...' : 'Save' }}</button></div>
          </div></div>
        }

        <!-- Schedule Modal -->
        @if (showModal() === 'schedule') {
          <div class="modal-backdrop" (click)="closeModal()"><div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header"><h3>{{ editingScheduleId ? 'Edit' : 'Add' }} Pay Schedule</h3><button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button></div>
            <div class="modal-body">
              <label class="form-label">Schedule Name</label>
              <input class="form-input" [(ngModel)]="schedForm.name" placeholder="e.g. Daily Cash, Weekly Friday" id="sched-name">
              <label class="form-label" style="margin-top:var(--space-sm);">Frequency</label>
              <select class="form-select" [(ngModel)]="schedForm.frequency" id="sched-freq">
                @for (f of payFrequencies; track f.value) { <option [value]="f.value">{{ f.label }}</option> }
              </select>
              @if (schedForm.frequency !== 'DAILY') {
                <label class="form-label" style="margin-top:var(--space-sm);">Pay Day {{ schedForm.frequency === 'WEEKLY' ? '(1=Mon, 7=Sun)' : '(1-31)' }}</label>
                <input class="form-input" type="number" [(ngModel)]="schedForm.pay_day" [min]="1" [max]="schedForm.frequency === 'WEEKLY' ? 7 : 31" id="sched-payday">
              }
              <label class="form-label" style="margin-top:var(--space-sm);">Cutoff Hour (24h)</label>
              <input class="form-input" type="number" [(ngModel)]="schedForm.cutoff_hour" min="0" max="23" id="sched-cutoff">
              <label class="form-label" style="margin-top:var(--space-sm);">
                <input type="checkbox" [(ngModel)]="schedForm.is_default" id="sched-default" style="margin-right:6px;"> Set as Default Schedule
              </label>
            </div>
            <div class="modal-footer"><button class="btn btn-ghost" (click)="closeModal()">Cancel</button><button class="btn btn-primary" (click)="submitSchedule()" [disabled]="submitting()" id="btn-submit-sched">{{ submitting() ? 'Saving...' : 'Save' }}</button></div>
          </div></div>
        }
      }
    </div>
  `,
  styles: [`
    .ts-section { padding: var(--space-lg) !important; margin-top: var(--space-md); }
    .ts-section-title { font-size: 1rem; font-weight: 700; margin-bottom: 4px; }
    .ts-section-desc { font-size: 0.8rem; color: var(--color-text-muted); margin-bottom: var(--space-md); }
    .ts-section-header { display: flex; justify-content: space-between; align-items: flex-start; }

    /* Tenant Selector */
    .tenant-selector {
      display: flex; align-items: center; gap: var(--space-md); flex-wrap: wrap;
      padding: var(--space-md) var(--space-lg);
      background: rgba(99,102,241,0.04); border: 1px solid rgba(99,102,241,0.15);
      border-radius: var(--radius-lg); margin-bottom: var(--space-lg);
    }
    .tenant-selector-label {
      display: flex; align-items: center; gap: 6px;
      font-size: 0.85rem; font-weight: 600; color: var(--color-text-primary); white-space: nowrap;
    }
    .tenant-selector-input { flex: 1; min-width: 280px; max-width: 480px; position: relative; z-index: 20; }
    .tenant-selector-badge {
      display: inline-flex; align-items: center; gap: 4px;
      padding: 4px 12px; border-radius: var(--radius-pill);
      background: rgba(34,197,94,0.1); color: #22c55e;
      font-size: 0.78rem; font-weight: 600;
    }

    .ts-industry-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: var(--space-md); }
    .ts-industry-card {
      display: flex; flex-direction: column; align-items: center; gap: 8px; padding: var(--space-lg);
      border: 2px solid var(--color-border); border-radius: var(--radius-lg);
      background: rgba(255,255,255,0.02); cursor: pointer; transition: all 0.2s; position: relative;
    }
    .ts-industry-card:hover { border-color: var(--color-accent); transform: translateY(-2px); }
    .ts-industry-card.selected { border-color: var(--color-accent); background: rgba(99,102,241,0.08); }
    .ts-industry-icon { font-size: 2rem; color: var(--color-text-secondary); }
    .ts-industry-card.selected .ts-industry-icon { color: var(--color-accent); }
    .ts-industry-label { font-size: 0.85rem; font-weight: 600; }
    .ts-industry-desc { font-size: 0.65rem; color: var(--color-text-muted); text-align: center; line-height: 1.35; opacity: 0.7; max-width: 180px; }
    .ts-industry-card:hover .ts-industry-desc { opacity: 1; }
    .ts-check { position: absolute; top: 8px; right: 8px; font-size: 18px; color: var(--color-success); }

    .ts-job-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: var(--space-md); }
    .ts-job-card {
      border: 1px solid var(--color-border); border-radius: var(--radius-md); padding: var(--space-md);
      background: rgba(255,255,255,0.02); transition: border-color 0.2s;
    }
    .ts-job-card:hover { border-color: var(--color-accent); }
    .ts-job-card.inactive { opacity: 0.5; }
    .ts-job-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
    .ts-job-code { font-size: 0.7rem; background: rgba(99,102,241,0.12); color: var(--color-accent); padding: 2px 8px; border-radius: 4px; }
    .ts-job-name { font-size: 0.9rem; font-weight: 600; margin-bottom: 6px; }
    .ts-job-actions { display: flex; gap: 4px; justify-content: flex-end; }

    .ts-schedule-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: var(--space-md); }
    .ts-schedule-card {
      border: 1px solid var(--color-border); border-radius: var(--radius-md); padding: var(--space-md);
      background: rgba(255,255,255,0.02); transition: border-color 0.2s;
    }
    .ts-schedule-card:hover { border-color: var(--color-accent); }
    .ts-schedule-card.is-default { border-color: var(--color-success); }
    .ts-schedule-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
    .ts-schedule-freq { display: flex; align-items: center; gap: 6px; font-size: 0.75rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: var(--color-text-muted); }
    .ts-schedule-name { font-size: 0.9rem; font-weight: 600; margin-bottom: 4px; }
    .ts-schedule-meta { display: flex; gap: var(--space-md); font-size: 0.75rem; color: var(--color-text-muted); margin-bottom: 8px; }

    .badge-primary { background: rgba(99,102,241,0.15); color: #818cf8; }
    .badge-facilitator { background: rgba(251,146,60,0.12); color: #fb923c; }
    .badge-support { background: rgba(168,85,247,0.12); color: #a855f7; }
    .badge-supervisor { background: rgba(56,189,248,0.12); color: #38bdf8; }

    @media (max-width: 768px) {
      .ts-industry-grid { grid-template-columns: repeat(3, 1fr); }
      .ts-section-header { flex-direction: column; gap: var(--space-sm); }
    }

    /* Bootstrap Preview (AD-13) */
    .ts-bootstrap-preview {
      margin-top: var(--space-lg);
      padding: var(--space-lg);
      border: 1px solid rgba(99,102,241,0.3);
      border-radius: var(--radius-lg);
      background: rgba(99,102,241,0.04);
      animation: fadeSlideUp 0.3s ease;
    }
    @keyframes fadeSlideUp {
      from { opacity: 0; transform: translateY(12px); }
      to { opacity: 1; transform: translateY(0); }
    }
    .ts-preview-grid {
      display: flex; flex-direction: column; gap: var(--space-md);
    }
    .ts-preview-item {}
    .ts-preview-label {
      display: block; font-size: 0.7rem; font-weight: 700;
      text-transform: uppercase; letter-spacing: 0.05em;
      color: var(--color-text-muted); margin-bottom: 6px;
    }
    .ts-preview-tags { display: flex; flex-wrap: wrap; gap: 6px; }
    .ts-preview-tag {
      font-size: 0.72rem; padding: 3px 10px; border-radius: 100px;
      background: rgba(255,255,255,0.06); border: 1px solid var(--color-border);
      color: var(--color-text-secondary); font-weight: 500;
    }
    .ts-preview-tag.cat-primary { background: rgba(99,102,241,0.12); color: #818cf8; border-color: rgba(99,102,241,0.3); }
    .ts-preview-tag.cat-facilitator { background: rgba(251,146,60,0.1); color: #fb923c; border-color: rgba(251,146,60,0.3); }
    .ts-preview-tag.cat-supervisor { background: rgba(56,189,248,0.1); color: #38bdf8; border-color: rgba(56,189,248,0.3); }
    .ts-preview-tag.cat-support { background: rgba(168,85,247,0.1); color: #a855f7; border-color: rgba(168,85,247,0.3); }

    .ts-bootstrap-result {
      display: flex; align-items: flex-start; gap: 10px;
      margin-top: var(--space-md); padding: var(--space-md);
      border-radius: var(--radius-md);
      background: rgba(34,197,94,0.06); border: 1px solid rgba(34,197,94,0.2);
      font-size: 0.82rem;
    }
    .ts-bootstrap-result div { font-size: 0.78rem; color: var(--color-text-secondary); }

    /* KYC Policy */
    .kyc-policy-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); }
    @media (max-width: 768px) { .kyc-policy-grid { grid-template-columns: 1fr; } }
    .kyc-policy-card {
      border: 1px solid var(--color-border); border-radius: var(--radius-md);
      padding: var(--space-lg); background: rgba(255,255,255,0.02);
    }
    .kyc-policy-header { display: flex; justify-content: space-between; align-items: flex-start; gap: var(--space-md); }
    .kyc-policy-title { font-size: 0.9rem; font-weight: 700; color: var(--color-text-primary); margin-bottom: 4px; }
    .kyc-policy-desc { font-size: 0.78rem; color: var(--color-text-muted); line-height: 1.4; }
    /* Toggle Switch */
    .toggle-switch { position: relative; display: inline-block; width: 48px; height: 26px; flex-shrink: 0; }
    .toggle-switch input { opacity: 0; width: 0; height: 0; }
    .toggle-slider {
      position: absolute; cursor: pointer; inset: 0;
      background: var(--color-border); border-radius: 26px; transition: all 0.25s;
    }
    .toggle-slider::before {
      content: ''; position: absolute; width: 20px; height: 20px;
      left: 3px; bottom: 3px; background: #fff; border-radius: 50%; transition: transform 0.25s;
    }
    .toggle-switch input:checked + .toggle-slider { background: var(--color-accent, #6366f1); }
    .toggle-switch input:checked + .toggle-slider::before { transform: translateX(22px); }
    /* Mode Options */
    .kyc-mode-options { display: flex; flex-direction: column; gap: var(--space-sm); }
    .kyc-mode-option {
      display: flex; align-items: flex-start; gap: var(--space-sm); padding: var(--space-sm) var(--space-md);
      border: 1px solid var(--color-border); border-radius: var(--radius-sm); cursor: pointer;
      transition: all 0.2s;
    }
    .kyc-mode-option input { display: none; }
    .kyc-mode-option:hover { border-color: var(--color-accent); }
    .kyc-mode-option.selected { border-color: var(--color-accent); background: rgba(99,102,241,0.06); }
    .kyc-mode-option div { display: flex; flex-direction: column; }
    .kyc-mode-option strong { font-size: 0.82rem; }
    .kyc-mode-option span { font-size: 0.72rem; color: var(--color-text-muted); }
    /* Actions Grid */
    .kyc-actions-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: var(--space-sm); }
    .kyc-action-item {
      display: flex; align-items: center; gap: var(--space-sm); padding: 10px var(--space-md);
      border: 1px solid var(--color-border); border-radius: var(--radius-sm); cursor: pointer;
      transition: all 0.2s; background: rgba(255,255,255,0.01);
    }
    .kyc-action-item:hover { border-color: var(--color-accent); }
    .kyc-action-item.checked { border-color: rgba(239,68,68,0.4); background: rgba(239,68,68,0.04); }
    .kyc-action-item input { accent-color: var(--color-danger, #ef4444); width: 16px; height: 16px; flex-shrink: 0; }
    .kyc-action-icon { font-size: 20px; color: var(--color-text-muted); flex-shrink: 0; }
    .kyc-action-item.checked .kyc-action-icon { color: var(--color-danger, #ef4444); }
    .kyc-action-info { display: flex; flex-direction: column; }
    .kyc-action-name { font-size: 0.82rem; font-weight: 600; color: var(--color-text-primary); }
    .kyc-action-desc { font-size: 0.68rem; color: var(--color-text-muted); }

    /* Top-Up Method / Channel Grid */
    .topup-methods-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: var(--space-md); }
    .topup-method-card {
      border: 1px solid var(--color-border); border-radius: var(--radius-lg);
      padding: var(--space-md); background: rgba(255,255,255,0.02);
      transition: all 0.25s ease;
    }
    .topup-method-card:hover { border-color: var(--color-accent); }
    .topup-method-card--disabled { opacity: 0.45; }
    .topup-method-header {
      display: flex; align-items: center; justify-content: space-between; gap: var(--space-sm);
    }
    .topup-method-info { display: flex; align-items: center; gap: var(--space-sm); }
    .topup-method-label { font-size: 0.9rem; font-weight: 700; color: var(--color-text-primary); }
    .topup-method-desc { font-size: 0.72rem; color: var(--color-text-muted); margin-top: 2px; }

    .topup-channels {
      margin-top: var(--space-md); padding-top: var(--space-sm);
      border-top: 1px dashed var(--color-border);
      animation: fadeSlideUp 0.25s ease;
    }
    .topup-channels-label {
      display: flex; align-items: center; gap: 4px;
      font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em;
      color: var(--color-text-muted); margin-bottom: var(--space-sm);
    }
    .topup-channel-item {
      display: flex; align-items: center; justify-content: space-between;
      padding: 8px 12px; border-radius: var(--radius-sm);
      border: 1px solid transparent; cursor: pointer;
      transition: all 0.2s; margin-bottom: 4px;
    }
    .topup-channel-item:hover { background: rgba(99,102,241,0.04); }
    .topup-channel--enabled {
      border-color: rgba(34,197,94,0.2); background: rgba(34,197,94,0.04);
    }
    .topup-channel-info { display: flex; align-items: center; gap: 8px; }
    .topup-channel-emoji { font-size: 1rem; }
    .topup-channel-name { font-size: 0.82rem; font-weight: 500; color: var(--color-text-primary); }

    /* Small toggle variant */
    .toggle-switch--sm { width: 36px; height: 20px; }
    .toggle-switch--sm .toggle-slider::before { width: 14px; height: 14px; }
    .toggle-switch--sm input:checked + .toggle-slider::before { transform: translateX(16px); }
  `]
})
export class TenantSettingsComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmDialogService);
  readonly orgCtx = inject(OrgContextService);

  sacco = signal<Organization | null>(null);
  jobTypes = signal<TenantJobType[]>([]);
  paySchedules = signal<PaySchedule[]>([]);
  loading = signal(true);
  showModal = signal<string | null>(null);
  submitting = signal(false);

  /** All organizations for the admin autocomplete selector */
  allOrgs = signal<Organization[]>([]);
  orgOptions = computed<AutocompleteOption[]>(() => {
    return this.allOrgs().map(o => ({
      value: o.id,
      label: o.name,
      sublabel: o.industry_type ? `${o.industry_type} · ${o.county || ''}` : o.county || '',
      badge: o.is_active ? 'Active' : 'Inactive',
      searchText: `${o.name} ${o.registration_number || ''} ${o.county || ''} ${o.industry_type || ''}`,
    }));
  });

  activeTab = 'industry';
  saccoId = '';
  editingJobId = '';
  editingScheduleId = '';

  readonly industries = INDUSTRIES;
  readonly payFrequencies = PAY_FREQUENCIES;
  readonly jobCategories = JOB_CATEGORIES;

  jobForm = { code: '', display_name: '', category: 'PRIMARY' as JobTypeCategory };
  schedForm = { name: '', frequency: 'DAILY' as PayFrequency, pay_day: 0, cutoff_hour: 17, is_default: false };

  // --- KYC Policy ---
  kycForm = {
    kyc_required: true,
    kyc_verification_mode: 'UPLOAD' as 'UPLOAD' | 'MANUAL',
    kyc_restricted_actions: [] as string[],
  };
  savingKYC = signal(false);

  // --- Finance Config ---
  financeForm = {
    topup_verification_mode: 'HYBRID' as 'API' | 'MANUAL' | 'HYBRID',
    allowed_topup_methods: [] as string[],
    allowed_topup_channels: [] as string[],
    handle_statutory_deductions: false,
    enabled_deductions: [] as string[],
    custom_deduction_labels: {} as Record<string, string>,
  };

  /** Standard deduction definitions shown in Settings */
  readonly standardDeductionDefs = [
    { code: 'LOAN',      label: 'Loan Repayment', desc: 'Loan instalments deducted before payout (e.g. SACCO loans, bank loans).' },
    { code: 'INSURANCE', label: 'Insurance',       desc: 'Insurance premiums deducted before payout.' },
    { code: 'OTHER',     label: 'Other',            desc: 'Any other agreed deduction not covered above.' },
  ];

  /** New custom deduction label input field */
  newDeductionLabel = '';

  /** Top-up method definitions with nested channels */
  readonly allTopUpMethods = [
    { id: 'mobile_money', icon: 'phone_android', label: 'Mobile Money', desc: 'Enable/disable mobile money providers', color: '#22c55e',
      channels: [
        { id: 'mpesa', label: 'M-Pesa (Safaricom)', emoji: '🟢' },
        { id: 'airtel', label: 'Airtel Money', emoji: '🔴' },
        { id: 'tkash', label: 'T-Kash (Telkom)', emoji: '🔵' },
      ] },
    { id: 'bank', icon: 'account_balance', label: 'Bank Transfer', desc: 'Enable/disable bank transfer channels', color: '#3b82f6',
      channels: [
        { id: 'kcb', label: 'KCB Bank', emoji: '🏦' },
        { id: 'equity', label: 'Equity Bank', emoji: '🏦' },
        { id: 'coop', label: 'Co-operative Bank', emoji: '🏦' },
        { id: 'rtgs', label: 'RTGS (Real-time)', emoji: '⚡' },
      ] },
    { id: 'card', icon: 'credit_card', label: 'Card Payment', desc: 'Enable/disable card networks', color: '#a855f7',
      channels: [
        { id: 'visa', label: 'Visa', emoji: '💳' },
        { id: 'mastercard', label: 'Mastercard', emoji: '💳' },
      ] },
  ];

  readonly allRestrictableActions: { code: string; label: string; icon: string; desc: string }[] = [
    { code: 'WALLET_WITHDRAW', label: 'Wallet Withdrawal', icon: 'account_balance_wallet', desc: 'Withdraw funds from wallet' },
    { code: 'WALLET_TRANSFER', label: 'Wallet Transfer', icon: 'swap_horiz', desc: 'Transfer funds between wallets' },
    { code: 'BILL_PAY', label: 'Bill Payment', icon: 'receipt_long', desc: 'Pay bills from wallet' },
    { code: 'LOAN_APPLY', label: 'Loan Application', icon: 'request_quote', desc: 'Apply for micro-loans' },
    { code: 'PAYOUT', label: 'Payout', icon: 'payments', desc: 'Receive salary/wage payouts' },
    { code: 'INSURANCE_ENROLL', label: 'Insurance Enrollment', icon: 'health_and_safety', desc: 'Enroll in insurance policies' },
    { code: 'ASSIGNMENT_ACCEPT', label: 'Accept Assignments', icon: 'assignment_turned_in', desc: 'Accept work assignments' },
    { code: 'PROFILE_EDIT', label: 'Edit Profile', icon: 'edit', desc: 'Edit profile information' },
    { code: 'DOCUMENT_UPLOAD', label: 'Upload Documents', icon: 'upload_file', desc: 'Upload documents beyond KYC' },
    { code: 'CREDIT_SCORE_VIEW', label: 'View Credit Score', icon: 'analytics', desc: 'Access credit score details' },
  ];

  ngOnInit(): void {
    const user = this.auth.currentUser();
    this.saccoId = user?.organization_id || '';
    if (this.saccoId) {
      this.loadAll();
    } else if (this.auth.isAdmin()) {
      // System admin — load all orgs for the selector
      this.api.getOrganizations({ per_page: '200' }).subscribe({
        next: r => {
          this.allOrgs.set(r.data);
          if (r.data.length > 0) {
            this.saccoId = r.data[0].id;
            this.loadAll();
          } else {
            this.loading.set(false);
          }
        },
        error: () => this.loading.set(false),
      });
    } else {
      // EMPLOYER without org — show empty state
      this.loading.set(false);
    }
  }

  isSystemAdmin(): boolean {
    return this.auth.isAdmin();
  }

  /** Called when the admin selects a different org from the autocomplete */
  onTenantSelected(orgId: string): void {
    if (!orgId || orgId === this.saccoId) return;
    this.saccoId = orgId;
    this.sacco.set(null);
    this.jobTypes.set([]);
    this.paySchedules.set([]);
    this.bootstrapPreview.set(null);
    this.bootstrapResult.set(null);
    this.loadAll();
  }

  loadAll(): void {
    this.loading.set(true);
    this.api.getOrganization(this.saccoId).subscribe({
      next: r => {
        this.sacco.set(r.data);
        this.loading.set(false);
        // Hydrate KYC form from tenant config
        const cfg = r.data.tenant_config;
        if (cfg) {
          this.kycForm.kyc_required = (cfg as any).kyc_required ?? true;
          this.kycForm.kyc_verification_mode = (cfg as any).kyc_verification_mode || 'UPLOAD';
          const actions = (cfg as any).kyc_restricted_actions as string[] | undefined;
          // Default: all actions restricted
          this.kycForm.kyc_restricted_actions = actions && actions.length > 0
            ? [...actions]
            : this.allRestrictableActions.map(a => a.code);
          // Hydrate finance form
          this.financeForm.topup_verification_mode = (cfg as any).topup_verification_mode || 'HYBRID';
          this.financeForm.allowed_topup_methods = (cfg as any).allowed_topup_methods || [];
          this.financeForm.allowed_topup_channels = (cfg as any).allowed_topup_channels || [];
          this.financeForm.handle_statutory_deductions = (cfg as any).handle_statutory_deductions === true;
          this.financeForm.enabled_deductions = (cfg as any).enabled_deductions || [];
          this.financeForm.custom_deduction_labels = (cfg as any).custom_deduction_labels || {};
        } else {
          // No config yet — default everything off
          this.kycForm.kyc_required = true;
          this.kycForm.kyc_verification_mode = 'UPLOAD';
          this.kycForm.kyc_restricted_actions = this.allRestrictableActions.map(a => a.code);
          this.financeForm.topup_verification_mode = 'HYBRID';
          this.financeForm.allowed_topup_methods = [];
          this.financeForm.allowed_topup_channels = [];
          this.financeForm.handle_statutory_deductions = false;
          this.financeForm.enabled_deductions = [];
          this.financeForm.custom_deduction_labels = {};
        }
      },
      error: () => this.loading.set(false),
    });
    this.api.getJobTypes(this.saccoId).subscribe({
      next: r => this.jobTypes.set(r.data || []),
    });
    this.api.getPaySchedules(this.saccoId).subscribe({
      next: r => this.paySchedules.set(r.data || []),
    });
  }

  // --- KYC Policy Methods ---
  isActionRestricted(code: string): boolean {
    return this.kycForm.kyc_restricted_actions.includes(code);
  }

  toggleAction(code: string): void {
    const idx = this.kycForm.kyc_restricted_actions.indexOf(code);
    if (idx >= 0) {
      this.kycForm.kyc_restricted_actions.splice(idx, 1);
    } else {
      this.kycForm.kyc_restricted_actions.push(code);
    }
    this.saveKYCConfig();
  }

  saveKYCConfig(): void {
    this.savingKYC.set(true);
    this.api.updateTenantConfig(this.saccoId, {
      tenant_config: {
        kyc_required: this.kycForm.kyc_required,
        kyc_verification_mode: this.kycForm.kyc_verification_mode,
        kyc_restricted_actions: this.kycForm.kyc_restricted_actions,
        topup_verification_mode: this.financeForm.topup_verification_mode,
        allowed_topup_methods: this.financeForm.allowed_topup_methods,
        allowed_topup_channels: this.financeForm.allowed_topup_channels,
        handle_statutory_deductions: this.financeForm.handle_statutory_deductions,
        enabled_deductions: this.financeForm.enabled_deductions,
        custom_deduction_labels: this.financeForm.custom_deduction_labels,
      },
    }).subscribe({
      next: r => {
        this.sacco.set(r.data);
        this.savingKYC.set(false);
        this.toast.success('KYC policy updated');
      },
      error: () => this.savingKYC.set(false),
    });
  }

  saveFinanceConfig(): void {
    this.api.updateTenantConfig(this.saccoId, {
      tenant_config: {
        kyc_required: this.kycForm.kyc_required,
        kyc_verification_mode: this.kycForm.kyc_verification_mode,
        kyc_restricted_actions: this.kycForm.kyc_restricted_actions,
        topup_verification_mode: this.financeForm.topup_verification_mode,
        allowed_topup_methods: this.financeForm.allowed_topup_methods,
        allowed_topup_channels: this.financeForm.allowed_topup_channels,
        handle_statutory_deductions: this.financeForm.handle_statutory_deductions,
        enabled_deductions: this.financeForm.enabled_deductions,
        custom_deduction_labels: this.financeForm.custom_deduction_labels,
      },
    }).subscribe({
      next: r => {
        this.sacco.set(r.data);
        this.toast.success('Finance settings updated');
      },
      error: () => this.toast.error('Failed to save finance settings'),
    });
  }

  // --- Deduction Type Methods ---

  isDeductionEnabled(code: string): boolean {
    return this.financeForm.enabled_deductions.includes(code);
  }

  toggleDeduction(code: string): void {
    const idx = this.financeForm.enabled_deductions.indexOf(code);
    if (idx >= 0) {
      this.financeForm.enabled_deductions.splice(idx, 1);
    } else {
      this.financeForm.enabled_deductions.push(code);
    }
    this.saveFinanceConfig();
  }

  /** Convert a human label to a safe uppercase code, e.g. "Motor Vehicle Loan" → "MOTOR_VEHICLE_LOAN" */
  private labelToCode(label: string): string {
    return label.trim().toUpperCase().replace(/[^A-Z0-9]+/g, '_').replace(/^_|_$/g, '');
  }

  addCustomDeduction(): void {
    const label = this.newDeductionLabel.trim();
    if (!label) return;
    const code = this.labelToCode(label);
    if (!code) return;
    // Prevent duplicates
    if (this.financeForm.custom_deduction_labels[code]) {
      this.toast.error(`A deduction with code ${code} already exists.`);
      return;
    }
    this.financeForm.custom_deduction_labels = { ...this.financeForm.custom_deduction_labels, [code]: label };
    // Enable it immediately
    if (!this.financeForm.enabled_deductions.includes(code)) {
      this.financeForm.enabled_deductions = [...this.financeForm.enabled_deductions, code];
    }
    this.newDeductionLabel = '';
    this.saveFinanceConfig();
    this.toast.success(`Custom deduction "${label}" added and enabled.`);
  }

  removeCustomDeduction(code: string): void {
    const label = this.financeForm.custom_deduction_labels[code] || code;
    const { [code]: _, ...rest } = this.financeForm.custom_deduction_labels;
    this.financeForm.custom_deduction_labels = rest;
    // Also disable it
    this.financeForm.enabled_deductions = this.financeForm.enabled_deductions.filter(c => c !== code);
    this.saveFinanceConfig();
    this.toast.success(`Deduction "${label}" removed.`);
  }

  isTopUpMethodEnabled(methodId: string): boolean {
    // Empty array = all methods enabled (default)
    if (this.financeForm.allowed_topup_methods.length === 0) return false;
    return this.financeForm.allowed_topup_methods.includes(methodId);
  }

  toggleTopUpMethod(methodId: string): void {
    const arr = this.financeForm.allowed_topup_methods;
    const method = this.allTopUpMethods.find(m => m.id === methodId);
    const idx = arr.indexOf(methodId);
    if (idx >= 0) {
      // Disabling this method — also remove all its channels
      arr.splice(idx, 1);
      if (method) {
        for (const ch of method.channels) {
          const ci = this.financeForm.allowed_topup_channels.indexOf(ch.id);
          if (ci >= 0) this.financeForm.allowed_topup_channels.splice(ci, 1);
        }
      }
    } else {
      // Enabling this method — also enable all its channels by default
      arr.push(methodId);
      if (method) {
        for (const ch of method.channels) {
          if (!this.financeForm.allowed_topup_channels.includes(ch.id)) {
            this.financeForm.allowed_topup_channels.push(ch.id);
          }
        }
      }
    }
    // Create new references so Angular detects the change
    this.financeForm.allowed_topup_methods = [...arr];
    this.financeForm.allowed_topup_channels = [...this.financeForm.allowed_topup_channels];
    this.saveFinanceConfig();
  }

  /** Check if a specific channel (provider) is enabled */
  isChannelEnabled(channelId: string): boolean {
    // If no channels are configured, all are enabled (backward compat)
    if (this.financeForm.allowed_topup_channels.length === 0) return true;
    return this.financeForm.allowed_topup_channels.includes(channelId);
  }

  /** Toggle a specific channel within a method */
  toggleChannel(channelId: string, methodId: string): void {
    const arr = this.financeForm.allowed_topup_channels;
    const idx = arr.indexOf(channelId);
    if (idx >= 0) {
      arr.splice(idx, 1);
    } else {
      arr.push(channelId);
    }
    this.financeForm.allowed_topup_channels = [...arr];

    // If all channels in this method are disabled, also disable the method
    const method = this.allTopUpMethods.find(m => m.id === methodId);
    if (method) {
      const anyEnabled = method.channels.some(ch => this.financeForm.allowed_topup_channels.includes(ch.id));
      const methodIdx = this.financeForm.allowed_topup_methods.indexOf(methodId);
      if (!anyEnabled && methodIdx >= 0) {
        this.financeForm.allowed_topup_methods.splice(methodIdx, 1);
        this.financeForm.allowed_topup_methods = [...this.financeForm.allowed_topup_methods];
      }
    }
    this.saveFinanceConfig();
  }

  bootstrapPreview = signal<any>(null);
  bootstrapResult = signal<BootstrapResult | null>(null);
  bootstrapping = signal(false);

  selectIndustry(industry: IndustryType): void {
    // Show preview of what will be seeded
    const tmpl = getIndustryTemplate(industry);
    this.bootstrapPreview.set(tmpl);
    this.bootstrapResult.set(null);

    // Always update the industry type immediately
    this.api.updateOrganization(this.saccoId, { industry_type: industry }).subscribe({
      next: r => {
        this.sacco.set(r.data);
        this.orgCtx.setIndustry(industry); // Update sidebar immediately
        this.toast.success(`Industry set to ${industry}`);
      },
    });
  }

  confirmBootstrap(): void {
    const industry = this.sacco()?.industry_type;
    if (!industry) return;

    this.bootstrapping.set(true);
    this.api.bootstrapIndustry(this.saccoId, industry).subscribe({
      next: r => {
        this.bootstrapResult.set(r.data);
        this.bootstrapping.set(false);
        this.toast.success('Industry template applied — job types and schedules seeded!');
        // Reload job types and schedules
        this.api.getJobTypes(this.saccoId).subscribe({ next: r => this.jobTypes.set(r.data || []) });
        this.api.getPaySchedules(this.saccoId).subscribe({ next: r => this.paySchedules.set(r.data || []) });
      },
      error: () => { this.bootstrapping.set(false); this.toast.error('Bootstrap failed'); },
    });
  }

  labelKeys(obj: Record<string, string>): string[] {
    return obj ? Object.keys(obj) : [];
  }

  // --- Job Types ---
  openJobModal(): void {
    this.editingJobId = '';
    this.jobForm = { code: '', display_name: '', category: 'PRIMARY' };
    this.showModal.set('job');
  }
  editJobType(j: TenantJobType): void {
    this.editingJobId = j.id;
    this.jobForm = { code: j.code, display_name: j.display_name, category: j.category };
    this.showModal.set('job');
  }
  submitJob(): void {
    if (!this.jobForm.code || !this.jobForm.display_name) { this.toast.error('Code and name required'); return; }
    this.submitting.set(true);
    const data = { ...this.jobForm, code: this.jobForm.code.toUpperCase() };
    const obs = this.editingJobId
      ? this.api.updateJobType(this.saccoId, this.editingJobId, data)
      : this.api.createJobType(this.saccoId, data);
    obs.subscribe({
      next: () => {
        this.toast.success(this.editingJobId ? 'Job type updated' : 'Job type created');
        this.closeModal(); this.submitting.set(false);
        this.api.getJobTypes(this.saccoId).subscribe({ next: r => this.jobTypes.set(r.data || []) });
      },
      error: () => this.submitting.set(false),
    });
  }
  deleteJobType(j: TenantJobType): void {
    this.confirm.danger('Delete Job Type', `Delete "${j.display_name}"? This cannot be undone.`).subscribe(r => {
      if (r.confirmed) {
        this.api.deleteJobType(this.saccoId, j.id).subscribe({
          next: () => { this.toast.success('Job type deleted'); this.api.getJobTypes(this.saccoId).subscribe({ next: r => this.jobTypes.set(r.data || []) }); },
        });
      }
    });
  }

  // --- Pay Schedules ---
  openScheduleModal(): void {
    this.editingScheduleId = '';
    this.schedForm = { name: '', frequency: 'DAILY', pay_day: 0, cutoff_hour: 17, is_default: false };
    this.showModal.set('schedule');
  }
  editSchedule(s: PaySchedule): void {
    this.editingScheduleId = s.id;
    this.schedForm = { name: s.name, frequency: s.frequency, pay_day: s.pay_day || 0, cutoff_hour: s.cutoff_hour, is_default: s.is_default };
    this.showModal.set('schedule');
  }
  submitSchedule(): void {
    if (!this.schedForm.name) { this.toast.error('Name required'); return; }
    this.submitting.set(true);
    const obs = this.editingScheduleId
      ? this.api.updatePaySchedule(this.saccoId, this.editingScheduleId, this.schedForm)
      : this.api.createPaySchedule(this.saccoId, this.schedForm);
    obs.subscribe({
      next: () => {
        this.toast.success(this.editingScheduleId ? 'Schedule updated' : 'Schedule created');
        this.closeModal(); this.submitting.set(false);
        this.api.getPaySchedules(this.saccoId).subscribe({ next: r => this.paySchedules.set(r.data || []) });
      },
      error: () => this.submitting.set(false),
    });
  }
  deleteSchedule(s: PaySchedule): void {
    this.confirm.danger('Delete Schedule', `Delete "${s.name}"?`).subscribe(r => {
      if (r.confirmed) {
        this.api.deletePaySchedule(this.saccoId, s.id).subscribe({
          next: () => { this.toast.success('Schedule deleted'); this.api.getPaySchedules(this.saccoId).subscribe({ next: r => this.paySchedules.set(r.data || []) }); },
        });
      }
    });
  }

  closeModal(): void { this.showModal.set(null); }

  categoryBadge(cat: JobTypeCategory): string {
    const map: Record<string, string> = { PRIMARY: 'badge-primary', FACILITATOR: 'badge-facilitator', SUPPORT: 'badge-support', SUPERVISOR: 'badge-supervisor' };
    return map[cat] || 'badge-neutral';
  }

  scheduleIcon(freq: PayFrequency): string {
    const map: Record<string, string> = { DAILY: 'today', WEEKLY: 'date_range', BI_WEEKLY: 'calendar_month', MONTHLY: 'event' };
    return map[freq] || 'schedule';
  }
}
