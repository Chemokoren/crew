import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { AuditLog } from '../../../core/models';

@Component({
  selector: 'app-platform-compliance',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './platform-compliance.component.html',
  styleUrl: './platform-compliance.component.scss',
})
export class PlatformComplianceComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  loading = signal(true);
  logs = signal<AuditLog[]>([]);
  totalLogs = signal(0);
  page = signal(1);
  perPage = 20;

  // Filters
  filterAction = signal('');
  filterEntity = signal('');
  filterUser = signal('');
  filterDateFrom = signal('');
  filterDateTo = signal('');

  // Stats
  totalActions = signal(0);
  uniqueUsers = signal(0);

  readonly actions = ['CREATE', 'UPDATE', 'DELETE', 'LOGIN', 'LOGOUT', 'APPROVE', 'REJECT', 'EXPORT'];
  readonly entities = ['user', 'crew_member', 'assignment', 'payroll', 'wallet', 'organization', 'document', 'loan', 'insurance'];

  ngOnInit() { this.loadLogs(); }

  loadLogs() {
    this.loading.set(true);
    const params: Record<string, string> = { page: String(this.page()), per_page: String(this.perPage) };
    if (this.filterAction()) params['action'] = this.filterAction();
    if (this.filterEntity()) params['resource'] = this.filterEntity();
    if (this.filterUser()) params['user_id'] = this.filterUser();

    this.api.getAuditLogs(params).subscribe({
      next: r => {
        this.logs.set(r.data || []);
        this.totalLogs.set(r.meta?.total || 0);
        this.totalActions.set(r.meta?.total || 0);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  applyFilters() { this.page.set(1); this.loadLogs(); }
  clearFilters() {
    this.filterAction.set(''); this.filterEntity.set(''); this.filterUser.set('');
    this.filterDateFrom.set(''); this.filterDateTo.set('');
    this.page.set(1); this.loadLogs();
  }

  nextPage() { if (this.page() * this.perPage < this.totalLogs()) { this.page.set(this.page() + 1); this.loadLogs(); } }
  prevPage() { if (this.page() > 1) { this.page.set(this.page() - 1); this.loadLogs(); } }
  get totalPages(): number { return Math.ceil(this.totalLogs() / this.perPage); }

  exportCSV() {
    const csv = ['Timestamp,Action,Entity,Entity ID,Actor,Details'];
    for (const log of this.logs()) {
      csv.push(`"${log.created_at}","${log.action}","${log.resource}","${log.resource_id}","${log.actor_id || ''}","${(log.details || '').replace(/"/g, '""')}"`);
    }
    const blob = new Blob([csv.join('\n')], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = `audit-logs-${new Date().toISOString().split('T')[0]}.csv`;
    a.click(); URL.revokeObjectURL(url);
    this.toast.success('Audit logs exported');
  }

  actionColor(action: string): string {
    switch (action) {
      case 'CREATE': return '#10b981'; case 'UPDATE': return '#6366f1'; case 'DELETE': return '#ef4444';
      case 'LOGIN': return '#3b82f6'; case 'APPROVE': return '#10b981'; case 'REJECT': return '#ef4444';
      default: return '#8b5cf6';
    }
  }
}
