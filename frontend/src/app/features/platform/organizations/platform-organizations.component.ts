import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { Organization } from '../../../core/models';

@Component({
  selector: 'app-platform-organizations',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title platform-title">Organizations</h1>
          <p class="page-subtitle">Manage and monitor all organizations on the platform</p>
        </div>
        <div class="page-actions">
          <span class="org-count-badge">{{ totalOrgs() }} organizations</span>
        </div>
      </div>

      <!-- Filters -->
      <div class="filters-bar">
        <div class="search-input-wrap">
          <span class="material-icons-round search-icon">search</span>
          <input class="form-input search-input" type="text" placeholder="Search by name, county, or reg number..."
                 [(ngModel)]="searchQuery" (ngModelChange)="onSearch()" id="org-search" />
        </div>
        <select class="form-select filter-select" [(ngModel)]="industryFilter" (ngModelChange)="onSearch()" id="org-industry-filter">
          <option value="">All Industries</option>
          <option value="TRANSPORT">Transport</option>
          <option value="CONSTRUCTION">Construction</option>
          <option value="HEALTH">Health</option>
          <option value="LOGISTICS">Logistics</option>
          <option value="AGRICULTURE">Agriculture</option>
          <option value="HOSPITALITY">Hospitality</option>
          <option value="GENERAL">General</option>
        </select>
        <select class="form-select filter-select" [(ngModel)]="statusFilter" (ngModelChange)="onSearch()" id="org-status-filter">
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>
      </div>

      <!-- Loading skeleton -->
      @if (loading()) {
        @for (i of [1,2,3,4]; track i) {
          <div class="skeleton" style="height: 80px; margin-bottom: 8px;"></div>
        }
      }

      <!-- Empty state -->
      @else if (filteredOrgs().length === 0) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">business</span>
          <div class="empty-title">No organizations found</div>
          <div class="empty-description">Adjust your search or filters</div>
        </div>
      }

      <!-- Org list -->
      @else {
        <div class="org-table-wrapper">
          <table class="data-table">
            <thead>
              <tr>
                <th>Organization</th>
                <th>Industry</th>
                <th>County</th>
                <th>Status</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              @for (org of filteredOrgs(); track org.id) {
                <tr>
                  <td>
                    <div class="org-cell">
                      <div class="org-cell-avatar">{{ org.name.charAt(0) }}</div>
                      <div class="org-cell-info">
                        <span class="org-cell-name">{{ org.name }}</span>
                        <span class="org-cell-reg">{{ org.registration_number }}</span>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span class="badge badge-info">
                      <span class="material-icons-round" style="font-size:12px;">{{ industryIcon(org.industry_type) }}</span>
                      {{ org.industry_type || 'General' }}
                    </span>
                  </td>
                  <td style="font-size: 0.8125rem;">{{ org.county }}</td>
                  <td>
                    <span class="badge" [ngClass]="org.is_active ? 'badge-success' : 'badge-danger'">
                      {{ org.is_active ? 'Active' : 'Inactive' }}
                    </span>
                  </td>
                  <td style="font-size: 0.8125rem; color: var(--color-text-muted);">
                    {{ org.created_at | relativeTime }}
                  </td>
                  <td>
                    <div class="action-btns">
                      <button class="btn btn-sm btn-ghost" style="color:#8b5cf6;" (click)="openConfigModal(org)"
                              title="Configure">
                        <span class="material-icons-round" style="font-size:16px;">settings</span>
                      </button>
                      <button class="btn btn-sm btn-ghost" style="color: var(--color-text-muted);" (click)="viewAsOrg(org)"
                              title="View as Organization">
                        <span class="material-icons-round" style="font-size:16px;">visibility</span>
                      </button>
                    </div>
                  </td>
                </tr>
              }
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        <div class="pagination">
          <button class="btn btn-sm btn-secondary" [disabled]="page() <= 1" (click)="changePage(-1)">← Prev</button>
          <span class="page-info">Page {{ page() }} of {{ totalPages() }}</span>
          <button class="btn btn-sm btn-secondary" [disabled]="page() >= totalPages()" (click)="changePage(1)">Next →</button>
        </div>
      }

      <!-- Config Modal -->
      @if (showConfigModal()) {
        <div class="modal-backdrop" (click)="showConfigModal.set(false)">
          <div class="modal-content" style="max-width: 600px;" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>Organization Configuration</h3>
              <button class="btn btn-ghost btn-icon" (click)="showConfigModal.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <div class="modal-body">
              @if (selectedOrg(); as org) {
                <div class="config-org-header">
                  <div class="org-cell-avatar" style="width:48px;height:48px;font-size:1.25rem;">{{ org.name.charAt(0) }}</div>
                  <div>
                    <div style="font-size:1.125rem;font-weight:700;color:var(--color-text-primary);">{{ org.name }}</div>
                    <div style="font-size:0.8125rem;color:var(--color-text-muted);">{{ org.registration_number }} • {{ org.county }}</div>
                  </div>
                </div>

                <div class="config-grid">
                  <div class="config-item">
                    <span class="config-label">Industry</span>
                    <span class="config-value">{{ org.industry_type || 'Not set' }}</span>
                  </div>
                  <div class="config-item">
                    <span class="config-label">Type</span>
                    <span class="config-value">{{ org.organization_type || 'SACCO' }}</span>
                  </div>
                  <div class="config-item">
                    <span class="config-label">Currency</span>
                    <span class="config-value">{{ org.currency }}</span>
                  </div>
                  <div class="config-item">
                    <span class="config-label">Status</span>
                    <span class="badge" [ngClass]="org.is_active ? 'badge-success' : 'badge-danger'">
                      {{ org.is_active ? 'Active' : 'Inactive' }}
                    </span>
                  </div>
                </div>

                <div style="margin-top: var(--space-lg); display: flex; gap: var(--space-sm);">
                  <a class="btn btn-secondary" [routerLink]="'/settings/tenant'" (click)="showConfigModal.set(false)">
                    <span class="material-icons-round">tune</span> Full Settings
                  </a>
                  <button class="btn btn-secondary" (click)="viewAsOrg(org); showConfigModal.set(false)">
                    <span class="material-icons-round">visibility</span> View as Org
                  </button>
                </div>
              }
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .platform-title {
      background: linear-gradient(135deg, #c084fc 0%, #f472b6 100%) !important;
      -webkit-background-clip: text !important;
      -webkit-text-fill-color: transparent !important;
      background-clip: text !important;
    }

    .org-count-badge {
      padding: 4px 14px; border-radius: var(--radius-full);
      background: rgba(139,92,246,0.12); color: #8b5cf6;
      font-size: 0.8125rem; font-weight: 600;
    }

    .filters-bar {
      display: flex; gap: var(--space-sm); flex-wrap: wrap;
      margin-bottom: var(--space-lg); align-items: center;
    }

    .search-input-wrap {
      position: relative; flex: 1; min-width: 240px; max-width: 400px;
    }
    .search-icon {
      position: absolute; left: 12px; top: 50%; transform: translateY(-50%);
      font-size: 18px; color: var(--color-text-muted);
    }
    .search-input { padding-left: 38px; }

    .filter-select { max-width: 180px; }

    .org-cell {
      display: flex; align-items: center; gap: var(--space-sm);
    }
    .org-cell-avatar {
      width: 36px; height: 36px; border-radius: var(--radius-md);
      background: linear-gradient(135deg, #8b5cf6, #ec4899);
      display: flex; align-items: center; justify-content: center;
      font-size: 0.875rem; font-weight: 700; color: #fff; flex-shrink: 0;
    }
    .org-cell-info { display: flex; flex-direction: column; }
    .org-cell-name { font-size: 0.875rem; font-weight: 600; color: var(--color-text-primary); }
    .org-cell-reg { font-size: 0.6875rem; color: var(--color-text-muted); }

    .action-btns { display: flex; gap: 2px; }

    .pagination {
      display: flex; align-items: center; justify-content: center;
      gap: var(--space-md); margin-top: var(--space-lg);
    }
    .page-info { font-size: 0.8125rem; color: var(--color-text-muted); }

    .org-table-wrapper { border-radius: var(--radius-lg); border: 1px solid var(--color-border); overflow-x: auto; }

    /* Config modal */
    .config-org-header {
      display: flex; align-items: center; gap: var(--space-md); margin-bottom: var(--space-lg);
    }
    .config-grid {
      display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md);
    }
    .config-item { display: flex; flex-direction: column; gap: 4px; }
    .config-label { font-size: 0.75rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--color-text-muted); }
    .config-value { font-size: 0.875rem; font-weight: 500; color: var(--color-text-primary); }

    @media (max-width: 640px) {
      .filters-bar { flex-direction: column; }
      .search-input-wrap { max-width: 100%; }
      .filter-select { max-width: 100%; width: 100%; }
    }
  `]
})
export class PlatformOrganizationsComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  orgs = signal<Organization[]>([]);
  loading = signal(true);
  page = signal(1);
  totalOrgs = signal(0);
  totalPages = signal(1);

  searchQuery = '';
  industryFilter = '';
  statusFilter = '';

  showConfigModal = signal(false);
  selectedOrg = signal<Organization | null>(null);

  filteredOrgs = computed(() => {
    let list = this.orgs();
    const q = this.searchQuery.toLowerCase().trim();
    if (q) {
      list = list.filter(o =>
        o.name.toLowerCase().includes(q) ||
        o.registration_number.toLowerCase().includes(q) ||
        o.county.toLowerCase().includes(q)
      );
    }
    if (this.industryFilter) {
      list = list.filter(o => o.industry_type === this.industryFilter);
    }
    if (this.statusFilter === 'active') {
      list = list.filter(o => o.is_active);
    } else if (this.statusFilter === 'inactive') {
      list = list.filter(o => !o.is_active);
    }
    return list;
  });

  ngOnInit(): void {
    this.loadOrgs();
  }

  loadOrgs(): void {
    this.loading.set(true);
    this.api.getOrganizations({ page: this.page().toString(), per_page: '25' }).subscribe({
      next: r => {
        this.orgs.set(r.data || []);
        this.totalOrgs.set(r.meta?.total || r.data?.length || 0);
        this.totalPages.set(r.meta?.total_pages || 1);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  onSearch(): void {
    // Client-side filtering is applied via computed signal
  }

  changePage(delta: number): void {
    this.page.update(p => p + delta);
    this.loadOrgs();
  }

  openConfigModal(org: Organization): void {
    this.selectedOrg.set(org);
    this.showConfigModal.set(true);
  }

  viewAsOrg(org: Organization): void {
    // Navigate to the org-level dashboard for this org
    window.open(`/employers/${org.id}`, '_blank');
  }

  industryIcon(type: string | undefined): string {
    const icons: Record<string, string> = {
      TRANSPORT: 'directions_bus', CONSTRUCTION: 'construction',
      HEALTH: 'health_and_safety', LOGISTICS: 'local_shipping',
      AGRICULTURE: 'agriculture', HOSPITALITY: 'hotel', GENERAL: 'business',
    };
    return icons[type || ''] || 'business';
  }
}
