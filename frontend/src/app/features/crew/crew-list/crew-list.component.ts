import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink, Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { AuthService } from '../../../core/services/auth.service';
import { OrgContextService } from '../../../core/services/org-context.service';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';
import { CrewMember, PaginationMeta, Organization, TenantJobType } from '../../../core/models';

@Component({
  selector: 'app-crew-list',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, AutocompleteComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">{{ orgCtx.workersLabel() }} Management</h1>
          <p class="page-subtitle">Manage your workforce — {{ templateRoleSummary() }}</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-secondary btn-sm" (click)="showBulkModal.set(true)" id="btn-bulk-import">
            <span class="material-icons-round">upload_file</span> Bulk Import
          </button>
          <button class="btn btn-primary" (click)="showCreateModal.set(true)" id="btn-add-crew">
            <span class="material-icons-round">person_add</span> Add {{ orgCtx.workerLabel() }}
          </button>
        </div>
      </div>

      <div class="filters-bar">
        <div class="search-input-wrapper">
          <span class="material-icons-round search-icon">search</span>
          <input class="form-input" placeholder="Search by name..." [(ngModel)]="searchQuery" (ngModelChange)="loadCrew()" id="crew-search" />
        </div>
        <!-- #68: Search by National ID -->
        <div class="search-input-wrapper">
          <span class="material-icons-round search-icon">badge</span>
          <input class="form-input" placeholder="Search National ID..." [(ngModel)]="nationalIdQuery" (keydown.enter)="searchNationalID()" id="crew-nid-search" />
          @if (nationalIdQuery) {
            <button class="nid-search-btn" (click)="searchNationalID()" [disabled]="searchingNID()" title="Search">
              <span class="material-icons-round">{{ searchingNID() ? 'hourglass_empty' : 'search' }}</span>
            </button>
          }
        </div>
        <div style="position: relative; z-index: 55; flex: 1; min-width: 140px; max-width: 180px;">
          <app-autocomplete [(ngModel)]="roleFilter" (ngModelChange)="loadCrew()" [options]="activeRoleOptions()" placeholder="— All Roles —" id="crew-role-filter"></app-autocomplete>
        </div>
        <div style="position: relative; z-index: 54; flex: 1; min-width: 140px; max-width: 180px;">
          <app-autocomplete [(ngModel)]="kycFilter" (ngModelChange)="loadCrew()" [options]="kycOptions" placeholder="— All KYC —" id="crew-kyc-filter"></app-autocomplete>
        </div>
        <!-- #71: Filter by Organization -->
        <div style="position: relative; z-index: 53; flex: 1; min-width: 180px; max-width: 240px;">
          <app-autocomplete [(ngModel)]="saccoFilter" (ngModelChange)="loadCrew()" [options]="saccoOptions()" placeholder="— All Organizations —" id="crew-sacco-filter"></app-autocomplete>
        </div>
      </div>

      @if (loading()) {
        <div class="data-table-wrapper">
          @for (i of [1,2,3,4,5]; track i) {
            <div class="skeleton" style="height: 56px; margin: 4px 0;"></div>
          }
        </div>
      } @else if (crewMembers().length === 0) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">groups</span>
          <div class="empty-title">No {{ orgCtx.workersLabel().toLowerCase() }} found</div>
          <div class="empty-description">Add your first {{ orgCtx.workerLabel().toLowerCase() }} to get started with workforce management.</div>
        </div>
      } @else {
        <div class="data-table-wrapper">
          <table class="data-table">
            <thead>
              <tr>
                <th>{{ orgCtx.workerLabel() }} ID</th>
                <th>Name</th>
                <th>Role</th>
                <th>KYC Status</th>
                <th>Status</th>
                <th>Joined</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              @for (member of crewMembers(); track member.id) {
                <tr>
                  <td><code style="color: var(--color-accent); font-size: 0.8rem;">{{ member.crew_id }}</code></td>
                  <td>
                    <div style="display: flex; align-items: center; gap: 8px;">
                      <div class="avatar-sm">{{ member.first_name.charAt(0) }}{{ member.last_name.charAt(0) }}</div>
                      <span style="color: var(--color-text-primary); font-weight: 500;">{{ member.full_name }}</span>
                    </div>
                  </td>
                  <td><span class="badge badge-accent">{{ member.role }}</span></td>
                  <td>
                    <span class="badge" [ngClass]="kycBadgeClass(member.kyc_status)">{{ member.kyc_status }}</span>
                  </td>
                  <td>
                    <span class="badge" [ngClass]="member.is_active ? 'badge-success' : 'badge-danger'">
                      {{ member.is_active ? 'Active' : 'Inactive' }}
                    </span>
                  </td>
                  <td style="color: var(--color-text-muted); font-size: 0.8125rem;">{{ member.created_at | date:'mediumDate' }}</td>
                  <td>
                    <div style="display: flex; gap: 4px;">
                      <a [routerLink]="['/crew', member.id]" class="btn btn-ghost btn-sm" id="view-crew-{{member.id}}">View</a>
                    </div>
                  </td>
                </tr>
              }
            </tbody>
          </table>
        </div>

        @if (meta()) {
          <div class="pagination">
            <button class="page-btn" [disabled]="currentPage() <= 1" (click)="goToPage(currentPage() - 1)">
              <span class="material-icons-round" style="font-size:16px;">chevron_left</span>
            </button>
            @for (p of pages(); track p) {
              <button class="page-btn" [class.active]="p === currentPage()" (click)="goToPage(p)">{{ p }}</button>
            }
            <button class="page-btn" [disabled]="currentPage() >= (meta()?.total_pages ?? 1)" (click)="goToPage(currentPage() + 1)">
              <span class="material-icons-round" style="font-size:16px;">chevron_right</span>
            </button>
          </div>
        }
      }

      <!-- Create Worker Modal -->
      @if (showCreateModal()) {
        <div class="modal-backdrop" (click)="showCreateModal.set(false)">
          <div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>Add {{ orgCtx.workerLabel() }}</h3>
              <button class="btn btn-ghost btn-icon" (click)="showCreateModal.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <form class="modal-body" (ngSubmit)="createCrew()" id="create-crew-form">
              <div class="form-group">
                <label class="form-label">National ID</label>
                <input class="form-input" [(ngModel)]="newCrew.national_id" name="nationalId" required placeholder="12345678" />
              </div>
              <div class="form-group">
                <label class="form-label">First Name</label>
                <input class="form-input" [(ngModel)]="newCrew.first_name" name="firstName" required placeholder="John" />
              </div>
              <div class="form-group">
                <label class="form-label">Last Name</label>
                <input class="form-input" [(ngModel)]="newCrew.last_name" name="lastName" required placeholder="Doe" />
              </div>
              <div class="form-group" style="position: relative; z-index: 60;">
                <label class="form-label">Role / Job Type</label>
                <app-autocomplete [(ngModel)]="newCrew.role" [options]="dynamicRoleOptions()" placeholder="Search role..." name="role" id="create-crew-role"></app-autocomplete>
              </div>
            </form>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="showCreateModal.set(false)">Cancel</button>
              <button class="btn btn-primary" (click)="createCrew()" [disabled]="creating()" id="submit-create-crew">
                @if (creating()) { Creating... } @else { Add {{ orgCtx.workerLabel() }} }
              </button>
            </div>
          </div>
        </div>
      }

      <!-- #69: Bulk Import Modal -->
      @if (showBulkModal()) {
        <div class="modal-backdrop" (click)="showBulkModal.set(false)">
          <div class="modal-content modal-lg" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>Bulk Import {{ orgCtx.workersLabel() }}</h3>
              <button class="btn btn-ghost btn-icon" (click)="closeBulkModal()">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <div class="modal-body">
              <p class="text-muted" style="margin-bottom: var(--space-sm);">
                Upload a CSV file or paste data below. Each row: <code>national_id, first_name, last_name, role</code>
              </p>
              <button class="template-download-btn" (click)="downloadBulkTemplate()" id="btn-download-template">
                <span class="material-icons-round" style="font-size:16px;">download</span>
                Download CSV Template (5 sample rows)
              </button>

              <!-- File upload -->
              <div class="upload-zone" (click)="fileInput.click()" (dragover)="$event.preventDefault()" (drop)="onFileDrop($event)">
                <span class="material-icons-round upload-icon">cloud_upload</span>
                <span class="upload-label">Click to upload CSV or drag & drop</span>
                <span class="upload-hint">CSV format: national_id, first_name, last_name, role</span>
                <input #fileInput type="file" accept=".csv" style="display:none" (change)="onFileSelect($event)" />
              </div>

              @if (bulkFileName) {
                <div class="file-chip">
                  <span class="material-icons-round" style="font-size:16px;">description</span>
                  {{ bulkFileName }} ({{ bulkRows().length }} rows)
                  <button class="btn btn-ghost btn-icon btn-xs" (click)="clearBulkFile()">
                    <span class="material-icons-round" style="font-size:14px;">close</span>
                  </button>
                </div>
              }

              <div class="divider-text"><span>or paste manually</span></div>

              <!-- Manual entry -->
              <textarea class="form-input bulk-textarea"
                placeholder="12345678, John, Doe, DRIVER&#10;87654321, Jane, Smith, CONDUCTOR"
                [(ngModel)]="bulkTextInput"
                (ngModelChange)="parseBulkText()"
                rows="4"></textarea>

              @if (bulkRows().length > 0) {
                <div class="bulk-preview">
                  <h4 style="margin-bottom:8px;font-size:0.875rem;color:var(--color-text-secondary);">Preview ({{ bulkRows().length }} members)</h4>
                  <div class="data-table-wrapper" style="max-height:200px;overflow:auto;">
                    <table class="data-table">
                      <thead><tr><th>National ID</th><th>First Name</th><th>Last Name</th><th>Role</th></tr></thead>
                      <tbody>
                        @for (row of bulkRows(); track $index) {
                          <tr>
                            <td>{{ row.national_id }}</td>
                            <td>{{ row.first_name }}</td>
                            <td>{{ row.last_name }}</td>
                            <td><span class="badge badge-accent">{{ row.role }}</span></td>
                          </tr>
                        }
                      </tbody>
                    </table>
                  </div>
                </div>
              }
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="closeBulkModal()">Cancel</button>
              <button class="btn btn-primary" (click)="submitBulkImport()" [disabled]="bulkImporting() || bulkRows().length === 0" id="submit-bulk-import">
                @if (bulkImporting()) { Importing... } @else {
                  <span class="material-icons-round">upload</span> Import {{ bulkRows().length }} {{ orgCtx.workersLabel() }}
                }
              </button>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .avatar-sm {
      width: 32px; height: 32px; border-radius: var(--radius-md);
      background: var(--gradient-accent); display: flex; align-items: center; justify-content: center;
      font-size: 0.6875rem; font-weight: 700; color: var(--color-text-inverse); flex-shrink: 0;
    }
    .nid-search-btn {
      position: absolute; right: 8px; top: 50%; transform: translateY(-50%);
      background: var(--color-accent); color: var(--color-bg); border: none;
      border-radius: var(--radius-sm); width: 28px; height: 28px;
      display: flex; align-items: center; justify-content: center; cursor: pointer;
      .material-icons-round { font-size: 16px; }
    }
    .search-input-wrapper { position: relative; }
    .upload-zone {
      border: 2px dashed var(--color-border); border-radius: var(--radius-lg);
      padding: var(--space-xl); text-align: center; cursor: pointer;
      transition: border-color 200ms, background 200ms;
      display: flex; flex-direction: column; align-items: center; gap: var(--space-xs);
      &:hover { border-color: var(--color-accent); background: rgba(0,210,255,0.04); }
    }
    .upload-icon { font-size: 36px; color: var(--color-accent); }
    .upload-label { font-size: 0.875rem; font-weight: 500; color: var(--color-text-primary); }
    .upload-hint { font-size: 0.75rem; color: var(--color-text-muted); }
    .file-chip {
      display: inline-flex; align-items: center; gap: 6px; margin-top: var(--space-sm);
      padding: 4px 12px; border-radius: var(--radius-full);
      background: var(--color-accent-10); color: var(--color-accent); font-size: 0.8125rem;
    }
    .divider-text {
      display: flex; align-items: center; gap: var(--space-md); margin: var(--space-md) 0;
      &::before, &::after { content: ''; flex: 1; border-top: 1px solid var(--color-border); }
      span { font-size: 0.75rem; color: var(--color-text-muted); }
    }
    .bulk-textarea { font-family: var(--font-mono, monospace); font-size: 0.8125rem; }
    .bulk-preview { margin-top: var(--space-md); }
    .modal-lg { max-width: 700px; }
    .btn-xs { padding: 0; width: 20px; height: 20px; }
    .template-download-btn {
      display: inline-flex; align-items: center; gap: 6px;
      padding: 6px 14px; margin-bottom: var(--space-md);
      border-radius: var(--radius-md); border: 1px dashed var(--color-accent);
      background: rgba(0,210,255,0.06); color: var(--color-accent);
      font-size: 0.8125rem; font-weight: 500; cursor: pointer;
      transition: background 200ms, border-color 200ms;
      &:hover { background: rgba(0,210,255,0.12); border-style: solid; }
    }
  `]
})
export class CrewListComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);
  private router = inject(Router);
  readonly orgCtx = inject(OrgContextService);

  crewMembers = signal<CrewMember[]>([]);
  meta = signal<PaginationMeta | null>(null);
  loading = signal(true);
  currentPage = signal(1);
  searchQuery = '';
  nationalIdQuery = '';
  roleFilter = '';
  kycFilter = '';
  saccoFilter = '';
  showCreateModal = signal(false);
  creating = signal(false);
  newCrew = { national_id: '', first_name: '', last_name: '', role: '' };

  // #68: NID search
  searchingNID = signal(false);

  // #69: Bulk import
  showBulkModal = signal(false);
  bulkImporting = signal(false);
  bulkRows = signal<Array<{ national_id: string; first_name: string; last_name: string; role: string }>>([]);
  bulkTextInput = '';
  bulkFileName = '';

  // #71: Organization filter
  saccos = signal<Organization[]>([]);
  saccoOptions = computed<AutocompleteOption[]>(() => this.saccos().map(s => ({
    value: s.id,
    label: s.name,
    searchText: s.name
  })));

  /** Fallback roles — only used when no tenant job types are loaded */
  private fallbackRoles: AutocompleteOption[] = [
    { value: 'DRIVER', label: 'Driver', searchText: 'driver' },
    { value: 'CONDUCTOR', label: 'Conductor', searchText: 'conductor' },
    { value: 'RIDER', label: 'Rider', searchText: 'rider' },
    { value: 'OTHER', label: 'Other', searchText: 'other' },
  ];

  /** Active role filter options — derived from tenant job types or industry template defaults */
  activeRoleOptions = computed<AutocompleteOption[]>(() => {
    const custom = this.tenantJobTypes();
    if (custom.length > 0) {
      return custom.map(jt => ({ value: jt.code, label: jt.display_name, searchText: `${jt.display_name} ${jt.code} ${jt.category}` }));
    }
    // Fall back to industry template defaults
    const tmpl = this.orgCtx.template();
    if (tmpl.default_job_types?.length) {
      return tmpl.default_job_types.map(jt => ({ value: jt.code, label: jt.display_name, searchText: `${jt.display_name} ${jt.code} ${jt.category}` }));
    }
    return this.fallbackRoles;
  });

  /** Summary of role types for the subtitle */
  templateRoleSummary = computed(() => {
    const tmpl = this.orgCtx.template();
    const names = tmpl.default_job_types.slice(0, 3).map(j => j.display_name.toLowerCase());
    if (names.length === 0) return 'your workforce';
    return names.join(', ') + (tmpl.default_job_types.length > 3 ? ' & more' : '');
  });

  kycOptions: AutocompleteOption[] = [
    { value: 'PENDING', label: 'Pending', searchText: 'pending' },
    { value: 'VERIFIED', label: 'Verified', searchText: 'verified' },
    { value: 'REJECTED', label: 'Rejected', searchText: 'rejected' },
  ];

  ngOnInit(): void {
    // Set default role from industry template
    const tmpl = this.orgCtx.template();
    if (tmpl.default_job_types?.length) {
      this.newCrew.role = tmpl.default_job_types[0].code;
    } else {
      this.newCrew.role = 'OTHER';
    }
    this.loadCrew();
    this.loadOrganizations();
    // Load job types scoped to the user's own org
    const userOrgId = this.auth.currentUser()?.organization_id;
    this.loadJobTypes(userOrgId);
  }

  loadCrew(): void {
    this.loading.set(true);
    const params: Record<string, string> = { page: this.currentPage().toString(), per_page: '20' };
    if (this.searchQuery) params['search'] = this.searchQuery;
    if (this.roleFilter) params['role'] = this.roleFilter;
    if (this.kycFilter) params['kyc_status'] = this.kycFilter;
    if (this.saccoFilter) params['organization_id'] = this.saccoFilter;

    this.api.getCrewMembers(params).subscribe({
      next: (res) => {
        this.crewMembers.set(res.data);
        this.meta.set(res.meta);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  loadOrganizations(): void {
    this.api.getOrganizations({ per_page: '100' }).subscribe({
      next: (res) => {
        this.saccos.set(res.data ?? []);
      },
    });
  }

  // F8: Dynamic job type loading
  tenantJobTypes = signal<TenantJobType[]>([]);
  dynamicRoleOptions = computed<AutocompleteOption[]>(() => {
    const custom = this.tenantJobTypes();
    if (custom.length > 0) {
      return custom.map(jt => ({ value: jt.code, label: jt.display_name, searchText: `${jt.display_name} ${jt.code} ${jt.category}` }));
    }
    // Fall back to industry template defaults
    const tmpl = this.orgCtx.template();
    if (tmpl.default_job_types?.length) {
      return tmpl.default_job_types.map(jt => ({ value: jt.code, label: jt.display_name, searchText: `${jt.display_name} ${jt.code} ${jt.category}` }));
    }
    return this.fallbackRoles;
  });

  private loadJobTypes(saccoId?: string): void {
    if (!saccoId) {
      // No org ID available — tenant job types won't load,
      // but dynamicRoleOptions falls back to industry template defaults.
      return;
    }
    this.fetchJobTypes(saccoId);
  }

  private fetchJobTypes(saccoId: string): void {
    this.api.getJobTypes(saccoId).subscribe({
      next: r => this.tenantJobTypes.set(r.data || []),
    });
  }

  // #68: Search by National ID
  searchNationalID(): void {
    if (!this.nationalIdQuery.trim()) return;
    this.searchingNID.set(true);
    this.api.searchByNationalID(this.nationalIdQuery.trim()).subscribe({
      next: (res) => {
        this.searchingNID.set(false);
        if (res.data?.id) {
          this.router.navigate(['/crew', res.data.id]);
        } else {
          this.toast.warning('No crew member found with that National ID');
        }
      },
      error: () => {
        this.searchingNID.set(false);
        this.toast.warning('No crew member found with that National ID');
      },
    });
  }

  createCrew(): void {
    if (!this.newCrew.national_id || !this.newCrew.first_name || !this.newCrew.last_name) {
      this.toast.warning('All fields are required');
      return;
    }
    this.creating.set(true);
    this.api.createCrewMember(this.newCrew).subscribe({
      next: () => {
        this.toast.success(this.orgCtx.workerLabel() + ' added successfully');
        this.showCreateModal.set(false);
        this.creating.set(false);
        const defaultRole = this.orgCtx.template().default_job_types?.[0]?.code || 'OTHER';
        this.newCrew = { national_id: '', first_name: '', last_name: '', role: defaultRole };
        this.loadCrew();
      },
      error: () => this.creating.set(false),
    });
  }

  // #69: Bulk import
  onFileSelect(event: Event): void {
    const file = (event.target as HTMLInputElement).files?.[0];
    if (file) this.parseCSVFile(file);
  }

  onFileDrop(event: DragEvent): void {
    event.preventDefault();
    const file = event.dataTransfer?.files?.[0];
    if (file) this.parseCSVFile(file);
  }

  private parseCSVFile(file: File): void {
    this.bulkFileName = file.name;
    const reader = new FileReader();
    reader.onload = () => {
      const text = reader.result as string;
      this.bulkTextInput = text;
      this.parseBulkText();
    };
    reader.readAsText(file);
  }

  parseBulkText(): void {
    const lines = this.bulkTextInput.split('\n').filter(l => l.trim());
    const rows = lines.map(line => {
      const parts = line.split(',').map(p => p.trim());
      return { national_id: parts[0] || '', first_name: parts[1] || '', last_name: parts[2] || '', role: (parts[3] || 'DRIVER').toUpperCase() };
    }).filter(r => r.national_id && r.first_name && r.last_name);
    this.bulkRows.set(rows);
  }

  clearBulkFile(): void {
    this.bulkFileName = '';
    this.bulkTextInput = '';
    this.bulkRows.set([]);
  }

  closeBulkModal(): void {
    this.showBulkModal.set(false);
    this.clearBulkFile();
  }

  /** Generate and download a CSV template with 5 sample rows using industry-specific roles */
  downloadBulkTemplate(): void {
    const roles = this.dynamicRoleOptions();
    const sampleNames = [
      { first: 'John', last: 'Kamau', nid: '12345678' },
      { first: 'Jane', last: 'Wanjiku', nid: '23456789' },
      { first: 'Peter', last: 'Otieno', nid: '34567890' },
      { first: 'Mary', last: 'Njeri', nid: '45678901' },
      { first: 'David', last: 'Kipchoge', nid: '56789012' },
    ];

    const header = 'national_id,first_name,last_name,role';
    const rows = sampleNames.map((s, i) => {
      const role = roles[i % roles.length]?.value || 'OTHER';
      return `${s.nid},${s.first},${s.last},${role}`;
    });

    const csv = [header, ...rows].join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `bulk_import_${this.orgCtx.workersLabel().toLowerCase().replace(/\s+/g, '_')}_template.csv`;
    a.click();
    URL.revokeObjectURL(url);
    this.toast.success('Template downloaded — fill it out and upload!');
  }

  submitBulkImport(): void {
    const rows = this.bulkRows();
    if (rows.length === 0) return;
    this.bulkImporting.set(true);
    this.api.bulkImportCrew(rows).subscribe({
      next: () => {
        this.toast.success(`${rows.length} crew members imported successfully`);
        this.closeBulkModal();
        this.bulkImporting.set(false);
        this.loadCrew();
      },
      error: () => this.bulkImporting.set(false),
    });
  }

  goToPage(page: number): void {
    this.currentPage.set(page);
    this.loadCrew();
  }

  pages(): number[] {
    const total = this.meta()?.total_pages ?? 1;
    return Array.from({ length: Math.min(total, 7) }, (_, i) => i + 1);
  }

  kycBadgeClass(status: string): string {
    switch (status) {
      case 'VERIFIED': return 'badge-success';
      case 'REJECTED': return 'badge-danger';
      default: return 'badge-warning';
    }
  }
}
