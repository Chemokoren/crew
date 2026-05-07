import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';
import { Wallet, WalletTransaction, PaginationMeta, CrewMember, SACCOFloat, Organization } from '../../../core/models';

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
        <div class="glass-card" style="margin-bottom:var(--space-md);padding:var(--space-sm) var(--space-md); display:flex; align-items:center; gap:var(--space-sm);">
          <span class="material-icons-round" style="color:var(--color-accent);">business</span>
          <label style="font-weight:500; font-size:0.85rem; white-space:nowrap;">Active Organization:</label>
          <select [(ngModel)]="selectedOrgId" (ngModelChange)="onOrgChanged($event)" class="form-control" style="flex:1; max-width:400px;" id="select-org">
            @for (org of organizations(); track org.id) {
              <option [value]="org.id">{{ org.name }}</option>
            }
          </select>
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
                @for (method of paymentMethods; track method.id) {
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

      <!-- Pay Employee Modal (with deductions before payout) -->
      @if (showModal() === 'payout') {
        <div class="modal-backdrop" (click)="closeModal()">
          <div class="modal-content" (click)="$event.stopPropagation()" style="max-width:560px;">
            <div class="modal-header">
              <h3 style="display:flex;align-items:center;gap:8px;">
                <span class="material-icons-round" style="color:var(--color-success);font-size:22px;">payments</span>
                Pay Employee
              </h3>
              <button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button>
            </div>
            <div class="modal-body">
              <div class="modal-info-banner modal-info-banner--success">
                <span class="material-icons-round" style="font-size:18px;flex-shrink:0;">info</span>
                <span>Pay an employee from your organization's float. <strong>Deductions are applied before payout</strong> — the employee receives only the net amount in their wallet.</span>
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
                <div class="deduction-row">
                  <label>NSSF</label>
                  <input type="number" class="form-input form-input--sm" [(ngModel)]="deductionNSSF" min="0" step="1" placeholder="0" (ngModelChange)="recalcNetPay()">
                </div>
                <div class="deduction-row">
                  <label>SHA (NHIF)</label>
                  <input type="number" class="form-input form-input--sm" [(ngModel)]="deductionSHA" min="0" step="1" placeholder="0" (ngModelChange)="recalcNetPay()">
                </div>
                <div class="deduction-row">
                  <label>Housing Levy</label>
                  <input type="number" class="form-input form-input--sm" [(ngModel)]="deductionHousing" min="0" step="1" placeholder="0" (ngModelChange)="recalcNetPay()">
                </div>
                <div class="deduction-row">
                  <label>Loan Repayment</label>
                  <input type="number" class="form-input form-input--sm" [(ngModel)]="deductionLoan" min="0" step="1" placeholder="0" (ngModelChange)="recalcNetPay()">
                </div>
                <div class="deduction-row">
                  <label>Insurance</label>
                  <input type="number" class="form-input form-input--sm" [(ngModel)]="deductionInsurance" min="0" step="1" placeholder="0" (ngModelChange)="recalcNetPay()">
                </div>
                <div class="deduction-row">
                  <label>Other</label>
                  <input type="number" class="form-input form-input--sm" [(ngModel)]="deductionOther" min="0" step="1" placeholder="0" (ngModelChange)="recalcNetPay()">
                </div>
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
  deductionLoan = 0;
  deductionInsurance = 0;
  deductionOther = 0;

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

  /** Payment method definitions */
  readonly paymentMethods = [
    { id: 'mobile_money' as const, icon: 'phone_android', label: 'Mobile Money', hint: 'M-Pesa, Airtel Money' },
    { id: 'bank' as const, icon: 'account_balance', label: 'Bank Transfer', hint: 'KCB, Equity, RTGS' },
    { id: 'card' as const, icon: 'credit_card', label: 'Card', hint: 'Visa, Mastercard' },
  ];

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
    return this.providers[method] || [];
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
    return this.auth.hasRole('SYSTEM_ADMIN', 'SACCO_ADMIN');
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
    return (this.deductionNSSF || 0) + (this.deductionSHA || 0) + (this.deductionHousing || 0) +
      (this.deductionLoan || 0) + (this.deductionInsurance || 0) + (this.deductionOther || 0);
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
    this.deductionLoan = 0;
    this.deductionInsurance = 0;
    this.deductionOther = 0;
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
        if (this.topupMethod === 'mobile_money') {
          // Mobile money: balance NOT yet credited — waiting for callback
          const msg = res?.data?.message || `STK push sent to ${this.topupPhone}. Check your phone to complete payment.`;
          this.toast.success(msg);
        } else {
          // Bank/card: balance credited immediately
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
    if (this.deductionLoan > 0) parts.push(`Loan: ${this.deductionLoan}`);
    if (this.deductionInsurance > 0) parts.push(`Insurance: ${this.deductionInsurance}`);
    if (this.deductionOther > 0) parts.push(`Other: ${this.deductionOther}`);
    const deductionSummary = parts.length > 0 ? ` | Deductions: ${parts.join(', ')}` : '';
    const desc = `Gross: ${this.modalAmount} KES, Net: ${net} KES${deductionSummary}${this.modalDescription ? ' | ' + this.modalDescription : ''}`;

    this.submitting.set(true);

    // Single atomic call — backend handles float debit + wallet credit in one DB transaction
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
