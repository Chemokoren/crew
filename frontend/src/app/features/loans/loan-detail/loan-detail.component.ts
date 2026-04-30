import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { LoanApplication, LoanStatus, CrewMember } from '../../../core/models';

interface TimelineStep { label: string; status: LoanStatus; icon: string; }

const LIFECYCLE_STEPS: TimelineStep[] = [
  { label: 'Applied',   status: 'APPLIED',   icon: 'description' },
  { label: 'Approved',  status: 'APPROVED',  icon: 'check_circle' },
  { label: 'Disbursed', status: 'DISBURSED', icon: 'account_balance' },
  { label: 'Repaid',    status: 'COMPLETED', icon: 'done_all' },
];

const STATUS_ORDER: Record<string, number> = {
  'APPLIED': 0, 'PENDING': 0, 'APPROVED': 1, 'REJECTED': -1,
  'DISBURSED': 2, 'REPAYING': 2, 'COMPLETED': 3, 'REPAID': 3, 'DEFAULTED': -2,
};

@Component({
  selector: 'app-loan-detail',
  standalone: true,
  imports: [CommonModule, FormsModule, CurrencyKesPipe, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <button class="btn btn-ghost btn-sm" (click)="goBack()" style="margin-bottom:var(--space-xs);">
            <span class="material-icons-round" style="font-size:16px;">arrow_back</span> Back to Loans
          </button>
          <h1 class="page-title">Loan Application</h1>
          <p class="page-subtitle">{{ crewName() }} — {{ loan()?.category || '' }}</p>
        </div>
        <div class="page-actions">
          @if (loan(); as l) {
            @if (l.status === 'DISBURSED' || l.status === 'REPAYING') {
              <button class="btn btn-primary" (click)="repay()" id="btn-repay-detail">
                <span class="material-icons-round">payment</span> Make Repayment
              </button>
            }
            @if (isAdmin()) {
              @if (l.status === 'APPLIED' || l.status === 'PENDING') {
                <button class="btn btn-secondary" (click)="openApproveModal()">Approve</button>
                <button class="btn btn-danger" (click)="rejectLoan()">Reject</button>
              }
              @if (l.status === 'APPROVED') {
                <button class="btn btn-primary" (click)="disburseLoan()">Disburse</button>
              }
            }
          }
        </div>
      </div>

      @if (loading()) {
        @for (i of [1,2,3]; track i) { <div class="skeleton" style="height:80px;margin:8px 0;border-radius:var(--radius-lg);"></div> }
      } @else if (loan(); as l) {

        <!-- Lifecycle Timeline (Task 142) -->
        <div class="glass-card" style="margin-bottom:var(--space-lg);padding:var(--space-lg) var(--space-xl) !important;">
          @if (isTerminal()) {
            <!-- Rejected / Defaulted state -->
            <div class="terminal-state" [class.rejected]="l.status === 'REJECTED'" [class.defaulted]="l.status === 'DEFAULTED'">
              <span class="material-icons-round terminal-icon">{{ l.status === 'REJECTED' ? 'block' : 'warning' }}</span>
              <span class="terminal-label">{{ l.status }}</span>
            </div>
          } @else {
            <div class="timeline-track">
              @for (step of lifecycleSteps; track step.status; let i = $index) {
                <div class="timeline-step" [class.active]="stepReached(step.status)" [class.current]="stepCurrent(step.status)">
                  <div class="step-circle">
                    @if (stepReached(step.status) && !stepCurrent(step.status)) {
                      <span class="material-icons-round" style="font-size:16px;">check</span>
                    } @else {
                      <span class="material-icons-round" style="font-size:16px;">{{ step.icon }}</span>
                    }
                  </div>
                  <span class="step-text">{{ step.label }}</span>
                </div>
                @if (i < lifecycleSteps.length - 1) {
                  <div class="timeline-connector" [class.active]="connectorActive(i)"></div>
                }
              }
            </div>
          }
        </div>

        <!-- Summary Cards -->
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); margin-bottom: var(--space-lg);">
          <div class="stat-card"><div class="stat-icon" style="background:rgba(168,85,247,0.12);color:#a855f7;"><span class="material-icons-round">request_quote</span></div><div class="stat-value">{{ (l.amount_requested_cents || l.amount_cents) | currencyKes }}</div><div class="stat-label">Requested</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">check_circle</span></div><div class="stat-value" style="color:var(--color-success);">{{ l.approved_amount_cents ? (l.approved_amount_cents | currencyKes) : '—' }}</div><div class="stat-label">Approved</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">payments</span></div><div class="stat-value" style="color:var(--color-accent);">{{ l.total_repaid_cents | currencyKes }}</div><div class="stat-label">Repaid</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(251,191,36,0.12);color:#fbbf24;"><span class="material-icons-round">schedule</span></div><div class="stat-value">{{ l.tenure_days }}d</div><div class="stat-label">Tenure</div></div>
        </div>

        <!-- Repayment Progress -->
        @if (l.approved_amount_cents && l.approved_amount_cents > 0) {
          <div class="glass-card" style="margin-bottom:var(--space-lg);">
            <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Repayment Progress</h3>
            <div class="progress-track">
              <div class="progress-fill" [style.width.%]="repaymentPct()"></div>
            </div>
            <div style="display:flex;justify-content:space-between;margin-top:6px;">
              <span style="font-size:0.75rem;color:var(--color-text-muted);">{{ repaymentPct() | number:'1.0-1' }}% repaid</span>
              <span style="font-size:0.75rem;color:var(--color-text-muted);">{{ remainingAmount() | currencyKes }} remaining</span>
            </div>
          </div>
        }

        <!-- Loan Details -->
        <div class="glass-card">
          <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Loan Details</h3>
          <div class="detail-grid">
            <div class="detail-row"><span class="detail-label">Crew Member</span><span class="detail-value">{{ crewName() }}</span></div>
            <div class="detail-row"><span class="detail-label">Category</span><span class="detail-value"><span class="badge badge-accent">{{ l.category }}</span></span></div>
            @if (l.purpose) { <div class="detail-row"><span class="detail-label">Purpose</span><span class="detail-value">{{ l.purpose }}</span></div> }
            <div class="detail-row"><span class="detail-label">Status</span><span class="detail-value"><span class="badge" [ngClass]="statusBadge(l.status)">{{ l.status }}</span></span></div>
            @if (l.interest_rate != null) { <div class="detail-row"><span class="detail-label">Interest Rate</span><span class="detail-value">{{ (l.interest_rate * 100) | number:'1.1-2' }}%</span></div> }
            @if (l.disbursed_at) { <div class="detail-row"><span class="detail-label">Disbursed</span><span class="detail-value">{{ l.disbursed_at | relativeTime }}</span></div> }
            @if (l.due_at || l.due_date) { <div class="detail-row"><span class="detail-label">Due Date</span><span class="detail-value">{{ (l.due_at || l.due_date) | date:'mediumDate' }}</span></div> }
            <div class="detail-row"><span class="detail-label">Applied</span><span class="detail-value">{{ l.created_at | relativeTime }}</span></div>
          </div>
        </div>
      }

      <!-- Approve Modal -->
      @if (showApproveModal()) {
        <div class="modal-backdrop" (click)="showApproveModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Approve Loan</h3><button class="btn btn-ghost btn-icon" (click)="showApproveModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group"><label class="form-label">Approved Amount (KES)</label>
              <input class="form-input" type="number" [(ngModel)]="approveForm.approved_amount" min="100" />
            </div>
            <div class="form-group"><label class="form-label">Interest Rate (%)</label>
              <input class="form-input" type="number" [(ngModel)]="approveForm.interest_rate" min="0" max="100" step="0.5" />
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" (click)="showApproveModal.set(false)">Cancel</button>
            <button class="btn btn-primary" (click)="submitApproval()" [disabled]="approving()">{{ approving() ? 'Approving...' : 'Approve' }}</button>
          </div>
        </div></div>
      }
    </div>
  `,
  styles: [`
    .detail-grid { display: grid; gap: 0; }
    .detail-row {
      display: flex; justify-content: space-between; align-items: center;
      padding: 10px 0; border-bottom: 1px solid var(--color-border);
      &:last-child { border-bottom: none; }
    }
    .detail-label { font-size: 0.8rem; color: var(--color-text-muted); font-weight: 500; }
    .detail-value { font-size: 0.875rem; color: var(--color-text-secondary); text-align: right; }

    /* Timeline */
    .timeline-track { display: flex; align-items: center; justify-content: center; gap: 0; }
    .timeline-step { display: flex; flex-direction: column; align-items: center; gap: 8px; min-width: 80px; }
    .step-circle {
      width: 40px; height: 40px; border-radius: 50%;
      display: flex; align-items: center; justify-content: center;
      background: var(--color-surface-alt); border: 2px solid var(--color-border);
      color: var(--color-text-muted); transition: all 0.3s ease;
    }
    .timeline-step.active .step-circle {
      background: var(--color-accent); color: #fff; border-color: var(--color-accent);
    }
    .timeline-step.current .step-circle {
      box-shadow: 0 0 0 4px rgba(0, 210, 255, 0.25); animation: pulse 2s infinite;
    }
    .step-text { font-size: 0.7rem; font-weight: 600; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.04em; }
    .timeline-step.active .step-text { color: var(--color-accent); }
    .timeline-connector {
      flex: 1; height: 2px; min-width: 32px; max-width: 100px;
      background: var(--color-border); transition: background 0.3s;
    }
    .timeline-connector.active { background: var(--color-accent); }

    /* Terminal states */
    .terminal-state {
      display: flex; align-items: center; justify-content: center; gap: var(--space-sm);
      padding: var(--space-md); border-radius: var(--radius-md);
    }
    .terminal-state.rejected { background: rgba(239,68,68,0.08); }
    .terminal-state.defaulted { background: rgba(251,191,36,0.08); }
    .terminal-icon { font-size: 28px; }
    .terminal-state.rejected .terminal-icon { color: #ef4444; }
    .terminal-state.defaulted .terminal-icon { color: #fbbf24; }
    .terminal-label {
      font-size: 1.25rem; font-weight: 800; text-transform: uppercase; letter-spacing: 0.06em;
    }
    .terminal-state.rejected .terminal-label { color: #ef4444; }
    .terminal-state.defaulted .terminal-label { color: #fbbf24; }

    /* Progress */
    .progress-track {
      height: 10px; border-radius: 5px; background: rgba(255,255,255,0.06); overflow: hidden;
    }
    .progress-fill {
      height: 100%; border-radius: 5px; background: var(--gradient-accent);
      transition: width 0.6s ease-out; min-width: 2px;
    }

    @media (max-width: 640px) {
      .timeline-step { min-width: 56px; }
      .step-circle { width: 32px; height: 32px; .material-icons-round { font-size: 14px !important; } }
      .step-text { font-size: 0.6rem; }
    }
  `]
})
export class LoanDetailComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  loan = signal<LoanApplication | null>(null);
  loading = signal(true);
  crewName = signal('—');
  showApproveModal = signal(false);
  approving = signal(false);
  approveForm = { approved_amount: 0, interest_rate: 8 };

  readonly lifecycleSteps = LIFECYCLE_STEPS;
  private loanId = '';

  ngOnInit(): void {
    this.loanId = this.route.snapshot.paramMap.get('id') || '';
    if (this.loanId) this.loadLoan();
  }

  isAdmin(): boolean { return this.auth.isAdmin(); }
  goBack(): void { this.router.navigate(['/loans']); }

  loadLoan(): void {
    this.loading.set(true);
    this.api.getLoans().subscribe({
      next: r => {
        const found = r.data.find(l => l.id === this.loanId);
        if (found) {
          this.loan.set(found);
          this.resolveCrewName(found.crew_member_id);
        }
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  private resolveCrewName(crewMemberId: string): void {
    this.api.getCrewMember(crewMemberId).subscribe({
      next: r => this.crewName.set(`${r.data.first_name} ${r.data.last_name}`),
      error: () => this.crewName.set(crewMemberId.slice(0, 8) + '...'),
    });
  }

  // --- Timeline logic ---
  stepReached(status: LoanStatus): boolean {
    const current = STATUS_ORDER[this.loan()?.status || ''] ?? -99;
    const target = STATUS_ORDER[status] ?? 99;
    return current >= target;
  }

  stepCurrent(status: LoanStatus): boolean {
    const l = this.loan();
    if (!l) return false;
    return STATUS_ORDER[l.status] === STATUS_ORDER[status];
  }

  connectorActive(index: number): boolean {
    const nextStatus = this.lifecycleSteps[index + 1]?.status;
    return nextStatus ? this.stepReached(nextStatus) : false;
  }

  isTerminal(): boolean {
    const s = this.loan()?.status;
    return s === 'REJECTED' || s === 'DEFAULTED';
  }

  // --- Repayment progress ---
  repaymentPct = computed(() => {
    const l = this.loan();
    if (!l?.approved_amount_cents || l.approved_amount_cents === 0) return 0;
    return Math.min(100, (l.total_repaid_cents / l.approved_amount_cents) * 100);
  });

  remainingAmount = computed(() => {
    const l = this.loan();
    if (!l?.approved_amount_cents) return 0;
    return Math.max(0, l.approved_amount_cents - l.total_repaid_cents);
  });

  statusBadge(s: string) {
    return s === 'REPAID' || s === 'COMPLETED' || s === 'APPROVED' ? 'badge-success'
      : s === 'REJECTED' || s === 'DEFAULTED' ? 'badge-danger'
      : s === 'DISBURSED' || s === 'REPAYING' ? 'badge-info'
      : 'badge-warning';
  }

  // --- Actions ---
  repay(): void {
    const l = this.loan();
    if (!l) return;
    const amt = prompt('Repayment amount (KES):');
    if (amt) {
      this.api.repayLoan(l.id, Math.round(parseFloat(amt) * 100)).subscribe({
        next: () => { this.toast.success('Repayment processed'); this.loadLoan(); },
      });
    }
  }

  openApproveModal(): void {
    const l = this.loan();
    if (!l) return;
    this.approveForm.approved_amount = (l.amount_requested_cents || l.amount_cents) / 100;
    this.approveForm.interest_rate = 8;
    this.showApproveModal.set(true);
  }

  submitApproval(): void {
    const l = this.loan();
    if (!l) return;
    this.approving.set(true);
    this.api.approveLoan(l.id, {
      approved_amount_cents: Math.round(this.approveForm.approved_amount * 100),
      interest_rate: this.approveForm.interest_rate / 100,
    }).subscribe({
      next: () => { this.toast.success('Loan approved'); this.showApproveModal.set(false); this.approving.set(false); this.loadLoan(); },
      error: () => this.approving.set(false),
    });
  }

  rejectLoan(): void {
    if (!confirm('Reject this loan application?')) return;
    this.api.rejectLoan(this.loanId).subscribe({
      next: () => { this.toast.success('Loan rejected'); this.loadLoan(); },
    });
  }

  disburseLoan(): void {
    if (!confirm('Disburse this loan?')) return;
    this.api.disburseLoan(this.loanId).subscribe({
      next: () => { this.toast.success('Loan disbursed'); this.loadLoan(); },
    });
  }
}
