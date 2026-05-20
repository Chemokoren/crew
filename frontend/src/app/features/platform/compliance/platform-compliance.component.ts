import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { AuditLog } from '../../../core/models';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';

@Component({
  selector: 'app-platform-compliance',
  standalone: true,
  imports: [CommonModule, FormsModule, AutocompleteComponent],
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

  userOptions = signal<AutocompleteOption[]>([]);
  expandedLogId = signal<string | null>(null);

  toggleDetails(id: string) {
    this.expandedLogId.update(current => current === id ? null : id);
  }

  formatJson(val: any): string {
    if (!val || val === 'null') return 'null';
    if (typeof val === 'string') {
      try { return JSON.stringify(JSON.parse(val), null, 2); } catch { return val; }
    }
    return JSON.stringify(val, null, 2);
  }

  getUserName(userId: string | undefined): string {
    if (!userId) return 'System';
    const opt = this.userOptions().find(o => o.value === userId);
    return opt ? opt.label : (userId.substring(0, 8) + '...');
  }

  getIpAddress(log: AuditLog): string {
    if (log.ip_address) return log.ip_address;
    if (log.new_value && typeof log.new_value === 'object' && log.new_value.ip_address) {
      return log.new_value.ip_address;
    }
    if (log.new_value && typeof log.new_value === 'string') {
      try { const parsed = JSON.parse(log.new_value); return parsed.ip_address || 'Unknown IP'; } catch {}
    }
    return 'Unknown IP';
  }

  getUserAgent(log: AuditLog): string {
    if (log.user_agent) return log.user_agent;
    if (log.new_value && typeof log.new_value === 'object' && log.new_value.user_agent) {
      return log.new_value.user_agent;
    }
    if (log.new_value && typeof log.new_value === 'string') {
      try { const parsed = JSON.parse(log.new_value); return parsed.user_agent || 'Unknown'; } catch {}
    }
    return 'Unknown';
  }

  readonly actionOptions: AutocompleteOption[] = [
    { value: 'CREATE', label: 'Create', sublabel: 'Resource creation events', searchText: 'create add new' },
    { value: 'UPDATE', label: 'Update', sublabel: 'Resource modification events', searchText: 'update edit modify change' },
    { value: 'DELETE', label: 'Delete', sublabel: 'Resource deletion events', searchText: 'delete remove destroy' },
    { value: 'LOGIN', label: 'Login', sublabel: 'User authentication events', searchText: 'login sign in auth' },
    { value: 'LOGOUT', label: 'Logout', sublabel: 'User sign-out events', searchText: 'logout sign out' },
    { value: 'APPROVE', label: 'Approve', sublabel: 'Approval workflow events', searchText: 'approve accept confirm' },
    { value: 'REJECT', label: 'Reject', sublabel: 'Rejection workflow events', searchText: 'reject deny decline' },
    { value: 'EXPORT', label: 'Export', sublabel: 'Data export events', searchText: 'export download csv' },
    { value: 'DENIED', label: 'Access Denied', sublabel: 'Permission denied events', searchText: 'denied permission block' },
    { value: 'TOPUP', label: 'Top Up', sublabel: 'Wallet top up events', searchText: 'topup credit float deposit' },
    { value: 'CREDIT', label: 'Credit', sublabel: 'Credit events', searchText: 'credit' },
    { value: 'DEBIT', label: 'Debit', sublabel: 'Debit events', searchText: 'debit' },
  ];

  readonly entityOptions: AutocompleteOption[] = [
    { value: 'user', label: 'User', sublabel: 'Platform user accounts', searchText: 'user account profile', badge: 'AUTH' },
    { value: 'crew_member', label: 'Crew Member', sublabel: 'Worker profiles', searchText: 'crew member worker employee', badge: 'HR' },
    { value: 'assignment', label: 'Assignment', sublabel: 'Work assignments & shifts', searchText: 'assignment shift task job work', badge: 'OPS' },
    { value: 'payroll', label: 'Payroll', sublabel: 'Payroll runs & disbursements', searchText: 'payroll salary payment run', badge: 'FIN' },
    { value: 'wallet', label: 'Wallet', sublabel: 'Wallet transactions', searchText: 'wallet balance transaction credit debit', badge: 'FIN' },
    { value: 'organization', label: 'Organization', sublabel: 'Organization settings', searchText: 'organization company tenant sacco', badge: 'ORG' },
    { value: 'document', label: 'Document', sublabel: 'KYC & uploaded documents', searchText: 'document kyc id file upload', badge: 'DOC' },
    { value: 'loan', label: 'Loan', sublabel: 'Loan applications & disbursements', searchText: 'loan credit borrow advance', badge: 'FIN' },
    { value: 'insurance', label: 'Insurance', sublabel: 'Insurance policies', searchText: 'insurance policy cover premium', badge: 'INS' },
  ];

  ngOnInit() { 
    this.loadLogs();
    this.loadUsers(); 
  }

  loadUsers() {
    this.api.getUsers({ per_page: '1000' }).subscribe({
      next: r => {
        const users = r.data || [];
        const opts: AutocompleteOption[] = users.map(u => ({
          value: u.id,
          label: u.email || u.phone || u.id,
          sublabel: u.system_role,
          searchText: `${u.id} ${u.email || ''} ${u.phone || ''} ${u.system_role}`
        }));
        this.userOptions.set(opts);
      }
    });
  }

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
    const csv = ['Timestamp,Action,Entity,Entity ID,User,IP,User Agent,Old Value,New Value'];
    for (const log of this.logs()) {
      csv.push(`"${log.created_at}","${log.action}","${log.resource}","${log.resource_id}","${log.user_id || ''}","${log.ip_address || ''}","${log.user_agent || ''}","${(JSON.stringify(log.old_value) || '').replace(/"/g, '""')}","${(JSON.stringify(log.new_value) || '').replace(/"/g, '""')}"`);
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
