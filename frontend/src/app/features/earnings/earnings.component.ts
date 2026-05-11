import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { CurrencyKesPipe } from '../../shared/pipes/currency-kes.pipe';
import { AutocompleteComponent, AutocompleteOption } from '../../shared/components/autocomplete/autocomplete.component';
import { Earning, CrewMember, DailySummary, WorkType } from '../../core/models';

type AggPeriod = 'daily' | 'weekly' | 'monthly';

interface ChartBar {
  label: string;
  value: number;
  percent: number;
}

@Component({
  selector: 'app-earnings',
  standalone: true,
  imports: [CommonModule, FormsModule, CurrencyKesPipe, AutocompleteComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Earnings Dashboard</h1><p class="page-subtitle">Track earnings across shifts and assignments</p></div>
      </div>

      <!-- Admin Filters -->
      <div class="filters-bar" style="flex-wrap:wrap;">
        @if (isAdmin()) {
          <div style="position: relative; z-index: 54; flex: 1; min-width: 220px; max-width: 260px;">
            <app-autocomplete [(ngModel)]="selectedCrewMemberId" (ngModelChange)="onCrewMemberChange()" [options]="crewOptions()" placeholder="— Search Crew Member —" id="filter-crew"></app-autocomplete>
          </div>
        }
        <input class="form-input" placeholder="Assignment ID" [(ngModel)]="filterAssignmentId" (ngModelChange)="loadEarnings()" id="filter-assignment" style="max-width:220px;" />
        <select class="form-select" [(ngModel)]="filterWorkType" (ngModelChange)="loadEarnings()" id="filter-work-type" style="max-width:160px;">
          <option value="">All Work Types</option>
          <option value="SHIFT">Shift</option><option value="DAILY">Daily</option>
          <option value="HOURLY">Hourly</option><option value="PER_TRIP">Per Trip</option>
          <option value="PROJECT">Project</option><option value="TASK">Task</option>
        </select>
        <input class="form-input" type="date" [(ngModel)]="dateFrom" (ngModelChange)="loadEarnings()" id="filter-from" style="max-width:170px;" />
        <input class="form-input" type="date" [(ngModel)]="dateTo" (ngModelChange)="loadEarnings()" id="filter-to" style="max-width:170px;" />
      </div>

      <!-- Summary Cards -->
      @if (summary()) {
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); margin-bottom: var(--space-lg);">
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">payments</span></div><div class="stat-value" style="color:var(--color-success);">{{ summary()!.total_earned_cents | currencyKes }}</div><div class="stat-label">Total Earned</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(239,68,68,0.12);color:#ef4444;"><span class="material-icons-round">remove_circle_outline</span></div><div class="stat-value" style="color:#ef4444;">{{ summary()!.total_deductions_cents | currencyKes }}</div><div class="stat-label">Deductions</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">account_balance_wallet</span></div><div class="stat-value" style="color:var(--color-accent);">{{ summary()!.net_amount_cents | currencyKes }}</div><div class="stat-label">Net Earnings</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(168,85,247,0.12);color:#a855f7;"><span class="material-icons-round">assignment_turned_in</span></div><div class="stat-value">{{ summary()!.assignment_count }}</div><div class="stat-label">Assignments</div></div>
        </div>
      }

      <!-- Chart Section -->
      <div class="glass-card" style="margin-bottom:var(--space-lg);">
        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:var(--space-md);">
          <h3 style="font-size:1rem;font-weight:600;">Earnings Over Time</h3>
          <div class="agg-toggle">
            <button class="agg-btn" [class.active]="aggPeriod === 'daily'" (click)="setAggPeriod('daily')">Daily</button>
            <button class="agg-btn" [class.active]="aggPeriod === 'weekly'" (click)="setAggPeriod('weekly')">Weekly</button>
            <button class="agg-btn" [class.active]="aggPeriod === 'monthly'" (click)="setAggPeriod('monthly')">Monthly</button>
          </div>
        </div>

        @if (chartLoading()) {
          <div class="skeleton" style="height:200px;border-radius:var(--radius-md);"></div>
        } @else if (chartData().length === 0) {
          <div class="empty-state" style="padding:var(--space-xl);"><span class="material-icons-round empty-icon">bar_chart</span><div class="empty-title">No chart data</div><div class="empty-subtitle">Select a crew member and date range to view earnings</div></div>
        } @else {
          <div class="chart-container">
            <div class="chart-y-axis">
              <span>{{ chartMax() | currencyKes }}</span>
              <span>{{ chartMid() | currencyKes }}</span>
              <span>KES 0</span>
            </div>
            <div class="chart-bars">
              @for (bar of chartData(); track bar.label) {
                <div class="chart-bar-wrapper" [title]="bar.label + ': ' + (bar.value | currencyKes)">
                  <div class="chart-bar" [style.height.%]="bar.percent">
                    <div class="chart-bar-fill"></div>
                  </div>
                  <div class="chart-bar-label">{{ bar.label }}</div>
                </div>
              }
            </div>
          </div>
        }
      </div>

      <!-- Earnings List (F12: grouped by work type/site) -->
      <div class="glass-card">
        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:var(--space-md);flex-wrap:wrap;gap:8px;">
          <h3 style="font-size:1rem;font-weight:600;">Earnings Records</h3>
          <div class="agg-toggle">
            <button class="agg-btn" [class.active]="groupBy === 'none'" (click)="groupBy = 'none'" id="group-none">All</button>
            <button class="agg-btn" [class.active]="groupBy === 'work_type'" (click)="groupBy = 'work_type'" id="group-type">By Work Type</button>
            <button class="agg-btn" [class.active]="groupBy === 'site'" (click)="groupBy = 'site'" id="group-site">By Site</button>
          </div>
        </div>
        @if (earningsLoading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:48px;margin:4px 0;"></div>} }
        @else if (earnings().length === 0) { <div class="empty-state" style="padding:var(--space-lg);"><span class="material-icons-round empty-icon">trending_up</span><div class="empty-title">No earnings recorded</div></div> }
        @else {
          <!-- Grouped Summary Cards -->
          @if (groupBy !== 'none') {
            <div class="group-summary-grid">
              @for (g of groupedEarnings(); track g.key) {
                <div class="group-summary-card">
                  <div class="group-key">
                    <span class="badge badge-accent">{{ g.key || 'Unspecified' }}</span>
                  </div>
                  <div class="group-value" style="color:var(--color-success);font-weight:700;font-size:0.9rem;">{{ g.totalFormatted }}</div>
                  <div class="group-count">{{ g.count }} record{{ g.count !== 1 ? 's' : '' }}</div>
                </div>
              }
            </div>
          }
          <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Date</th><th>Type</th><th>Work Type</th><th>Site</th><th>Amount</th><th>Verified</th></tr></thead>
            <tbody>@for(e of earnings();track e.id){<tr>
              <td>{{ e.earned_at | date:'mediumDate' }}</td>
              <td><span class="badge badge-accent">{{ e.earning_type }}</span></td>
              <td><span class="badge" style="background:rgba(168,85,247,0.12);color:#a855f7;">{{ e.work_type || 'SHIFT' }}</span></td>
              <td style="max-width:140px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{ e.work_site || '—' }}</td>
              <td style="font-weight:600;color:var(--color-success);">{{ e.amount_cents | currencyKes }}</td>
              <td><span class="badge" [ngClass]="e.is_verified ? 'badge-success' : 'badge-warning'">{{ e.is_verified ? 'Yes' : 'Pending' }}</span></td>
            </tr>}</tbody></table></div>
        }
      </div>
    </div>
  `,
  styles: [`
    .agg-toggle { display: flex; gap: 2px; background: var(--color-surface-alt); border-radius: var(--radius-md); padding: 2px; }
    .agg-btn {
      padding: 4px 14px; border: none; background: transparent; color: var(--color-text-muted);
      font-size: 0.75rem; font-weight: 500; border-radius: var(--radius-sm); cursor: pointer; transition: all 0.2s;
      &.active { background: var(--color-accent); color: #fff; }
      &:hover:not(.active) { color: var(--color-text-primary); }
    }
    .chart-container { display: flex; gap: var(--space-sm); height: 220px; }
    .chart-y-axis {
      display: flex; flex-direction: column; justify-content: space-between; align-items: flex-end;
      font-size: 0.65rem; color: var(--color-text-muted); min-width: 70px; padding-bottom: 20px;
    }
    .chart-bars { display: flex; flex: 1; gap: 3px; align-items: flex-end; padding-bottom: 20px; border-bottom: 1px solid var(--color-border); position: relative; }
    .chart-bar-wrapper { flex: 1; display: flex; flex-direction: column; align-items: center; min-width: 0; }
    .chart-bar { width: 100%; max-width: 40px; min-height: 2px; border-radius: 4px 4px 0 0; overflow: hidden; transition: height 0.4s ease; }
    .chart-bar-fill { width: 100%; height: 100%; background: linear-gradient(180deg, var(--color-accent) 0%, rgba(0,210,255,0.4) 100%); border-radius: 4px 4px 0 0; }
    .chart-bar-label { font-size: 0.6rem; color: var(--color-text-muted); margin-top: 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; text-align: center; }
    .chart-bar-wrapper:hover .chart-bar-fill { background: linear-gradient(180deg, #a855f7 0%, rgba(168,85,247,0.5) 100%); }
    .group-summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: var(--space-sm); margin-bottom: var(--space-md); }
    .group-summary-card {
      background: var(--color-surface-alt); border-radius: var(--radius-md); padding: var(--space-md);
      display: flex; flex-direction: column; gap: 2px;
    }
    .group-key { margin-bottom: 4px; }
    .group-count { font-size: 0.7rem; color: var(--color-text-muted); }
    input[type="date"] { color-scheme: dark; cursor: pointer; }
    input[type="date"]::-webkit-calendar-picker-indicator { filter: invert(0.7) sepia(1) saturate(5) hue-rotate(175deg); cursor: pointer; }
  `]
})
export class EarningsComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);

  // State
  earnings = signal<Earning[]>([]);
  earningsLoading = signal(false);
  summary = signal<DailySummary | null>(null);
  chartData = signal<ChartBar[]>([]);
  chartLoading = signal(false);
  crewMembers = signal<CrewMember[]>([]);
  crewOptions = computed<AutocompleteOption[]>(() => this.crewMembers().map(c => ({
    value: c.id,
    label: `${c.first_name} ${c.last_name}`,
    sublabel: `ID: ${c.crew_id}`,
    searchText: `${c.first_name} ${c.last_name} ${c.crew_id}`
  })));

  // Filters
  selectedCrewMemberId = '';
  filterAssignmentId = '';
  filterWorkType = '';
  dateFrom = '';
  dateTo = '';
  aggPeriod: AggPeriod = 'daily';
  groupBy: 'none' | 'work_type' | 'site' = 'none';

  // Computed
  chartMax = computed(() => {
    const max = Math.max(...this.chartData().map(b => b.value), 0);
    return max || 100;
  });
  chartMid = computed(() => Math.round(this.chartMax() / 2));

  isAdmin(): boolean {
    return this.auth.hasRole('SYSTEM_ADMIN', 'EMPLOYER');
  }

  ngOnInit(): void {
    // Set default date range: last 30 days
    const now = new Date();
    this.dateTo = now.toISOString().slice(0, 10);
    const from = new Date(now);
    from.setDate(from.getDate() - 30);
    this.dateFrom = from.toISOString().slice(0, 10);

    if (this.isAdmin()) {
      this.api.getCrewMembers({ per_page: '200' }).subscribe({
        next: r => this.crewMembers.set(r.data),
      });
    } else {
      // Non-admin: load their own earnings (backend filters by auth token)
      this.loadEarnings();
    }

    this.loadEarnings();
  }

  onCrewMemberChange(): void {
    this.loadAll();
  }

  loadAll(): void {
    this.loadEarnings();
    this.loadSummary();
    this.loadChartData();
  }

  loadEarnings(): void {
    this.earningsLoading.set(true);
    const p: Record<string, string> = { per_page: '100' };
    if (this.selectedCrewMemberId) p['crew_member_id'] = this.selectedCrewMemberId;
    if (this.filterAssignmentId) p['assignment_id'] = this.filterAssignmentId;
    if (this.filterWorkType) p['work_type'] = this.filterWorkType;
    if (this.dateFrom) p['date_from'] = this.dateFrom;
    if (this.dateTo) p['date_to'] = this.dateTo;
    this.api.getEarnings(p).subscribe({
      next: r => {
        this.earnings.set(r.data || []);
        this.earningsLoading.set(false);
        // Build chart from raw earnings if no crew member selected for summary
        if (!this.selectedCrewMemberId) {
          this.buildChartFromEarnings(r.data || []);
        }
      },
      error: () => this.earningsLoading.set(false),
    });
  }

  loadSummary(): void {
    if (!this.selectedCrewMemberId) { this.summary.set(null); return; }
    this.api.getEarningSummary(this.selectedCrewMemberId, this.dateTo || undefined).subscribe({
      next: r => this.summary.set(r.data),
      error: () => this.summary.set(null),
    });
  }

  loadChartData(): void {
    if (!this.selectedCrewMemberId) return;
    this.chartLoading.set(true);
    // Fetch all earnings for this crew member in date range, then aggregate client-side
    const p: Record<string, string> = { per_page: '500', crew_member_id: this.selectedCrewMemberId };
    if (this.dateFrom) p['date_from'] = this.dateFrom;
    if (this.dateTo) p['date_to'] = this.dateTo;
    this.api.getEarnings(p).subscribe({
      next: r => {
        this.buildChartFromEarnings(r.data || []);
        this.chartLoading.set(false);
      },
      error: () => this.chartLoading.set(false),
    });
  }

  setAggPeriod(period: AggPeriod): void {
    this.aggPeriod = period;
    // Rebuild chart with existing earnings data
    this.buildChartFromEarnings(this.earnings());
  }

  // F12: Group earnings by work type or site
  groupedEarnings(): { key: string; count: number; total: number; totalFormatted: string }[] {
    const map = new Map<string, { count: number; total: number }>();
    for (const e of this.earnings()) {
      const key = this.groupBy === 'work_type' ? (e.work_type || 'SHIFT') : (e.work_site || 'Unspecified');
      const existing = map.get(key);
      if (existing) { existing.count++; existing.total += e.amount_cents; }
      else { map.set(key, { count: 1, total: e.amount_cents }); }
    }
    return [...map.entries()]
      .map(([key, v]) => ({ key, ...v, totalFormatted: `KES ${(v.total / 100).toFixed(2)}` }))
      .sort((a, b) => b.total - a.total);
  }

  private buildChartFromEarnings(earnings: Earning[]): void {
    if (!earnings.length) { this.chartData.set([]); return; }

    // Group by period
    const groups = new Map<string, number>();
    for (const e of earnings) {
      const key = this.getGroupKey(e.earned_at || e.created_at);
      groups.set(key, (groups.get(key) || 0) + e.amount_cents);
    }

    // Fill in missing periods
    this.fillMissingPeriods(groups);

    // Sort by key and build chart bars
    const sorted = [...groups.entries()].sort((a, b) => a[0].localeCompare(b[0]));
    const maxVal = Math.max(...sorted.map(([, v]) => v), 1);
    const bars: ChartBar[] = sorted.map(([key, val]) => ({
      label: this.formatLabel(key),
      value: val,
      percent: Math.max((val / maxVal) * 100, 1),
    }));

    // Limit to last 30 bars
    this.chartData.set(bars.slice(-30));
  }

  private getGroupKey(dateStr: string): string {
    const d = new Date(dateStr);
    switch (this.aggPeriod) {
      case 'daily':
        return d.toISOString().slice(0, 10);
      case 'weekly': {
        // ISO week: Monday-based
        const day = d.getDay() || 7;
        const mon = new Date(d);
        mon.setDate(d.getDate() - day + 1);
        return mon.toISOString().slice(0, 10);
      }
      case 'monthly':
        return d.toISOString().slice(0, 7);
    }
  }

  private fillMissingPeriods(groups: Map<string, number>): void {
    if (!this.dateFrom || !this.dateTo) return;
    const start = new Date(this.dateFrom);
    const end = new Date(this.dateTo);
    const cur = new Date(start);

    while (cur <= end) {
      const key = this.getGroupKey(cur.toISOString());
      if (!groups.has(key)) groups.set(key, 0);
      switch (this.aggPeriod) {
        case 'daily': cur.setDate(cur.getDate() + 1); break;
        case 'weekly': cur.setDate(cur.getDate() + 7); break;
        case 'monthly': cur.setMonth(cur.getMonth() + 1); break;
      }
    }
  }

  private formatLabel(key: string): string {
    switch (this.aggPeriod) {
      case 'daily': {
        const d = new Date(key + 'T00:00:00');
        return d.toLocaleDateString('en', { month: 'short', day: 'numeric' });
      }
      case 'weekly': {
        const d = new Date(key + 'T00:00:00');
        return 'W' + d.toLocaleDateString('en', { month: 'short', day: 'numeric' });
      }
      case 'monthly': {
        const [y, m] = key.split('-');
        return new Date(+y, +m - 1).toLocaleDateString('en', { month: 'short', year: '2-digit' });
      }
    }
  }
}
