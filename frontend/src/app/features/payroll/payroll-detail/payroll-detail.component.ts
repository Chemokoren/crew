import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { PayrollRun, PayrollEntry, PayrollStatus, SACCO, CrewMember } from '../../../core/models';
import { ToastService } from '../../../core/services/toast.service';

const STATUS_STEPS: PayrollStatus[] = ['DRAFT', 'PROCESSED', 'APPROVED', 'SUBMITTED', 'COMPLETED'];

@Component({
  selector: 'app-payroll-detail',
  standalone: true,
  imports: [CommonModule, CurrencyKesPipe, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <button class="btn btn-ghost btn-sm" (click)="goBack()" style="margin-bottom:var(--space-xs);"><span class="material-icons-round" style="font-size:16px;">arrow_back</span> Back to Payroll</button>
          <h1 class="page-title">Payroll Run</h1>
          <p class="page-subtitle">{{ periodLabel() }} — {{ saccoName() }}</p>
        </div>
        <div class="page-actions">
          @if (run()?.status === 'DRAFT') { <button class="btn btn-secondary" (click)="processRun()" id="btn-process">Process</button> }
          @if (run()?.status === 'PROCESSED') { <button class="btn btn-primary" (click)="approveRun()" id="btn-approve">Approve</button> }
          @if (run()?.status === 'APPROVED') { <button class="btn btn-primary" (click)="submitRun()" id="btn-submit">Submit to PerPay</button> }
        </div>
      </div>

      @if (loading()) {
        @for (i of [1,2,3]; track i) { <div class="skeleton" style="height:80px;margin:8px 0;border-radius:var(--radius-lg);"></div> }
      } @else if (run(); as r) {

        <!-- Status Progression (Task 128) -->
        <div class="glass-card" style="margin-bottom:var(--space-lg);padding:var(--space-lg) var(--space-xl) !important;">
          <div class="status-track">
            @for (step of statusSteps; track step; let i = $index) {
              <div class="status-step" [class.active]="stepIndex(step) <= currentStepIndex()" [class.current]="step === r.status">
                <div class="step-dot">
                  @if (stepIndex(step) < currentStepIndex()) { <span class="material-icons-round" style="font-size:14px;">check</span> }
                  @else { {{ i + 1 }} }
                </div>
                <span class="step-label">{{ step }}</span>
              </div>
              @if (i < statusSteps.length - 1) { <div class="step-line" [class.active]="stepIndex(step) < currentStepIndex()"></div> }
            }
          </div>
        </div>

        <!-- Summary Cards -->
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); margin-bottom: var(--space-lg);">
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">payments</span></div><div class="stat-value" style="color:var(--color-success);">{{ r.total_gross_cents | currencyKes }}</div><div class="stat-label">Gross Pay</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(239,68,68,0.12);color:#ef4444;"><span class="material-icons-round">remove_circle_outline</span></div><div class="stat-value" style="color:#ef4444;">{{ r.total_deductions_cents | currencyKes }}</div><div class="stat-label">Total Deductions</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">account_balance_wallet</span></div><div class="stat-value" style="color:var(--color-accent);">{{ r.total_net_cents | currencyKes }}</div><div class="stat-label">Net Pay</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(168,85,247,0.12);color:#a855f7;"><span class="material-icons-round">people</span></div><div class="stat-value">{{ r.entry_count }}</div><div class="stat-label">Entries</div></div>
        </div>

        <!-- Run Details -->
        <div class="glass-card" style="margin-bottom:var(--space-lg);">
          <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Run Details</h3>
          <div class="detail-grid">
            <div class="detail-row"><span class="detail-label">SACCO</span><span class="detail-value">{{ saccoName() }}</span></div>
            <div class="detail-row"><span class="detail-label">Period</span><span class="detail-value">{{ r.period_start | date:'mediumDate' }} — {{ r.period_end | date:'mediumDate' }}</span></div>
            <div class="detail-row"><span class="detail-label">Status</span><span class="detail-value"><span class="badge" [ngClass]="statusBadge(r.status)">{{ r.status }}</span></span></div>
            <div class="detail-row"><span class="detail-label">Created</span><span class="detail-value">{{ r.created_at | relativeTime }}</span></div>
          </div>
        </div>

        <!-- Entries Table (Task 127) -->
        <div class="glass-card">
          <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Payroll Entries — Per-Crew Breakdown</h3>
          @if (entriesLoading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:48px;margin:4px 0;"></div>} }
          @else if (entries().length === 0) {
            <div class="empty-state" style="padding:var(--space-lg);"><span class="material-icons-round empty-icon">receipt_long</span><div class="empty-title">No entries yet</div><div class="empty-subtitle">Process the payroll run to generate entries</div></div>
          } @else {
            <div class="data-table-wrapper"><table class="data-table">
              <thead><tr><th>Crew Member</th><th>Gross</th><th>SHA</th><th>NSSF</th><th>Housing</th><th>Other</th><th>Net Pay</th></tr></thead>
              <tbody>
                @for (e of entries(); track e.id) {
                  <tr>
                    <td style="font-weight:500;color:var(--color-text-primary);">{{ crewName(e.crew_member_id) }}</td>
                    <td>{{ e.gross_earnings_cents | currencyKes }}</td>
                    <td class="text-danger">{{ e.sha_deduction_cents | currencyKes }}</td>
                    <td class="text-danger">{{ e.nssf_deduction_cents | currencyKes }}</td>
                    <td class="text-danger">{{ e.housing_levy_deduction_cents | currencyKes }}</td>
                    <td class="text-danger">{{ e.other_deductions_cents | currencyKes }}</td>
                    <td style="font-weight:600;color:var(--color-success);">{{ e.net_pay_cents | currencyKes }}</td>
                  </tr>
                }
                <tr style="font-weight:700;border-top:2px solid var(--color-border);">
                  <td>TOTAL</td>
                  <td>{{ totalGross() | currencyKes }}</td>
                  <td class="text-danger">{{ totalSHA() | currencyKes }}</td>
                  <td class="text-danger">{{ totalNSSF() | currencyKes }}</td>
                  <td class="text-danger">{{ totalHousing() | currencyKes }}</td>
                  <td class="text-danger">{{ totalOther() | currencyKes }}</td>
                  <td style="color:var(--color-success);">{{ totalNet() | currencyKes }}</td>
                </tr>
              </tbody>
            </table></div>
          }
        </div>
      }
    </div>
  `,
  styles: [`
    .detail-grid { display: grid; gap: 0; }
    .detail-row { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; border-bottom: 1px solid var(--color-border); &:last-child { border-bottom: none; } }
    .detail-label { font-size: 0.8rem; color: var(--color-text-muted); font-weight: 500; }
    .detail-value { font-size: 0.875rem; color: var(--color-text-secondary); text-align: right; }

    /* Status Track */
    .status-track { display: flex; align-items: center; gap: 0; justify-content: center; }
    .status-step { display: flex; flex-direction: column; align-items: center; gap: 6px; min-width: 72px; }
    .step-dot {
      width: 32px; height: 32px; border-radius: 50%;
      display: flex; align-items: center; justify-content: center;
      font-size: 0.7rem; font-weight: 700; color: var(--color-text-muted);
      background: var(--color-surface-alt); border: 2px solid var(--color-border);
      transition: all 0.3s ease;
    }
    .status-step.active .step-dot {
      background: var(--color-accent); color: #fff; border-color: var(--color-accent);
    }
    .status-step.current .step-dot {
      box-shadow: 0 0 0 4px rgba(0, 210, 255, 0.25); animation: pulse 2s infinite;
    }
    .step-label { font-size: 0.65rem; font-weight: 600; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.04em; }
    .status-step.active .step-label { color: var(--color-accent); }
    .step-line {
      flex: 1; height: 2px; min-width: 24px; max-width: 80px;
      background: var(--color-border); transition: background 0.3s;
    }
    .step-line.active { background: var(--color-accent); }

    .text-danger { color: #ef4444; }

    @media (max-width: 640px) {
      .status-step { min-width: 50px; }
      .step-label { font-size: 0.55rem; }
      .step-dot { width: 26px; height: 26px; font-size: 0.6rem; }
    }
  `]
})
export class PayrollDetailComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  run = signal<PayrollRun | null>(null);
  entries = signal<PayrollEntry[]>([]);
  loading = signal(true);
  entriesLoading = signal(false);
  saccoName = signal('—');
  private crewMap = new Map<string, string>();
  runId = '';

  readonly statusSteps = STATUS_STEPS;

  periodLabel = computed(() => {
    const r = this.run();
    if (!r) return '';
    return `${new Date(r.period_start).toLocaleDateString('en', { month: 'short', day: 'numeric' })} — ${new Date(r.period_end).toLocaleDateString('en', { month: 'short', day: 'numeric', year: 'numeric' })}`;
  });

  currentStepIndex = computed(() => this.stepIndex(this.run()?.status || 'DRAFT'));

  // Entry totals
  totalGross = computed(() => this.entries().reduce((s, e) => s + e.gross_earnings_cents, 0));
  totalSHA = computed(() => this.entries().reduce((s, e) => s + e.sha_deduction_cents, 0));
  totalNSSF = computed(() => this.entries().reduce((s, e) => s + e.nssf_deduction_cents, 0));
  totalHousing = computed(() => this.entries().reduce((s, e) => s + e.housing_levy_deduction_cents, 0));
  totalOther = computed(() => this.entries().reduce((s, e) => s + e.other_deductions_cents, 0));
  totalNet = computed(() => this.entries().reduce((s, e) => s + e.net_pay_cents, 0));

  ngOnInit(): void {
    this.runId = this.route.snapshot.paramMap.get('id') || '';
    if (this.runId) {
      this.loadRun();
      this.loadEntries();
      // Pre-load crew members for name resolution
      this.api.getCrewMembers({ per_page: '200' }).subscribe({
        next: r => { for (const cm of r.data) this.crewMap.set(cm.id, `${cm.first_name} ${cm.last_name}`); },
      });
    }
  }

  goBack(): void { this.router.navigate(['/payroll']); }

  stepIndex(status: string): number { return STATUS_STEPS.indexOf(status as PayrollStatus); }

  crewName(id: string): string { return this.crewMap.get(id) || id.slice(0, 8) + '...'; }

  statusBadge(s: string) {
    return s === 'COMPLETED' || s === 'APPROVED' ? 'badge-success' : s === 'FAILED' ? 'badge-danger' : s === 'SUBMITTED' ? 'badge-info' : 'badge-warning';
  }

  loadRun(): void {
    this.loading.set(true);
    this.api.getPayrollRun(this.runId).subscribe({
      next: (r) => {
        this.run.set(r.data);
        if (r.data.sacco_id) {
          this.api.getSACCO(r.data.sacco_id).subscribe({
            next: sr => this.saccoName.set(sr.data.name),
            error: () => this.saccoName.set(r.data.sacco_id.slice(0, 8) + '...'),
          });
        }
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  loadEntries(): void {
    this.entriesLoading.set(true);
    this.api.getPayrollEntries(this.runId).subscribe({
      next: (r) => { this.entries.set(r.data || []); this.entriesLoading.set(false); },
      error: () => this.entriesLoading.set(false),
    });
  }

  processRun(): void {
    this.api.processPayrollRun(this.runId).subscribe({
      next: () => { this.toast.success('Payroll processed'); this.loadRun(); this.loadEntries(); },
    });
  }

  approveRun(): void {
    this.api.approvePayrollRun(this.runId).subscribe({
      next: () => { this.toast.success('Payroll approved'); this.loadRun(); },
    });
  }

  submitRun(): void {
    this.api.submitPayrollRun(this.runId).subscribe({
      next: () => { this.toast.success('Payroll submitted to PerPay'); this.loadRun(); },
    });
  }
}
