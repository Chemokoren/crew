import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { SystemStats, Organization, AuditLog } from '../../../core/models';

@Component({
  selector: 'app-command-center',
  standalone: true,
  imports: [CommonModule, RouterLink, CurrencyKesPipe, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title platform-title">Command Center</h1>
          <p class="page-subtitle">Platform-wide health, metrics, and operational overview</p>
        </div>
        <div class="page-actions">
          <span class="live-badge">
            <span class="live-dot"></span> Live
          </span>
        </div>
      </div>

      <!-- System Health -->
      @if (stats(); as s) {
        <div class="stats-grid">
          <div class="stat-card platform-stat">
            <div class="stat-icon" style="background: rgba(139, 92, 246, 0.12); color: #8b5cf6;">
              <span class="material-icons-round">people</span>
            </div>
            <div class="stat-value">{{ s.total_users | number }}</div>
            <div class="stat-label">Total Users</div>
            <div class="stat-change" style="background: rgba(16,185,129,0.12); color: #10b981;">
              <span class="material-icons-round" style="font-size:14px;">trending_up</span> Platform-wide
            </div>
          </div>

          <div class="stat-card platform-stat">
            <div class="stat-icon" style="background: rgba(236, 72, 153, 0.12); color: #ec4899;">
              <span class="material-icons-round">verified_user</span>
            </div>
            <div class="stat-value">{{ s.active_users | number }}</div>
            <div class="stat-label">Active Users</div>
          </div>

          <div class="stat-card platform-stat">
            <div class="stat-icon" style="background: rgba(99, 102, 241, 0.12); color: #6366f1;">
              <span class="material-icons-round">business</span>
            </div>
            <div class="stat-value">{{ s.total_saccos | number }}</div>
            <div class="stat-label">Organizations</div>
          </div>

          <div class="stat-card platform-stat">
            <div class="stat-icon" style="background: rgba(16, 185, 129, 0.12); color: #10b981;">
              <span class="material-icons-round">groups</span>
            </div>
            <div class="stat-value">{{ s.total_crew | number }}</div>
            <div class="stat-label">Workers</div>
          </div>

          <div class="stat-card platform-stat">
            <div class="stat-icon" style="background: rgba(245, 158, 11, 0.12); color: #f59e0b;">
              <span class="material-icons-round">account_balance_wallet</span>
            </div>
            <div class="stat-value">{{ s.total_wallet_balance_cents | currencyKes }}</div>
            <div class="stat-label">Total Float</div>
          </div>

          <div class="stat-card platform-stat">
            <div class="stat-icon" style="background: rgba(34, 211, 238, 0.12); color: #22d3ee;">
              <span class="material-icons-round">directions_bus</span>
            </div>
            <div class="stat-value">{{ s.total_vehicles | number }}</div>
            <div class="stat-label">Fleet Vehicles</div>
          </div>
        </div>
      } @else {
        <div class="stats-grid">
          @for (i of [1,2,3,4,5,6]; track i) {
            <div class="skeleton" style="height: 140px;"></div>
          }
        </div>
      }

      <div class="cc-grid">
        <!-- Quick Actions -->
        <div class="cc-section">
          <h2 class="cc-section-title">
            <span class="material-icons-round" style="font-size:20px; color:#8b5cf6;">bolt</span>
            Quick Actions
          </h2>
          <div class="quick-actions-grid">
            <a routerLink="/platform/organizations" class="quick-action glass-card" id="cc-orgs">
              <div class="qa-icon" style="background: rgba(139,92,246,0.12); color:#8b5cf6;">
                <span class="material-icons-round">business</span>
              </div>
              <span class="qa-label">Manage Organizations</span>
              <span class="qa-desc">View, configure, and monitor all orgs</span>
            </a>
            <a routerLink="/platform/users" class="quick-action glass-card" id="cc-users">
              <div class="qa-icon" style="background: rgba(236,72,153,0.12); color:#ec4899;">
                <span class="material-icons-round">person_search</span>
              </div>
              <span class="qa-label">User Lookup</span>
              <span class="qa-desc">Search, reset, and assist users</span>
            </a>
            <a routerLink="/platform/finance" class="quick-action glass-card" id="cc-finance">
              <div class="qa-icon" style="background: rgba(16,185,129,0.12); color:#10b981;">
                <span class="material-icons-round">account_balance</span>
              </div>
              <span class="qa-label">Float Oversight</span>
              <span class="qa-desc">Monitor float across all organizations</span>
            </a>
            <a routerLink="/platform/compliance" class="quick-action glass-card" id="cc-audit">
              <div class="qa-icon" style="background: rgba(245,158,11,0.12); color:#f59e0b;">
                <span class="material-icons-round">history</span>
              </div>
              <span class="qa-label">Audit Trail</span>
              <span class="qa-desc">Review system-wide activity logs</span>
            </a>
            <a routerLink="/platform/settings" class="quick-action glass-card" id="cc-settings">
              <div class="qa-icon" style="background: rgba(99,102,241,0.12); color:#6366f1;">
                <span class="material-icons-round">settings</span>
              </div>
              <span class="qa-label">System Settings</span>
              <span class="qa-desc">Statutory rates, feature flags</span>
            </a>
            <a routerLink="/platform/team" class="quick-action glass-card" id="cc-team">
              <div class="qa-icon" style="background: rgba(34,211,238,0.12); color:#22d3ee;">
                <span class="material-icons-round">group_work</span>
              </div>
              <span class="qa-label">Platform Team</span>
              <span class="qa-desc">Manage support & admin staff</span>
            </a>
          </div>
        </div>

        <!-- Recent Audit Activity -->
        <div class="cc-section">
          <h2 class="cc-section-title">
            <span class="material-icons-round" style="font-size:20px; color:#f59e0b;">history</span>
            Recent Activity
          </h2>
          @if (auditLoading()) {
            @for (i of [1,2,3]; track i) {
              <div class="skeleton" style="height: 48px; margin-bottom: 8px;"></div>
            }
          } @else if (recentAudits().length === 0) {
            <div class="empty-state" style="padding: var(--space-lg);">
              <span class="material-icons-round empty-icon" style="font-size: 36px;">history</span>
              <div class="empty-title" style="font-size: 0.875rem;">No recent activity</div>
            </div>
          } @else {
            <div class="audit-feed glass-card">
              @for (log of recentAudits(); track log.id) {
                <div class="audit-row">
                  <div class="audit-action-icon" [ngClass]="auditIconClass(log.action)">
                    <span class="material-icons-round">{{ auditIcon(log.action) }}</span>
                  </div>
                  <div class="audit-info">
                    <span class="audit-action">{{ log.action }}</span>
                    <span class="audit-resource">
                      <span class="badge badge-neutral">{{ log.resource }}</span>
                    </span>
                  </div>
                  <span class="audit-time">{{ log.created_at | relativeTime }}</span>
                </div>
              }
              <a routerLink="/platform/compliance" class="feed-view-all">View full audit trail →</a>
            </div>
          }
        </div>
      </div>

      <!-- Organizations Overview -->
      <div class="cc-section" style="margin-top: var(--space-xl);">
        <div class="cc-section-header">
          <h2 class="cc-section-title">
            <span class="material-icons-round" style="font-size:20px; color:#6366f1;">business</span>
            Organizations
          </h2>
          <a routerLink="/platform/organizations" class="btn btn-sm btn-secondary">View All →</a>
        </div>
        @if (orgsLoading()) {
          @for (i of [1,2,3]; track i) {
            <div class="skeleton" style="height: 60px; margin-bottom: 8px;"></div>
          }
        } @else {
          <div class="org-cards-grid">
            @for (org of topOrgs(); track org.id) {
              <a [routerLink]="'/platform/organizations'" class="org-card glass-card">
                <div class="org-card-header">
                  <div class="org-avatar">{{ org.name.charAt(0) }}</div>
                  <div class="org-meta">
                    <span class="org-name">{{ org.name }}</span>
                    <span class="org-type">{{ org.industry_type || org.organization_type || 'General' }}</span>
                  </div>
                </div>
                <div class="org-card-footer">
                  <span class="badge" [ngClass]="org.is_active ? 'badge-success' : 'badge-danger'">
                    {{ org.is_active ? 'Active' : 'Inactive' }}
                  </span>
                  <span class="org-county">{{ org.county }}</span>
                </div>
              </a>
            }
          </div>
        }
      </div>
    </div>
  `,
  styles: [`
    .platform-title {
      background: linear-gradient(135deg, #c084fc 0%, #f472b6 100%) !important;
      -webkit-background-clip: text !important;
      -webkit-text-fill-color: transparent !important;
      background-clip: text !important;
    }

    .live-badge {
      display: inline-flex; align-items: center; gap: 6px;
      padding: 4px 12px; border-radius: var(--radius-full);
      background: rgba(16,185,129,0.12); color: #10b981;
      font-size: 0.75rem; font-weight: 600;
    }
    .live-dot {
      width: 6px; height: 6px; background: #10b981;
      border-radius: 50%; animation: pulse 2s infinite;
    }

    .platform-stat::before {
      background: linear-gradient(135deg, #8b5cf6 0%, #ec4899 100%) !important;
    }

    .cc-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: var(--space-xl);
      margin-top: var(--space-md);
    }

    .cc-section-title {
      display: flex; align-items: center; gap: var(--space-sm);
      font-family: var(--font-heading); font-size: 1.125rem; font-weight: 600;
      color: var(--color-text-secondary); margin-bottom: var(--space-md);
    }

    .cc-section-header {
      display: flex; align-items: center; justify-content: space-between;
      margin-bottom: var(--space-md);
    }
    .cc-section-header .cc-section-title { margin-bottom: 0; }

    /* Quick Actions */
    .quick-actions-grid {
      display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-sm);
    }

    .quick-action {
      display: flex; flex-direction: column; gap: 6px;
      padding: var(--space-md) !important; text-decoration: none; cursor: pointer;
    }
    .qa-icon {
      width: 36px; height: 36px; border-radius: var(--radius-md);
      display: flex; align-items: center; justify-content: center;
      .material-icons-round { font-size: 18px; }
    }
    .qa-label { font-size: 0.8125rem; font-weight: 600; color: var(--color-text-primary); }
    .qa-desc { font-size: 0.6875rem; color: var(--color-text-muted); }

    /* Audit Feed */
    .audit-feed { padding: 0 !important; overflow: hidden; }
    .audit-row {
      display: flex; align-items: center; gap: var(--space-md);
      padding: 12px var(--space-md);
      border-bottom: 1px solid var(--color-border);
      &:last-of-type { border-bottom: none; }
    }
    .audit-action-icon {
      width: 32px; height: 32px; border-radius: var(--radius-md);
      display: flex; align-items: center; justify-content: center; flex-shrink: 0;
      .material-icons-round { font-size: 16px; }
      &.action-create { background: rgba(16,185,129,0.12); color: #10b981; }
      &.action-update { background: rgba(99,102,241,0.12); color: #6366f1; }
      &.action-delete { background: rgba(239,68,68,0.12); color: #ef4444; }
      &.action-default { background: rgba(139,92,246,0.12); color: #8b5cf6; }
    }
    .audit-info {
      flex: 1; display: flex; flex-direction: column; gap: 2px; min-width: 0;
    }
    .audit-action { font-size: 0.8125rem; font-weight: 500; color: var(--color-text-primary); }
    .audit-resource { font-size: 0.75rem; }
    .audit-time { font-size: 0.6875rem; color: var(--color-text-muted); white-space: nowrap; }
    .feed-view-all {
      display: block; text-align: center; padding: 10px;
      font-size: 0.8125rem; color: #8b5cf6; text-decoration: none;
      border-top: 1px solid var(--color-border);
      &:hover { background: rgba(139,92,246,0.04); }
    }

    /* Org Cards */
    .org-cards-grid {
      display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: var(--space-md);
    }
    .org-card {
      text-decoration: none; cursor: pointer; padding: var(--space-md) !important;
    }
    .org-card-header {
      display: flex; align-items: center; gap: var(--space-sm); margin-bottom: var(--space-sm);
    }
    .org-avatar {
      width: 36px; height: 36px; border-radius: var(--radius-md);
      background: linear-gradient(135deg, #8b5cf6 0%, #ec4899 100%);
      display: flex; align-items: center; justify-content: center;
      font-size: 0.875rem; font-weight: 700; color: #fff; flex-shrink: 0;
    }
    .org-meta { display: flex; flex-direction: column; min-width: 0; }
    .org-name {
      font-size: 0.875rem; font-weight: 600; color: var(--color-text-primary);
      white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    }
    .org-type {
      font-size: 0.6875rem; color: var(--color-text-muted); text-transform: capitalize;
    }
    .org-card-footer {
      display: flex; align-items: center; justify-content: space-between;
    }
    .org-county { font-size: 0.6875rem; color: var(--color-text-muted); }

    @media (max-width: 900px) {
      .cc-grid { grid-template-columns: 1fr; }
      .quick-actions-grid { grid-template-columns: 1fr; }
    }

    @media (max-width: 480px) {
      .org-cards-grid { grid-template-columns: 1fr; }
    }
  `]
})
export class CommandCenterComponent implements OnInit {
  private api = inject(ApiService);
  auth = inject(AuthService);

  stats = signal<SystemStats | null>(null);
  topOrgs = signal<Organization[]>([]);
  orgsLoading = signal(true);
  recentAudits = signal<AuditLog[]>([]);
  auditLoading = signal(true);

  ngOnInit(): void {
    this.api.getSystemStats().subscribe({
      next: r => this.stats.set(r.data),
    });

    this.api.getOrganizations({ per_page: '6' }).subscribe({
      next: r => { this.topOrgs.set(r.data || []); this.orgsLoading.set(false); },
      error: () => this.orgsLoading.set(false),
    });

    this.api.getAuditLogs({ per_page: '8' }).subscribe({
      next: r => { this.recentAudits.set(r.data || []); this.auditLoading.set(false); },
      error: () => this.auditLoading.set(false),
    });
  }

  auditIcon(action: string): string {
    const a = action.toLowerCase();
    if (a.includes('create') || a.includes('add')) return 'add_circle';
    if (a.includes('update') || a.includes('edit')) return 'edit';
    if (a.includes('delete') || a.includes('remove')) return 'delete';
    if (a.includes('login')) return 'login';
    if (a.includes('approve')) return 'check_circle';
    if (a.includes('reject')) return 'cancel';
    return 'info';
  }

  auditIconClass(action: string): string {
    const a = action.toLowerCase();
    if (a.includes('create') || a.includes('add') || a.includes('approve')) return 'action-create';
    if (a.includes('update') || a.includes('edit')) return 'action-update';
    if (a.includes('delete') || a.includes('remove') || a.includes('reject')) return 'action-delete';
    return 'action-default';
  }
}
