import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { ToastService } from '../../core/services/toast.service';
import { AutocompleteComponent, AutocompleteOption } from '../../shared/components/autocomplete/autocomplete.component';
import { ConfirmDialogService } from '../../shared/components/confirm-dialog/confirm-dialog.component';
import { Organization, Assignment } from '../../core/models';

interface WorkSite {
  name: string;
  project_ref: string;
  assignments: number;
  latestDate: string;
}

@Component({
  selector: 'app-work-sites',
  standalone: true,
  imports: [CommonModule, FormsModule, AutocompleteComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Work Sites & Projects</h1>
          <p class="page-subtitle">Manage construction sites, farm blocks, warehouses, and project locations</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-primary" (click)="showCreateModal.set(true)" id="btn-new-site">
            <span class="material-icons-round">add_location</span> Add Site
          </button>
        </div>
      </div>

      <div class="filters-bar" style="margin-bottom:var(--space-lg);">
        <div class="search-input-wrapper">
          <span class="material-icons-round search-icon">search</span>
          <input class="form-input" placeholder="Search sites or projects..." [(ngModel)]="searchQuery" (ngModelChange)="filterSites()" id="site-search" />
        </div>
        <div style="position:relative;z-index:55;flex:1;min-width:180px;max-width:220px;">
          <app-autocomplete [(ngModel)]="saccoFilter" (ngModelChange)="loadAssignments()" [options]="saccoOptions()" placeholder="— All Orgs —" inputId="site-sacco-filter"></app-autocomplete>
        </div>
      </div>

      @if (loading()) {
        @for (i of [1,2,3]; track i) { <div class="skeleton" style="height:120px;margin-bottom:var(--space-sm);border-radius:var(--radius-lg);"></div> }
      } @else if (filteredSites().length === 0) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">location_off</span>
          <div class="empty-title">No work sites found</div>
          <div class="empty-description">Work sites appear here when assignments include a work_site or project_ref field.</div>
        </div>
      } @else {
        <div class="sites-grid">
          @for (site of filteredSites(); track site.name) {
            <div class="glass-card site-card">
              <div class="site-icon-wrapper">
                <span class="material-icons-round">location_on</span>
              </div>
              <div class="site-content">
                <div class="site-name">{{ site.name }}</div>
                @if (site.project_ref) {
                  <div class="site-project">
                    <span class="material-icons-round" style="font-size:14px;">folder</span>
                    {{ site.project_ref }}
                  </div>
                }
                <div class="site-meta">
                  <span class="site-stat">
                    <span class="material-icons-round" style="font-size:14px;">assignment</span>
                    {{ site.assignments }} assignment{{ site.assignments !== 1 ? 's' : '' }}
                  </span>
                  <span class="site-stat">
                    <span class="material-icons-round" style="font-size:14px;">schedule</span>
                    Latest: {{ site.latestDate | date:'mediumDate' }}
                  </span>
                </div>
              </div>
              <div class="site-badge">
                <span class="badge badge-accent">{{ site.assignments }}</span>
              </div>
            </div>
          }
        </div>
      }

      <!-- Create Site Modal (shortcut: pre-fill an assignment with site) -->
      @if (showCreateModal()) {
        <div class="modal-backdrop" (click)="showCreateModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Add Work Site</h3><button class="btn btn-ghost btn-icon" (click)="showCreateModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group"><label class="form-label">Site Name</label><input class="form-input" [(ngModel)]="newSite.name" placeholder="e.g. Kilimani Road Phase 2" id="new-site-name" /></div>
            <div class="form-group"><label class="form-label">Project Reference</label><input class="form-input" [(ngModel)]="newSite.project_ref" placeholder="e.g. PRJ-2026-003" id="new-site-ref" /></div>
            <p class="text-muted" style="font-size:0.8rem;">Work sites are created implicitly when assignments use the <code>work_site</code> field. You can also create assignments with this site pre-filled from the Bulk Assignment page.</p>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" (click)="showCreateModal.set(false)">Cancel</button>
            <button class="btn btn-primary" (click)="createSite()" [disabled]="!newSite.name" id="btn-save-site">Save Site</button>
          </div>
        </div></div>
      }
    </div>
  `,
  styles: [`
    .search-input-wrapper { position: relative; }
    .sites-grid { display: flex; flex-direction: column; gap: var(--space-sm); }
    .site-card {
      display: flex; align-items: center; gap: var(--space-md); padding: var(--space-lg) !important;
      transition: border-color 200ms;
      &:hover { border-color: var(--color-accent) !important; }
    }
    .site-icon-wrapper {
      width: 48px; height: 48px; border-radius: var(--radius-md);
      background: rgba(251,146,60,0.12); color: #fb923c;
      display: flex; align-items: center; justify-content: center; flex-shrink: 0;
      .material-icons-round { font-size: 24px; }
    }
    .site-content { flex: 1; min-width: 0; }
    .site-name { font-size: 0.9375rem; font-weight: 700; color: var(--color-text-primary); margin-bottom: 2px; }
    .site-project { display: flex; align-items: center; gap: 4px; font-size: 0.8rem; color: var(--color-accent); margin-bottom: 4px; }
    .site-meta { display: flex; gap: var(--space-md); flex-wrap: wrap; }
    .site-stat { display: flex; align-items: center; gap: 4px; font-size: 0.75rem; color: var(--color-text-muted); }
    .site-badge { flex-shrink: 0; }

    @media (max-width: 768px) {
      .site-card { flex-wrap: wrap; }
    }
  `]
})
export class WorkSitesComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  saccos = signal<Organization[]>([]);
  assignments = signal<Assignment[]>([]);
  loading = signal(true);
  searchQuery = '';
  saccoFilter = '';
  showCreateModal = signal(false);
  newSite = { name: '', project_ref: '' };

  saccoOptions = computed<AutocompleteOption[]>(() =>
    this.saccos().map(s => ({ value: s.id, label: s.name, searchText: s.name }))
  );

  // Derive work sites from assignment data
  allSites = computed<WorkSite[]>(() => {
    const map = new Map<string, WorkSite>();
    for (const a of this.assignments()) {
      const name = a.work_site || '';
      if (!name) continue;
      const existing = map.get(name);
      if (existing) {
        existing.assignments++;
        if (a.shift_date > existing.latestDate) existing.latestDate = a.shift_date;
        if (a.project_ref && !existing.project_ref) existing.project_ref = a.project_ref;
      } else {
        map.set(name, { name, project_ref: a.project_ref || '', assignments: 1, latestDate: a.shift_date });
      }
    }
    return [...map.values()].sort((a, b) => b.assignments - a.assignments);
  });

  filteredSites = computed<WorkSite[]>(() => {
    const q = this.searchQuery.toLowerCase();
    if (!q) return this.allSites();
    return this.allSites().filter(s =>
      s.name.toLowerCase().includes(q) || s.project_ref.toLowerCase().includes(q)
    );
  });

  ngOnInit(): void {
    this.api.getOrganizations({ per_page: '200' }).subscribe({ next: r => this.saccos.set(r.data) });
    this.loadAssignments();
  }

  loadAssignments(): void {
    this.loading.set(true);
    const params: Record<string, string> = { per_page: '500' };
    if (this.saccoFilter) params['organization_id'] = this.saccoFilter;
    this.api.getAssignments(params).subscribe({
      next: r => { this.assignments.set(r.data || []); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  filterSites(): void {
    // Triggers computed re-evaluation via searchQuery binding
  }

  createSite(): void {
    // Sites are derived from assignments — just store locally for now
    this.toast.success(`Site "${this.newSite.name}" noted. Use Bulk Assignment to create assignments with this site.`);
    this.showCreateModal.set(false);
    this.newSite = { name: '', project_ref: '' };
  }
}
