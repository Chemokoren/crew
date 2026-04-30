import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { LoanApplication, LoanTier, CrewMember, LoanCategory } from '../../../core/models';

@Component({
  selector: 'app-loan-list', standalone: true,
  imports: [CommonModule, FormsModule, CurrencyKesPipe, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Loans</h1><p class="page-subtitle">Apply, track, and manage loan applications</p></div>
        <div class="page-actions">
          <button class="btn btn-ghost" routerLink="/credit" style="color:var(--color-text-muted);">
            <span class="material-icons-round">credit_score</span> Credit Score
          </button>
          <button class="btn btn-primary" (click)="openApplyModal()" id="btn-apply-loan">
            <span class="material-icons-round">add</span> Apply for Loan
          </button>
        </div>
      </div>

      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:56px;margin:4px 0;"></div>} }
      @else if (items().length === 0) {
        <div class="empty-state"><span class="material-icons-round empty-icon">savings</span>
          <div class="empty-title">No loan applications</div>
          <div class="empty-description">Apply for a loan based on your earnings history and credit score.</div>
        </div>
      } @else {
        <div class="data-table-wrapper"><table class="data-table"><thead><tr>
          <th>Category</th><th>Requested</th><th>Approved</th><th>Tenure</th><th>Status</th><th>Repaid</th><th>Created</th><th>Actions</th>
        </tr></thead><tbody>
          @for(l of items();track l.id){<tr class="clickable-row" (click)="viewLoan(l)">
            <td><span class="badge badge-accent">{{l.category}}</span></td>
            <td style="font-weight:600;">{{(l.amount_requested_cents || l.amount_cents)|currencyKes}}</td>
            <td>{{l.approved_amount_cents ? (l.approved_amount_cents|currencyKes) : '—'}}</td>
            <td>{{l.tenure_days}} days</td>
            <td><span class="badge" [ngClass]="statusBadge(l.status)">{{l.status}}</span></td>
            <td>{{l.total_repaid_cents|currencyKes}}</td>
            <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{l.created_at|relativeTime}}</td>
            <td>
              <div style="display:flex;gap:4px;flex-wrap:wrap;" (click)="$event.stopPropagation()">
                <button class="btn btn-sm btn-ghost" style="color:var(--color-accent);" (click)="viewLoan(l)">View</button>
                @if(l.status==='DISBURSED'||l.status==='REPAYING'){<button class="btn btn-sm btn-primary" (click)="repay(l)">Repay</button>}
                @if(isAdmin()) {
                  @if(l.status==='APPLIED'||l.status==='PENDING'){
                    <button class="btn btn-sm btn-secondary" (click)="openApproveModal(l)">Approve</button>
                    <button class="btn btn-sm btn-danger" (click)="rejectLoan(l)">Reject</button>
                  }
                  @if(l.status==='APPROVED'){
                    <button class="btn btn-sm btn-primary" (click)="disburseLoan(l)">Disburse</button>
                  }
                }
              </div>
            </td>
          </tr>}
        </tbody></table></div>
      }

      <!-- Apply for Loan Modal (Task 138) -->
      @if (showApplyModal()) {
        <div class="modal-backdrop" (click)="showApplyModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Apply for a Loan</h3><button class="btn btn-ghost btn-icon" (click)="showApplyModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            @if (isAdmin()) {
              <div class="form-group"><label class="form-label">Crew Member</label>
                <select class="form-select" [(ngModel)]="applyForm.crew_member_id" id="select-crew-loan">
                  <option value="">— Select —</option>
                  @for (c of crewMembers(); track c.id) { <option [value]="c.id">{{ c.first_name }} {{ c.last_name }}</option> }
                </select>
              </div>
            }
            <div class="form-group"><label class="form-label">Category</label>
              <select class="form-select" [(ngModel)]="applyForm.category">
                @for (cat of loanCategories; track cat) { <option [value]="cat">{{ cat }}</option> }
              </select>
            </div>
            <div class="form-group"><label class="form-label">Amount (KES)</label>
              <input class="form-input" type="number" [(ngModel)]="applyForm.amount" min="100" placeholder="e.g. 5000" />
            </div>
            <div class="form-group"><label class="form-label">Tenure (days)</label>
              <input class="form-input" type="number" [(ngModel)]="applyForm.tenure_days" min="1" max="30" placeholder="e.g. 14" />
            </div>
            <div class="form-group"><label class="form-label">Purpose (optional)</label>
              <textarea class="form-textarea" [(ngModel)]="applyForm.purpose" rows="2" placeholder="What's the loan for?"></textarea>
            </div>
            @if (applyTier(); as t) {
              <div class="tier-info-banner">
                <span class="material-icons-round" style="color:var(--color-accent);font-size:18px;">info</span>
                <span>Your tier: <strong>{{ t.grade }}</strong> — max KES {{ t.max_loan_kes | number:'1.0-0' }} at {{ t.interest_rate }}%</span>
              </div>
            }
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" (click)="showApplyModal.set(false)">Cancel</button>
            <button class="btn btn-primary" (click)="submitApplication()" [disabled]="applying()">{{ applying() ? 'Submitting...' : 'Submit Application' }}</button>
          </div>
        </div></div>
      }

      <!-- Approve Loan Modal (Task 139) -->
      @if (showApproveModal()) {
        <div class="modal-backdrop" (click)="showApproveModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Approve Loan</h3><button class="btn btn-ghost btn-icon" (click)="showApproveModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group"><label class="form-label">Requested Amount</label>
              <div class="form-input" style="background:var(--color-surface-alt);cursor:not-allowed;">{{ (selectedLoan()?.amount_requested_cents || selectedLoan()?.amount_cents || 0) | currencyKes }}</div>
            </div>
            <div class="form-group"><label class="form-label">Approved Amount (KES)</label>
              <input class="form-input" type="number" [(ngModel)]="approveForm.approved_amount" min="100" />
            </div>
            <div class="form-group"><label class="form-label">Interest Rate (%)</label>
              <input class="form-input" type="number" [(ngModel)]="approveForm.interest_rate" min="0" max="100" step="0.5" />
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" (click)="showApproveModal.set(false)">Cancel</button>
            <button class="btn btn-primary" (click)="submitApproval()" [disabled]="approving()">{{ approving() ? 'Approving...' : 'Approve Loan' }}</button>
          </div>
        </div></div>
      }
    </div>`,
  styles: [`
    .clickable-row { cursor: pointer; transition: background 0.15s; &:hover { background: rgba(255,255,255,0.02); } }
    .tier-info-banner {
      display: flex; align-items: center; gap: var(--space-sm);
      padding: 10px 14px; border-radius: var(--radius-md);
      background: rgba(0,210,255,0.06); border: 1px solid rgba(0,210,255,0.15);
      font-size: 0.8125rem; color: var(--color-text-secondary);
    }
  `],
})
export class LoanListComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);
  private router = inject(Router);

  items = signal<LoanApplication[]>([]); loading = signal(true);
  crewMembers = signal<CrewMember[]>([]);
  showApplyModal = signal(false); applying = signal(false);
  showApproveModal = signal(false); approving = signal(false);
  selectedLoan = signal<LoanApplication | null>(null);
  applyTier = signal<LoanTier | null>(null);

  readonly loanCategories: LoanCategory[] = ['PERSONAL', 'EMERGENCY', 'EDUCATION', 'BUSINESS', 'ASSET'];

  applyForm = { crew_member_id: '', amount: 0, tenure_days: 14, category: 'PERSONAL' as LoanCategory, purpose: '' };
  approveForm = { approved_amount: 0, interest_rate: 8 };

  ngOnInit() {
    this.load();
    if (this.isAdmin()) {
      this.api.getCrewMembers({ per_page: '200' }).subscribe({ next: r => this.crewMembers.set(r.data) });
    }
  }

  isAdmin(): boolean { return this.auth.isAdmin(); }

  load() {
    this.loading.set(true);
    this.api.getLoans().subscribe({
      next: r => { this.items.set(r.data); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  viewLoan(l: LoanApplication) { this.router.navigate(['/loans', l.id]); }

  openApplyModal(): void {
    const user = this.auth.currentUser();
    if (user?.crew_member_id) {
      this.applyForm.crew_member_id = user.crew_member_id;
      this.api.getLoanTier(user.crew_member_id).subscribe({
        next: r => this.applyTier.set(r.data),
        error: () => this.applyTier.set(null),
      });
    }
    this.showApplyModal.set(true);
  }

  submitApplication(): void {
    const crewId = this.applyForm.crew_member_id || this.auth.currentUser()?.crew_member_id;
    if (!crewId || !this.applyForm.amount) return;
    this.applying.set(true);
    this.api.applyForLoan({
      crew_member_id: crewId,
      amount_cents: Math.round(this.applyForm.amount * 100),
      tenure_days: this.applyForm.tenure_days,
      category: this.applyForm.category,
      purpose: this.applyForm.purpose || undefined,
    }).subscribe({
      next: () => {
        this.toast.success('Loan application submitted');
        this.showApplyModal.set(false); this.applying.set(false);
        this.applyForm = { crew_member_id: '', amount: 0, tenure_days: 14, category: 'PERSONAL', purpose: '' };
        this.load();
      },
      error: () => this.applying.set(false),
    });
  }

  // --- Admin actions (Tasks 139, 140, 141) ---
  openApproveModal(l: LoanApplication): void {
    this.selectedLoan.set(l);
    this.approveForm.approved_amount = (l.amount_requested_cents || l.amount_cents) / 100;
    this.approveForm.interest_rate = 8;
    this.showApproveModal.set(true);
  }

  submitApproval(): void {
    const loan = this.selectedLoan();
    if (!loan) return;
    this.approving.set(true);
    this.api.approveLoan(loan.id, {
      approved_amount_cents: Math.round(this.approveForm.approved_amount * 100),
      interest_rate: this.approveForm.interest_rate / 100,
    }).subscribe({
      next: () => { this.toast.success('Loan approved'); this.showApproveModal.set(false); this.approving.set(false); this.load(); },
      error: () => this.approving.set(false),
    });
  }

  rejectLoan(l: LoanApplication): void {
    if (!confirm('Reject this loan application?')) return;
    this.api.rejectLoan(l.id).subscribe({
      next: () => { this.toast.success('Loan rejected'); this.load(); },
    });
  }

  disburseLoan(l: LoanApplication): void {
    if (!confirm('Disburse this loan? Funds will be credited to the crew member\'s wallet.')) return;
    this.api.disburseLoan(l.id).subscribe({
      next: () => { this.toast.success('Loan disbursed'); this.load(); },
    });
  }

  repay(l: LoanApplication) {
    const amt = prompt('Repayment amount (KES):');
    if (amt) {
      this.api.repayLoan(l.id, Math.round(parseFloat(amt) * 100)).subscribe({
        next: () => { this.toast.success('Repayment processed'); this.load(); },
      });
    }
  }

  statusBadge(s: string) {
    return s === 'REPAID' || s === 'COMPLETED' || s === 'APPROVED' ? 'badge-success'
      : s === 'REJECTED' || s === 'DEFAULTED' ? 'badge-danger'
      : s === 'DISBURSED' || s === 'REPAYING' ? 'badge-info'
      : 'badge-warning';
  }
}
