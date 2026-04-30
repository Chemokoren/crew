import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink, Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { CrewMember, PaginationMeta, SACCO } from '../../../core/models';

@Component({
  selector: 'app-crew-list',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Crew Management</h1>
          <p class="page-subtitle">Manage your workforce — drivers, conductors, and riders</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-secondary btn-sm" (click)="showBulkModal.set(true)" id="btn-bulk-import">
            <span class="material-icons-round">upload_file</span> Bulk Import
          </button>
          <button class="btn btn-primary" (click)="showCreateModal.set(true)" id="btn-add-crew">
            <span class="material-icons-round">person_add</span> Add Crew
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
        <select class="form-select" [(ngModel)]="roleFilter" (ngModelChange)="loadCrew()" id="crew-role-filter">
          <option value="">All Roles</option>
          <option value="DRIVER">Driver</option>
          <option value="CONDUCTOR">Conductor</option>
          <option value="RIDER">Rider</option>
          <option value="OTHER">Other</option>
        </select>
        <select class="form-select" [(ngModel)]="kycFilter" (ngModelChange)="loadCrew()" id="crew-kyc-filter">
          <option value="">All KYC</option>
          <option value="PENDING">Pending</option>
          <option value="VERIFIED">Verified</option>
          <option value="REJECTED">Rejected</option>
        </select>
        <!-- #71: Filter by SACCO -->
        <select class="form-select" [(ngModel)]="saccoFilter" (ngModelChange)="loadCrew()" id="crew-sacco-filter">
          <option value="">All SACCOs</option>
          @for (s of saccos(); track s.id) {
            <option [value]="s.id">{{ s.name }}</option>
          }
        </select>
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
          <div class="empty-title">No crew members found</div>
          <div class="empty-description">Add your first crew member to get started with workforce management.</div>
        </div>
      } @else {
        <div class="data-table-wrapper">
          <table class="data-table">
            <thead>
              <tr>
                <th>Crew ID</th>
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

      <!-- Create Crew Modal -->
      @if (showCreateModal()) {
        <div class="modal-backdrop" (click)="showCreateModal.set(false)">
          <div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>Add Crew Member</h3>
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
              <div class="form-group">
                <label class="form-label">Role</label>
                <select class="form-select" [(ngModel)]="newCrew.role" name="role" required>
                  <option value="DRIVER">Driver</option>
                  <option value="CONDUCTOR">Conductor</option>
                  <option value="RIDER">Rider</option>
                  <option value="OTHER">Other</option>
                </select>
              </div>
            </form>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="showCreateModal.set(false)">Cancel</button>
              <button class="btn btn-primary" (click)="createCrew()" [disabled]="creating()" id="submit-create-crew">
                @if (creating()) { Creating... } @else { Add Crew Member }
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
              <h3>Bulk Import Crew</h3>
              <button class="btn btn-ghost btn-icon" (click)="closeBulkModal()">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <div class="modal-body">
              <p class="text-muted" style="margin-bottom: var(--space-md);">
                Upload a CSV file or paste data below. Each row: <code>national_id, first_name, last_name, role</code>
              </p>

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
                  <span class="material-icons-round">upload</span> Import {{ bulkRows().length }} Members
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
  `]
})
export class CrewListComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private router = inject(Router);

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
  newCrew = { national_id: '', first_name: '', last_name: '', role: 'DRIVER' };

  // #68: NID search
  searchingNID = signal(false);

  // #69: Bulk import
  showBulkModal = signal(false);
  bulkImporting = signal(false);
  bulkRows = signal<Array<{ national_id: string; first_name: string; last_name: string; role: string }>>([]);
  bulkTextInput = '';
  bulkFileName = '';

  // #71: SACCO filter
  saccos = signal<SACCO[]>([]);

  ngOnInit(): void {
    this.loadCrew();
    this.loadSACCOs();
  }

  loadCrew(): void {
    this.loading.set(true);
    const params: Record<string, string> = { page: this.currentPage().toString(), per_page: '20' };
    if (this.searchQuery) params['search'] = this.searchQuery;
    if (this.roleFilter) params['role'] = this.roleFilter;
    if (this.kycFilter) params['kyc_status'] = this.kycFilter;
    if (this.saccoFilter) params['sacco_id'] = this.saccoFilter;

    this.api.getCrewMembers(params).subscribe({
      next: (res) => {
        this.crewMembers.set(res.data);
        this.meta.set(res.meta);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  loadSACCOs(): void {
    this.api.getSACCOs({ per_page: '100' }).subscribe({
      next: (res) => this.saccos.set(res.data ?? []),
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
        this.toast.success('Crew member added successfully');
        this.showCreateModal.set(false);
        this.creating.set(false);
        this.newCrew = { national_id: '', first_name: '', last_name: '', role: 'DRIVER' };
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
