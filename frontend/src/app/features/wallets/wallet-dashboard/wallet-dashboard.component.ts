import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';
import { Wallet, WalletTransaction, PaginationMeta, CrewMember, SACCOFloat, SACCOFloatTransaction, Organization } from '../../../core/models';

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
            <button class="btn btn-primary" (click)="openModal('topup')" id="btn-topup-float"
              title="Top Up — Add funds to your organization's float account. This is the source for employee payouts.">
              <span class="material-icons-round">account_balance</span> Top Up Float
            </button>
            <button class="btn btn-secondary" (click)="openModal('payout')" id="btn-payout"
              title="Pay Employee — Send wages to a crew member's wallet. Deductions (NSSF, SHA, etc.) are applied before payout.">
              <span class="material-icons-round">send</span> Pay Employee
            </button>
            <button class="btn btn-ghost" (click)="openModal('credit')" id="btn-credit-wallet"
              title="Credit — Manually add money into a crew member's wallet for corrections or reversals.">
              <span class="material-icons-round">add_circle</span> Credit
            </button>
          }
          @if (wallet()) {
            <button class="btn btn-ghost" (click)="exportCSV()" id="btn-export-csv">
              <span class="material-icons-round">download</span> CSV
            </button>
          }
        </div>
      </div>

      <!-- Admin: Organization Selector (SYSTEM_ADMIN without org_id) -->
      @if (showOrgSelector()) {
        <div class="glass-card" style="margin-bottom:var(--space-md);padding:var(--space-sm) var(--space-md); display:flex; align-items:center; gap:var(--space-sm); position:relative; z-index:55;">
          <span class="material-icons-round" style="color:var(--color-accent);">business</span>
          <label style="font-weight:500; font-size:0.85rem; white-space:nowrap;">Active Organization:</label>
          <app-autocomplete [(ngModel)]="selectedOrgId" (ngModelChange)="onOrgChanged($event)" [options]="orgOptions()" placeholder="— Search organization —" id="select-org" style="flex:1; max-width:400px;"></app-autocomplete>
        </div>
      }

      <!-- Admin: Organization Float Balance -->
      @if (isAdmin() && orgFloat()) {
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); margin-bottom: var(--space-lg);">
          <div class="stat-card" style="border-left: 3px solid var(--color-accent);">
            <div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">account_balance</span></div>
            <div class="stat-value">{{ orgFloat()!.balance_cents | currencyKes }}</div>
            <div class="stat-label">Organization Float</div>
          </div>
        </div>

        <!-- Recent Sync Status -->
        @if (recentSyncedTopUps().length > 0) {
          <div class="glass-card" style="margin-bottom:var(--space-lg);padding:var(--space-md);">
            <div style="display:flex;align-items:center;justify-content:space-between;margin:0 0 var(--space-sm) 0;">
              <h3 style="display:flex;align-items:center;gap:8px;margin:0;font-size:0.95rem;font-weight:600;">
                <span class="material-icons-round" style="color:var(--color-success);font-size:20px;">sync</span>
                Recent Sync Activity
              </h3>
            </div>
            <div style="display:flex;flex-direction:column;gap:6px;">
              @for (stx of recentSyncedTopUps(); track stx.id) {
                <div class="sync-status-row" [class.sync-status-row--completed]="stx.status === 'COMPLETED'" [class.sync-status-row--failed]="stx.status === 'FAILED'">
                  <span class="material-icons-round" style="font-size:18px;">{{ stx.status === 'COMPLETED' ? 'check_circle' : 'cancel' }}</span>
                  <div style="flex:1;min-width:0;">
                    <div style="font-size:0.8rem;font-weight:500;color:var(--color-text-primary);">{{ stx.amount_cents | currencyKes }}</div>
                    <div style="font-size:0.68rem;color:var(--color-text-muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{ stx.reference || 'No reference' }}</div>
                  </div>
                  <span class="sync-badge" [class.sync-badge--callback]="stx.sync_method === 'CALLBACK'" [class.sync-badge--poll]="stx.sync_method === 'POLL'" [class.sync-badge--manual]="stx.sync_method === 'MANUAL'">
                    <span class="material-icons-round" style="font-size:12px;">{{ getSyncIcon(stx.sync_method) }}</span>
                    {{ getSyncLabel(stx.sync_method) }}
                  </span>
                  <span class="sync-status-chip" [class.sync-status-chip--synced]="stx.status === 'COMPLETED'" [class.sync-status-chip--failed]="stx.status === 'FAILED'">
                    {{ stx.status === 'COMPLETED' ? 'Synced' : 'Failed' }}
                  </span>
                  <div style="font-size:0.68rem;color:var(--color-text-muted);white-space:nowrap;">{{ stx.synced_at ? (stx.synced_at | relativeTime) : (stx.created_at | relativeTime) }}</div>
                </div>
              }
            </div>
          </div>
        }

        <!-- Pending Top-Up Approvals -->
        @if (pendingTopUps().length > 0) {
          <div class="glass-card" style="margin-bottom:var(--space-lg);padding:var(--space-md);">
            <div style="display:flex;align-items:center;justify-content:space-between;margin:0 0 var(--space-sm) 0;">
              <h3 style="display:flex;align-items:center;gap:8px;margin:0;font-size:0.95rem;font-weight:600;">
                <span class="material-icons-round" style="color:#f59e0b;font-size:20px;">pending_actions</span>
                Pending Top-Up Approvals
                <span style="background:#f59e0b;color:#fff;font-size:0.7rem;padding:2px 8px;border-radius:100px;font-weight:700;">{{ pendingTopUps().length }}</span>
              </h3>
              <button class="btn-sync" (click)="pollPendingSTK()" [disabled]="polling()" id="btn-poll-stk" title="Check payment gateway for completed payments">
                <span class="material-icons-round" [class.spin]="polling()" style="font-size:16px;">sync</span>
                {{ polling() ? 'Checking...' : 'Sync Payments' }}
              </button>
            </div>
            @if (pollResult()) {
              <div class="poll-result" [class.poll-result--success]="pollResult()!.confirmed > 0" [class.poll-result--warning]="pollResult()!.confirmed === 0">
                <span class="material-icons-round" style="font-size:16px;">{{ pollResult()!.confirmed > 0 ? 'check_circle' : 'info' }}</span>
                <span>{{ pollResult()!.message }}</span>
                @if (pollResult()!.confirmed > 0) {
                  <span style="font-weight:600;"> — {{ pollResult()!.confirmed }} payment(s) confirmed!</span>
                }
              </div>
            }
            <p style="font-size:0.75rem;color:var(--color-text-muted);margin:0 0 var(--space-sm);">
              These top-ups require verification before the float balance is credited. Use <strong>Sync Payments</strong> to auto-check with the payment gateway, or confirm/reject manually.
            </p>
            <div style="display:flex;flex-direction:column;gap:8px;">
              @for (ptx of pendingTopUps(); track ptx.id) {
                <div style="display:flex;align-items:center;gap:var(--space-sm);padding:10px 14px;border-radius:var(--radius-md);border:1.5px solid rgba(251,191,36,0.4);background:rgba(251,191,36,0.05);">
                  <span class="material-icons-round" style="color:#f59e0b;font-size:20px;">schedule</span>
                  <div style="flex:1;min-width:0;">
                    <div style="font-size:0.82rem;font-weight:500;color:var(--color-text-primary);">{{ ptx.amount_cents | currencyKes }}</div>
                    <div style="font-size:0.7rem;color:var(--color-text-muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{ ptx.reference || 'No reference' }}</div>
                  </div>
                  <div style="font-size:0.7rem;color:var(--color-text-muted);white-space:nowrap;">{{ ptx.created_at | relativeTime }}</div>
                  <span class="sync-status-chip sync-status-chip--pending">
                    <span class="material-icons-round" style="font-size:11px;">sync_disabled</span>
                    Not Synced
                  </span>
                  <button class="btn-sync-sm" (click)="pollSingleTx(ptx.id)" [disabled]="pollingTxId() === ptx.id" title="Check this payment's status with JamboPay">
                    <span class="material-icons-round" [class.spin]="pollingTxId() === ptx.id" style="font-size:14px;">sync</span>
                  </button>
                  <button class="btn btn-primary" style="padding:4px 12px;font-size:0.72rem;min-height:28px;" (click)="confirmPendingTopUp(ptx.id)" id="btn-confirm-{{ptx.id}}">
                    <span class="material-icons-round" style="font-size:14px;">check</span> Confirm
                  </button>
                  <button class="btn btn-ghost" style="padding:4px 10px;font-size:0.72rem;min-height:28px;color:var(--color-danger);" (click)="rejectPendingTopUp(ptx.id)" id="btn-reject-{{ptx.id}}">
                    <span class="material-icons-round" style="font-size:14px;">close</span> Reject
                  </button>
                </div>
              }
            </div>
          </div>
        }
      }

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

      <!-- Crew Member Quick Actions -->
      @if (!isAdmin() && wallet()) {
        <div class="qa-grid" style="margin-top:var(--space-lg);">
          <button class="qa-card" (click)="openCrewModal('withdraw')" id="qa-withdraw">
            <div class="qa-icon qa-icon--withdraw"><span class="material-icons-round">savings</span></div>
            <span class="qa-label">Withdraw</span>
            <span class="qa-hint">To M-Pesa or Bank</span>
          </button>
          <button class="qa-card" (click)="openCrewModal('transfer')" id="qa-transfer">
            <div class="qa-icon qa-icon--transfer"><span class="material-icons-round">swap_horiz</span></div>
            <span class="qa-label">Transfer</span>
            <span class="qa-hint">To another wallet</span>
          </button>
          <button class="qa-card" (click)="openCrewModal('airtime')" id="qa-airtime">
            <div class="qa-icon qa-icon--airtime"><span class="material-icons-round">phone_android</span></div>
            <span class="qa-label">Buy Airtime</span>
            <span class="qa-hint">Safaricom, Airtel, Telkom</span>
          </button>
          <button class="qa-card" (click)="openCrewModal('bills')" id="qa-bills">
            <div class="qa-icon qa-icon--bills"><span class="material-icons-round">receipt_long</span></div>
            <span class="qa-label">Pay Bills</span>
            <span class="qa-hint">KPLC, Water, TV</span>
          </button>
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
            <option value="ADJUSTMENT">Adjustment</option>
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
                <option value="ADJUSTMENT">Adjustment — Manual balance correction by admin</option>
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

      <!-- Top Up Float Modal -->
      @if (showModal() === 'topup') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:540px;">
            <div class="modal-header">
              <h3 style="display:flex;align-items:center;gap:8px;">
                <span class="material-icons-round" style="color:var(--color-accent);font-size:22px;">account_balance</span>
                Top Up Organization Float
              </h3>
              <button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button>
            </div>
            <div class="modal-body">
              <div class="modal-info-banner modal-info-banner--success">
                <span class="material-icons-round" style="font-size:18px;flex-shrink:0;">info</span>
                <span>Add funds to your organization's float. Choose a payment method below.</span>
              </div>

              <!-- Payment Method Selection -->
              <label class="form-label" style="margin-top:var(--space-md);">Payment Method <span class="field-required">*</span></label>
              <div class="pm-grid">
                @for (method of availablePaymentMethods(); track method.id) {
                  <button class="pm-card" [class.pm-card--active]="topupMethod === method.id" (click)="selectTopUpMethod(method.id)" type="button">
                    <span class="material-icons-round pm-icon">{{ method.icon }}</span>
                    <span class="pm-label">{{ method.label }}</span>
                    <span class="pm-hint">{{ method.hint }}</span>
                  </button>
                }
              </div>

              <!-- Provider Sub-options -->
              @if (topupMethod) {
                <div class="pm-providers" style="margin-top:var(--space-md);">
                  <label class="form-label">{{ topupMethod === 'mobile_money' ? 'Mobile Provider' : topupMethod === 'bank' ? 'Bank' : 'Card Type' }} <span class="field-required">*</span></label>
                  <div class="provider-chips">
                    @for (p of getProviders(topupMethod); track p.id) {
                      <button class="provider-chip" [class.provider-chip--active]="topupProvider === p.id" (click)="topupProvider = p.id" type="button">
                        <span class="provider-chip-icon">{{ p.emoji }}</span>
                        <span>{{ p.label }}</span>
                      </button>
                    }
                  </div>
                </div>
              }

              <!-- Dynamic fields per method -->
              @if (topupProvider) {
                <div style="margin-top:var(--space-md);">
                  @if (topupMethod === 'mobile_money') {
                    <label class="form-label">Phone Number <span class="field-required">*</span></label>
                    <p class="field-hint">The M-Pesa/Airtel number to initiate STK push.</p>
                    <input type="tel" class="form-input" [(ngModel)]="topupPhone" placeholder="e.g. 0712345678" id="topup-phone">
                  }
                  @if (topupMethod === 'bank') {
                    @if (topupProvider === 'rtgs') {
                      <label class="form-label">RTGS Reference <span class="field-required">*</span></label>
                      <p class="field-hint">Enter the RTGS transfer reference from your bank.</p>
                      <input type="text" class="form-input" [(ngModel)]="topupBankRef" placeholder="e.g. RTGS2026050700123" id="topup-bank-ref">
                    } @else {
                      <label class="form-label">Account / Transaction Reference <span class="field-required">*</span></label>
                      <p class="field-hint">Paybill or bank transfer reference number.</p>
                      <input type="text" class="form-input" [(ngModel)]="topupBankRef" placeholder="e.g. TXN-20260507-001" id="topup-bank-ref">
                    }
                  }
                  @if (topupMethod === 'card') {
                    <div class="modal-info-banner modal-info-banner--warning" style="margin-bottom:var(--space-sm);">
                      <span class="material-icons-round" style="font-size:18px;flex-shrink:0;">credit_card</span>
                      <span>You will be redirected to a secure payment gateway to complete the card transaction.</span>
                    </div>
                  }
                </div>
              }

              <!-- Amount -->
              @if (topupProvider) {
                <label class="form-label" style="margin-top:var(--space-md);">Amount (KES) <span class="field-required">*</span></label>
                <p class="field-hint">Enter the amount to add to the organization's float.</p>
                <input type="number" class="form-input" [(ngModel)]="modalAmount" min="1" step="1" placeholder="e.g. 100,000" id="modal-topup-amount">
                <label class="form-label" style="margin-top:var(--space-md);">Reference Note (optional)</label>
                <input type="text" class="form-input" [(ngModel)]="modalDescription" placeholder="e.g. May 2026 payroll funding" id="modal-topup-ref">
              }
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
              <button class="btn btn-primary" (click)="submitTopUp()" [disabled]="submitting() || !topupProvider || modalAmount <= 0" id="btn-submit-topup">
                <span class="material-icons-round" style="font-size:18px;">{{ submitting() ? 'hourglass_empty' : getTopUpIcon() }}</span>
                {{ submitting() ? 'Processing...' : 'Top Up via ' + getProviderLabel(topupProvider) }}
              </button>
            </div>
          </div>
        </div>
      }

      <!-- Pay Employee Modal (Single or Bulk) -->
      @if (showModal() === 'payout') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:620px;">
            <div class="modal-header">
              <h3 style="display:flex;align-items:center;gap:8px;">
                <span class="material-icons-round" style="color:var(--color-success);font-size:22px;">payments</span>
                Pay Employee
              </h3>
              <button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button>
            </div>

            <!-- Tab toggle -->
            <div style="display:flex;gap:0;border-bottom:1px solid var(--color-border);padding:0 var(--space-lg);">
              <button class="payout-tab" [class.payout-tab--active]="payoutTab() === 'single'"
                      (click)="payoutTab.set('single')" id="tab-single-pay">
                <span class="material-icons-round" style="font-size:16px;">person</span> Single Employee
              </button>
              <button class="payout-tab" [class.payout-tab--active]="payoutTab() === 'bulk'"
                      (click)="payoutTab.set('bulk')" id="tab-bulk-pay">
                <span class="material-icons-round" style="font-size:16px;">group</span> Bulk Upload
              </button>
            </div>

            <!-- ── SINGLE EMPLOYEE TAB ── -->
            @if (payoutTab() === 'single') {
              <div class="modal-body">
                <div class="modal-info-banner" [class.modal-info-banner--success]="statutoryEnabled()"
                     [style]="!statutoryEnabled() ? 'background:rgba(99,102,241,0.07);color:#6366f1;border:1px solid rgba(99,102,241,0.2);' : ''">
                  <span class="material-icons-round" style="font-size:18px;flex-shrink:0;">info</span>
                  @if (statutoryEnabled()) {
                    <span>Pay an employee from the org float. <strong>Statutory deductions (NSSF, SHA, Housing Levy) apply</strong> — the employee receives the net amount in their wallet.</span>
                  } @else {
                    <span><strong>Informal worker mode</strong> — statutory deductions are not applied. The employee receives the full gross amount. Enable statutory handling in organization settings for formal sector payroll.</span>
                  }
                </div>
                <div style="position:relative; z-index: 54; margin-top:var(--space-md);">
                  <label class="form-label">Employee <span class="field-required">*</span></label>
                  <app-autocomplete [(ngModel)]="modalCrewId" [options]="crewOptions()" placeholder="— Search Employee —"></app-autocomplete>
                </div>
                <label class="form-label" style="margin-top:var(--space-md);">Gross Amount (KES) <span class="field-required">*</span></label>
                <p class="field-hint">Total earnings before deductions.</p>
                <input type="number" class="form-input" [(ngModel)]="modalAmount" min="1" step="1" placeholder="e.g. 15000" id="modal-payout-gross" (ngModelChange)="recalcNetPay()">

                <!-- Deductions Section -->
                <div style="margin-top:var(--space-md);padding:var(--space-md);background:var(--color-bg-secondary);border-radius:var(--radius-md);border:1px solid var(--color-border);">
                  <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:var(--space-sm);">
                    <label class="form-label" style="margin:0;font-size:0.85rem;">Deductions</label>
                    <span style="font-size:0.72rem;color:var(--color-text-muted);">Applied before payout</span>
                  </div>
                  <!-- Statutory deductions — only shown when employer has formal mode enabled -->
                  @if (statutoryEnabled()) {
                    <div class="deduction-row">
                      <label>NSSF <span style="font-size:0.65rem;color:var(--color-accent);">statutory</span></label>
                      <input type="number" class="form-input form-input--sm" [(ngModel)]="deductionNSSF" min="0" step="1" placeholder="0" (ngModelChange)="recalcNetPay()">
                    </div>
                    <div class="deduction-row">
                      <label>SHA (NHIF) <span style="font-size:0.65rem;color:var(--color-accent);">statutory</span></label>
                      <input type="number" class="form-input form-input--sm" [(ngModel)]="deductionSHA" min="0" step="1" placeholder="0" (ngModelChange)="recalcNetPay()">
                    </div>
                    <div class="deduction-row">
                      <label>Housing Levy <span style="font-size:0.65rem;color:var(--color-accent);">statutory</span></label>
                      <input type="number" class="form-input form-input--sm" [(ngModel)]="deductionHousing" min="0" step="1" placeholder="0" (ngModelChange)="recalcNetPay()">
                    </div>
                  }
                  <!-- Non-statutory deductions — only shown when configured in tenant settings -->
                  @if (activeDeductions().length > 0) {
                    @for (ded of activeDeductions(); track ded.code) {
                      <div class="deduction-row">
                        <label>{{ ded.label }}</label>
                        <input type="number" class="form-input form-input--sm"
                               [ngModel]="deductionValues[ded.code] || 0"
                               (ngModelChange)="deductionValues[ded.code] = +$event; recalcNetPay()"
                               min="0" step="1" placeholder="0">
                      </div>
                    }
                  } @else if (!statutoryEnabled()) {
                    <div style="padding:8px 0;font-size:0.75rem;color:var(--color-text-muted);display:flex;align-items:center;gap:6px;">
                      <span class="material-icons-round" style="font-size:15px;">info</span>
                      No deductions configured. Enable them in <strong>Settings → Finance → Deduction Types</strong>.
                    </div>
                  }
                </div>

                <!-- Net Pay Summary -->
                <div class="net-pay-summary" style="margin-top:var(--space-md);">
                  <div class="net-row"><span>Gross Pay</span><span>{{ (modalAmount || 0) | number:'1.0-0' }} KES</span></div>
                  <div class="net-row net-row--deduction"><span>Total Deductions</span><span>- {{ totalDeductions() | number:'1.0-0' }} KES</span></div>
                  <div class="net-row net-row--total"><span><strong>Net Pay (to wallet)</strong></span><span><strong>{{ netPay() | number:'1.0-0' }} KES</strong></span></div>
                </div>
                <label class="form-label" style="margin-top:var(--space-md);">Note (optional)</label>
                <input type="text" class="form-input" [(ngModel)]="modalDescription" placeholder="e.g. May 2026 wages" id="modal-payout-desc">
              </div>
              <div class="modal-footer">
                <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
                <button class="btn btn-primary" (click)="submitEmployeePayout()" [disabled]="submitting() || netPay() <= 0" id="btn-submit-payout">
                  <span class="material-icons-round" style="font-size:18px;">{{ submitting() ? 'hourglass_empty' : 'payments' }}</span>
                  {{ submitting() ? 'Processing...' : 'Pay ' + (netPay() | number:'1.0-0') + ' KES' }}
                </button>
              </div>
            }

            <!-- ── BULK UPLOAD TAB ── -->
            @if (payoutTab() === 'bulk') {
              <div class="modal-body">
                <div class="modal-info-banner" style="background:rgba(99,102,241,0.07);color:#6366f1;border:1px solid rgba(99,102,241,0.2);">
                  <span class="material-icons-round" style="font-size:18px;flex-shrink:0;">upload_file</span>
                  <span>Upload a <strong>.csv</strong> file to pay multiple employees at once. Each row is processed as an individual atomic payout (float debit + wallet credit).</span>
                </div>

                <!-- Template download -->
                <div style="display:flex;align-items:center;gap:var(--space-sm);margin-top:var(--space-md);padding:10px 14px;border-radius:var(--radius-md);border:1px dashed var(--color-border);background:var(--color-bg-secondary);">
                  <span class="material-icons-round" style="color:var(--color-text-muted);font-size:20px;">table_chart</span>
                  <div style="flex:1;">
                    <div style="font-size:0.8rem;font-weight:600;color:var(--color-text-primary);">CSV Format: <code style="font-size:0.75rem;background:rgba(0,0,0,0.08);padding:2px 6px;border-radius:4px;">crew_id, gross_amount, description</code></div>
                    <div style="font-size:0.7rem;color:var(--color-text-muted);margin-top:2px;">Use crew IDs like CRW-00001. Gross = KES (not cents). Description is optional.</div>
                  </div>
                  <button class="btn btn-ghost btn-sm" (click)="downloadBulkTemplate()" id="btn-download-template" style="white-space:nowrap;font-size:0.75rem;">
                    <span class="material-icons-round" style="font-size:14px;">download</span> Template
                  </button>
                </div>

                <!-- File input -->
                <div style="margin-top:var(--space-md);">
                  <label class="form-label">Upload CSV File <span class="field-required">*</span></label>
                  <label style="display:flex;align-items:center;gap:var(--space-sm);padding:12px 16px;border:2px dashed var(--color-border);border-radius:var(--radius-md);cursor:pointer;transition:border-color 0.2s;background:var(--color-bg-secondary);" for="bulk-file-input"
                         [style.borderColor]="bulkFileName ? 'var(--color-accent)' : ''">
                    <span class="material-icons-round" style="color:var(--color-accent);font-size:22px;">cloud_upload</span>
                    <span style="font-size:0.82rem;color:var(--color-text-secondary);">{{ bulkFileName || 'Click to select .csv file' }}</span>
                    <input id="bulk-file-input" type="file" accept=".csv,.txt" style="display:none;" (change)="onBulkFileChange($event)">
                  </label>
                </div>

                <!-- Preview table -->
                @if (bulkRows().length > 0) {
                  <div style="margin-top:var(--space-md);">
                    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:6px;">
                      <span style="font-size:0.8rem;font-weight:600;color:var(--color-text-primary);">Preview ({{ bulkRows().length }} rows, {{ bulkValidRowCount() }} valid)</span>
                    </div>
                    <div style="max-height:220px;overflow-y:auto;border:1px solid var(--color-border);border-radius:var(--radius-md);">
                      <table style="width:100%;border-collapse:collapse;font-size:0.77rem;">
                        <thead style="position:sticky;top:0;background:var(--color-bg-secondary);">
                          <tr>
                            <th style="padding:6px 10px;text-align:left;font-weight:600;color:var(--color-text-secondary);">Employee</th>
                            <th style="padding:6px 10px;text-align:right;font-weight:600;color:var(--color-text-secondary);">Gross (KES)</th>
                            <th style="padding:6px 10px;text-align:left;font-weight:600;color:var(--color-text-secondary);">Note</th>
                            <th style="padding:6px 10px;text-align:center;font-weight:600;color:var(--color-text-secondary);">Status</th>
                          </tr>
                        </thead>
                        <tbody>
                          @for (row of bulkRows(); track $index) {
                            <tr [style.background]="row.error ? 'rgba(239,68,68,0.04)' : 'transparent'">
                              <td style="padding:6px 10px;color:var(--color-text-primary);">{{ row.name }}</td>
                              <td style="padding:6px 10px;text-align:right;font-weight:500;">{{ row.gross | number:'1.0-0' }}</td>
                              <td style="padding:6px 10px;color:var(--color-text-muted);max-width:120px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{ row.description || '—' }}</td>
                              <td style="padding:6px 10px;text-align:center;">
                                @if (row.error) {
                                  <span style="color:var(--color-danger);font-size:0.7rem;display:flex;align-items:center;gap:3px;justify-content:center;">
                                    <span class="material-icons-round" style="font-size:14px;">error</span> {{ row.error }}
                                  </span>
                                } @else {
                                  <span style="color:var(--color-success);font-size:0.7rem;display:flex;align-items:center;gap:3px;justify-content:center;">
                                    <span class="material-icons-round" style="font-size:14px;">check_circle</span> OK
                                  </span>
                                }
                              </td>
                            </tr>
                          }
                        </tbody>
                      </table>
                    </div>
                  </div>
                }

                <!-- Bulk result summary -->
                @if (bulkSubmitResult(); as result) {
                  <div style="margin-top:var(--space-md);padding:12px 16px;border-radius:var(--radius-md);"
                       [style.background]="result.failed === 0 ? 'rgba(34,197,94,0.07)' : 'rgba(251,191,36,0.07)'"
                       [style.border]="result.failed === 0 ? '1px solid rgba(34,197,94,0.25)' : '1px solid rgba(251,191,36,0.3)'">
                    <div style="display:flex;align-items:center;gap:8px;font-size:0.85rem;font-weight:600;" [style.color]="result.failed === 0 ? '#16a34a' : '#92400e'">
                      <span class="material-icons-round" style="font-size:18px;">{{ result.failed === 0 ? 'check_circle' : 'warning' }}</span>
                      {{ result.succeeded }} of {{ result.total }} payouts succeeded{{ result.failed > 0 ? ' · ' + result.failed + ' failed' : '' }}
                    </div>
                  </div>
                }
              </div>
              <div class="modal-footer">
                <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
                <button class="btn btn-primary" (click)="submitBulkPayout()"
                        [disabled]="submitting() || bulkValidRowCount() === 0" id="btn-submit-bulk-payout">
                  <span class="material-icons-round" style="font-size:18px;">{{ submitting() ? 'hourglass_empty' : 'rocket_launch' }}</span>
                  {{ submitting() ? 'Processing...' : 'Pay ' + bulkValidRowCount() + ' Employees' }}
                </button>
              </div>
            }
          </div>
        </div>
      }

      <!-- Withdraw Modal -->
      @if (showModal() === 'withdraw') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:480px;">
            <div class="modal-header">
              <h3 style="display:flex;align-items:center;gap:8px;"><span class="material-icons-round" style="color:#10b981;font-size:22px;">savings</span> Withdraw Funds</h3>
              <button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button>
            </div>
            <div class="modal-body">
              <div class="modal-info-banner modal-info-banner--success">
                <span class="material-icons-round" style="font-size:18px;flex-shrink:0;">info</span>
                <span>Withdraw money from your wallet to M-Pesa or bank account.</span>
              </div>
              <label class="form-label" style="margin-top:var(--space-md);">Withdraw To <span class="field-required">*</span></label>
              <div class="provider-chips" style="margin-top:6px;">
                @for (ch of withdrawChannels; track ch.id) {
                  <button class="provider-chip" [class.provider-chip--active]="crewChannel === ch.id" (click)="crewChannel = ch.id" type="button">
                    <span class="provider-chip-icon">{{ ch.emoji }}</span><span>{{ ch.label }}</span>
                  </button>
                }
              </div>
              @if (crewChannel) {
                <div style="margin-top:var(--space-md);">
                  @if (crewChannel === 'mpesa') {
                    <label class="form-label">M-Pesa Number <span class="field-required">*</span></label>
                    <input type="tel" class="form-input" [(ngModel)]="crewPhone" placeholder="e.g. 0712345678" id="withdraw-phone">
                  }
                  @if (crewChannel === 'bank') {
                    <label class="form-label">Bank Name <span class="field-required">*</span></label>
                    <select class="form-select" [(ngModel)]="crewBankCode">
                      <option value="">— Select Bank —</option>
                      <option value="01">KCB Bank</option>
                      <option value="02">Equity Bank</option>
                      <option value="11">Co-op Bank</option>
                      <option value="31">Stanbic Bank</option>
                      <option value="10">NCBA</option>
                      <option value="12">Absa Bank</option>
                      <option value="63">DTB</option>
                    </select>
                    <label class="form-label" style="margin-top:var(--space-sm);">Account Number <span class="field-required">*</span></label>
                    <input type="text" class="form-input" [(ngModel)]="crewBankAccount" placeholder="e.g. 1234567890" id="withdraw-account">
                  }
                  <label class="form-label" style="margin-top:var(--space-md);">Amount (KES) <span class="field-required">*</span></label>
                  <p class="field-hint">Available: {{ wallet()!.balance_cents | currencyKes }}</p>
                  <input type="number" class="form-input" [(ngModel)]="modalAmount" min="1" step="1" placeholder="e.g. 500" id="withdraw-amount">
                </div>
              }
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
              <button class="btn btn-primary" (click)="submitWithdraw()" [disabled]="submitting() || !crewChannel || modalAmount <= 0" id="btn-submit-withdraw">
                <span class="material-icons-round" style="font-size:18px;">{{ submitting() ? 'hourglass_empty' : 'savings' }}</span>
                {{ submitting() ? 'Processing...' : 'Withdraw' }}
              </button>
            </div>
          </div>
        </div>
      }

      <!-- Transfer Modal -->
      @if (showModal() === 'transfer') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:480px;">
            <div class="modal-header">
              <h3 style="display:flex;align-items:center;gap:8px;"><span class="material-icons-round" style="color:#6366f1;font-size:22px;">swap_horiz</span> Transfer to Wallet</h3>
              <button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button>
            </div>
            <div class="modal-body">
              <div class="modal-info-banner" style="background:rgba(99,102,241,0.08);color:#6366f1;border:1px solid rgba(99,102,241,0.2);">
                <span class="material-icons-round" style="font-size:18px;flex-shrink:0;">info</span>
                <span>Send money to another crew member's wallet instantly.</span>
              </div>
              <div style="position:relative; z-index: 54; margin-top:var(--space-md);">
                <label class="form-label">Recipient <span class="field-required">*</span></label>
                <p class="field-hint">Search by name or ID to find the recipient.</p>
                <app-autocomplete [(ngModel)]="modalCrewId" [options]="crewOptions()" placeholder="— Search Crew Member —"></app-autocomplete>
              </div>
              <label class="form-label" style="margin-top:var(--space-md);">Amount (KES) <span class="field-required">*</span></label>
              <p class="field-hint">Available: {{ wallet()!.balance_cents | currencyKes }}</p>
              <input type="number" class="form-input" [(ngModel)]="modalAmount" min="1" step="1" placeholder="e.g. 200" id="transfer-amount">
              <label class="form-label" style="margin-top:var(--space-md);">Note (optional)</label>
              <input type="text" class="form-input" [(ngModel)]="modalDescription" placeholder="e.g. Lunch money" id="transfer-note">
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
              <button class="btn btn-primary" (click)="submitTransfer()" [disabled]="submitting() || !modalCrewId || modalAmount <= 0" id="btn-submit-transfer" style="background:#6366f1;">
                <span class="material-icons-round" style="font-size:18px;">{{ submitting() ? 'hourglass_empty' : 'send' }}</span>
                {{ submitting() ? 'Processing...' : 'Send Money' }}
              </button>
            </div>
          </div>
        </div>
      }

      <!-- Buy Airtime Modal -->
      @if (showModal() === 'airtime') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:480px;">
            <div class="modal-header">
              <h3 style="display:flex;align-items:center;gap:8px;"><span class="material-icons-round" style="color:#f59e0b;font-size:22px;">phone_android</span> Buy Airtime</h3>
              <button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button>
            </div>
            <div class="modal-body">
              <label class="form-label">Network <span class="field-required">*</span></label>
              <div class="provider-chips" style="margin-top:6px;">
                @for (n of airtimeNetworks; track n.id) {
                  <button class="provider-chip" [class.provider-chip--active]="crewNetwork === n.id" (click)="crewNetwork = n.id" type="button">
                    <span class="provider-chip-icon">{{ n.emoji }}</span><span>{{ n.label }}</span>
                  </button>
                }
              </div>
              @if (crewNetwork) {
                <label class="form-label" style="margin-top:var(--space-md);">Phone Number <span class="field-required">*</span></label>
                <p class="field-hint">Buy for yourself or another number.</p>
                <input type="tel" class="form-input" [(ngModel)]="crewPhone" placeholder="e.g. 0712345678" id="airtime-phone">
                <label class="form-label" style="margin-top:var(--space-md);">Amount (KES) <span class="field-required">*</span></label>
                <div class="airtime-presets">
                  @for (a of [20, 50, 100, 200, 500, 1000]; track a) {
                    <button class="preset-chip" [class.preset-chip--active]="modalAmount === a" (click)="modalAmount = a" type="button">{{ a }}</button>
                  }
                </div>
                <input type="number" class="form-input" [(ngModel)]="modalAmount" min="5" step="1" placeholder="Custom amount" id="airtime-amount" style="margin-top:8px;">
              }
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
              <button class="btn btn-primary" (click)="submitAirtime()" [disabled]="submitting() || !crewNetwork || !crewPhone || modalAmount < 5" id="btn-submit-airtime" style="background:#f59e0b;">
                <span class="material-icons-round" style="font-size:18px;">{{ submitting() ? 'hourglass_empty' : 'phone_android' }}</span>
                {{ submitting() ? 'Processing...' : 'Buy KES ' + (modalAmount || 0) + ' Airtime' }}
              </button>
            </div>
          </div>
        </div>
      }

      <!-- Pay Bills Modal -->
      @if (showModal() === 'bills') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:520px;">
            <div class="modal-header">
              <h3 style="display:flex;align-items:center;gap:8px;"><span class="material-icons-round" style="color:#ef4444;font-size:22px;">receipt_long</span> Pay Bills</h3>
              <button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button>
            </div>
            <div class="modal-body">
              <label class="form-label">Bill Category <span class="field-required">*</span></label>
              <div class="pm-grid" style="grid-template-columns:repeat(3,1fr);margin-top:8px;">
                @for (cat of billCategories; track cat.id) {
                  <button class="pm-card" [class.pm-card--active]="billCategory === cat.id" (click)="selectBillCategory(cat.id)" type="button" style="padding:12px 6px;">
                    <span class="material-icons-round pm-icon" style="font-size:24px;">{{ cat.icon }}</span>
                    <span class="pm-label" style="font-size:0.75rem;">{{ cat.label }}</span>
                  </button>
                }
              </div>
              @if (billCategory) {
                <label class="form-label" style="margin-top:var(--space-md);">Provider <span class="field-required">*</span></label>
                <div class="provider-chips" style="margin-top:6px;">
                  @for (p of getBillProviders(billCategory); track p.id) {
                    <button class="provider-chip" [class.provider-chip--active]="billProvider === p.id" (click)="billProvider = p.id" type="button">
                      <span class="provider-chip-icon">{{ p.emoji }}</span><span>{{ p.label }}</span>
                    </button>
                  }
                </div>
              }
              @if (billProvider) {
                <label class="form-label" style="margin-top:var(--space-md);">Account / Meter Number <span class="field-required">*</span></label>
                <p class="field-hint">Enter your {{ billCategory === 'electricity' ? 'meter' : 'account' }} number as shown on your bill.</p>
                <input type="text" class="form-input" [(ngModel)]="billAccountNo" placeholder="e.g. 12345678" id="bill-account">
                <label class="form-label" style="margin-top:var(--space-md);">Amount (KES) <span class="field-required">*</span></label>
                <input type="number" class="form-input" [(ngModel)]="modalAmount" min="1" step="1" placeholder="e.g. 500" id="bill-amount">
              }
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" (click)="closeModal()">Cancel</button>
              <button class="btn btn-primary" (click)="submitBillPayment()" [disabled]="submitting() || !billProvider || !billAccountNo || modalAmount <= 0" id="btn-submit-bill" style="background:#ef4444;">
                <span class="material-icons-round" style="font-size:18px;">{{ submitting() ? 'hourglass_empty' : 'receipt_long' }}</span>
                {{ submitting() ? 'Processing...' : 'Pay Bill' }}
              </button>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .lookup-row { display: flex; align-items: center; gap: var(--space-md); }
    .qa-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
    .qa-card { display: flex; flex-direction: column; align-items: center; gap: 8px; padding: 20px 12px; border-radius: var(--radius-lg, 14px); border: 1.5px solid var(--color-border); background: var(--color-bg-primary, #fff); cursor: pointer; transition: all 0.22s ease; text-align: center; box-shadow: 0 1px 3px rgba(0,0,0,0.04); }
    .qa-card:hover { transform: translateY(-3px); box-shadow: 0 6px 20px rgba(0,0,0,0.08); border-color: transparent; }
    .qa-card:active { transform: translateY(-1px); }
    .qa-icon { width: 48px; height: 48px; border-radius: 14px; display: flex; align-items: center; justify-content: center; }
    .qa-icon .material-icons-round { font-size: 24px; color: #fff; }
    .qa-icon--withdraw { background: linear-gradient(135deg, #10b981, #059669); }
    .qa-icon--transfer { background: linear-gradient(135deg, #6366f1, #4f46e5); }
    .qa-icon--airtime { background: linear-gradient(135deg, #f59e0b, #d97706); }
    .qa-icon--bills { background: linear-gradient(135deg, #ef4444, #dc2626); }
    .qa-label { font-size: 0.85rem; font-weight: 600; color: var(--color-text-primary); }
    .qa-hint { font-size: 0.68rem; color: var(--color-text-muted); line-height: 1.3; }
    .airtime-presets { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
    .preset-chip { padding: 8px 16px; border-radius: 100px; border: 1.5px solid var(--color-border); background: var(--color-bg-secondary); cursor: pointer; font-size: 0.82rem; font-weight: 600; color: var(--color-text-secondary); transition: all 0.15s ease; }
    .preset-chip:hover { border-color: #f59e0b; color: #f59e0b; }
    .preset-chip--active { border-color: #f59e0b; background: rgba(245,158,11,0.1); color: #f59e0b; box-shadow: 0 0 0 2px rgba(245,158,11,0.15); }
    @media (max-width: 600px) { .qa-grid { grid-template-columns: repeat(2, 1fr); } }
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
    .deduction-row { display: flex; align-items: center; justify-content: space-between; gap: var(--space-sm); margin-bottom: 6px; }
    .deduction-row label { font-size: 0.78rem; color: var(--color-text-secondary); min-width: 110px; }
    .form-input--sm { max-width: 120px; padding: 6px 10px; font-size: 0.8rem; text-align: right; }
    .net-pay-summary { padding: var(--space-md); background: var(--color-bg-tertiary, rgba(0,0,0,0.03)); border-radius: var(--radius-md); border: 1px solid var(--color-border); }
    .net-row { display: flex; justify-content: space-between; font-size: 0.82rem; padding: 4px 0; color: var(--color-text-secondary); }
    .net-row--deduction { color: var(--color-danger); }
    .net-row--total { border-top: 1px solid var(--color-border); margin-top: 6px; padding-top: 8px; font-size: 0.9rem; color: var(--color-text-primary); }
    .pm-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-top: 8px; }
    .pm-card { display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 16px 8px; border-radius: var(--radius-lg, 12px); border: 2px solid var(--color-border); background: var(--color-bg-secondary, #f8f9fb); cursor: pointer; transition: all 0.2s ease; text-align: center; }
    .pm-card:hover { border-color: var(--color-accent); background: rgba(0,210,255,0.04); transform: translateY(-1px); }
    .pm-card--active { border-color: var(--color-accent); background: rgba(0,210,255,0.08); box-shadow: 0 0 0 3px rgba(0,210,255,0.15); }
    .pm-icon { font-size: 28px; color: var(--color-accent); }
    .pm-card--active .pm-icon { color: var(--color-accent); }
    .pm-label { font-size: 0.82rem; font-weight: 600; color: var(--color-text-primary); }
    .pm-hint { font-size: 0.65rem; color: var(--color-text-muted); line-height: 1.3; }
    .provider-chips { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 6px; }
    .provider-chip { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 100px; border: 1.5px solid var(--color-border); background: var(--color-bg-secondary, #f8f9fb); cursor: pointer; font-size: 0.8rem; font-weight: 500; color: var(--color-text-secondary); transition: all 0.18s ease; }
    .provider-chip:hover { border-color: var(--color-accent); color: var(--color-text-primary); }
    .provider-chip--active { border-color: var(--color-accent); background: rgba(0,210,255,0.1); color: var(--color-accent); font-weight: 600; box-shadow: 0 0 0 2px rgba(0,210,255,0.12); }
    .provider-chip-icon { font-size: 1.1rem; }
    @media (max-width: 600px) { .tx-time, .tx-balance { display: none; } .pm-grid { grid-template-columns: 1fr; } }
    .btn-sync { display: inline-flex; align-items: center; gap: 6px; padding: 6px 14px; border-radius: 100px; border: 1.5px solid var(--color-accent); background: rgba(0,210,255,0.06); color: var(--color-accent); font-size: 0.75rem; font-weight: 600; cursor: pointer; transition: all 0.2s ease; white-space: nowrap; }
    .btn-sync:hover:not(:disabled) { background: rgba(0,210,255,0.14); transform: translateY(-1px); box-shadow: 0 3px 12px rgba(0,210,255,0.15); }
    .btn-sync:disabled { opacity: 0.6; cursor: not-allowed; }
    .spin { animation: spin 1s linear infinite; }
    @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
    .poll-result { display: flex; align-items: center; gap: 8px; padding: 8px 12px; border-radius: var(--radius-md); font-size: 0.75rem; margin-bottom: var(--space-sm); transition: all 0.3s ease; }
    .poll-result--success { background: rgba(34,197,94,0.08); color: #16a34a; border: 1px solid rgba(34,197,94,0.25); }
    .poll-result--warning { background: rgba(251,191,36,0.08); color: #92400e; border: 1px solid rgba(251,191,36,0.25); }
    .btn-sync-sm { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 50%; border: 1.5px solid var(--color-border); background: var(--color-bg-secondary); color: var(--color-accent); cursor: pointer; transition: all 0.2s ease; flex-shrink: 0; padding: 0; }
    .btn-sync-sm:hover:not(:disabled) { border-color: var(--color-accent); background: rgba(0,210,255,0.1); transform: rotate(90deg); }
    .btn-sync-sm:disabled { opacity: 0.5; cursor: not-allowed; }
    /* --- Sync Status Styles --- */
    .sync-status-chip { display: inline-flex; align-items: center; gap: 4px; padding: 2px 10px; border-radius: 100px; font-size: 0.65rem; font-weight: 600; white-space: nowrap; letter-spacing: 0.02em; }
    .sync-status-chip--synced { background: rgba(34,197,94,0.1); color: #16a34a; border: 1px solid rgba(34,197,94,0.3); }
    .sync-status-chip--pending { background: rgba(251,191,36,0.1); color: #92400e; border: 1px solid rgba(251,191,36,0.3); }
    .sync-status-chip--failed { background: rgba(239,68,68,0.1); color: #dc2626; border: 1px solid rgba(239,68,68,0.3); }
    .sync-badge { display: inline-flex; align-items: center; gap: 3px; padding: 2px 8px; border-radius: 6px; font-size: 0.62rem; font-weight: 600; white-space: nowrap; text-transform: uppercase; letter-spacing: 0.04em; }
    .sync-badge--callback { background: rgba(99,102,241,0.1); color: #6366f1; border: 1px solid rgba(99,102,241,0.25); }
    .sync-badge--poll { background: rgba(0,210,255,0.1); color: var(--color-accent); border: 1px solid rgba(0,210,255,0.25); }
    .sync-badge--manual { background: rgba(245,158,11,0.1); color: #d97706; border: 1px solid rgba(245,158,11,0.25); }
    .sync-status-row { display: flex; align-items: center; gap: var(--space-sm); padding: 8px 12px; border-radius: var(--radius-md); transition: all 0.2s ease; }
    .sync-status-row--completed { border: 1px solid rgba(34,197,94,0.2); background: rgba(34,197,94,0.03); }
    .sync-status-row--completed > .material-icons-round:first-child { color: #16a34a; }
    .sync-status-row--failed { border: 1px solid rgba(239,68,68,0.2); background: rgba(239,68,68,0.03); }
    .sync-status-row--failed > .material-icons-round:first-child { color: #dc2626; }
    /* --- Payout Modal Tabs --- */
    .payout-tab { display: inline-flex; align-items: center; gap: 6px; padding: 10px 18px; border: none; border-bottom: 2px solid transparent; background: none; cursor: pointer; font-size: 0.82rem; font-weight: 500; color: var(--color-text-muted); transition: all 0.18s ease; margin-bottom: -1px; }
    .payout-tab:hover { color: var(--color-text-primary); }
    .payout-tab--active { color: var(--color-accent); border-bottom-color: var(--color-accent); font-weight: 600; }
  `]
})
export class WalletDashboardComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);

  wallet = signal<Wallet | null>(null);
  orgFloat = signal<SACCOFloat | null>(null);
  organizations = signal<Organization[]>([]);
  selectedOrgId = '';
  showOrgSelector = computed(() =>
    this.isAdmin() && this.organizations().length > 1 && !this.auth.currentUser()?.organization_id
  );
  /** Allowed top-up methods from tenant config (empty = all allowed) */
  allowedTopUpMethods = signal<string[]>([]);
  /** Allowed top-up channels from tenant config (empty = all within allowed methods) */
  allowedTopUpChannels = signal<string[]>([]);
  transactions = signal<WalletTransaction[]>([]);
  txMeta = signal<PaginationMeta | null>(null);
  loadingTxs = signal(true);
  pendingTopUps = signal<SACCOFloatTransaction[]>([]);
  /** Recently synced (completed/failed) top-ups — shows last 10 for audit visibility */
  recentSyncedTopUps = signal<SACCOFloatTransaction[]>([]);
  polling = signal(false);
  pollingTxId = signal<string | null>(null);
  pollResult = signal<{ message: string; confirmed: number; failed: number; skipped: number } | null>(null);
  crewMembers = signal<CrewMember[]>([]);
  crewOptions = computed<AutocompleteOption[]>(() => this.crewMembers().map(c => ({
    value: c.id,
    label: `${c.first_name} ${c.last_name}`,
    sublabel: `ID: ${c.crew_id}`,
    searchText: `${c.first_name} ${c.last_name} ${c.crew_id}`
  })));
  orgOptions = computed<AutocompleteOption[]>(() => this.organizations().map(org => ({
    value: org.id,
    label: org.name,
    sublabel: org.id,
    searchText: `${org.name} ${org.id}`
  })));

  showModal = signal<'credit' | 'topup' | 'payout' | 'withdraw' | 'transfer' | 'airtime' | 'bills' | null>(null);
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

  // Deduction fields (applied before payout)
  deductionNSSF = 0;
  deductionSHA = 0;
  deductionHousing = 0;

  /** Dynamic deduction values keyed by deduction code (e.g. 'LOAN', 'INSURANCE', 'MOTOR_VEHICLE_LOAN') */
  deductionValues: Record<string, number> = {};

  /** Whether this tenant handles statutory deductions (NSSF, SHA, Housing Levy). Default: false = informal mode. */
  statutoryEnabled = signal(false);

  /** Enabled non-statutory deduction codes from tenant config. Default: [] (none active). */
  enabledDeductions = signal<string[]>([]);

  /** Custom deduction labels from tenant config. e.g. { 'MOTOR_VEHICLE_LOAN': 'Motor Vehicle Loan' } */
  customDeductionLabels = signal<Record<string, string>>({});

  /** Standard deduction definitions */
  readonly standardDeductions = [
    { code: 'LOAN', label: 'Loan Repayment' },
    { code: 'INSURANCE', label: 'Insurance' },
    { code: 'OTHER', label: 'Other' },
  ];

  /** Resolved list of all active deductions for this tenant (standard + custom), in order */
  activeDeductions = computed(() => {
    const enabled = this.enabledDeductions();
    const customLabels = this.customDeductionLabels();
    // Standard ones that are enabled
    const std = this.standardDeductions
      .filter(d => enabled.includes(d.code))
      .map(d => ({ code: d.code, label: d.label }));
    // Custom ones
    const custom = Object.entries(customLabels)
      .filter(([code]) => enabled.includes(code))
      .map(([code, label]) => ({ code, label }));
    return [...std, ...custom];
  });

  /** Get label for a deduction code */
  deductionLabel(code: string): string {
    const std = this.standardDeductions.find(d => d.code === code);
    if (std) return std.label;
    return this.customDeductionLabels()[code] || code;
  }


  /** Tab in the Pay Employee modal: 'single' or 'bulk' */
  payoutTab = signal<'single' | 'bulk'>('single');

  bulkFile: File | null = null;
  bulkFileName = '';
  bulkRows = signal<Array<{ crew_member_id: string; name: string; gross: number; net: number; description: string; error?: string }>>([]);
  bulkSubmitResult = signal<{ total: number; succeeded: number; failed: number; results?: any } | null>(null);
  bulkIdempotencyPrefix = '';


  // Top-up payment method fields
  topupMethod: 'mobile_money' | 'bank' | 'card' | '' = '';
  topupProvider = '';
  topupPhone = '';
  topupBankRef = '';

  // Crew member action fields
  crewChannel = '';
  crewPhone = '';
  crewBankCode = '';
  crewBankAccount = '';
  crewNetwork = '';
  billCategory = '';
  billProvider = '';
  billAccountNo = '';

  /** Withdrawal channel options */
  readonly withdrawChannels = [
    { id: 'mpesa', label: 'M-Pesa', emoji: '🟢' },
    { id: 'bank', label: 'Bank Account', emoji: '🏦' },
  ];

  /** Airtime network options */
  readonly airtimeNetworks = [
    { id: 'safaricom', label: 'Safaricom', emoji: '🟢' },
    { id: 'airtel', label: 'Airtel', emoji: '🔴' },
    { id: 'telkom', label: 'Telkom', emoji: '🔵' },
  ];

  /** Bill payment categories */
  readonly billCategories = [
    { id: 'electricity', icon: 'bolt', label: 'Electricity' },
    { id: 'water', icon: 'water_drop', label: 'Water' },
    { id: 'tv', icon: 'tv', label: 'TV & Internet' },
    { id: 'rent', icon: 'home', label: 'Rent' },
    { id: 'insurance', icon: 'shield', label: 'Insurance' },
    { id: 'other', icon: 'more_horiz', label: 'Other' },
  ];

  private readonly billProviders: Record<string, { id: string; label: string; emoji: string }[]> = {
    electricity: [
      { id: 'kplc_prepaid', label: 'KPLC Prepaid', emoji: '⚡' },
      { id: 'kplc_postpaid', label: 'KPLC Postpaid', emoji: '⚡' },
    ],
    water: [
      { id: 'nairobi_water', label: 'Nairobi Water', emoji: '💧' },
      { id: 'eldowas', label: 'ELDOWAS', emoji: '💧' },
      { id: 'other_water', label: 'Other', emoji: '💧' },
    ],
    tv: [
      { id: 'dstv', label: 'DStv', emoji: '📺' },
      { id: 'gotv', label: 'GOtv', emoji: '📺' },
      { id: 'startimes', label: 'StarTimes', emoji: '📺' },
      { id: 'zuku', label: 'Zuku', emoji: '🌐' },
      { id: 'safaricom_home', label: 'Safaricom Home', emoji: '🌐' },
    ],
    rent: [
      { id: 'paybill_rent', label: 'Paybill', emoji: '🏠' },
    ],
    insurance: [
      { id: 'nhif', label: 'SHA/NHIF', emoji: '🛡️' },
      { id: 'jubilee', label: 'Jubilee', emoji: '🛡️' },
      { id: 'britam', label: 'Britam', emoji: '🛡️' },
      { id: 'aar', label: 'AAR', emoji: '🛡️' },
    ],
    other: [
      { id: 'custom_paybill', label: 'Custom Paybill', emoji: '📝' },
    ],
  };

  getBillProviders(category: string): { id: string; label: string; emoji: string }[] {
    return this.billProviders[category] || [];
  }

  selectBillCategory(cat: string): void {
    this.billCategory = cat;
    this.billProvider = '';
    this.billAccountNo = '';
  }

  /** All payment method definitions */
  readonly paymentMethods = [
    { id: 'mobile_money' as const, icon: 'phone_android', label: 'Mobile Money', hint: 'M-Pesa, Airtel Money, T-Kash' },
    { id: 'bank' as const, icon: 'account_balance', label: 'Bank Transfer', hint: 'KCB, Equity, Co-op, RTGS' },
    { id: 'card' as const, icon: 'credit_card', label: 'Card', hint: 'Visa, Mastercard' },
  ];

  /** Payment methods filtered by tenant config, with dynamic hints based on enabled channels */
  availablePaymentMethods = computed(() => {
    const allowedMethods = this.allowedTopUpMethods();
    const allowedChannels = this.allowedTopUpChannels();

    let methods = this.paymentMethods;
    if (allowedMethods && allowedMethods.length > 0) {
      methods = methods.filter(m => allowedMethods.includes(m.id));
    }

    // Generate dynamic hints based on allowed channels
    if (allowedChannels && allowedChannels.length > 0) {
      return methods.map(m => {
        const allProviders = this.providers[m.id] || [];
        const filtered = allProviders.filter(p => allowedChannels.includes(p.id));
        return {
          ...m,
          hint: filtered.length > 0
            ? filtered.map(p => p.label).join(', ')
            : m.hint,
        };
      });
    }
    return methods;
  });

  private readonly providers: Record<string, { id: string; label: string; emoji: string }[]> = {
    mobile_money: [
      { id: 'mpesa', label: 'M-Pesa', emoji: '🟢' },
      { id: 'airtel', label: 'Airtel Money', emoji: '🔴' },
      { id: 'tkash', label: 'T-Kash', emoji: '🔵' },
    ],
    bank: [
      { id: 'kcb', label: 'KCB', emoji: '🏦' },
      { id: 'equity', label: 'Equity', emoji: '🏦' },
      { id: 'coop', label: 'Co-op Bank', emoji: '🏦' },
      { id: 'rtgs', label: 'RTGS', emoji: '⚡' },
    ],
    card: [
      { id: 'visa', label: 'Visa', emoji: '💳' },
      { id: 'mastercard', label: 'Mastercard', emoji: '💳' },
    ],
  };

  getProviders(method: string): { id: string; label: string; emoji: string }[] {
    const all = this.providers[method] || [];
    const channels = this.allowedTopUpChannels();
    // If no channel config, return all providers
    if (!channels || channels.length === 0) return all;
    // Filter to only allowed channels
    return all.filter(p => channels.includes(p.id));
  }

  selectTopUpMethod(method: 'mobile_money' | 'bank' | 'card'): void {
    this.topupMethod = method;
    this.topupProvider = '';
    this.topupPhone = '';
    this.topupBankRef = '';
  }

  getTopUpIcon(): string {
    if (this.topupMethod === 'mobile_money') return 'phone_android';
    if (this.topupMethod === 'bank') return 'account_balance';
    if (this.topupMethod === 'card') return 'credit_card';
    return 'account_balance';
  }

  getProviderLabel(providerId: string): string {
    for (const list of Object.values(this.providers)) {
      const found = list.find(p => p.id === providerId);
      if (found) return found.label;
    }
    return 'Selected Provider';
  }

  isAdmin(): boolean {
    return this.auth.hasRole('SYSTEM_ADMIN', 'EMPLOYER');
  }

  /** Get Material icon name for a sync method */
  getSyncIcon(method?: string): string {
    switch (method) {
      case 'CALLBACK': return 'webhook';
      case 'POLL':     return 'sync';
      case 'MANUAL':   return 'person';
      default:         return 'help_outline';
    }
  }

  /** Get human-readable label for a sync method */
  getSyncLabel(method?: string): string {
    switch (method) {
      case 'CALLBACK': return 'Callback';
      case 'POLL':     return 'Poll';
      case 'MANUAL':   return 'Manual';
      default:         return 'Unknown';
    }
  }

  /** Resolve the active organization ID from user profile or selector */
  getActiveOrgId(): string | undefined {
    return this.auth.currentUser()?.organization_id || this.selectedOrgId || undefined;
  }

  /** Handle org selector change (SYSTEM_ADMIN) */
  onOrgChanged(orgId: string): void {
    this.selectedOrgId = orgId;
    this.loadOrgFloat();
  }

  totalDeductions(): number {
    const statutory = (this.deductionNSSF || 0) + (this.deductionSHA || 0) + (this.deductionHousing || 0);
    const nonStatutory = Object.values(this.deductionValues).reduce((s, v) => s + (v || 0), 0);
    return statutory + nonStatutory;
  }

  netPay(): number {
    return Math.max(0, (this.modalAmount || 0) - this.totalDeductions());
  }

  recalcNetPay(): void { /* triggers change detection via ngModel bindings */ }

  ngOnInit(): void {
    const user = this.auth.currentUser();

    if (this.isAdmin()) {
      // Load crew members for lookup dropdown
      this.api.getCrewMembers({ per_page: '200' }).subscribe({
        next: (res) => this.crewMembers.set(res.data),
      });
      // Load organization float balance
      if (user?.organization_id) {
        this.selectedOrgId = user.organization_id;
        this.loadOrgFloat();
      } else {
        // SYSTEM_ADMIN: fetch all orgs and auto-select the first
        this.api.getOrganizations({ per_page: '50' }).subscribe({
          next: (res) => {
            this.organizations.set(res.data || []);
            if (res.data?.length) {
              this.selectedOrgId = res.data[0].id;
              this.loadOrgFloat();
            }
          },
        });
      }
      this.loadingTxs.set(false);
    } else if (user?.crew_member_id) {
      this.activeCrewId = user.crew_member_id;
      this.loadWallet();
      this.loadTransactions();
      // Load crew members for wallet-to-wallet transfers
      this.api.getCrewMembers({ per_page: '200' }).subscribe({
        next: (res) => this.crewMembers.set(res.data),
      });
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

  loadOrgFloat(): void {
    const orgId = this.getActiveOrgId();
    if (!orgId) return;
    this.api.getSACCOFloat(orgId).subscribe({
      next: (res) => this.orgFloat.set(res.data),
    });
    // Load org config to get allowed top-up methods
    this.api.getOrganization(orgId).subscribe({
      next: (res) => {
        const cfg = res.data?.tenant_config;
        this.allowedTopUpMethods.set(cfg?.allowed_topup_methods || []);
        this.allowedTopUpChannels.set(cfg?.allowed_topup_channels || []);
        this.statutoryEnabled.set(cfg?.handle_statutory_deductions === true);
        this.enabledDeductions.set(cfg?.enabled_deductions || []);
        this.customDeductionLabels.set(cfg?.custom_deduction_labels || {});
      },
    });
    // Also load pending float transactions for approval + recently synced
    this.api.getFloatTransactions(orgId, { per_page: '50' }).subscribe({
      next: (res) => {
        const allTxs = res.data || [];
        const pending = allTxs.filter((tx: SACCOFloatTransaction) => tx.status === 'PENDING');
        this.pendingTopUps.set(pending);
        // Show recently synced (completed/failed with a sync_method) — last 10
        const synced = allTxs
          .filter((tx: SACCOFloatTransaction) =>
            (tx.status === 'COMPLETED' || tx.status === 'FAILED') && tx.sync_method
          )
          .slice(0, 10);
        this.recentSyncedTopUps.set(synced);
      },
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
  openModal(type: 'credit' | 'topup' | 'payout'): void {
    this.resetModalFields();
    this.showModal.set(type);
  }

  openCrewModal(type: 'withdraw' | 'transfer' | 'airtime' | 'bills'): void {
    this.resetModalFields();
    // Pre-fill phone from user's profile if available
    const user = this.auth.currentUser();
    if (user?.phone) this.crewPhone = user.phone;
    this.showModal.set(type);
  }

  private resetModalFields(): void {
    this.modalAmount = 0;
    this.modalDescription = '';
    this.modalCrewId = this.activeCrewId || '';
    this.modalCategory = 'EARNING';
    this.deductionNSSF = 0;
    this.deductionSHA = 0;
    this.deductionHousing = 0;
    this.deductionValues = {};
    this.topupMethod = '';
    this.topupProvider = '';
    this.topupPhone = '';
    this.topupBankRef = '';
    this.crewChannel = '';
    this.crewPhone = '';
    this.crewBankCode = '';
    this.crewBankAccount = '';
    this.crewNetwork = '';
    this.billCategory = '';
    this.billProvider = '';
    this.billAccountNo = '';
    // Bulk payout reset
    this.payoutTab.set('single');
    this.bulkFile = null;
    this.bulkFileName = '';
    this.bulkRows.set([]);
    this.bulkSubmitResult.set(null);
    this.bulkIdempotencyPrefix = 'bulk-' + Date.now();
  }

  closeModal(): void {
    this.showModal.set(null);
  }

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

  submitTopUp(): void {
    if (this.modalAmount <= 0) {
      this.toast.error('Enter a valid amount');
      return;
    }
    if (!this.topupProvider) {
      this.toast.error('Select a payment method and provider');
      return;
    }
    if (this.topupMethod === 'mobile_money' && !this.topupPhone) {
      this.toast.error('Enter a phone number');
      return;
    }
    if (this.topupMethod === 'bank' && !this.topupBankRef) {
      this.toast.error('Enter a bank/RTGS reference');
      return;
    }
    const orgId = this.getActiveOrgId();
    if (!orgId) {
      this.toast.error('No organization selected. Please select an organization first.');
      return;
    }

    // Build reference with payment details
    const providerLabel = this.getProviderLabel(this.topupProvider);
    const channelInfo = this.topupMethod === 'mobile_money'
      ? `${providerLabel} (${this.topupPhone})`
      : this.topupMethod === 'bank'
        ? `${providerLabel} Ref: ${this.topupBankRef}`
        : `${providerLabel} Card`;
    const reference = `${channelInfo}${this.modalDescription ? ' | ' + this.modalDescription : ''}`;

    this.submitting.set(true);
    this.api.topupSACCOFloat(orgId, {
      amount_cents: Math.round(this.modalAmount * 100),
      idempotency_key: this.generateIdempotencyKey(),
      method: this.topupMethod,
      provider: this.topupProvider,
      phone_number: this.topupPhone || undefined,
      bank_ref: this.topupBankRef || undefined,
      reference,
    }).subscribe({
      next: (res: any) => {
        const status = res?.data?.status;
        if (status === 'PENDING') {
          const msg = res?.data?.message || 'Top-up recorded as pending. It must be confirmed by an admin.';
          this.toast.success(msg);
          // Reload pending list immediately so the new tx appears
          this.loadOrgFloat();
          // Auto-poll after 15s for mobile money STK push — gives M-Pesa time to complete
          if (this.topupMethod === 'mobile_money') {
            this.toast.info('We\u2019ll auto-check the payment status in 15 seconds...');
            setTimeout(() => this.pollPendingSTK(), 15000);
          }
        } else {
          this.toast.success(`Float topped up via ${providerLabel}`);
          this.loadOrgFloat();
        }
        this.closeModal();
        this.submitting.set(false);
      },
      error: (err: any) => {
        const msg = err?.error?.message || 'Top-up failed. Please try again.';
        this.toast.error(msg);
        this.submitting.set(false);
      },
    });
  }

  /** Confirm a PENDING float top-up (admin approval) */
  confirmPendingTopUp(txId: string): void {
    const orgId = this.getActiveOrgId();
    if (!orgId) return;
    this.api.confirmTopUp(orgId, txId).subscribe({
      next: () => {
        this.toast.success('Top-up confirmed. Float balance credited.');
        this.loadOrgFloat();
      },
      error: (err: any) => {
        const msg = err?.error?.message || 'Failed to confirm top-up.';
        this.toast.error(msg);
      },
    });
  }

  /** Reject a PENDING float top-up (admin denial) */
  rejectPendingTopUp(txId: string): void {
    const reason = prompt('Reason for rejecting this top-up:');
    if (reason === null) return; // User cancelled
    const orgId = this.getActiveOrgId();
    if (!orgId) return;
    this.api.rejectTopUp(orgId, txId, reason).subscribe({
      next: () => {
        this.toast.success('Top-up rejected.');
        this.loadOrgFloat();
      },
      error: (err: any) => {
        const msg = err?.error?.message || 'Failed to reject top-up.';
        this.toast.error(msg);
      },
    });
  }

  /** Poll JamboPay for pending STK transaction statuses */
  pollPendingSTK(): void {
    const orgId = this.getActiveOrgId();
    if (!orgId) return;
    this.polling.set(true);
    this.pollResult.set(null);
    this.api.pollSTK(orgId).subscribe({
      next: (res: any) => {
        const data = res.data || res;
        this.pollResult.set({
          message: data.message || 'Poll completed',
          confirmed: data.confirmed || 0,
          failed: data.failed || 0,
          skipped: data.skipped || 0,
        });
        if (data.confirmed > 0) {
          this.toast.success(`${data.confirmed} payment(s) confirmed via gateway sync!`);
          this.loadOrgFloat();
        } else if (data.checked === 0) {
          this.toast.info('No pending payments to check.');
        } else {
          this.toast.info('Payments still processing. You can confirm manually or try again later.');
        }
        this.polling.set(false);
        // Clear result after 10 seconds
        setTimeout(() => this.pollResult.set(null), 10000);
      },
      error: (err: any) => {
        const msg = err?.error?.message || 'Failed to check payment status.';
        this.toast.error(msg);
        this.polling.set(false);
      },
    });
  }

  /** Poll JamboPay for a single pending STK transaction */
  pollSingleTx(txId: string): void {
    const orgId = this.getActiveOrgId();
    if (!orgId) return;
    this.pollingTxId.set(txId);
    this.api.pollSingleSTK(orgId, txId).subscribe({
      next: (res: any) => {
        const data = res.data || res;
        if (data.action === 'confirmed') {
          this.toast.success('Payment confirmed! Float balance updated.');
          this.loadOrgFloat();
        } else if (data.action === 'failed') {
          this.toast.error('Payment failed: ' + (data.jp_status || 'Unknown'));
          this.loadOrgFloat();
        } else if (data.action === 'still_pending') {
          this.toast.info('Payment still processing. Try again shortly or confirm manually.');
        } else if (data.error?.includes('GATEWAY_UNREACHABLE')) {
          this.toast.info('Gateway auto-sync unavailable. Use Confirm ✓ if you received the M-Pesa SMS.');
        } else if (data.error) {
          this.toast.info('Could not verify payment. Use Confirm ✓ if you received the M-Pesa SMS.');
        }
        this.pollingTxId.set(null);
      },
      error: (err: any) => {
        const msg = err?.error?.message || 'Failed to check payment status.';
        this.toast.error(msg);
        this.pollingTxId.set(null);
      },
    });
  }

  /** Pay employee: atomically debit org float (gross) + credit wallet (net) */
  submitEmployeePayout(): void {
    if (!this.modalCrewId || this.modalAmount <= 0) {
      this.toast.error('Select an employee and enter a gross amount');
      return;
    }
    const net = this.netPay();
    if (net <= 0) {
      this.toast.error('Net pay must be greater than zero');
      return;
    }

    // Build description with deduction breakdown
    const parts: string[] = [];
    if (this.deductionNSSF > 0) parts.push(`NSSF: ${this.deductionNSSF}`);
    if (this.deductionSHA > 0) parts.push(`SHA: ${this.deductionSHA}`);
    if (this.deductionHousing > 0) parts.push(`Housing: ${this.deductionHousing}`);
    for (const [code, val] of Object.entries(this.deductionValues)) {
      if ((val || 0) > 0) parts.push(`${this.deductionLabel(code)}: ${val}`);
    }
    const deductionSummary = parts.length > 0 ? ` | Deductions: ${parts.join(', ')}` : '';
    const desc = `Gross: ${this.modalAmount} KES, Net: ${net} KES${deductionSummary}${this.modalDescription ? ' | ' + this.modalDescription : ''}`;

    this.submitting.set(true);

    this.api.employeePayout({
      crew_member_id: this.modalCrewId,
      gross_cents: Math.round(this.modalAmount * 100),
      net_cents: Math.round(net * 100),
      idempotency_key: this.generateIdempotencyKey(),
      description: desc,
    }).subscribe({
      next: () => {
        this.toast.success(`KES ${net.toLocaleString()} paid to employee wallet (KES ${this.modalAmount.toLocaleString()} debited from float)`);
        this.closeModal();
        this.submitting.set(false);
        this.loadWallet();
        this.loadTransactions();
        this.loadOrgFloat();
      },
      error: (err: any) => {
        const msg = err?.error?.message || 'Employee payout failed. No funds were moved.';
        this.toast.error(msg);
        this.submitting.set(false);
      },
    });
  }

  /** Handle CSV file selection for bulk payout */
  onBulkFileChange(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input?.files?.[0];
    if (!file) return;
    this.bulkFile = file;
    this.bulkFileName = file.name;
    this.bulkSubmitResult.set(null);

    const reader = new FileReader();
    reader.onload = (e) => {
      const text = e.target?.result as string;
      const parsed = this.parseCSV(text);
      this.bulkRows.set(parsed);
    };
    reader.readAsText(file);
  }

  /** Parse CSV text into bulk payout rows. Expected columns: crew_id,gross_amount,description */
  private parseCSV(text: string): Array<{ crew_member_id: string; name: string; gross: number; net: number; description: string; error?: string }> {
    const lines = text.split('\n').map(l => l.trim()).filter(l => l.length > 0);
    if (lines.length === 0) return [];

    // Skip header row if it starts with 'crew_id' or 'crew'
    const startIdx = lines[0].toLowerCase().startsWith('crew') ? 1 : 0;
    const rows: Array<{ crew_member_id: string; name: string; gross: number; net: number; description: string; error?: string }> = [];

    for (let i = startIdx; i < lines.length; i++) {
      const cols = lines[i].split(',').map(c => c.trim().replace(/^"|"$/g, ''));
      const crewIdOrCode = cols[0] || '';
      const grossRaw = parseFloat(cols[1] || '0');
      const desc = cols[2] || '';

      if (!crewIdOrCode) continue;

      // Look up UUID by crew_id string or match full UUID
      const match = this.crewMembers().find(c =>
        c.crew_id === crewIdOrCode || c.id === crewIdOrCode
      );

      const gross = isNaN(grossRaw) || grossRaw <= 0 ? 0 : grossRaw;
      const row: { crew_member_id: string; name: string; gross: number; net: number; description: string; error?: string } = {
        crew_member_id: match?.id || '',
        name: match ? `${match.first_name} ${match.last_name}` : crewIdOrCode,
        gross,
        net: gross,  // no deductions applied in bulk mode; admins set same gross=net
        description: desc,
        error: !match ? `Unknown crew ID: ${crewIdOrCode}` : gross <= 0 ? 'Invalid amount' : undefined,
      };
      rows.push(row);
    }
    return rows;
  }

  /** Download a CSV template for bulk payouts */
  downloadBulkTemplate(): void {
    const header = 'crew_id,gross_amount,description';
    const sample = this.crewMembers().slice(0, 2)
      .map(c => `${c.crew_id},15000,May 2026 wages`)
      .join('\n') || 'CRW-00001,15000,May 2026 wages';
    const csv = `${header}\n${sample}`;
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'bulk_payout_template.csv';
    a.click();
    URL.revokeObjectURL(url);
  }

  /** Count valid bulk rows (no errors and non-zero amount) */
  bulkValidRowCount(): number {
    return this.bulkRows().filter(r => !r.error && r.gross > 0).length;
  }

  /** Submit the bulk payout */
  submitBulkPayout(): void {
    const validRows = this.bulkRows().filter(r => !r.error && r.gross > 0);
    if (validRows.length === 0) {
      this.toast.error('No valid rows to process. Fix errors in the preview table.');
      return;
    }
    this.submitting.set(true);
    this.bulkSubmitResult.set(null);

    const payouts = validRows.map(r => ({
      crew_member_id: r.crew_member_id,
      gross_cents: Math.round(r.gross * 100),
      net_cents: Math.round(r.net * 100),
      description: r.description || 'Bulk payout',
    }));

    this.api.bulkEmployeePayout({
      payouts,
      idempotency_prefix: this.bulkIdempotencyPrefix,
    }).subscribe({
      next: (res: any) => {
        const data = res?.data || res;
        this.bulkSubmitResult.set(data);
        if (data.failed === 0) {
          this.toast.success(`All ${data.succeeded} payouts processed successfully!`);
        } else {
          this.toast.warning(`${data.succeeded} succeeded, ${data.failed} failed. See results below.`);
        }
        this.submitting.set(false);
        this.loadOrgFloat();
      },
      error: (err: any) => {
        const msg = err?.error?.message || 'Bulk payout failed.';
        this.toast.error(msg);
        this.submitting.set(false);
      },
    });
  }

  // --- Crew Member Actions ---
  submitWithdraw(): void {
    if (this.modalAmount <= 0 || !this.crewChannel) {
      this.toast.error('Select a withdrawal channel and enter an amount');
      return;
    }
    if (this.crewChannel === 'mpesa' && !this.crewPhone) {
      this.toast.error('Enter M-Pesa phone number');
      return;
    }
    if (this.crewChannel === 'bank' && (!this.crewBankCode || !this.crewBankAccount)) {
      this.toast.error('Select a bank and enter account number');
      return;
    }
    const channel = this.crewChannel === 'mpesa' ? 'MOMO_B2C' : 'BANK';
    const desc = this.crewChannel === 'mpesa'
      ? `Withdrawal to M-Pesa ${this.crewPhone}`
      : `Withdrawal to Bank (code: ${this.crewBankCode}) Acc: ${this.crewBankAccount}`;
    this.submitting.set(true);
    this.api.debitWallet({
      crew_member_id: this.activeCrewId,
      amount_cents: Math.round(this.modalAmount * 100),
      category: 'WITHDRAWAL',
      description: desc,
    }, this.generateIdempotencyKey()).subscribe({
      next: () => {
        this.toast.success(`KES ${this.modalAmount.toLocaleString()} withdrawal initiated via ${channel}`);
        this.closeModal();
        this.submitting.set(false);
        this.loadWallet();
        this.loadTransactions();
      },
      error: () => this.submitting.set(false),
    });
  }

  submitTransfer(): void {
    if (!this.modalCrewId || this.modalAmount <= 0) {
      this.toast.error('Select a recipient and enter an amount');
      return;
    }
    if (this.modalCrewId === this.activeCrewId) {
      this.toast.error('Cannot transfer to yourself');
      return;
    }
    this.submitting.set(true);
    const note = this.modalDescription || 'Wallet transfer';

    // Single atomic call — backend debits sender + credits recipient in one DB transaction
    this.api.walletTransfer({
      to_crew_member_id: this.modalCrewId,
      amount_cents: Math.round(this.modalAmount * 100),
      idempotency_key: this.generateIdempotencyKey(),
      description: note,
    }).subscribe({
      next: () => {
        this.toast.success(`KES ${this.modalAmount.toLocaleString()} sent successfully`);
        this.closeModal();
        this.submitting.set(false);
        this.loadWallet();
        this.loadTransactions();
      },
      error: (err: any) => {
        const msg = err?.error?.message || 'Transfer failed. No funds were moved.';
        this.toast.error(msg);
        this.submitting.set(false);
      },
    });
  }

  submitAirtime(): void {
    if (!this.crewNetwork || !this.crewPhone || this.modalAmount < 5) {
      this.toast.error('Select network, enter phone and amount');
      return;
    }
    const networkLabel = this.airtimeNetworks.find(n => n.id === this.crewNetwork)?.label || this.crewNetwork;
    this.submitting.set(true);
    this.api.debitWallet({
      crew_member_id: this.activeCrewId,
      amount_cents: Math.round(this.modalAmount * 100),
      category: 'WITHDRAWAL',
      description: `Airtime: KES ${this.modalAmount} ${networkLabel} to ${this.crewPhone}`,
    }, this.generateIdempotencyKey()).subscribe({
      next: () => {
        this.toast.success(`KES ${this.modalAmount} airtime sent to ${this.crewPhone}`);
        this.closeModal();
        this.submitting.set(false);
        this.loadWallet();
        this.loadTransactions();
      },
      error: () => this.submitting.set(false),
    });
  }

  submitBillPayment(): void {
    if (!this.billProvider || !this.billAccountNo || this.modalAmount <= 0) {
      this.toast.error('Fill in all bill payment fields');
      return;
    }
    const providerLabel = this.getBillProviders(this.billCategory).find(p => p.id === this.billProvider)?.label || this.billProvider;
    this.submitting.set(true);
    this.api.debitWallet({
      crew_member_id: this.activeCrewId,
      amount_cents: Math.round(this.modalAmount * 100),
      category: 'WITHDRAWAL',
      description: `Bill Payment: ${providerLabel} Acc: ${this.billAccountNo} KES ${this.modalAmount}`,
    }, this.generateIdempotencyKey()).subscribe({
      next: () => {
        this.toast.success(`Bill payment of KES ${this.modalAmount.toLocaleString()} sent to ${providerLabel}`);
        this.closeModal();
        this.submitting.set(false);
        this.loadWallet();
        this.loadTransactions();
      },
      error: () => this.submitting.set(false),
    });
  }
}
