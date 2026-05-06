import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../core/services/api.service';
import { ToastService } from '../../core/services/toast.service';
import { AutocompleteComponent, AutocompleteOption } from '../../shared/components/autocomplete/autocomplete.component';
import { ConfirmDialogService } from '../../shared/components/confirm-dialog/confirm-dialog.component';
import { TooltipDirective } from '../../shared/directives/tooltip.directive';
import { Organization, WorkSite } from '../../core/models';

@Component({
  selector: 'app-work-sites',
  standalone: true,
  imports: [CommonModule, FormsModule, AutocompleteComponent, TooltipDirective],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Work Sites &amp; Projects</h1>
          <p class="page-subtitle">Manage construction sites, farm blocks, warehouses, and project locations</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-primary" (click)="openCreate()" id="btn-new-site">
            <span class="material-icons-round">add_location</span> Add Site
          </button>
        </div>
      </div>

      <!-- Filters -->
      <div class="filters-bar" style="margin-bottom:var(--space-lg);">
        <div class="search-input-wrapper" style="flex:1;min-width:180px;max-width:340px;">
          <span class="material-icons-round search-icon">search</span>
          <input class="form-input" placeholder="Search sites or projects..."
                 [ngModel]="searchQuery()" (ngModelChange)="searchQuery.set($event)"
                 id="site-search" style="padding-left:36px;" />
        </div>
        <div style="position:relative;z-index:55;flex:1;min-width:180px;max-width:220px;">
          <app-autocomplete [ngModel]="saccoFilter()" (ngModelChange)="onOrgChange($event)"
                            [options]="saccoOptions()" placeholder="— All Orgs —"
                            inputId="site-sacco-filter"></app-autocomplete>
        </div>
        @if (searchQuery() || saccoFilter()) {
          <button class="btn btn-ghost btn-sm" (click)="clearFilters()" id="btn-clear-filters">
            <span class="material-icons-round" style="font-size:16px;">filter_alt_off</span> Clear
          </button>
        }
      </div>

      <!-- Content -->
      @if (loading()) {
        @for (i of [1,2,3,4]; track i) {
          <div class="skeleton" style="height:100px;margin-bottom:8px;border-radius:var(--radius-lg);"></div>
        }
      } @else if (filteredSites().length === 0) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">location_off</span>
          <div class="empty-title">No work sites found</div>
          <div class="empty-description">
            @if (searchQuery() || saccoFilter()) {
              No sites match your filters. <button class="btn btn-ghost btn-sm" (click)="clearFilters()">Clear filters</button>
            } @else {
              Create a work site or assign crew to a location via Bulk Assignment.
            }
          </div>
        </div>
      } @else {
        <div class="sites-grid">
          @for (site of filteredSites(); track site.id) {
            <div class="glass-card site-card" [class.inactive]="!site.is_active">
              <div class="site-icon-wrapper">
                <span class="material-icons-round">{{ site.is_active ? 'location_on' : 'location_off' }}</span>
              </div>
              <div class="site-content">
                <div class="site-name">
                  {{ site.name }}
                  @if (!site.is_active) {
                    <span class="badge badge-muted" style="margin-left:6px;font-size:0.65rem;">Inactive</span>
                  }
                </div>
                @if (site.project_ref) {
                  <div class="site-project">
                    <span class="material-icons-round" style="font-size:14px;">folder</span>
                    {{ site.project_ref }}
                  </div>
                }
                @if (site.address) {
                  <div class="site-address">
                    <span class="material-icons-round" style="font-size:13px;">place</span>
                    {{ site.address }}
                  </div>
                }
                @if (site.description) {
                  <div class="site-desc">{{ site.description }}</div>
                }
                <div class="site-meta">
                  <span class="site-stat">
                    <span class="material-icons-round" style="font-size:13px;">schedule</span>
                    Added {{ site.created_at | date:'mediumDate' }}
                  </span>
                </div>
              </div>
              <div class="site-actions" (click)="$event.stopPropagation()">
                <button class="btn btn-sm btn-ghost" (click)="openEdit(site)" id="edit-{{site.id}}"
                        appTooltip="Edit this site" tooltipPosition="left">
                  <span class="material-icons-round" style="font-size:16px;">edit</span>
                </button>
                <button class="btn btn-sm btn-ghost" (click)="goBulkAssign(site)" id="assign-{{site.id}}"
                        appTooltip="Create bulk assignments for this site" tooltipPosition="left">
                  <span class="material-icons-round" style="font-size:16px;">rocket_launch</span>
                </button>
                <button class="btn btn-sm btn-danger" (click)="deleteSite(site)" id="delete-{{site.id}}"
                        appTooltip="Delete this site" tooltipPosition="left">
                  <span class="material-icons-round" style="font-size:16px;">delete</span>
                </button>
              </div>
            </div>
          }
        </div>
      }

      <!-- Create Modal -->
      @if (showCreateModal()) {
        <div class="modal-backdrop" (click)="showCreateModal.set(false)">
          <div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>Add Work Site</h3>
              <button class="btn btn-ghost btn-icon" (click)="showCreateModal.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <div class="modal-body">
              <div class="form-group">
                <label class="form-label">Site Name *</label>
                <input class="form-input" [(ngModel)]="form.name" placeholder="e.g. Kilimani Road Phase 2" id="new-site-name" />
              </div>
              <div class="form-group">
                <label class="form-label">Project Reference <span style="color:var(--color-text-muted);font-weight:400;">(optional)</span></label>
                <input class="form-input" [(ngModel)]="form.project_ref" placeholder="e.g. PRJ-2026-003" id="new-site-ref" />
              </div>
              <div class="form-group">
                <label class="form-label">Address <span style="color:var(--color-text-muted);font-weight:400;">(optional)</span></label>
                <input class="form-input" [(ngModel)]="form.address" placeholder="e.g. Westlands, Nairobi" id="new-site-address" />
              </div>
              <div class="form-group">
                <label class="form-label">Description <span style="color:var(--color-text-muted);font-weight:400;">(optional)</span></label>
                <textarea class="form-input" [(ngModel)]="form.description" rows="2" placeholder="Brief notes about this site" id="new-site-desc" style="resize:vertical;"></textarea>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="showCreateModal.set(false)">Cancel</button>
              <div style="flex:1;"></div>
              <button class="btn btn-ghost btn-sm" (click)="createThenBulk()" [disabled]="!form.name || saving()" id="btn-create-bulk">
                <span class="material-icons-round" style="font-size:15px;">rocket_launch</span> Save &amp; Bulk Assign
              </button>
              <button class="btn btn-primary" (click)="createSite()" [disabled]="!form.name || saving()" id="btn-save-site">
                {{ saving() ? 'Saving...' : 'Save Site' }}
              </button>
            </div>
          </div>
        </div>
      }

      <!-- Edit Modal -->
      @if (showEditModal()) {
        <div class="modal-backdrop" (click)="showEditModal.set(false)">
          <div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>Edit Work Site</h3>
              <button class="btn btn-ghost btn-icon" (click)="showEditModal.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <div class="modal-body">
              <div class="form-group">
                <label class="form-label">Site Name *</label>
                <input class="form-input" [(ngModel)]="editForm.name" placeholder="Site name" id="edit-site-name" />
              </div>
              <div class="form-group">
                <label class="form-label">Project Reference</label>
                <input class="form-input" [(ngModel)]="editForm.project_ref" placeholder="e.g. PRJ-2026-003" id="edit-site-ref" />
              </div>
              <div class="form-group">
                <label class="form-label">Address</label>
                <input class="form-input" [(ngModel)]="editForm.address" placeholder="e.g. Westlands, Nairobi" id="edit-site-address" />
              </div>
              <div class="form-group">
                <label class="form-label">Description</label>
                <textarea class="form-input" [(ngModel)]="editForm.description" rows="2" id="edit-site-desc" style="resize:vertical;"></textarea>
              </div>
              <div class="form-group">
                <label class="form-label" style="display:flex;align-items:center;gap:8px;cursor:pointer;">
                  <input type="checkbox" [(ngModel)]="editForm.is_active" id="edit-site-active" />
                  Site is active
                </label>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="showEditModal.set(false)">Cancel</button>
              <button class="btn btn-primary" (click)="updateSite()" [disabled]="!editForm.name || saving()" id="btn-save-edit">
                {{ saving() ? 'Saving...' : 'Save Changes' }}
              </button>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .search-input-wrapper { position: relative; }
    .search-icon {
      position: absolute; left: 10px; top: 50%; transform: translateY(-50%);
      font-size: 18px; color: var(--color-text-muted); pointer-events: none; z-index: 1;
    }
    .filters-bar { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }

    .sites-grid { display: flex; flex-direction: column; gap: var(--space-sm); }
    .site-card {
      display: flex; align-items: flex-start; gap: var(--space-md);
      padding: var(--space-lg) !important;
      transition: border-color 200ms;
      &:hover { border-color: var(--color-accent) !important; }
    }
    .site-card.inactive { opacity: 0.6; }

    .site-icon-wrapper {
      width: 48px; height: 48px; border-radius: var(--radius-md); flex-shrink: 0;
      background: rgba(251,146,60,0.12); color: #fb923c;
      display: flex; align-items: center; justify-content: center;
      .material-icons-round { font-size: 24px; }
    }
    .site-content { flex: 1; min-width: 0; }
    .site-name { font-size: 0.9375rem; font-weight: 700; color: var(--color-text-primary); margin-bottom: 2px; display: flex; align-items: center; }
    .site-project { display: flex; align-items: center; gap: 4px; font-size: 0.8rem; color: var(--color-accent); margin-bottom: 4px; }
    .site-address { display: flex; align-items: center; gap: 4px; font-size: 0.78rem; color: var(--color-text-secondary); margin-bottom: 4px; }
    .site-desc { font-size: 0.78rem; color: var(--color-text-muted); margin-bottom: 4px; }
    .site-meta { display: flex; gap: var(--space-md); flex-wrap: wrap; }
    .site-stat { display: flex; align-items: center; gap: 4px; font-size: 0.75rem; color: var(--color-text-muted); }
    .site-actions { display: flex; flex-direction: column; gap: 4px; flex-shrink: 0; }

    @media (max-width: 768px) {
      .site-card { flex-wrap: wrap; }
      .site-actions { flex-direction: row; }
    }
  `]
})
export class WorkSitesComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private router = inject(Router);
  private dialog = inject(ConfirmDialogService);

  sites = signal<WorkSite[]>([]);
  saccos = signal<Organization[]>([]);
  loading = signal(true);
  saving = signal(false);
  searchQuery = signal('');
  saccoFilter = signal('');
  showCreateModal = signal(false);
  showEditModal = signal(false);
  editingId = '';

  form = { name: '', project_ref: '', address: '', description: '' };
  editForm = { name: '', project_ref: '', address: '', description: '', is_active: true };

  saccoOptions = computed<AutocompleteOption[]>(() =>
    this.saccos().map(s => ({ value: s.id, label: s.name, searchText: s.name }))
  );

  filteredSites = computed<WorkSite[]>(() => {
    const q = this.searchQuery().toLowerCase();
    let list = this.sites();
    if (q) {
      list = list.filter(s =>
        s.name.toLowerCase().includes(q) ||
        (s.project_ref || '').toLowerCase().includes(q) ||
        (s.address || '').toLowerCase().includes(q)
      );
    }
    return list;
  });

  ngOnInit(): void {
    this.api.getOrganizations({ per_page: '200' }).subscribe({ next: r => this.saccos.set(r.data) });
    this.load();
  }

  load(): void {
    this.loading.set(true);
    const params: Record<string, string> = { per_page: '100' };
    const sf = this.saccoFilter();
    if (sf) params['organization_id'] = sf;
    this.api.getWorkSites(params).subscribe({
      next: r => { this.sites.set(r.data || []); this.loading.set(false); },
      error: () => { this.toast.error('Failed to load work sites'); this.loading.set(false); },
    });
  }

  onOrgChange(val: string): void {
    this.saccoFilter.set(val);
    this.load();
  }

  clearFilters(): void {
    this.searchQuery.set('');
    this.saccoFilter.set('');
    this.load();
  }

  openCreate(): void {
    this.form = { name: '', project_ref: '', address: '', description: '' };
    this.showCreateModal.set(true);
  }

  createSite(thenNavigate = false): void {
    if (!this.form.name) return;
    this.saving.set(true);
    this.api.createWorkSite(this.form).subscribe({
      next: r => {
        this.sites.update(list => [r.data, ...list]);
        this.toast.success(`Site "${r.data.name}" created successfully.`);
        this.showCreateModal.set(false);
        this.saving.set(false);
        if (thenNavigate) {
          this.router.navigate(['/assignments/bulk'], {
            queryParams: { work_site: r.data.name, project_ref: r.data.project_ref || '' }
          });
        }
      },
      error: (err: any) => {
        this.toast.error(err?.error?.error?.message || 'Failed to create site');
        this.saving.set(false);
      },
    });
  }

  createThenBulk(): void { this.createSite(true); }

  openEdit(site: WorkSite): void {
    this.editingId = site.id;
    this.editForm = {
      name: site.name,
      project_ref: site.project_ref || '',
      address: site.address || '',
      description: site.description || '',
      is_active: site.is_active,
    };
    this.showEditModal.set(true);
  }

  updateSite(): void {
    if (!this.editForm.name) return;
    this.saving.set(true);
    this.api.updateWorkSite(this.editingId, this.editForm).subscribe({
      next: r => {
        this.sites.update(list => list.map(s => s.id === this.editingId ? r.data : s));
        this.toast.success('Site updated.');
        this.showEditModal.set(false);
        this.saving.set(false);
      },
      error: (err: any) => {
        this.toast.error(err?.error?.error?.message || 'Failed to update site');
        this.saving.set(false);
      },
    });
  }

  deleteSite(site: WorkSite): void {
    this.dialog.confirm(
      'Delete Work Site',
      `Are you sure you want to delete "${site.name}"? This cannot be undone.`,
      { confirmText: 'Delete', variant: 'danger', icon: 'delete_forever' }
    ).subscribe(result => {
      if (!result.confirmed) return;
      this.api.deleteWorkSite(site.id).subscribe({
        next: () => {
          this.sites.update(list => list.filter(s => s.id !== site.id));
          this.toast.success(`Site "${site.name}" deleted.`);
        },
        error: (err: any) => this.toast.error(err?.error?.error?.message || 'Failed to delete site'),
      });
    });
  }

  goBulkAssign(site: WorkSite): void {
    this.router.navigate(['/assignments/bulk'], {
      queryParams: { work_site: site.name, project_ref: site.project_ref || '' }
    });
  }
}
