import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { SystemStats, AuditLog } from '../../../core/models';

@Component({
  selector: 'app-admin-dashboard', standalone: true,
  imports: [CommonModule, CurrencyKesPipe, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">System Administration</h1><p class="page-subtitle">Platform monitoring, user management, and audit logs</p></div>
      </div>

      @if (stats(); as s) {
        <div class="stats-grid">
          <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">people</span></div><div class="stat-value">{{s.total_users}}</div><div class="stat-label">Total Users</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">verified_user</span></div><div class="stat-value">{{s.active_users}}</div><div class="stat-label">Active Users</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-info-light);color:var(--color-info);"><span class="material-icons-round">groups</span></div><div class="stat-value">{{s.total_crew}}</div><div class="stat-label">Crew Members</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-warning-light);color:var(--color-warning);"><span class="material-icons-round">account_balance_wallet</span></div><div class="stat-value">{{s.total_wallet_balance_cents|currencyKes}}</div><div class="stat-label">Total Wallet Float</div></div>
        </div>
      }

      <div class="glass-card" style="margin-top:var(--space-lg);">
        <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Recent Audit Logs</h3>
        @if (logsLoading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:40px;margin:4px 0;"></div>} }
        @else if (logs().length === 0) { <p class="text-muted" style="padding:var(--space-md);">No audit logs found.</p> }
        @else {
          <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Action</th><th>Resource</th><th>Time</th></tr></thead>
            <tbody>@for(l of logs();track l.id){<tr>
              <td style="font-weight:500;color:var(--color-text-primary);">{{l.action}}</td>
              <td><span class="badge badge-neutral">{{l.resource}}</span></td>
              <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{l.created_at|relativeTime}}</td>
            </tr>}</tbody></table></div>
        }
      </div>
    </div>`,
})
export class AdminDashboardComponent implements OnInit {
  private api = inject(ApiService);
  stats = signal<SystemStats | null>(null);
  logs = signal<AuditLog[]>([]);
  logsLoading = signal(true);

  ngOnInit() {
    this.api.getSystemStats().subscribe({next: r => this.stats.set(r.data)});
    this.api.getAuditLogs({per_page: '20'}).subscribe({
      next: r => { this.logs.set(r.data); this.logsLoading.set(false); },
      error: () => this.logsLoading.set(false),
    });
  }
}
