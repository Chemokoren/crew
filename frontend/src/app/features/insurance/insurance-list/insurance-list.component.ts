import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { InsurancePolicy, CrewMember } from '../../../core/models';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';

@Component({
  selector: 'app-insurance-list', standalone: true,
  imports: [CommonModule, FormsModule, CurrencyKesPipe, AutocompleteComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Insurance</h1><p class="page-subtitle">Manage insurance policies and premium deductions</p></div>
        @if (isAdmin()) {
          <button class="btn btn-primary" (click)="showCreateModal.set(true)" id="btn-create-policy">
            <span class="material-icons-round">add</span> Create Policy
          </button>
        }
      </div>

      <!-- Filters (Task 146) -->
      <div class="filters-bar">
        @if (isAdmin()) {
          <div style="position:relative; min-width: 220px; z-index: 50;">
            <app-autocomplete
              [(ngModel)]="filterCrewId"
              (ngModelChange)="applyFilters()"
              [options]="crewOptions()"
              placeholder="All Crew Members"
              inputId="filter-crew"
            ></app-autocomplete>
          </div>
        }
        <select class="form-select filter-select" [(ngModel)]="filterStatus" (ngModelChange)="applyFilters()" id="filter-status">
          <option value="">All Statuses</option>
          <option value="ACTIVE">Active</option>
          <option value="LAPSED">Lapsed</option>
          <option value="CLAIMED">Claimed</option>
        </select>
        <select class="form-select filter-select" [(ngModel)]="filterType" (ngModelChange)="applyFilters()" id="filter-type">
          <option value="">All Types</option>
          @for (t of policyTypes(); track t) { <option [value]="t">{{ t }}</option> }
        </select>
        @if (filterCrewId || filterStatus || filterType) {
          <button class="btn btn-ghost btn-sm" (click)="clearFilters()" style="color:var(--color-text-muted);">
            <span class="material-icons-round" style="font-size:16px;">close</span> Clear
          </button>
        }
      </div>

      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:56px;margin:4px 0;"></div>} }
      @else if (filtered().length === 0) {
        <div class="empty-state"><span class="material-icons-round empty-icon">health_and_safety</span>
          <div class="empty-title">No insurance policies</div>
          <div class="empty-description">
            @if (hasActiveFilters()) { No policies match the current filters. }
            @else { Create a policy to start managing insurance for crew members. }
          </div>
        </div>
      } @else {
        <!-- Summary Cards -->
        <div class="stats-grid" style="grid-template-columns:repeat(auto-fit,minmax(140px,1fr));margin-bottom:var(--space-lg);">
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">verified_user</span></div><div class="stat-value" style="color:var(--color-success);">{{ activeCount() }}</div><div class="stat-label">Active</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(239,68,68,0.12);color:#ef4444;"><span class="material-icons-round">gpp_bad</span></div><div class="stat-value" style="color:#ef4444;">{{ lapsedCount() }}</div><div class="stat-label">Lapsed</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(168,85,247,0.12);color:#a855f7;"><span class="material-icons-round">category</span></div><div class="stat-value">{{ policyTypes().length }}</div><div class="stat-label">Types</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">group</span></div><div class="stat-value" style="color:var(--color-accent);">{{ uniqueCrewCount() }}</div><div class="stat-label">Members</div></div>
        </div>

        <div class="data-table-wrapper"><table class="data-table"><thead><tr>
          <th>Provider</th><th>Type</th><th>Premium</th><th>Frequency</th><th>Period</th><th>Status</th>
          @if (isAdmin()) { <th>Actions</th> }
        </tr></thead><tbody>
          @for(p of filtered();track p.id){<tr>
            <td style="font-weight:500;color:var(--color-text-primary);">{{p.provider}}</td>
            <td><span class="badge badge-accent">{{p.policy_type}}</span></td>
            <td>{{p.premium_cents|currencyKes}}</td>
            <td style="text-transform:capitalize;">{{(p.frequency || '—').toLowerCase()}}</td>
            <td style="font-size:0.8125rem;">{{p.start_date|date:'mediumDate'}} — {{p.end_date|date:'mediumDate'}}</td>
            <td><span class="badge" [ngClass]="statusBadge(p.status)">{{p.status}}</span></td>
            @if (isAdmin()) {
              <td>
                @if (p.status === 'ACTIVE') {
                  <button class="btn btn-sm btn-danger" (click)="lapsePolicy(p)" [disabled]="lapsing()">
                    <span class="material-icons-round" style="font-size:14px;">cancel</span> Lapse
                  </button>
                }
              </td>
            }
          </tr>}</tbody></table></div>
      }

      <!-- Create Policy Modal (Task 144) -->
      @if (showCreateModal()) {
        <div class="modal-backdrop" (click)="showCreateModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Create Insurance Policy</h3><button class="btn btn-ghost btn-icon" (click)="showCreateModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group" style="position:relative;"><label class="form-label">Crew Member *</label>
              <app-autocomplete
                [(ngModel)]="createForm.crew_member_id"
                [options]="crewOptions()"
                placeholder="Search crew members..."
                inputId="create-crew"
              ></app-autocomplete>
            </div>
            <div class="form-row">
              <div class="form-group"><label class="form-label">Provider *</label>
                <select class="form-select" [(ngModel)]="createForm.provider" required>
                  <option value="">— Select Provider —</option>
                  @for (p of configuredProviders; track p) { <option [value]="p">{{ p }}</option> }
                </select>
              </div>
              <div class="form-group" style="position:relative;"><label class="form-label">Policy Type *</label>
                <app-autocomplete
                  [(ngModel)]="createForm.policy_type"
                  [options]="policyTypeOptions"
                  placeholder="Search policy types..."
                ></app-autocomplete>
              </div>
            </div>
            <div class="form-row">
              <div class="form-group"><label class="form-label">Premium (KES) *</label>
                <input class="form-input" type="number" [(ngModel)]="createForm.premium" min="1" placeholder="e.g. 500" required />
              </div>
              <div class="form-group"><label class="form-label">Frequency *</label>
                <select class="form-select" [(ngModel)]="createForm.frequency" required>
                  <option value="MONTHLY">Monthly</option>
                  <option value="QUARTERLY">Quarterly</option>
                  <option value="ANNUALLY">Annually</option>
                </select>
              </div>
            </div>
            <div class="form-row">
              <div class="form-group"><label class="form-label">Start Date *</label>
                <input class="form-input" type="date" [(ngModel)]="createForm.start_date" required />
              </div>
              <div class="form-group"><label class="form-label">End Date *</label>
                <input class="form-input" type="date" [(ngModel)]="createForm.end_date" required />
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" (click)="showCreateModal.set(false)">Cancel</button>
            <button class="btn btn-primary" (click)="submitCreate()" [disabled]="creating()">{{ creating() ? 'Creating...' : 'Create Policy' }}</button>
          </div>
        </div></div>
      }
    </div>`,
  styles: [`
    .filters-bar {
      display: flex; gap: var(--space-sm); flex-wrap: wrap; margin-bottom: var(--space-lg);
      align-items: center;
    }
    .filter-select { min-width: 160px; max-width: 220px; }
    .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); }
    @media (max-width: 640px) { .form-row { grid-template-columns: 1fr; } .filter-select { min-width: 100%; } }
  `],
})
export class InsuranceListComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);

  items = signal<InsurancePolicy[]>([]);
  crewMembers = signal<CrewMember[]>([]);
  loading = signal(true);
  showCreateModal = signal(false);
  creating = signal(false);
  lapsing = signal(false);

  filterCrewId = '';
  filterStatus = '';
  filterType = '';

  createForm = {
    crew_member_id: '', provider: '', policy_type: '',
    premium: 0, frequency: 'MONTHLY', start_date: '', end_date: '',
  };

  configuredProviders = [
    'Britam', 'NHIF', 'Jubilee Insurance', 'CIC Insurance', 
    'APA Insurance', 'Sanlam', 'Madison', 'ICEA Lion', 'Pioneer Assurance'
  ];

  policyTypeOptions: AutocompleteOption[] = [
    { value: 'HEALTH', label: 'Health', searchText: 'Health Medical' },
    { value: 'ACCIDENT', label: 'Accident', searchText: 'Accident Personal' },
    { value: 'LIFE', label: 'Life', searchText: 'Life Insurance' },
    { value: 'MOTOR', label: 'Motor', searchText: 'Motor Vehicle Auto' },
    { value: 'GROUP_LIFE', label: 'Group Life', searchText: 'Group Life' }
  ];

  crewOptions = computed<AutocompleteOption[]>(() => {
    return this.crewMembers().map(c => ({
      value: c.id,
      label: `${c.first_name} ${c.last_name}`,
      sublabel: `ID: ${c.crew_id || ''}`,
      badge: c.role,
      searchText: `${c.first_name} ${c.last_name} ${c.crew_id || ''}`
    }));
  });

  ngOnInit() {
    this.load();
    if (this.isAdmin()) {
      this.api.getCrewMembers({ per_page: '200' }).subscribe({ next: r => this.crewMembers.set(r.data) });
    }
  }

  isAdmin(): boolean { return this.auth.isAdmin(); }

  load(): void {
    this.loading.set(true);
    const params: Record<string, string> = {};
    if (this.filterCrewId) params['crew_member_id'] = this.filterCrewId;
    this.api.getInsurancePolicies(params).subscribe({
      next: r => { this.items.set(r.data); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  // --- Filters (Task 146) ---
  filtered = computed(() => {
    let list = this.items();
    if (this.filterStatus) list = list.filter(p => p.status === this.filterStatus);
    if (this.filterType) list = list.filter(p => p.policy_type === this.filterType);
    return list;
  });

  policyTypes = computed(() => [...new Set(this.items().map(p => p.policy_type))].sort());
  activeCount = computed(() => this.items().filter(p => p.status === 'ACTIVE').length);
  lapsedCount = computed(() => this.items().filter(p => p.status === 'LAPSED').length);
  uniqueCrewCount = computed(() => new Set(this.items().map(p => p.crew_member_id)).size);
  hasActiveFilters(): boolean { return !!(this.filterCrewId || this.filterStatus || this.filterType); }

  applyFilters(): void {
    // crew_member_id is server-side, status/type are client-side
    if (this.filterCrewId !== '') {
      this.load(); // re-fetch from server with crew filter
    }
  }

  clearFilters(): void {
    this.filterCrewId = '';
    this.filterStatus = '';
    this.filterType = '';
    this.load();
  }

  // --- Create Policy (Task 144) ---
  submitCreate(): void {
    const f = this.createForm;
    if (!f.crew_member_id || !f.provider || !f.policy_type || !f.premium || !f.start_date || !f.end_date) {
      this.toast.error('Please fill in all required fields');
      return;
    }
    this.creating.set(true);
    this.api.createInsurancePolicy({
      crew_member_id: f.crew_member_id,
      provider: f.provider,
      policy_type: f.policy_type,
      frequency: f.frequency,
      premium_cents: Math.round(f.premium * 100),
      start_date: f.start_date,
      end_date: f.end_date,
    }).subscribe({
      next: () => {
        this.toast.success('Insurance policy created');
        this.showCreateModal.set(false);
        this.creating.set(false);
        this.createForm = { crew_member_id: '', provider: '', policy_type: '', premium: 0, frequency: 'MONTHLY', start_date: '', end_date: '' };
        this.load();
      },
      error: () => this.creating.set(false),
    });
  }

  // --- Lapse Policy (Task 145) ---
  lapsePolicy(p: InsurancePolicy): void {
    if (!confirm(`Mark this ${p.policy_type} policy from ${p.provider} as lapsed?`)) return;
    this.lapsing.set(true);
    this.api.lapseInsurancePolicy(p.id).subscribe({
      next: () => {
        this.toast.success('Policy marked as lapsed');
        this.lapsing.set(false);
        this.load();
      },
      error: () => this.lapsing.set(false),
    });
  }

  statusBadge(s: string): string {
    return s === 'ACTIVE' ? 'badge-success' : s === 'CLAIMED' ? 'badge-info' : 'badge-danger';
  }
}
