import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { UssdManagementComponent } from '../ussd-management/ussd-management.component';
import { SystemStats, AuditLog, AdminUser, NotificationTemplate, StatutoryRate } from '../../../core/models';

type Tab = 'overview' | 'users' | 'audit' | 'templates' | 'rates' | 'ussd';

@Component({
  selector: 'app-admin-dashboard', standalone: true,
  imports: [CommonModule, FormsModule, CurrencyKesPipe, RelativeTimePipe, UssdManagementComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">System Administration</h1><p class="page-subtitle">Platform monitoring, user management, and audit logs</p></div>
      </div>

      <!-- Tab Nav -->
      <div class="tab-bar">
        @for (t of tabs; track t.key) {
          <button class="tab-btn" [class.active]="activeTab()===t.key" (click)="activeTab.set(t.key)">
            <span class="material-icons-round" style="font-size:18px;">{{t.icon}}</span> {{t.label}}
          </button>
        }
      </div>

      <!-- OVERVIEW TAB -->
      @if (activeTab()==='overview') {
        @if (stats(); as s) {
          <div class="stats-grid">
            <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">people</span></div><div class="stat-value">{{s.total_users}}</div><div class="stat-label">Total Users</div></div>
            <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">verified_user</span></div><div class="stat-value">{{s.active_users}}</div><div class="stat-label">Active Users</div></div>
            <div class="stat-card"><div class="stat-icon" style="background:var(--color-info-light);color:var(--color-info);"><span class="material-icons-round">groups</span></div><div class="stat-value">{{s.total_crew}}</div><div class="stat-label">Crew Members</div></div>
            <div class="stat-card"><div class="stat-icon" style="background:var(--color-warning-light);color:var(--color-warning);"><span class="material-icons-round">account_balance_wallet</span></div><div class="stat-value">{{s.total_wallet_balance_cents|currencyKes}}</div><div class="stat-label">Total Wallet Float</div></div>
          </div>
        }
      }

      <!-- USERS TAB (Tasks 154, 155, 156, 161) -->
      @if (activeTab()==='users') {
        <div class="section-header">
          <input class="form-input search-input" type="text" placeholder="Search by phone..." [(ngModel)]="userSearch" (ngModelChange)="filterUsers()" />
        </div>
        @if (usersLoading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:48px;margin:4px 0;"></div>} }
        @else {
          <div class="data-table-wrapper"><table class="data-table"><thead><tr>
            <th>Phone</th><th>Role</th><th>Status</th><th>Last Login</th><th>Created</th><th>Actions</th>
          </tr></thead><tbody>
            @for(u of filteredUsers();track u.id){<tr>
              <td style="font-weight:500;color:var(--color-text-primary);">{{u.phone}}</td>
              <td><span class="badge badge-accent">{{u.system_role}}</span></td>
              <td><span class="badge" [ngClass]="u.is_active?'badge-success':'badge-danger'">{{u.is_active?'Active':'Disabled'}}</span></td>
              <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{u.last_login_at ? (u.last_login_at|relativeTime) : 'Never'}}</td>
              <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{u.created_at|relativeTime}}</td>
              <td><div style="display:flex;gap:4px;flex-wrap:wrap;">
                @if (u.is_active) {
                  <button class="btn btn-sm btn-danger" (click)="toggleAccount(u)">Disable</button>
                } @else {
                  <button class="btn btn-sm btn-secondary" (click)="toggleAccount(u)">Enable</button>
                }
                <button class="btn btn-sm btn-ghost" style="color:var(--color-accent);" (click)="openResetModal(u)">Reset PW</button>
              </div></td>
            </tr>}</tbody></table></div>
        }
      }

      <!-- AUDIT LOG TAB (Tasks 159, 160) -->
      @if (activeTab()==='audit') {
        <div class="filters-bar">
          <input class="form-input filter-input" type="text" placeholder="Filter by resource..." [(ngModel)]="auditResource" />
          <input class="form-input filter-input" type="text" placeholder="Filter by actor ID..." [(ngModel)]="auditActor" />
          <input class="form-input filter-input" type="date" [(ngModel)]="auditDateFrom" placeholder="From" />
          <input class="form-input filter-input" type="date" [(ngModel)]="auditDateTo" placeholder="To" />
          <button class="btn btn-secondary btn-sm" (click)="loadAuditLogs()">Apply</button>
          @if (auditResource||auditActor||auditDateFrom||auditDateTo) {
            <button class="btn btn-ghost btn-sm" (click)="clearAuditFilters()" style="color:var(--color-text-muted);"><span class="material-icons-round" style="font-size:16px;">close</span> Clear</button>
          }
        </div>
        @if (logsLoading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:40px;margin:4px 0;"></div>} }
        @else if (logs().length===0) { <div class="empty-state"><span class="material-icons-round empty-icon">history</span><div class="empty-title">No audit logs found</div></div> }
        @else {
          <div class="data-table-wrapper"><table class="data-table"><thead><tr>
            <th>Action</th><th>Resource</th><th>Actor</th><th>Details</th><th>Time</th>
          </tr></thead><tbody>
            @for(l of logs();track l.id){<tr>
              <td style="font-weight:500;color:var(--color-text-primary);">{{l.action}}</td>
              <td><span class="badge badge-neutral">{{l.resource}}</span></td>
              <td style="font-size:0.75rem;color:var(--color-text-muted);font-family:monospace;">{{(l.user_id||'').slice(0,8)}}...</td>
              <td style="font-size:0.8125rem;max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{(l.new_value ? (l.new_value | json) : l.old_value ? (l.old_value | json) : '—')}}</td>
              <td style="font-size:0.8125rem;color:var(--color-text-muted);white-space:nowrap;">{{l.created_at|relativeTime}}</td>
            </tr>}</tbody></table></div>
          <div class="pagination">
            <button class="btn btn-sm btn-secondary" [disabled]="auditPage<=1" (click)="auditPage=auditPage-1;loadAuditLogs()">← Prev</button>
            <span class="page-info">Page {{auditPage}}</span>
            <button class="btn btn-sm btn-secondary" [disabled]="logs().length<20" (click)="auditPage=auditPage+1;loadAuditLogs()">Next →</button>
          </div>
        }
      }

      <!-- TEMPLATES TAB (Task 158) -->
      @if (activeTab()==='templates') {
        <div class="section-header">
          <button class="btn btn-primary" (click)="openTemplateModal()"><span class="material-icons-round">add</span> Create Template</button>
        </div>
        @if (templatesLoading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:48px;margin:4px 0;"></div>} }
        @else if (templates().length===0) { <div class="empty-state"><span class="material-icons-round empty-icon">description</span><div class="empty-title">No templates</div></div> }
        @else {
          <div class="data-table-wrapper"><table class="data-table"><thead><tr>
            <th>Event</th><th>Channel</th><th>Title</th><th>Active</th><th>Actions</th>
          </tr></thead><tbody>
            @for(t of templates();track t.id){<tr>
              <td style="font-weight:500;color:var(--color-text-primary);">{{t.event_name}}</td>
              <td><span class="badge" [ngClass]="t.channel==='SMS'?'badge-success':t.channel==='PUSH'?'badge-info':'badge-accent'">{{t.channel}}</span></td>
              <td style="font-size:0.8125rem;">{{t.title_template}}</td>
              <td><span class="badge" [ngClass]="t.is_active?'badge-success':'badge-danger'">{{t.is_active?'Yes':'No'}}</span></td>
              <td><button class="btn btn-sm btn-ghost" style="color:var(--color-accent);" (click)="editTemplate(t)">Edit</button></td>
            </tr>}</tbody></table></div>
        }
      }

      <!-- STATUTORY RATES TAB (Task 157) -->
      @if (activeTab()==='rates') {
        @if (ratesLoading()) { @for(i of [1,2];track i){<div class="skeleton" style="height:60px;margin:4px 0;"></div>} }
        @else if (rates().length===0) { <div class="empty-state"><span class="material-icons-round empty-icon">calculate</span><div class="empty-title">No statutory rates</div></div> }
        @else {
          <div class="rates-grid">
            @for(r of rates();track r.id){
              <div class="glass-card rate-card">
                <div class="rate-name">{{r.name}}</div>
                <div class="rate-value">{{r.rate * 100 | number:'1.1-2'}}%</div>
                <div class="rate-meta">Effective: {{r.effective_from|date:'mediumDate'}}</div>
              </div>
            }
          </div>
        }
      }

      <!-- USSD TAB -->
      @if (activeTab()==='ussd') {
        <app-ussd-management />
      }

      <!-- Reset Password Modal -->
      @if (showResetModal()) {
        <div class="modal-backdrop" (click)="showResetModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Reset Password</h3><button class="btn btn-ghost btn-icon" (click)="showResetModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <p style="font-size:0.8125rem;color:var(--color-text-muted);margin-bottom:var(--space-md);">Resetting password for <strong>{{selectedUser()?.phone}}</strong></p>
            <div class="form-group"><label class="form-label">New Password (min 8 chars)</label>
              <input class="form-input" type="password" [(ngModel)]="newPassword" minlength="8" placeholder="Enter new password" />
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" (click)="showResetModal.set(false)">Cancel</button>
            <button class="btn btn-primary" (click)="submitReset()" [disabled]="resetting()||newPassword.length<8">{{resetting()?'Resetting...':'Reset Password'}}</button>
          </div>
        </div></div>
      }

      <!-- Template Modal -->
      @if (showTemplateModal()) {
        <div class="modal-backdrop" (click)="showTemplateModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>{{editingTemplate()?'Edit':'Create'}} Template</h3><button class="btn btn-ghost btn-icon" (click)="showTemplateModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group"><label class="form-label">Event Name *</label><input class="form-input" [(ngModel)]="tplForm.event_name" placeholder="e.g. LOAN_APPROVED" /></div>
            <div class="form-row">
              <div class="form-group"><label class="form-label">Channel *</label>
                <select class="form-select" [(ngModel)]="tplForm.channel"><option value="SMS">SMS</option><option value="PUSH">Push</option><option value="IN_APP">In-App</option></select>
              </div>
              <div class="form-group"><label class="form-label">Active</label>
                <select class="form-select" [(ngModel)]="tplForm.is_active"><option [ngValue]="true">Yes</option><option [ngValue]="false">No</option></select>
              </div>
            </div>
            <div class="form-group"><label class="form-label">Title Template *</label><input class="form-input" [(ngModel)]="tplForm.title_template" placeholder="e.g. Loan approved" /></div>
            <div class="form-group"><label class="form-label">Body Template *</label><textarea class="form-textarea" [(ngModel)]="tplForm.body_template" rows="3" placeholder="Use template variables like {name}, {amount}"></textarea></div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" (click)="showTemplateModal.set(false)">Cancel</button>
            <button class="btn btn-primary" (click)="submitTemplate()" [disabled]="savingTpl()">{{savingTpl()?'Saving...':'Save Template'}}</button>
          </div>
        </div></div>
      }
    </div>`,
  styles: [`
    .tab-bar{display:flex;gap:2px;background:var(--color-surface-alt);border-radius:var(--radius-md);padding:3px;margin-bottom:var(--space-lg);overflow-x:auto;}
    .tab-btn{padding:8px 16px;border:none;background:none;border-radius:var(--radius-sm);font-size:0.8125rem;font-weight:500;color:var(--color-text-muted);cursor:pointer;transition:all var(--transition-fast);display:flex;align-items:center;gap:6px;white-space:nowrap;}
    .tab-btn:hover{color:var(--color-text-primary);}
    .tab-btn.active{background:var(--color-accent);color:#fff;}
    .section-header{display:flex;justify-content:space-between;align-items:center;margin-bottom:var(--space-md);}
    .search-input{max-width:300px;}
    .filters-bar{display:flex;gap:var(--space-sm);flex-wrap:wrap;margin-bottom:var(--space-lg);align-items:center;}
    .filter-input{max-width:180px;}
    .pagination{display:flex;align-items:center;justify-content:center;gap:var(--space-md);margin-top:var(--space-md);}
    .page-info{font-size:0.8125rem;color:var(--color-text-muted);}
    .rates-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:var(--space-md);}
    .rate-card{text-align:center;padding:var(--space-lg)!important;}
    .rate-name{font-size:0.875rem;font-weight:600;color:var(--color-text-primary);margin-bottom:var(--space-xs);}
    .rate-value{font-size:1.5rem;font-weight:800;background:var(--gradient-accent);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;}
    .rate-meta{font-size:0.7rem;color:var(--color-text-muted);margin-top:var(--space-xs);}
    .form-row{display:grid;grid-template-columns:1fr 1fr;gap:var(--space-md);}

    @media(max-width:640px){.filter-input{max-width:100%;}.form-row{grid-template-columns:1fr;}}
  `]
})
export class AdminDashboardComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  activeTab = signal<Tab>('overview');
  stats = signal<SystemStats | null>(null);
  users = signal<AdminUser[]>([]); usersLoading = signal(false);
  logs = signal<AuditLog[]>([]); logsLoading = signal(true);
  templates = signal<NotificationTemplate[]>([]); templatesLoading = signal(false);
  rates = signal<StatutoryRate[]>([]); ratesLoading = signal(false);

  showResetModal = signal(false); selectedUser = signal<AdminUser | null>(null);
  resetting = signal(false); newPassword = '';
  showTemplateModal = signal(false); editingTemplate = signal(false); savingTpl = signal(false);

  userSearch = '';
  auditResource = ''; auditActor = ''; auditDateFrom = ''; auditDateTo = ''; auditPage = 1;

  tplForm = { id: '', event_name: '', channel: 'SMS', title_template: '', body_template: '', is_active: true };



  readonly tabs: { key: Tab; label: string; icon: string }[] = [
    { key: 'overview', label: 'Overview', icon: 'dashboard' },
    { key: 'users', label: 'Users', icon: 'people' },
    { key: 'audit', label: 'Audit Logs', icon: 'history' },
    { key: 'templates', label: 'Templates', icon: 'description' },
    { key: 'rates', label: 'Statutory Rates', icon: 'calculate' },
    { key: 'ussd', label: 'USSD', icon: 'sim_card' },
  ];

  ngOnInit() {
    this.api.getSystemStats().subscribe({ next: r => this.stats.set(r.data) });
    this.loadAuditLogs();
    this.loadUsers();
    this.loadTemplates();
    this.loadRates();
  }

  // --- Users (Tasks 154, 155, 156, 161) ---
  loadUsers() {
    this.usersLoading.set(true);
    this.api.getUsers({ per_page: '100' }).subscribe({
      next: r => { this.users.set(r.data); this.usersLoading.set(false); },
      error: () => this.usersLoading.set(false),
    });
  }

  filteredUsers = computed(() => {
    const q = this.userSearch.toLowerCase();
    return q ? this.users().filter(u => u.phone.includes(q)) : this.users();
  });

  filterUsers() { /* triggers computed */ }

  toggleAccount(u: AdminUser) {
    const action = u.is_active ? 'disable' : 'enable';
    if (!confirm(`${action.charAt(0).toUpperCase() + action.slice(1)} account for ${u.phone}?`)) return;
    const obs = u.is_active ? this.api.disableAccount(u.id) : this.api.enableAccount(u.id);
    obs.subscribe({ next: () => { this.toast.success(`Account ${action}d`); this.loadUsers(); } });
  }

  openResetModal(u: AdminUser) { this.selectedUser.set(u); this.newPassword = ''; this.showResetModal.set(true); }

  submitReset() {
    const u = this.selectedUser();
    if (!u || this.newPassword.length < 8) return;
    this.resetting.set(true);
    this.api.resetPassword(u.id, this.newPassword).subscribe({
      next: () => { this.toast.success('Password reset'); this.showResetModal.set(false); this.resetting.set(false); },
      error: () => this.resetting.set(false),
    });
  }

  // --- Audit Logs (Tasks 159, 160) ---
  loadAuditLogs() {
    this.logsLoading.set(true);
    const params: Record<string, string> = { page: this.auditPage.toString(), per_page: '20' };
    if (this.auditResource) params['resource'] = this.auditResource;
    this.api.getAuditLogs(params).subscribe({
      next: r => { this.logs.set(r.data); this.logsLoading.set(false); },
      error: () => this.logsLoading.set(false),
    });
  }

  clearAuditFilters() {
    this.auditResource = ''; this.auditActor = ''; this.auditDateFrom = ''; this.auditDateTo = ''; this.auditPage = 1;
    this.loadAuditLogs();
  }

  // --- Templates (Task 158) ---
  loadTemplates() {
    this.templatesLoading.set(true);
    this.api.getNotificationTemplates().subscribe({
      next: r => { this.templates.set(r.data || []); this.templatesLoading.set(false); },
      error: () => this.templatesLoading.set(false),
    });
  }

  openTemplateModal() {
    this.tplForm = { id: '', event_name: '', channel: 'SMS', title_template: '', body_template: '', is_active: true };
    this.editingTemplate.set(false); this.showTemplateModal.set(true);
  }

  editTemplate(t: NotificationTemplate) {
    this.tplForm = { ...t };
    this.editingTemplate.set(true); this.showTemplateModal.set(true);
  }

  submitTemplate() {
    this.savingTpl.set(true);
    const obs = this.editingTemplate()
      ? this.api.updateNotificationTemplate(this.tplForm)
      : this.api.createNotificationTemplate(this.tplForm);
    obs.subscribe({
      next: () => { this.toast.success(`Template ${this.editingTemplate() ? 'updated' : 'created'}`); this.showTemplateModal.set(false); this.savingTpl.set(false); this.loadTemplates(); },
      error: () => this.savingTpl.set(false),
    });
  }

  // --- Statutory Rates (Task 157) ---
  loadRates() {
    this.ratesLoading.set(true);
    this.api.getStatutoryRates().subscribe({
      next: r => { this.rates.set(r.data || []); this.ratesLoading.set(false); },
      error: () => this.ratesLoading.set(false),
    });
  }


}
