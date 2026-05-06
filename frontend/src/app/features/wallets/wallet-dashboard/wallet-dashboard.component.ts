import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';
import { Wallet, WalletTransaction, PaginationMeta, CrewMember } from '../../../core/models';

@Component({
  selector: 'app-wallet-dashboard',
  standalone: true,
  imports: [CommonModule, FormsModule, CurrencyKesPipe, RelativeTimePipe, AutocompleteComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Wallet</h1>
          <p class="page-subtitle">Manage your digital wallet and transactions</p>
        </div>
        <div class="page-actions">
          @if (isAdmin()) {
            <button class="btn btn-primary" (click)="openModal('credit')" id="btn-credit-wallet"
              title="Credit — Add money INTO a crew member's wallet. Use this to record wages earned, loan disbursements, or to top up a member's balance.">
              <span class="material-icons-round">add_circle</span> Credit
            </button>
            <button class="btn btn-secondary" (click)="openModal('debit')" id="btn-debit-wallet"
              title="Debit — Deduct money FROM a crew member's wallet. Use this to record withdrawals, insurance premium deductions, or to correct an over-credit.">
              <span class="material-icons-round">remove_circle</span> Debit
            </button>
          }
          @if (wallet()) {
            <button class="btn btn-ghost" (click)="openModal('payout')" id="btn-payout">
              <span class="material-icons-round">send</span> Payout
            </button>
            <button class="btn btn-ghost" (click)="exportCSV()" id="btn-export-csv">
              <span class="material-icons-round">download</span> CSV
            </button>
          }
        </div>
      </div>

      <!-- Admin: Crew member lookup -->
      @if (isAdmin()) {
        <div class="glass-card" style="margin-bottom:var(--space-lg);padding:var(--space-md); position: relative; z-index: 54;">
          <div class="lookup-row">
            <span class="material-icons-round" style="color:var(--color-accent);">person_search</span>
            <app-autocomplete [(ngModel)]="selectedCrewId" (ngModelChange)="onCrewSelected($event)" [options]="crewOptions()" placeholder="— Search crew member to view wallet —" id="select-crew-lookup" style="flex:1;"></app-autocomplete>
          </div>
        </div>
      }

      <!-- Balance cards -->
      @if (wallet(); as w) {
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));">
          <div class="stat-card">
            <div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">account_balance_wallet</span></div>
            <div class="stat-value">{{ w.balance_cents | currencyKes }}</div>
            <div class="stat-label">Available Balance</div>
          </div>
          <div class="stat-card">
            <div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">arrow_downward</span></div>
            <div class="stat-value">{{ w.total_credited_cents | currencyKes }}</div>
            <div class="stat-label">Total Credited</div>
          </div>
          <div class="stat-card">
            <div class="stat-icon" style="background:var(--color-danger-light);color:var(--color-danger);"><span class="material-icons-round">arrow_upward</span></div>
            <div class="stat-value">{{ w.total_debited_cents | currencyKes }}</div>
            <div class="stat-label">Total Debited</div>
          </div>
        </div>
      }

      <!-- Transaction filters + list -->
      <div class="glass-card" style="margin-top:var(--space-lg);">
        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:var(--space-md);flex-wrap:wrap;gap:var(--space-sm);">
          <h3 style="font-size:1rem;font-weight:600;margin:0;">Transaction History</h3>
          @if (txMeta(); as m) {
            <span style="font-size:0.75rem;color:var(--color-text-muted);">{{ m.total }} transactions</span>
          }
        </div>

        <!-- Filters bar -->
        <div class="filters-bar" style="margin-bottom:var(--space-md);">
          <select class="form-select" [(ngModel)]="filterType" (ngModelChange)="loadTransactions()" id="filter-tx-type">
            <option value="">All Types</option>
            <option value="CREDIT">Credit</option>
            <option value="DEBIT">Debit</option>
          </select>
          <select class="form-select" [(ngModel)]="filterCategory" (ngModelChange)="loadTransactions()" id="filter-tx-category">
            <option value="">All Categories</option>
            <option value="EARNING">Earning</option>
            <option value="WITHDRAWAL">Withdrawal</option>
            <option value="DEDUCTION">Deduction</option>
            <option value="TOP_UP">Top Up</option>
            <option value="REVERSAL">Reversal</option>
            <option value="LOAN">Loan</option>
          </select>
          <input type="date" class="form-input" [(ngModel)]="filterDateFrom" (ngModelChange)="loadTransactions()" placeholder="From" id="filter-date-from" style="max-width:160px;">
          <input type="date" class="form-input" [(ngModel)]="filterDateTo" (ngModelChange)="loadTransactions()" placeholder="To" id="filter-date-to" style="max-width:160px;">
        </div>

        @if (loadingTxs()) {
          @for (i of [1,2,3,4]; track i) { <div class="skeleton" style="height:48px;margin:4px 0;"></div> }
        } @else if (transactions().length === 0) {
          <div class="empty-state" style="padding:var(--space-xl);">
            <span class="material-icons-round empty-icon">receipt_long</span>
            <div class="empty-title">No transactions found</div>
            <div class="empty-subtitle">Adjust your filters or select a crew member</div>
          </div>
        } @else {
          <div class="tx-list">
            @for (tx of transactions(); track tx.id) {
              <div class="tx-item">
                <div class="tx-icon" [class.credit]="tx.transaction_type === 'CREDIT'" [class.debit]="tx.transaction_type === 'DEBIT'">
                  <span class="material-icons-round">{{ tx.transaction_type === 'CREDIT' ? 'arrow_downward' : 'arrow_upward' }}</span>
                </div>
                <div class="tx-info">
                  <span class="tx-category">{{ tx.category }}</span>
                  <span class="tx-description">{{ tx.description || tx.reference || '—' }}</span>
                </div>
                <div class="tx-amount" [class.text-success]="tx.transaction_type === 'CREDIT'" [class.text-danger]="tx.transaction_type === 'DEBIT'">
                  {{ tx.transaction_type === 'CREDIT' ? '+' : '-' }}{{ tx.amount_cents | currencyKes }}
                </div>
                <div class="tx-balance">Bal: {{ tx.balance_after_cents | currencyKes }}</div>
                <div class="tx-time">{{ tx.created_at | relativeTime }}</div>
              </div>
            }
          </div>

          <!-- Pagination -->
          @if (txMeta(); as m) {
            @if (m.total_pages > 1) {
              <div class="pagination" style="margin-top:var(--space-md);">
                <button class="page-btn" [disabled]="txPage === 1" (click)="goToPage(txPage - 1)">← Prev</button>
                <span style="font-size:0.8rem;color:var(--color-text-muted);">Page {{ txPage }} of {{ m.total_pages }}</span>
                <button class="page-btn" [disabled]="txPage >= m.total_pages" (click)="goToPage(txPage + 1)">Next →</button>
              </div>
            }
          }
        }
      </div>

      <!-- ==================== MODALS ==================== -->

      <!-- Credit Wallet Modal -->
      @if (showModal() === 'credit') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:480px;">
            <div class="modal-header">
              <h3 style="display:flex;align-items:center;gap:8px;">
                <span class="material-icons-round" style="color:var(--color-success);font-size:22px;">add_circle</span>
                Credit Wallet
              </h3>
              <button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button>
            </div>
            <div class="modal-body">
              <!-- Info banner -->
              <div class="modal-info-banner modal-info-banner--success">
                <span class="material-icons-round" style="font-size:18px;flex-shrink:0;">info</span>
                <span><strong>Crediting</strong> adds money <em>into</em> the member's wallet. Their balance will increase by the amount you enter. Use this to record wages, top-ups, or approved loan disbursements.</span>
              </div>
              <div style="position:relative; z-index: 54; margin-top:var(--space-md);">
                <label class="form-label">Crew Member <span class="field-required">*</span></label>
                <p class="field-hint">Search by name or staff ID — the money will be added to this person's wallet.</p>
                <app-autocomplete [(ngModel)]="modalCrewId" [options]="crewOptions()" placeholder="— Search Crew Member —"></app-autocomplete>
              </div>
              <label class="form-label" style="margin-top:var(--space-md);">Amount (KES) <span class="field-required">*</span></label>
              <p class="field-hint">Enter the exact amount in Kenyan Shillings (e.g. 1500 for KES 1,500).</p>
              <input type="number" class="form-input" [(ngModel)]="modalAmount" min="1" step="1" placeholder="e.g. 1500" id="modal-credit-amount">
              <label class="form-label" style="margin-top:var(--space-md);">Category <span class="field-required">*</span></label>
              <p class="field-hint">Choose what this credit is for:</p>
              <select class="form-select" [(ngModel)]="modalCategory" id="modal-credit-category">
                <option value="EARNING">Earning — Daily or shift wages earned by the member</option>
                <option value="TOP_UP">Top Up — Manual float top-up or balance correction</option>
                <option value="REVERSAL">Reversal — Cancelling a previous incorrect debit</option>
                <option value="LOAN">Loan — Disbursement of an approved loan to the member</option>
              </select>
              <label class="form-label" style="margin-top:var(--space-md);">Note (optional)</label>
              <p class="field-hint">Add a short note so the member and admin can see why this credit was made (e.g. "Shift 6am-2pm, Westlands route").</p>
              <input type="text" class="form-input" [(ngModel)]="modalDescription" placeholder="e.g. Shift 6am–2pm, Route 111" id="modal-credit-desc">
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
              <button class="btn btn-primary" (click)="submitCredit()" [disabled]="submitting()" id="btn-submit-credit"
                title="Confirm: This will add the entered amount into the selected member's wallet immediately.">
                <span class="material-icons-round" style="font-size:18px;">{{ submitting() ? 'hourglass_empty' : 'add_circle' }}</span>
                {{ submitting() ? 'Processing...' : 'Credit Wallet' }}
              </button>
            </div>
          </div>
        </div>
      }

      <!-- Debit Wallet Modal -->
      @if (showModal() === 'debit') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:480px;">
            <div class="modal-header">
              <h3 style="display:flex;align-items:center;gap:8px;">
                <span class="material-icons-round" style="color:var(--color-danger);font-size:22px;">remove_circle</span>
                Debit Wallet
              </h3>
              <button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button>
            </div>
            <div class="modal-body">
              <!-- Info banner -->
              <div class="modal-info-banner modal-info-banner--warning">
                <span class="material-icons-round" style="font-size:18px;flex-shrink:0;">warning_amber</span>
                <span><strong>Debiting</strong> removes money <em>from</em> the member's wallet. Their balance will decrease. Use this only to record withdrawals they've taken, statutory deductions (e.g. NSSF, NHIF), or to reverse an accidental over-credit.</span>
              </div>
              <div style="position:relative; z-index: 54; margin-top:var(--space-md);">
                <label class="form-label">Crew Member <span class="field-required">*</span></label>
                <p class="field-hint">Search by name or staff ID — the amount will be deducted from this person's wallet.</p>
                <app-autocomplete [(ngModel)]="modalCrewId" [options]="crewOptions()" placeholder="— Search Crew Member —"></app-autocomplete>
              </div>
              <label class="form-label" style="margin-top:var(--space-md);">Amount (KES) <span class="field-required">*</span></label>
              <p class="field-hint">Enter the exact amount to deduct. The member's wallet must have enough balance.</p>
              <input type="number" class="form-input" [(ngModel)]="modalAmount" min="1" step="1" placeholder="e.g. 200" id="modal-debit-amount">
              <label class="form-label" style="margin-top:var(--space-md);">Category <span class="field-required">*</span></label>
              <p class="field-hint">Choose the reason for this deduction:</p>
              <select class="form-select" [(ngModel)]="modalCategory" id="modal-debit-category">
                <option value="WITHDRAWAL">Withdrawal — Member has taken out cash or M-Pesa payment</option>
                <option value="DEDUCTION">Deduction — Statutory deductions e.g. NSSF, NHIF, insurance premium</option>
                <option value="REVERSAL">Reversal — Undoing a previous incorrect credit entry</option>
              </select>
              <label class="form-label" style="margin-top:var(--space-md);">Reason <span class="field-required">*</span></label>
              <p class="field-hint">Always provide a clear reason. This appears in the member's transaction history (e.g. "NSSF deduction May 2026").</p>
              <input type="text" class="form-input" [(ngModel)]="modalDescription" placeholder="e.g. NSSF deduction May 2026" id="modal-debit-desc">
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
              <button class="btn btn-danger" (click)="submitDebit()" [disabled]="submitting()" id="btn-submit-debit"
                title="Confirm: This will deduct the entered amount from the selected member's wallet immediately. This action is logged and cannot be automatically undone.">
                <span class="material-icons-round" style="font-size:18px;">{{ submitting() ? 'hourglass_empty' : 'remove_circle' }}</span>
                {{ submitting() ? 'Processing...' : 'Debit Wallet' }}
              </button>
            </div>
          </div>
        </div>
      }

      <!-- Payout Modal -->
      @if (showModal() === 'payout') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:520px;">
            <div class="modal-header"><h3>Initiate Payout</h3><button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button></div>
            <div class="modal-body">
              <label class="form-label">Payout Channel</label>
              <select class="form-select" [(ngModel)]="payoutChannel" (ngModelChange)="onChannelChange()" id="modal-payout-channel">
                <option value="MOMO_B2C">M-Pesa (B2C)</option>
                <option value="BANK">Bank Transfer</option>
                <option value="MOMO_B2B">Paybill / Till</option>
              </select>
              <label class="form-label" style="margin-top:var(--space-sm);">Amount (KES)</label>
              <input type="number" class="form-input" [(ngModel)]="modalAmount" min="1" step="1" placeholder="e.g. 1000" id="modal-payout-amount">
              <label class="form-label" style="margin-top:var(--space-sm);">Recipient Name</label>
              <input type="text" class="form-input" [(ngModel)]="payoutRecipient" placeholder="Full name" id="modal-payout-name">

              @if (payoutChannel === 'MOMO_B2C') {
                <label class="form-label" style="margin-top:var(--space-sm);">Recipient Phone</label>
                <input type="tel" class="form-input" [(ngModel)]="payoutPhone" placeholder="+254712345678" id="modal-payout-phone">
              }
              @if (payoutChannel === 'BANK') {
                <label class="form-label" style="margin-top:var(--space-sm);">Bank Code</label>
                <input type="text" class="form-input" [(ngModel)]="payoutBankCode" placeholder="e.g. 01" id="modal-payout-bankcode">
                <label class="form-label" style="margin-top:var(--space-sm);">Account Number</label>
                <input type="text" class="form-input" [(ngModel)]="payoutBankAccount" placeholder="Account number" id="modal-payout-account">
              }
              @if (payoutChannel === 'MOMO_B2B') {
                <label class="form-label" style="margin-top:var(--space-sm);">Paybill Number</label>
                <input type="text" class="form-input" [(ngModel)]="payoutPaybill" placeholder="e.g. 888880" id="modal-payout-paybill">
                <label class="form-label" style="margin-top:var(--space-sm);">Account Reference</label>
                <input type="text" class="form-input" [(ngModel)]="payoutPaybillRef" placeholder="Account ref" id="modal-payout-ref">
              }
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
              <button class="btn btn-primary" (click)="submitPayout()" [disabled]="submitting()" id="btn-submit-payout">
                {{ submitting() ? 'Processing...' : 'Send Payout' }}
              </button>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .lookup-row { display: flex; align-items: center; gap: var(--space-md); }
    .tx-list { display: flex; flex-direction: column; }
    .tx-item { display: flex; align-items: center; gap: var(--space-md); padding: 12px 0; border-bottom: 1px solid var(--color-border); &:last-child { border-bottom: none; } }
    .tx-icon { width: 36px; height: 36px; border-radius: var(--radius-md); display: flex; align-items: center; justify-content: center; flex-shrink: 0; .material-icons-round { font-size: 18px; } }
    .tx-icon.credit { background: var(--color-success-light); color: var(--color-success); }
    .tx-icon.debit { background: var(--color-danger-light); color: var(--color-danger); }
    .tx-info { flex: 1; display: flex; flex-direction: column; min-width: 0; }
    .tx-category { font-size: 0.875rem; font-weight: 500; color: var(--color-text-primary); }
    .tx-description { font-size: 0.75rem; color: var(--color-text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .tx-amount { font-weight: 600; font-size: 0.875rem; white-space: nowrap; }
    .tx-balance { font-size: 0.7rem; color: var(--color-text-muted); white-space: nowrap; }
    .tx-time { font-size: 0.75rem; color: var(--color-text-muted); white-space: nowrap; }
    .form-label { display: block; font-size: 0.8rem; font-weight: 500; color: var(--color-text-secondary); margin-bottom: 2px; }
    .field-hint { font-size: 0.72rem; color: var(--color-text-muted); margin: 0 0 6px; line-height: 1.4; }
    .field-required { color: var(--color-danger); margin-left: 2px; }
    .modal-info-banner { display: flex; align-items: flex-start; gap: 10px; padding: 12px 14px; border-radius: var(--radius-md); font-size: 0.78rem; line-height: 1.5; }
    .modal-info-banner--success { background: var(--color-success-light); color: var(--color-success); border: 1px solid rgba(34,197,94,0.25); }
    .modal-info-banner--warning { background: rgba(251,191,36,0.1); color: #92400e; border: 1px solid rgba(251,191,36,0.35); }
    @media (max-width: 600px) { .tx-time, .tx-balance { display: none; } }
  `]
})
export class WalletDashboardComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);

  wallet = signal<Wallet | null>(null);
  transactions = signal<WalletTransaction[]>([]);
  txMeta = signal<PaginationMeta | null>(null);
  loadingTxs = signal(true);
  crewMembers = signal<CrewMember[]>([]);
  crewOptions = computed<AutocompleteOption[]>(() => this.crewMembers().map(c => ({
    value: c.id,
    label: `${c.first_name} ${c.last_name}`,
    sublabel: `ID: ${c.crew_id}`,
    searchText: `${c.first_name} ${c.last_name} ${c.crew_id}`
  })));

  showModal = signal<'credit' | 'debit' | 'payout' | null>(null);
  submitting = signal(false);

  // Active crew member ID for wallet view
  activeCrewId = '';
  selectedCrewId = '';

  // Filters
  filterType = '';
  filterCategory = '';
  filterDateFrom = '';
  filterDateTo = '';
  txPage = 1;

  // Modal fields
  modalCrewId = '';
  modalAmount = 0;
  modalCategory = 'EARNING';
  modalDescription = '';

  // Payout fields
  payoutChannel = 'MOMO_B2C';
  payoutRecipient = '';
  payoutPhone = '';
  payoutBankCode = '';
  payoutBankAccount = '';
  payoutPaybill = '';
  payoutPaybillRef = '';

  isAdmin(): boolean {
    return this.auth.hasRole('SYSTEM_ADMIN', 'SACCO_ADMIN');
  }

  ngOnInit(): void {
    const user = this.auth.currentUser();

    if (this.isAdmin()) {
      // Load crew members for lookup dropdown
      this.api.getCrewMembers({ per_page: '200' }).subscribe({
        next: (res) => this.crewMembers.set(res.data),
      });
      this.loadingTxs.set(false);
    } else if (user?.crew_member_id) {
      this.activeCrewId = user.crew_member_id;
      this.loadWallet();
      this.loadTransactions();
    } else {
      this.loadingTxs.set(false);
    }
  }

  onCrewSelected(crewId: string): void {
    this.activeCrewId = crewId;
    this.txPage = 1;
    if (crewId) {
      this.loadWallet();
      this.loadTransactions();
    } else {
      this.wallet.set(null);
      this.transactions.set([]);
      this.txMeta.set(null);
    }
  }

  loadWallet(): void {
    if (!this.activeCrewId) return;
    this.api.getWalletBalance(this.activeCrewId).subscribe({
      next: (res) => this.wallet.set(res.data),
      error: () => this.wallet.set(null),
    });
  }

  loadTransactions(): void {
    if (!this.activeCrewId) return;
    this.loadingTxs.set(true);
    const params: Record<string, string> = {
      page: String(this.txPage),
      per_page: '20',
    };
    if (this.filterType) params['transaction_type'] = this.filterType;
    if (this.filterCategory) params['category'] = this.filterCategory;
    if (this.filterDateFrom) params['date_from'] = this.filterDateFrom;
    if (this.filterDateTo) params['date_to'] = this.filterDateTo;

    this.api.getWalletTransactions(this.activeCrewId, params).subscribe({
      next: (res) => {
        this.transactions.set(res.data || []);
        this.txMeta.set(res.meta);
        this.loadingTxs.set(false);
      },
      error: () => {
        this.transactions.set([]);
        this.loadingTxs.set(false);
      },
    });
  }

  goToPage(page: number): void {
    this.txPage = page;
    this.loadTransactions();
  }

  exportCSV(): void {
    if (!this.activeCrewId) return;
    this.api.exportWalletCSV(this.activeCrewId).subscribe({
      next: (blob) => {
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'wallet_statement.csv';
        a.click();
        URL.revokeObjectURL(url);
        this.toast.success('CSV exported');
      },
    });
  }

  // --- Modals ---
  openModal(type: 'credit' | 'debit' | 'payout'): void {
    this.modalAmount = 0;
    this.modalDescription = '';
    this.modalCrewId = this.activeCrewId || '';
    this.modalCategory = type === 'credit' ? 'EARNING' : 'WITHDRAWAL';
    this.payoutChannel = 'MOMO_B2C';
    this.payoutRecipient = '';
    this.payoutPhone = '';
    this.payoutBankCode = '';
    this.payoutBankAccount = '';
    this.payoutPaybill = '';
    this.payoutPaybillRef = '';
    this.showModal.set(type);
  }

  closeModal(): void {
    this.showModal.set(null);
  }

  onChannelChange(): void { /* template handles conditional fields */ }

  private generateIdempotencyKey(): string {
    return crypto.randomUUID();
  }

  submitCredit(): void {
    if (!this.modalCrewId || this.modalAmount <= 0) {
      this.toast.error('Select a crew member and enter a valid amount');
      return;
    }
    this.submitting.set(true);
    this.api.creditWallet({
      crew_member_id: this.modalCrewId,
      amount_cents: Math.round(this.modalAmount * 100),
      category: this.modalCategory,
      description: this.modalDescription,
    }, this.generateIdempotencyKey()).subscribe({
      next: () => {
        this.toast.success('Wallet credited successfully');
        this.closeModal();
        this.submitting.set(false);
        this.loadWallet();
        this.loadTransactions();
      },
      error: () => this.submitting.set(false),
    });
  }

  submitDebit(): void {
    if (!this.modalCrewId || this.modalAmount <= 0) {
      this.toast.error('Select a crew member and enter a valid amount');
      return;
    }
    this.submitting.set(true);
    this.api.debitWallet({
      crew_member_id: this.modalCrewId,
      amount_cents: Math.round(this.modalAmount * 100),
      category: this.modalCategory,
      description: this.modalDescription,
    }, this.generateIdempotencyKey()).subscribe({
      next: () => {
        this.toast.success('Wallet debited successfully');
        this.closeModal();
        this.submitting.set(false);
        this.loadWallet();
        this.loadTransactions();
      },
      error: () => this.submitting.set(false),
    });
  }

  submitPayout(): void {
    const crewId = this.activeCrewId || this.auth.currentUser()?.crew_member_id;
    if (!crewId || this.modalAmount <= 0 || !this.payoutRecipient) {
      this.toast.error('Fill in all required fields');
      return;
    }
    this.submitting.set(true);
    const data: Record<string, unknown> = {
      amount_cents: Math.round(this.modalAmount * 100),
      channel: this.payoutChannel,
      recipient_name: this.payoutRecipient,
    };
    if (this.payoutChannel === 'MOMO_B2C') data['recipient_phone'] = this.payoutPhone;
    if (this.payoutChannel === 'BANK') { data['bank_code'] = this.payoutBankCode; data['bank_account'] = this.payoutBankAccount; }
    if (this.payoutChannel === 'MOMO_B2B') { data['paybill_number'] = this.payoutPaybill; data['paybill_ref'] = this.payoutPaybillRef; }

    this.api.initiatePayout(crewId, data, this.generateIdempotencyKey()).subscribe({
      next: () => {
        this.toast.success('Payout initiated successfully');
        this.closeModal();
        this.submitting.set(false);
        this.loadWallet();
        this.loadTransactions();
      },
      error: () => this.submitting.set(false),
    });
  }
}
