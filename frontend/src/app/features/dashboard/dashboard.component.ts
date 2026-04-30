import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { catchError } from 'rxjs/operators';
import { of } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiService } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { CurrencyKesPipe } from '../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../shared/pipes/relative-time.pipe';
import { SystemStats, Wallet, WalletTransaction, DailySummary } from '../../core/models';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, RouterLink, CurrencyKesPipe, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Dashboard</h1>
          <p class="page-subtitle">{{ greeting() }} — here's your operations overview</p>
        </div>
        <div class="page-actions">
          <span class="live-badge"><span class="live-dot"></span> Live</span>
        </div>
      </div>

      <!-- Admin Stats -->
      @if (auth.isAdmin() && stats()) {
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-icon" style="background: rgba(0, 210, 255, 0.12); color: var(--color-accent);">
              <span class="material-icons-round">groups</span>
            </div>
            <div class="stat-value">{{ stats()!.total_crew | number }}</div>
            <div class="stat-label">Active Crew Members</div>
            <div class="stat-change badge-success" style="background:var(--color-success-light);color:var(--color-success);">
              <span class="material-icons-round" style="font-size:14px;">trending_up</span> Active
            </div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" style="background: var(--color-success-light); color: var(--color-success);">
              <span class="material-icons-round">account_balance_wallet</span>
            </div>
            <div class="stat-value">{{ stats()!.total_wallet_balance_cents | currencyKes }}</div>
            <div class="stat-label">Total Wallet Balance</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" style="background: var(--color-info-light); color: var(--color-info);">
              <span class="material-icons-round">business</span>
            </div>
            <div class="stat-value">{{ stats()!.total_saccos | number }}</div>
            <div class="stat-label">Registered SACCOs</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" style="background: var(--color-warning-light); color: var(--color-warning);">
              <span class="material-icons-round">directions_bus</span>
            </div>
            <div class="stat-value">{{ stats()!.total_vehicles | number }}</div>
            <div class="stat-label">Fleet Vehicles</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" style="background: rgba(123, 97, 255, 0.12); color: #7b61ff;">
              <span class="material-icons-round">assignment</span>
            </div>
            <div class="stat-value">{{ stats()!.total_assignments | number }}</div>
            <div class="stat-label">Total Assignments</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" style="background: rgba(236, 72, 153, 0.12); color: #ec4899;">
              <span class="material-icons-round">people</span>
            </div>
            <div class="stat-value">{{ stats()!.total_users | number }}</div>
            <div class="stat-label">Platform Users</div>
          </div>
        </div>
      }

      <!-- Crew Member Wallet Overview -->
      @if (!auth.isAdmin() && crewWallet()) {
        <div class="wallet-hero glass-card">
          <div class="wallet-hero-left">
            <span class="wallet-hero-label">Your Wallet Balance</span>
            <span class="wallet-hero-value">{{ crewWallet()!.balance_cents | currencyKes }}</span>
            <span class="wallet-hero-currency">{{ crewWallet()!.currency }}</span>
          </div>
          <div class="wallet-hero-actions">
            <a routerLink="/wallets" class="btn btn-primary" id="go-to-wallet">
              <span class="material-icons-round">account_balance_wallet</span> View Wallet
            </a>
          </div>
        </div>

        <div class="stats-grid" style="margin-top: var(--space-md);">
          <div class="stat-card">
            <div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);">
              <span class="material-icons-round">arrow_downward</span>
            </div>
            <div class="stat-value">{{ crewWallet()!.total_credited_cents | currencyKes }}</div>
            <div class="stat-label">Total Earned</div>
          </div>
          <div class="stat-card">
            <div class="stat-icon" style="background:var(--color-danger-light);color:var(--color-danger);">
              <span class="material-icons-round">arrow_upward</span>
            </div>
            <div class="stat-value">{{ crewWallet()!.total_debited_cents | currencyKes }}</div>
            <div class="stat-label">Total Withdrawn</div>
          </div>
          <!-- #55: Active Assignments Count -->
          <div class="stat-card">
            <div class="stat-icon" style="background:rgba(123,97,255,0.12);color:#7b61ff;">
              <span class="material-icons-round">assignment</span>
            </div>
            <div class="stat-value">{{ activeAssignments() }}</div>
            <div class="stat-label">Active Assignments</div>
          </div>
        </div>
      }

      <!-- #54: Today's Earnings Summary -->
      @if (!auth.isAdmin() && todayEarnings()) {
        <div class="section-block">
          <h2 class="section-title">
            <span class="material-icons-round" style="font-size:20px;color:var(--color-success);">payments</span>
            Today's Earnings
          </h2>
          <div class="today-card glass-card">
            <div class="today-main">
              <div class="today-amount">{{ todayEarnings()!.total_earned_cents | currencyKes }}</div>
              <div class="today-meta">
                <span class="today-tag"><span class="material-icons-round">event</span> {{ todayEarnings()!.assignment_count }} assignment{{ todayEarnings()!.assignment_count !== 1 ? 's' : '' }}</span>
                <span class="today-tag"><span class="material-icons-round">account_balance_wallet</span> Net: {{ todayEarnings()!.net_amount_cents | currencyKes }}</span>
              </div>
            </div>
          </div>
        </div>
      }

      <!-- #57: Earnings Sparkline — Last 7 Days -->
      @if (!auth.isAdmin() && earningsHistory().length > 0) {
        <div class="section-block">
          <h2 class="section-title">
            <span class="material-icons-round" style="font-size:20px;color:#ec4899;">show_chart</span>
            Earnings — Last 7 Days
          </h2>
          <div class="sparkline-card glass-card">
            <div class="sparkline-bars">
              @for (day of earningsHistory(); track day.date) {
                <div class="sparkline-col">
                  <div class="sparkline-bar-wrapper">
                    <div class="sparkline-bar" [style.height.%]="barHeight(day.total_earned_cents)"
                         [title]="(day.total_earned_cents | currencyKes)">
                    </div>
                  </div>
                  <span class="sparkline-label">{{ dayLabel(day.date) }}</span>
                  <span class="sparkline-amount">{{ day.total_earned_cents | currencyKes }}</span>
                </div>
              }
            </div>
            <div class="sparkline-summary">
              <span>7-day total: <strong>{{ weekTotal() | currencyKes }}</strong></span>
              <span>Avg: <strong>{{ weekAvg() | currencyKes }}</strong>/day</span>
            </div>
          </div>
        </div>
      }

      <!-- #56: Recent Transactions Widget -->
      @if (!auth.isAdmin() && recentTxns().length > 0) {
        <div class="section-block">
          <h2 class="section-title">
            <span class="material-icons-round" style="font-size:20px;color:var(--color-info);">receipt_long</span>
            Recent Transactions
          </h2>
          <div class="txn-list glass-card">
            @for (txn of recentTxns(); track txn.id) {
              <div class="txn-row">
                <div class="txn-icon" [class.credit]="txn.transaction_type === 'CREDIT'" [class.debit]="txn.transaction_type === 'DEBIT'">
                  <span class="material-icons-round">{{ txn.transaction_type === 'CREDIT' ? 'arrow_downward' : 'arrow_upward' }}</span>
                </div>
                <div class="txn-details">
                  <span class="txn-desc">{{ txn.description || txn.category }}</span>
                  <span class="txn-time">{{ txn.created_at | relativeTime }}</span>
                </div>
                <div class="txn-amount" [class.credit]="txn.transaction_type === 'CREDIT'" [class.debit]="txn.transaction_type === 'DEBIT'">
                  {{ txn.transaction_type === 'CREDIT' ? '+' : '-' }}{{ txn.amount_cents | currencyKes }}
                </div>
              </div>
            }
            <a routerLink="/wallets" class="txn-view-all">View all transactions →</a>
          </div>
        </div>
      }

      <!-- Quick Actions -->
      <div class="section-block">
        <h2 class="section-title">
          <span class="material-icons-round" style="font-size:20px;color:var(--color-accent);">bolt</span>
          Quick Actions
        </h2>
        <div class="actions-grid">
          @if (auth.hasRole('SYSTEM_ADMIN', 'SACCO_ADMIN')) {
            <a routerLink="/crew" class="action-card glass-card" id="quick-crew">
              <div class="action-icon-wrapper" style="background:rgba(0,210,255,0.12);color:var(--color-accent);">
                <span class="material-icons-round">person_add</span>
              </div>
              <span class="action-label">Manage Crew</span>
              <span class="action-desc">Add & manage members</span>
            </a>
            <a routerLink="/assignments" class="action-card glass-card" id="quick-assignments">
              <div class="action-icon-wrapper" style="background:rgba(123,97,255,0.12);color:#7b61ff;">
                <span class="material-icons-round">assignment_add</span>
              </div>
              <span class="action-label">Assignments</span>
              <span class="action-desc">Create shift assignments</span>
            </a>
            <a routerLink="/payroll" class="action-card glass-card" id="quick-payroll">
              <div class="action-icon-wrapper" style="background:var(--color-warning-light);color:var(--color-warning);">
                <span class="material-icons-round">receipt_long</span>
              </div>
              <span class="action-label">Run Payroll</span>
              <span class="action-desc">Process deductions</span>
            </a>
            <a routerLink="/saccos" class="action-card glass-card" id="quick-saccos">
              <div class="action-icon-wrapper" style="background:var(--color-info-light);color:var(--color-info);">
                <span class="material-icons-round">business</span>
              </div>
              <span class="action-label">SACCOs</span>
              <span class="action-desc">Manage cooperatives</span>
            </a>
          }
          <a routerLink="/wallets" class="action-card glass-card" id="quick-wallet">
            <div class="action-icon-wrapper" style="background:var(--color-success-light);color:var(--color-success);">
              <span class="material-icons-round">account_balance_wallet</span>
            </div>
            <span class="action-label">Wallet</span>
            <span class="action-desc">Balance & transactions</span>
          </a>
          <a routerLink="/earnings" class="action-card glass-card" id="quick-earnings">
            <div class="action-icon-wrapper" style="background:rgba(236,72,153,0.12);color:#ec4899;">
              <span class="material-icons-round">trending_up</span>
            </div>
            <span class="action-label">Earnings</span>
            <span class="action-desc">View earnings reports</span>
          </a>
          <a routerLink="/loans" class="action-card glass-card" id="quick-loans">
            <div class="action-icon-wrapper" style="background:rgba(251,191,36,0.12);color:#fbbf24;">
              <span class="material-icons-round">savings</span>
            </div>
            <span class="action-label">Loans</span>
            <span class="action-desc">Apply & track loans</span>
          </a>
          <a routerLink="/insurance" class="action-card glass-card" id="quick-insurance">
            <div class="action-icon-wrapper" style="background:rgba(34,211,238,0.12);color:#22d3ee;">
              <span class="material-icons-round">health_and_safety</span>
            </div>
            <span class="action-label">Insurance</span>
            <span class="action-desc">Manage policies</span>
          </a>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .live-badge {
      display: inline-flex; align-items: center; gap: 6px;
      padding: 4px 12px; border-radius: var(--radius-full);
      background: var(--color-success-light); color: var(--color-success);
      font-size: 0.75rem; font-weight: 600;
    }
    .live-dot {
      width: 6px; height: 6px; background: var(--color-success);
      border-radius: 50%; animation: pulse 2s infinite;
    }

    /* Wallet Hero */
    .wallet-hero {
      display: flex; align-items: center; justify-content: space-between;
      padding: var(--space-xl) !important;
      background: var(--gradient-accent-soft) !important;
      border: 1px solid rgba(0, 210, 255, 0.15) !important;
      margin-bottom: var(--space-md);
    }
    .wallet-hero-left { display: flex; flex-direction: column; }
    .wallet-hero-label { font-size: 0.8125rem; color: var(--color-text-muted); margin-bottom: 4px; }
    .wallet-hero-value {
      font-family: var(--font-heading); font-size: 2rem; font-weight: 800;
      background: var(--gradient-accent); -webkit-background-clip: text;
      -webkit-text-fill-color: transparent; background-clip: text;
    }
    .wallet-hero-currency { font-size: 0.75rem; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.1em; }

    /* Section */
    .section-block { margin-top: var(--space-xl); }
    .section-title {
      display: flex; align-items: center; gap: var(--space-sm);
      font-family: var(--font-heading); font-size: 1.125rem; font-weight: 600;
      color: var(--color-text-secondary); margin-bottom: var(--space-md);
    }

    /* Quick Actions */
    .actions-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: var(--space-md); }
    .action-card { display: flex; flex-direction: column; align-items: flex-start; gap: var(--space-sm); padding: var(--space-lg) !important; text-decoration: none; cursor: pointer; }
    .action-icon-wrapper {
      width: 40px; height: 40px; border-radius: var(--radius-md);
      display: flex; align-items: center; justify-content: center;
      .material-icons-round { font-size: 20px; }
    }
    .action-label { font-size: 0.875rem; font-weight: 600; color: var(--color-text-primary); }
    .action-desc { font-size: 0.75rem; color: var(--color-text-muted); }

    /* #54: Today's Earnings */
    .today-card { padding: var(--space-lg) !important; }
    .today-amount {
      font-family: var(--font-heading); font-size: 1.75rem; font-weight: 800;
      background: linear-gradient(135deg, #34d399, #10b981);
      -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
    }
    .today-meta { display: flex; gap: var(--space-md); margin-top: var(--space-sm); }
    .today-tag {
      display: inline-flex; align-items: center; gap: 4px;
      font-size: 0.8125rem; color: var(--color-text-muted);
      .material-icons-round { font-size: 16px; }
    }

    /* #57: Sparkline Chart */
    .sparkline-card { padding: var(--space-lg) !important; }
    .sparkline-bars { display: flex; align-items: flex-end; gap: 8px; height: 160px; }
    .sparkline-col {
      flex: 1; display: flex; flex-direction: column; align-items: center; gap: 4px;
      min-width: 0;
    }
    .sparkline-bar-wrapper { width: 100%; flex: 1; display: flex; align-items: flex-end; justify-content: center; }
    .sparkline-bar {
      width: 100%; max-width: 40px; min-height: 4px; border-radius: 4px 4px 0 0;
      background: linear-gradient(180deg, var(--color-accent), rgba(0, 210, 255, 0.3));
      transition: height 600ms cubic-bezier(0.4, 0, 0.2, 1);
    }
    .sparkline-label { font-size: 0.6875rem; color: var(--color-text-muted); font-weight: 500; }
    .sparkline-amount { font-size: 0.625rem; color: var(--color-text-muted); white-space: nowrap; }
    .sparkline-summary {
      display: flex; justify-content: space-between; margin-top: var(--space-md);
      padding-top: var(--space-sm); border-top: 1px solid var(--color-border);
      font-size: 0.8125rem; color: var(--color-text-muted);
      strong { color: var(--color-text-primary); }
    }

    /* #56: Recent Transactions */
    .txn-list { padding: 0 !important; overflow: hidden; }
    .txn-row {
      display: flex; align-items: center; gap: var(--space-md);
      padding: var(--space-md) var(--space-lg);
      border-bottom: 1px solid var(--color-border);
      transition: background 150ms ease;
      &:last-of-type { border-bottom: none; }
      &:hover { background: rgba(255, 255, 255, 0.02); }
    }
    .txn-icon {
      width: 36px; height: 36px; border-radius: var(--radius-md);
      display: flex; align-items: center; justify-content: center; flex-shrink: 0;
      .material-icons-round { font-size: 18px; }
      &.credit { background: var(--color-success-light); color: var(--color-success); }
      &.debit { background: var(--color-danger-light); color: var(--color-danger); }
    }
    .txn-details { flex: 1; display: flex; flex-direction: column; min-width: 0; }
    .txn-desc { font-size: 0.8125rem; font-weight: 500; color: var(--color-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .txn-time { font-size: 0.6875rem; color: var(--color-text-muted); }
    .txn-amount {
      font-family: var(--font-heading); font-size: 0.875rem; font-weight: 700; white-space: nowrap;
      &.credit { color: var(--color-success); }
      &.debit { color: var(--color-danger); }
    }
    .txn-view-all {
      display: block; text-align: center; padding: var(--space-sm) var(--space-lg);
      font-size: 0.8125rem; color: var(--color-accent); text-decoration: none;
      border-top: 1px solid var(--color-border);
      &:hover { background: rgba(0, 210, 255, 0.04); }
    }

    @media (max-width: 768px) {
      .wallet-hero { flex-direction: column; align-items: flex-start; gap: var(--space-md); }
      .wallet-hero-value { font-size: 1.5rem; }
      .actions-grid { grid-template-columns: 1fr 1fr; }
      .sparkline-bars { height: 120px; }
      .sparkline-amount { display: none; }
    }
    @media (max-width: 480px) {
      .actions-grid { grid-template-columns: 1fr; }
      .today-meta { flex-direction: column; gap: var(--space-xs); }
    }
  `]
})
export class DashboardComponent implements OnInit {
  private api = inject(ApiService);
  private http = inject(HttpClient);
  auth = inject(AuthService);

  stats = signal<SystemStats | null>(null);
  crewWallet = signal<Wallet | null>(null);
  todayEarnings = signal<DailySummary | null>(null);
  activeAssignments = signal(0);
  recentTxns = signal<WalletTransaction[]>([]);
  earningsHistory = signal<DailySummary[]>([]);

  private maxEarning = 0;
  private readonly API = environment.apiUrl;
  private readonly silentHeaders = { 'X-Skip-Error-Toast': 'true' };

  ngOnInit(): void {
    if (this.auth.isAdmin()) {
      this.api.getSystemStats().subscribe({ next: (res) => this.stats.set(res.data) });
    }

    const user = this.auth.currentUser();
    if (!user?.crew_member_id) return;
    const cmId = user.crew_member_id;

    // All crew dashboard calls use silent headers to avoid toast spam on 404
    this.http.get<any>(`${this.API}/wallets/${cmId}`, { headers: this.silentHeaders }).pipe(
      catchError(() => of(null))
    ).subscribe(res => { if (res?.data) this.crewWallet.set(res.data); });

    // #54: Today's earnings
    const today = new Date().toISOString().slice(0, 10);
    this.http.get<any>(`${this.API}/earnings/summary/${cmId}`, { headers: this.silentHeaders, params: { date: today } }).pipe(
      catchError(() => of(null))
    ).subscribe(res => { if (res?.data) this.todayEarnings.set(res.data); });

    // #55: Active assignments count
    this.http.get<any>(`${this.API}/assignments`, { headers: this.silentHeaders, params: { crew_member_id: cmId, status: 'ACTIVE', limit: '1' } }).pipe(
      catchError(() => of(null))
    ).subscribe(res => { if (res) this.activeAssignments.set(res.meta?.total ?? res.data?.length ?? 0); });

    // #56: Recent transactions
    this.http.get<any>(`${this.API}/wallets/${cmId}/transactions`, { headers: this.silentHeaders, params: { limit: '5' } }).pipe(
      catchError(() => of(null))
    ).subscribe(res => { if (res?.data) this.recentTxns.set(res.data); });

    // #57: Earnings history — last 7 days
    this.loadEarningsHistory(cmId);
  }

  private loadEarningsHistory(cmId: string): void {
    const days: DailySummary[] = [];
    let pending = 7;
    const emptyDay = (date: string): DailySummary => ({ id: '', crew_member_id: '', date, total_earned_cents: 0, total_deductions_cents: 0, net_amount_cents: 0, currency: 'KES', assignment_count: 0, is_processed: false, created_at: '' });

    for (let i = 6; i >= 0; i--) {
      const d = new Date();
      d.setDate(d.getDate() - i);
      const dateStr = d.toISOString().slice(0, 10);
      this.http.get<any>(`${this.API}/earnings/summary/${cmId}`, { headers: this.silentHeaders, params: { date: dateStr } }).pipe(
        catchError(() => of(null))
      ).subscribe(res => {
        days.push(res?.data ?? emptyDay(dateStr));
        pending--;
        if (pending === 0) {
          const sorted = days.sort((a, b) => a.date.localeCompare(b.date));
          this.maxEarning = Math.max(...sorted.map(d => d.total_earned_cents), 1);
          this.earningsHistory.set(sorted);
        }
      });
    }
  }

  barHeight(cents: number): number {
    return this.maxEarning > 0 ? Math.max((cents / this.maxEarning) * 100, 3) : 3;
  }

  dayLabel(dateStr: string): string {
    const d = new Date(dateStr + 'T00:00:00');
    return d.toLocaleDateString('en', { weekday: 'short' });
  }

  weekTotal(): number {
    return this.earningsHistory().reduce((s, d) => s + d.total_earned_cents, 0);
  }

  weekAvg(): number {
    const h = this.earningsHistory();
    return h.length > 0 ? Math.round(this.weekTotal() / h.length) : 0;
  }

  greeting(): string {
    const hour = new Date().getHours();
    if (hour < 12) return 'Good morning';
    if (hour < 17) return 'Good afternoon';
    return 'Good evening';
  }
}
