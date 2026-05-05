import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../core/services/api.service';
import { ToastService } from '../../core/services/toast.service';
import { AutocompleteComponent, AutocompleteOption } from '../../shared/components/autocomplete/autocomplete.component';
import { CrewMember, Organization, TenantJobType, JobTypeCategory } from '../../core/models';

@Component({
  selector: 'app-facilitators',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, AutocompleteComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Facilitators & Support Staff</h1>
          <p class="page-subtitle">Manage supervisors, facilitators, and support roles across your organization</p>
        </div>
      </div>

      <!-- Category Tabs -->
      <div class="category-tabs">
        <button class="tab-btn" [class.active]="activeCategory() === 'ALL'" (click)="activeCategory.set('ALL')" id="tab-all">
          <span class="material-icons-round" style="font-size:16px;">people</span> All ({{ allStaff().length }})
        </button>
        <button class="tab-btn" [class.active]="activeCategory() === 'FACILITATOR'" (click)="activeCategory.set('FACILITATOR')" id="tab-facilitator">
          <span class="material-icons-round" style="font-size:16px;">support_agent</span> Facilitators ({{ facilitatorCount() }})
        </button>
        <button class="tab-btn" [class.active]="activeCategory() === 'SUPERVISOR'" (click)="activeCategory.set('SUPERVISOR')" id="tab-supervisor">
          <span class="material-icons-round" style="font-size:16px;">manage_accounts</span> Supervisors ({{ supervisorCount() }})
        </button>
        <button class="tab-btn" [class.active]="activeCategory() === 'SUPPORT'" (click)="activeCategory.set('SUPPORT')" id="tab-support">
          <span class="material-icons-round" style="font-size:16px;">handyman</span> Support ({{ supportCount() }})
        </button>
      </div>

      <!-- Filters -->
      <div class="filters-bar" style="margin-bottom:var(--space-lg);">
        <div class="search-input-wrapper">
          <span class="material-icons-round search-icon">search</span>
          <input class="form-input" placeholder="Search by name..." [ngModel]="searchQuery()" (ngModelChange)="searchQuery.set($event)" id="facilitator-search" />
        </div>
        <div style="position:relative;z-index:55;flex:1;min-width:180px;max-width:220px;">
          <app-autocomplete [ngModel]="saccoFilter()" (ngModelChange)="saccoFilter.set($event); load()" [options]="saccoOptions()" placeholder="— All Orgs —" inputId="fac-sacco-filter"></app-autocomplete>
        </div>
      </div>

      @if (loading()) {
        @for (i of [1,2,3,4]; track i) { <div class="skeleton" style="height:80px;margin-bottom:var(--space-xs);border-radius:var(--radius-lg);"></div> }
      } @else if (filteredStaff().length === 0) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">support_agent</span>
          <div class="empty-title">No support staff found</div>
          <div class="empty-description">Add crew members with FACILITATOR, SUPERVISOR, or SUPPORT roles to see them here.</div>
        </div>
      } @else {
        <div class="staff-grid">
          @for (member of filteredStaff(); track member.id) {
            <div class="glass-card staff-card">
              <div class="staff-avatar">{{ member.first_name.charAt(0) }}{{ member.last_name.charAt(0) }}</div>
              <div class="staff-content">
                <div class="staff-name">{{ member.full_name }}</div>
                <div class="staff-meta">
                  <span class="badge" [ngClass]="categoryBadge(getRoleCategory(member.role))">{{ member.role }}</span>
                  <span class="badge" [ngClass]="categoryBadge(getRoleCategory(member.role))" style="opacity:0.7;">{{ getRoleCategory(member.role) }}</span>
                  <span class="staff-id">{{ member.crew_id }}</span>
                </div>
              </div>
              <div class="staff-status">
                <span class="badge" [ngClass]="member.is_active ? 'badge-success' : 'badge-danger'">{{ member.is_active ? 'Active' : 'Inactive' }}</span>
              </div>
              <a [routerLink]="['/crew', member.id]" class="btn btn-ghost btn-sm" id="view-{{ member.id }}">
                <span class="material-icons-round" style="font-size:16px;">open_in_new</span>
              </a>
            </div>
          }
        </div>
      }
    </div>
  `,
  styles: [`
    .category-tabs {
      display: flex; gap: 2px; background: var(--color-surface-alt); border-radius: var(--radius-md);
      padding: 3px; margin-bottom: var(--space-lg); overflow-x: auto;
    }
    .tab-btn {
      display: flex; align-items: center; gap: 6px;
      padding: 8px 16px; border: none; background: transparent; color: var(--color-text-muted);
      font-size: 0.8125rem; font-weight: 500; border-radius: var(--radius-sm); cursor: pointer;
      transition: all 0.2s; white-space: nowrap;
      &.active { background: var(--gradient-accent); color: #fff; }
      &:hover:not(.active) { color: var(--color-text-primary); }
    }
    .search-input-wrapper { position: relative; }
    .staff-grid { display: flex; flex-direction: column; gap: var(--space-xs); }
    .staff-card {
      display: flex; align-items: center; gap: var(--space-md); padding: var(--space-md) var(--space-lg) !important;
      transition: border-color 200ms;
      &:hover { border-color: var(--color-accent) !important; }
    }
    .staff-avatar {
      width: 40px; height: 40px; border-radius: var(--radius-md);
      background: var(--gradient-accent); display: flex; align-items: center; justify-content: center;
      font-size: 0.75rem; font-weight: 700; color: var(--color-text-inverse); flex-shrink: 0;
    }
    .staff-content { flex: 1; min-width: 0; }
    .staff-name { font-size: 0.9rem; font-weight: 600; color: var(--color-text-primary); margin-bottom: 2px; }
    .staff-meta { display: flex; align-items: center; gap: var(--space-xs); flex-wrap: wrap; }
    .staff-id { font-size: 0.7rem; color: var(--color-text-muted); font-family: var(--font-mono, monospace); }
    .staff-status { flex-shrink: 0; }

    .badge-facilitator { background: rgba(99,102,241,0.12); color: #6366f1; }
    .badge-supervisor { background: rgba(251,146,60,0.12); color: #fb923c; }
    .badge-support { background: rgba(168,85,247,0.12); color: #a855f7; }

    @media (max-width: 768px) {
      .staff-card { flex-wrap: wrap; }
      .category-tabs { flex-wrap: nowrap; }
    }
  `]
})
export class FacilitatorsComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  crewMembers = signal<CrewMember[]>([]);
  jobTypes = signal<TenantJobType[]>([]);
  saccos = signal<Organization[]>([]);
  loading = signal(true);
  searchQuery = signal('');
  saccoFilter = signal('');
  activeCategory = signal<'ALL' | JobTypeCategory>('ALL');

  saccoOptions = computed<AutocompleteOption[]>(() =>
    this.saccos().map(s => ({ value: s.id, label: s.name, searchText: s.name }))
  );

  // Non-primary roles map (derived from job types)
  private supportRoles = computed<Map<string, JobTypeCategory>>(() => {
    const map = new Map<string, JobTypeCategory>();
    for (const jt of this.jobTypes()) {
      if (jt.category !== 'PRIMARY') map.set(jt.code, jt.category);
    }
    // Fallback patterns for orgs without custom job types
    const defaults: Record<string, JobTypeCategory> = {
      SUPERVISOR: 'SUPERVISOR', FACILITATOR: 'FACILITATOR', SUPPORT: 'SUPPORT',
      BOOKING_AGENT: 'FACILITATOR', DISPATCHER: 'FACILITATOR', FOREMAN: 'SUPERVISOR',
      SITE_MANAGER: 'SUPERVISOR', NURSE: 'SUPPORT', CLEANER: 'SUPPORT',
      OFFICE_ADMIN: 'SUPPORT',
    };
    for (const [code, cat] of Object.entries(defaults)) {
      if (!map.has(code)) map.set(code, cat);
    }
    return map;
  });

  // All non-primary crew members
  allStaff = computed<CrewMember[]>(() =>
    this.crewMembers().filter(c => this.supportRoles().has(c.role))
  );

  filteredStaff = computed<CrewMember[]>(() => {
    let list = this.allStaff();
    const category = this.activeCategory();
    if (category !== 'ALL') {
      list = list.filter(c => this.getRoleCategory(c.role) === category);
    }
    const q = this.searchQuery().toLowerCase();
    if (q) list = list.filter(c => c.full_name.toLowerCase().includes(q));
    return list;
  });

  facilitatorCount = computed(() => this.allStaff().filter(c => this.getRoleCategory(c.role) === 'FACILITATOR').length);
  supervisorCount = computed(() => this.allStaff().filter(c => this.getRoleCategory(c.role) === 'SUPERVISOR').length);
  supportCount = computed(() => this.allStaff().filter(c => this.getRoleCategory(c.role) === 'SUPPORT').length);

  getRoleCategory(role: string): JobTypeCategory {
    return this.supportRoles().get(role) || 'SUPPORT';
  }

  categoryBadge(cat: JobTypeCategory): string {
    switch (cat) {
      case 'FACILITATOR': return 'badge-facilitator';
      case 'SUPERVISOR': return 'badge-supervisor';
      case 'SUPPORT': return 'badge-support';
      default: return 'badge-accent';
    }
  }

  ngOnInit(): void {
    this.api.getOrganizations({ per_page: '200' }).subscribe({
      next: r => {
        this.saccos.set(r.data);
        if (r.data?.length) {
          this.api.getJobTypes(r.data[0].id).subscribe({ next: jr => this.jobTypes.set(jr.data || []) });
        }
      },
    });
    this.load();
  }

  load(): void {
    this.loading.set(true);
    const params: Record<string, string> = { per_page: '500' };
    if (this.saccoFilter()) params['organization_id'] = this.saccoFilter();
    this.api.getCrewMembers(params).subscribe({
      next: r => { this.crewMembers.set(r.data); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }
}
