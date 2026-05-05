import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { ToastService } from '../../core/services/toast.service';
import { AuthService } from '../../core/services/auth.service';
import { ConfirmDialogService } from '../../shared/components/confirm-dialog/confirm-dialog.component';
import { Organization, TenantJobType, PaySchedule, IndustryType, PayFrequency, JobTypeCategory, BootstrapResult } from '../../core/models';
import { getIndustryTemplate, INDUSTRY_TEMPLATES, INDUSTRY_ICONS } from '../../core/config/industry-templates';
import { OrgContextService } from '../../core/services/org-context.service';

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
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Tenant Settings</h1>
          <p class="page-subtitle">Configure industry, job types, and pay schedules for your organization</p>
        </div>
      </div>

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

  activeTab = 'industry';
  saccoId = '';
  editingJobId = '';
  editingScheduleId = '';

  readonly industries = INDUSTRIES;
  readonly payFrequencies = PAY_FREQUENCIES;
  readonly jobCategories = JOB_CATEGORIES;

  jobForm = { code: '', display_name: '', category: 'PRIMARY' as JobTypeCategory };
  schedForm = { name: '', frequency: 'DAILY' as PayFrequency, pay_day: 0, cutoff_hour: 17, is_default: false };

  ngOnInit(): void {
    const user = this.auth.currentUser();
    this.saccoId = user?.organization_id || '';
    if (this.saccoId) {
      this.loadAll();
    } else if (this.auth.isAdmin()) {
      // System admin — load first sacco or show selector
      this.api.getOrganizations({ per_page: '1' }).subscribe({
        next: r => {
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
      // SACCO_ADMIN without org — show empty state
      this.loading.set(false);
    }
  }

  loadAll(): void {
    this.loading.set(true);
    this.api.getOrganization(this.saccoId).subscribe({
      next: r => { this.sacco.set(r.data); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
    this.api.getJobTypes(this.saccoId).subscribe({
      next: r => this.jobTypes.set(r.data || []),
    });
    this.api.getPaySchedules(this.saccoId).subscribe({
      next: r => this.paySchedules.set(r.data || []),
    });
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
