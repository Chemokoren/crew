import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { AuthService } from '../../../core/services/auth.service';
import { ConfirmDialogService } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { PaySchedule, PayPeriod, PayFrequency } from '../../../core/models';

@Component({
  selector: 'app-pay-schedule-dashboard',
  standalone: true,
  imports: [CommonModule, FormsModule, CurrencyKesPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Pay Schedule Dashboard</h1>
          <p class="page-subtitle">Manage pay schedules, periods, and upcoming payroll runs</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-primary" (click)="openGenerateModal()" id="btn-generate-period">
            <span class="material-icons-round">add_circle</span> Generate Period
          </button>
        </div>
      </div>

      @if (loading()) {
        @for (i of [1,2,3]; track i) { <div class="skeleton" style="height:160px;margin-bottom:var(--space-md);border-radius:var(--radius-lg);"></div> }
      } @else if (schedules().length === 0) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">event_busy</span>
          <div class="empty-title">No pay schedules configured</div>
          <div class="empty-description">Go to Tenant Settings to create pay schedules for your organization.</div>
        </div>
      } @else {
        <!-- Schedule Cards with Periods -->
        @for (sched of schedules(); track sched.id) {
          <div class="glass-card psd-sched-card">
            <div class="psd-sched-header">
              <div class="psd-sched-info">
                <span class="material-icons-round psd-sched-icon">{{ freqIcon(sched.frequency) }}</span>
                <div>
                  <div class="psd-sched-name">{{ sched.name }}</div>
                  <div class="psd-sched-meta">
                    <span class="badge badge-accent">{{ sched.frequency }}</span>
                    @if (sched.is_default) { <span class="badge badge-success">Default</span> }
                    @if (sched.pay_day) { <span>Pay day: {{ sched.pay_day }}</span> }
                    <span>Cutoff: {{ sched.cutoff_hour }}:00</span>
                  </div>
                </div>
              </div>
              <button class="btn btn-secondary btn-sm" (click)="generatePeriod(sched)" id="gen-{{ sched.id }}">
                <span class="material-icons-round" style="font-size:16px;">playlist_add</span> New Period
              </button>
            </div>

            <!-- Periods for this schedule -->
            @if (getPeriods(sched.id).length > 0) {
              <div class="psd-periods">
                <div class="psd-period-header-row">
                  <span>Period</span><span>Status</span><span>Workers</span><span>Total Paid</span><span>Actions</span>
                </div>
                @for (p of getPeriods(sched.id); track p.id) {
                  <div class="psd-period-row" [class.is-open]="p.status === 'OPEN'" [class.is-closed]="p.status === 'CLOSED'">
                    <span class="psd-period-dates">{{ p.period_start | date:'mediumDate' }} — {{ p.period_end | date:'mediumDate' }}</span>
                    <span>
                      <span class="badge" [ngClass]="p.status === 'OPEN' ? 'badge-success' : p.status === 'PROCESSING' ? 'badge-warning' : 'badge-neutral'">{{ p.status }}</span>
                    </span>
                    <span class="psd-period-count">{{ p.worker_count || 0 }}</span>
                    <span class="psd-period-total">{{ (p.total_amount_cents || 0) | currencyKes }}</span>
                    <span>
                      @if (p.status === 'OPEN') {
                        <button class="btn btn-sm btn-primary" (click)="runPayroll(sched, p)" [id]="'run-' + p.id">
                          <span class="material-icons-round" style="font-size:14px;">play_arrow</span> Run
                        </button>
                        <button class="btn btn-sm btn-ghost" (click)="closePeriod(p)" [id]="'close-' + p.id" style="color:var(--color-warning);">
                          <span class="material-icons-round" style="font-size:14px;">lock</span>
                        </button>
                      }
                      @if (p.status === 'CLOSED') {
                        <span class="material-icons-round" style="font-size:16px;color:var(--color-text-muted);">lock</span>
                      }
                    </span>
                  </div>
                }
              </div>
            } @else {
              <div class="psd-no-periods">
                <span class="material-icons-round" style="font-size:20px;color:var(--color-text-muted);">event_note</span>
                <span>No periods generated yet</span>
              </div>
            }
          </div>
        }
      }

      <!-- Generate Period Modal -->
      @if (showGenerateModal()) {
        <div class="modal-backdrop" (click)="showGenerateModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Generate Pay Period</h3><button class="btn-ghost" (click)="showGenerateModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <label class="form-label">Schedule</label>
            <select class="form-select" [(ngModel)]="genScheduleId" id="gen-sched-select">
              @for (s of schedules(); track s.id) { <option [value]="s.id">{{ s.name }} ({{ s.frequency }})</option> }
            </select>
            <label class="form-label" style="margin-top:var(--space-sm);">Reference Date</label>
            <input class="form-input" type="date" [(ngModel)]="genRefDate" id="gen-ref-date" />
          </div>
          <div class="modal-footer"><button class="btn btn-ghost" (click)="showGenerateModal.set(false)">Cancel</button><button class="btn btn-primary" (click)="submitGenerate()" [disabled]="submitting()" id="btn-submit-gen">{{ submitting() ? 'Generating...' : 'Generate Period' }}</button></div>
        </div></div>
      }
    </div>
  `,
  styles: [`
    .psd-sched-card { padding: var(--space-lg) !important; margin-bottom: var(--space-md); }
    .psd-sched-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-md); }
    .psd-sched-info { display: flex; align-items: center; gap: var(--space-md); }
    .psd-sched-icon { font-size: 28px; color: var(--color-accent); }
    .psd-sched-name { font-size: 1rem; font-weight: 700; }
    .psd-sched-meta { display: flex; align-items: center; gap: var(--space-sm); font-size: 0.75rem; color: var(--color-text-muted); margin-top: 2px; }

    .psd-periods { border-top: 1px solid var(--color-border); }
    .psd-period-header-row {
      display: grid; grid-template-columns: 2fr 1fr 0.8fr 1fr 1fr;
      gap: 8px; padding: 8px 0; font-size: 0.7rem; font-weight: 600;
      text-transform: uppercase; letter-spacing: 0.05em; color: var(--color-text-muted);
    }
    .psd-period-row {
      display: grid; grid-template-columns: 2fr 1fr 0.8fr 1fr 1fr;
      gap: 8px; padding: 10px 0; align-items: center;
      border-top: 1px solid rgba(255,255,255,0.03); font-size: 0.85rem;
    }
    .psd-period-row.is-open { border-left: 3px solid var(--color-success); padding-left: 10px; }
    .psd-period-dates { font-weight: 500; color: var(--color-text-primary); }
    .psd-period-count { font-weight: 600; color: var(--color-accent); }
    .psd-period-total { font-weight: 600; color: var(--color-success); }
    .psd-no-periods { display: flex; align-items: center; gap: 8px; padding: var(--space-md) 0; font-size: 0.85rem; color: var(--color-text-muted); border-top: 1px solid var(--color-border); }

    input[type="date"] { color-scheme: dark; cursor: pointer; }
    input[type="date"]::-webkit-calendar-picker-indicator { filter: invert(0.7) sepia(1) saturate(5) hue-rotate(175deg); cursor: pointer; }

    @media (max-width: 768px) {
      .psd-period-header-row, .psd-period-row { grid-template-columns: 1fr 1fr; }
      .psd-sched-header { flex-direction: column; align-items: flex-start; gap: var(--space-sm); }
    }
  `]
})
export class PayScheduleDashboardComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmDialogService);

  schedules = signal<PaySchedule[]>([]);
  periods = signal<PayPeriod[]>([]);
  loading = signal(true);
  showGenerateModal = signal(false);
  submitting = signal(false);
  saccoId = '';
  genScheduleId = '';
  genRefDate = '';

  ngOnInit(): void {
    const user = this.auth.currentUser();
    this.saccoId = user?.organization_id || '';
    if (!this.saccoId) {
      this.api.getOrganizations({ per_page: '1' }).subscribe({
        next: r => { if (r.data.length) { this.saccoId = r.data[0].id; this.loadData(); } else { this.loading.set(false); } },
        error: () => this.loading.set(false),
      });
    } else {
      this.loadData();
    }
  }

  loadData(): void {
    this.loading.set(true);
    this.api.getPaySchedules(this.saccoId).subscribe({
      next: r => {
        this.schedules.set(r.data || []);
        if (r.data?.length) this.genScheduleId = r.data[0].id;
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
    this.api.getPayPeriods({ organization_id: this.saccoId, per_page: '100' }).subscribe({
      next: r => this.periods.set(r.data || []),
    });
    const today = new Date();
    this.genRefDate = today.toISOString().slice(0, 10);
  }

  getPeriods(scheduleId: string): PayPeriod[] {
    return this.periods().filter(p => p.pay_schedule_id === scheduleId);
  }

  freqIcon(freq: PayFrequency): string {
    const m: Record<string, string> = { DAILY: 'today', WEEKLY: 'date_range', BI_WEEKLY: 'calendar_month', MONTHLY: 'event' };
    return m[freq] || 'schedule';
  }

  openGenerateModal(): void { this.showGenerateModal.set(true); }

  generatePeriod(sched: PaySchedule): void {
    this.genScheduleId = sched.id;
    this.showGenerateModal.set(true);
  }

  submitGenerate(): void {
    if (!this.genScheduleId || !this.genRefDate) { this.toast.error('Select schedule and date'); return; }
    this.submitting.set(true);
    this.api.generatePayPeriod(this.genScheduleId, { reference_date: this.genRefDate }).subscribe({
      next: () => {
        this.toast.success('Pay period generated');
        this.showGenerateModal.set(false);
        this.submitting.set(false);
        this.loadData();
      },
      error: () => this.submitting.set(false),
    });
  }

  closePeriod(p: PayPeriod): void {
    this.confirm.danger('Close Period', `Close this pay period? No more earnings can be recorded.`).subscribe(r => {
      if (r.confirmed) {
        this.api.closePayPeriod(p.id).subscribe({
          next: () => { this.toast.success('Period closed'); this.loadData(); },
        });
      }
    });
  }

  runPayroll(sched: PaySchedule, p: PayPeriod): void {
    this.confirm.confirm('Run Payroll', `Process payroll for this period? This will calculate deductions and credit wallets.`).subscribe(r => {
      if (r.confirmed) {
        this.api.runScheduledPayroll(sched.id, { reference_date: p.period_start, organization_id: this.saccoId }).subscribe({
          next: () => { this.toast.success('Payroll run initiated'); this.loadData(); },
        });
      }
    });
  }
}
