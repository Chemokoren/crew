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

  actionOptions: AutocompleteOption[] = [];

  private loadActionOptions() {
    // Build action options dynamically from actual data
    // Start with known action types from the database, plus common CRUD actions
    this.actionOptions = [
      { value: 'permission.denied', label: 'Access Denied', sublabel: 'Permission denied events', searchText: 'denied permission block access forbidden' },
      { value: 'INITIATE_TOPUP', label: 'Initiate Top-Up', sublabel: 'Float top-up initiation events', searchText: 'topup credit float deposit stk mpesa' },
      { value: 'CREDIT', label: 'Credit', sublabel: 'Wallet credit events', searchText: 'credit add fund' },
      { value: 'CREDIT_FLOAT', label: 'Credit Float', sublabel: 'Organization float credit events', searchText: 'credit float org sacco fund' },
      { value: 'DEBIT', label: 'Debit', sublabel: 'Wallet debit events', searchText: 'debit withdraw' },
      { value: 'PAYOUT_REVERSED', label: 'Payout Reversed', sublabel: 'Reversed payout events', searchText: 'payout reverse refund' },
      { value: 'role.assigned', label: 'Role Assigned', sublabel: 'User role assignment events', searchText: 'role assign rbac permission' },
      { value: 'CREATE', label: 'Create', sublabel: 'Resource creation events', searchText: 'create add new' },
      { value: 'UPDATE', label: 'Update', sublabel: 'Resource modification events', searchText: 'update edit modify change' },
      { value: 'DELETE', label: 'Delete', sublabel: 'Resource deletion events', searchText: 'delete remove destroy' },
      { value: 'LOGIN', label: 'Login', sublabel: 'User authentication events', searchText: 'login sign in auth' },
      { value: 'LOGOUT', label: 'Logout', sublabel: 'User sign-out events', searchText: 'logout sign out' },
      { value: 'APPROVE', label: 'Approve', sublabel: 'Approval workflow events', searchText: 'approve accept confirm' },
      { value: 'REJECT', label: 'Reject', sublabel: 'Rejection workflow events', searchText: 'reject deny decline' },
      { value: 'EXPORT', label: 'Export', sublabel: 'Data export events', searchText: 'export download csv' },
    ];
  }

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
    this.loadActionOptions();
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
    
    const cleanJson = (val: any) => {
      if (!val || val === 'null') return '';
      if (typeof val === 'string') {
        try { return JSON.stringify(JSON.parse(val)).replace(/"/g, '""'); } catch { return val.replace(/"/g, '""'); }
      }
      return JSON.stringify(val).replace(/"/g, '""');
    };

    for (const log of this.logs()) {
      const entityId = log.resource_id || '';
      const user = this.getUserName(log.user_id).replace(/"/g, '""');
      const ip = this.getIpAddress(log).replace(/"/g, '""');
      const ua = this.getUserAgent(log).replace(/"/g, '""');
      const oldVal = cleanJson(log.old_value);
      const newVal = cleanJson(log.new_value);
      
      csv.push(`"${log.created_at}","${log.action}","${log.resource}","${entityId}","${user}","${ip}","${ua}","${oldVal}","${newVal}"`);
    }
    
    const blob = new Blob([csv.join('\n')], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = `audit-logs-${new Date().toISOString().split('T')[0]}.csv`;
    a.click(); URL.revokeObjectURL(url);
    this.toast.success('Audit logs exported');
  }

  actionColor(action: string): string {
    const a = (action || '').toLowerCase();
    if (a.includes('create')) return '#10b981';
    if (a.includes('update')) return '#6366f1';
    if (a.includes('delete')) return '#ef4444';
    if (a.includes('login')) return '#3b82f6';
    if (a.includes('approve')) return '#10b981';
    if (a.includes('reject')) return '#ef4444';
    if (a.includes('denied')) return '#f59e0b';
    if (a.includes('topup')) return '#06b6d4';
    if (a.includes('credit')) return '#10b981';
    if (a.includes('debit')) return '#f97316';
    if (a.includes('payout')) return '#8b5cf6';
    if (a.includes('role')) return '#6366f1';
    return '#8b5cf6';
  }
}
