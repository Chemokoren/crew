import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { ConfirmDialogService } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { TooltipDirective } from '../../../shared/directives/tooltip.directive';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';
import { Assignment, PaginationMeta, CrewMember, Vehicle, SACCO } from '../../../core/models';

@Component({
  selector: 'app-assignment-list',
  standalone: true,
  imports: [CommonModule, FormsModule, TooltipDirective, AutocompleteComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Assignments</h1>
          <p class="page-subtitle">Track crew shifts, vehicles, and earnings</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-primary" (click)="openCreateModal()" id="btn-create-assignment"
                  appTooltip="Create a new shift assignment for a crew member">
            <span class="material-icons-round">add</span> New Assignment
          </button>
        </div>
      </div>

      <!-- Filters -->
      <div class="filters-bar">
        <select class="form-select" [(ngModel)]="statusFilter" (ngModelChange)="load()" id="assignment-status-filter">
          <option value="">All Status</option>
          <option value="SCHEDULED">Scheduled</option>
          <option value="ACTIVE">Active</option>
          <option value="COMPLETED">Completed</option>
          <option value="CANCELLED">Cancelled</option>
        </select>
        <input class="form-input" type="date" [(ngModel)]="dateFilter" (ngModelChange)="load()" style="max-width:180px;" id="assignment-date-filter" />

        <!-- Filter by Crew Member -->
        <div style="position: relative; z-index: 54; flex: 1; min-width: 200px; max-width: 260px;">
          <app-autocomplete [(ngModel)]="crewMemberFilter" (ngModelChange)="load()" [options]="crewMemberOptions()" placeholder="— All Crew Members —" id="assignment-crew-filter"></app-autocomplete>
        </div>

        <!-- Filter by SACCO -->
        <div style="position: relative; z-index: 53; flex: 1; min-width: 200px; max-width: 260px;">
          <app-autocomplete [(ngModel)]="saccoFilter" (ngModelChange)="load()" [options]="saccoOptions()" placeholder="— All SACCOs —" id="assignment-sacco-filter"></app-autocomplete>
        </div>

        @if (hasActiveFilters()) {
          <button class="btn btn-ghost btn-sm" (click)="clearFilters()" id="btn-clear-filters" style="white-space:nowrap;"
                  appTooltip="Remove all active filters">
            <span class="material-icons-round" style="font-size:16px;">filter_alt_off</span> Clear
          </button>
        }
      </div>

      @if (loading()) {
        @for (i of [1,2,3]; track i) { <div class="skeleton" style="height:56px;margin:4px 0;"></div> }
      } @else if (items().length === 0) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">assignment</span>
          <div class="empty-title">No assignments found</div>
          <div class="empty-description">
            @if (hasActiveFilters()) {
              No assignments match your current filters. Try adjusting or clearing them.
            } @else {
              Create shift assignments to start tracking crew operations and earnings.
            }
          </div>
        </div>
      } @else {
        <div class="data-table-wrapper">
          <table class="data-table">
            <thead><tr>
              <th>Crew Member</th><th>Vehicle</th><th>SACCO</th><th>Route</th><th>Shift Date</th><th>Status</th><th>Earning Model</th><th>Actions</th>
            </tr></thead>
            <tbody>
              @for (a of items(); track a.id) {
                <tr class="clickable-row" (click)="viewDetail(a)" id="row-{{a.id}}">
                  <td>
                    <div class="cell-primary">{{ a.crew_member_name || '—' }}</div>
                  </td>
                  <td>
                    <span class="badge badge-accent" style="font-family:var(--font-mono, monospace);font-size:0.75rem;">{{ a.vehicle_registration_no || '—' }}</span>
                  </td>
                  <td style="color:var(--color-text-secondary);font-size:0.8125rem;">{{ a.sacco_name || '—' }}</td>
                  <td style="color:var(--color-text-muted);font-size:0.8125rem;">{{ a.route_name || '—' }}</td>
                  <td style="color:var(--color-text-primary);font-weight:500;">{{ a.shift_date | date:'mediumDate' }}</td>
                  <td><span class="badge" [ngClass]="statusBadge(a.status)">{{ a.status }}</span></td>
                  <td><span class="badge badge-accent">{{ a.earning_model }}</span></td>
                  <td>
                    <div style="display:flex;gap:4px;" (click)="$event.stopPropagation()">
                      <button class="btn btn-sm btn-ghost" (click)="viewDetail(a)" id="view-{{a.id}}"
                              appTooltip="View full assignment details" tooltipPosition="left">
                        <span class="material-icons-round" style="font-size:16px;">visibility</span>
                      </button>
                      @if (a.status === 'ACTIVE' || a.status === 'SCHEDULED') {
                        <button class="btn btn-sm btn-primary" (click)="completeAssignment(a)" id="complete-{{a.id}}"
                                appTooltip="Record revenue and mark assignment as completed" tooltipPosition="top">
                          Complete
                        </button>
                        <button class="btn btn-sm btn-danger" (click)="cancelAssignment(a)" id="cancel-{{a.id}}"
                                appTooltip="Cancel this assignment with a reason" tooltipPosition="top">
                          Cancel
                        </button>
                      }
                    </div>
                  </td>
                </tr>
              }
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        @if (meta()?.total_pages && meta()!.total_pages > 1) {
          <div class="pagination-bar">
            <span class="pagination-info">Page {{ meta()!.page }} of {{ meta()!.total_pages }} ({{ meta()!.total }} total)</span>
            <div class="pagination-actions">
              <button class="btn btn-sm btn-ghost" [disabled]="meta()!.page <= 1" (click)="goToPage(meta()!.page - 1)"
                      appTooltip="Go to previous page">← Prev</button>
              <button class="btn btn-sm btn-ghost" [disabled]="meta()!.page >= meta()!.total_pages" (click)="goToPage(meta()!.page + 1)"
                      appTooltip="Go to next page">Next →</button>
            </div>
          </div>
        }
      }

      <!-- Create Modal with Autocomplete Selectors -->
      @if (showCreateModal()) {
        <div class="modal-backdrop" (click)="showCreateModal.set(false)">
          <div class="modal-content modal-content-lg" (click)="$event.stopPropagation()">
            <div class="modal-header"><h3>Create Assignment</h3><button class="btn btn-ghost btn-icon" (click)="showCreateModal.set(false)"><span class="material-icons-round">close</span></button></div>
            <div class="modal-body">
              <!-- Crew Member Autocomplete -->
              <div class="form-group">
                <label class="form-label">Crew Member</label>
                <app-autocomplete
                  [options]="crewMemberOptions()"
                  placeholder="Search by name, ID, role..."
                  inputId="create-crew-select"
                  [(ngModel)]="newAssignment.crew_member_id"
                ></app-autocomplete>
              </div>

              <!-- Vehicle Autocomplete -->
              <div class="form-group">
                <label class="form-label">Vehicle</label>
                <app-autocomplete
                  [options]="vehicleOptions()"
                  placeholder="Search by registration, type..."
                  inputId="create-vehicle-select"
                  [(ngModel)]="newAssignment.vehicle_id"
                ></app-autocomplete>
              </div>

              <!-- SACCO Autocomplete -->
              <div class="form-group">
                <label class="form-label">SACCO</label>
                <app-autocomplete
                  [options]="saccoOptions()"
                  placeholder="Search by SACCO name..."
                  inputId="create-sacco-select"
                  [(ngModel)]="newAssignment.sacco_id"
                ></app-autocomplete>
              </div>

              <!-- Shift Date — Date picker -->
              <div class="form-group">
                <label class="form-label">
                  <span class="material-icons-round form-label-icon">calendar_today</span>
                  Shift Date
                </label>
                <input
                  class="form-input"
                  type="date"
                  [(ngModel)]="newAssignment.shift_date"
                  id="create-shift-date"
                />
              </div>

              <!-- Shift Start — DateTime picker -->
              <div class="form-group">
                <label class="form-label">
                  <span class="material-icons-round form-label-icon">schedule</span>
                  Shift Start
                </label>
                <input
                  class="form-input"
                  type="datetime-local"
                  [(ngModel)]="newAssignment.shift_start"
                  id="create-shift-start"
                  step="60"
                />
              </div>

              <div class="form-group"><label class="form-label">Earning Model</label>
                <select class="form-select" [(ngModel)]="newAssignment.earning_model">
                  <option value="FIXED">Fixed</option><option value="COMMISSION">Commission</option><option value="HYBRID">Hybrid</option>
                </select>
              </div>
              @if (newAssignment.earning_model === 'FIXED' || newAssignment.earning_model === 'HYBRID') {
                <div class="form-group"><label class="form-label">Fixed Amount (KES)</label><input class="form-input" type="number" [(ngModel)]="fixedAmount" placeholder="500" /></div>
              }
              @if (newAssignment.earning_model === 'COMMISSION' || newAssignment.earning_model === 'HYBRID') {
                <div class="form-group"><label class="form-label">Commission Rate (%)</label><input class="form-input" type="number" [(ngModel)]="commissionRate" step="0.01" placeholder="10" /></div>
              }
              <div class="form-group"><label class="form-label">Notes (optional)</label><input class="form-input" [(ngModel)]="newAssignment.notes" placeholder="Optional shift notes" /></div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="showCreateModal.set(false)">Cancel</button>
              <button class="btn btn-primary" (click)="createAssignment()" [disabled]="creating() || !canCreate()">{{ creating() ? 'Creating...' : 'Create' }}</button>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .filters-bar {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      align-items: center;
    }
    .filters-bar .form-select,
    .filters-bar .form-input {
      max-width: 200px;
    }

    .clickable-row {
      cursor: pointer;
      transition: background var(--transition-fast);
    }
    .clickable-row:hover {
      background: rgba(255, 255, 255, 0.02) !important;
    }

    .cell-primary {
      font-weight: 600;
      color: var(--color-text-primary);
    }

    .pagination-bar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-top: var(--space-md);
      padding: var(--space-sm) 0;
    }
    .pagination-info {
      font-size: 0.8125rem;
      color: var(--color-text-muted);
    }
    .pagination-actions { display: flex; gap: 4px; }

    /* Make modal wider for autocomplete fields */
    .modal-content-lg {
      max-width: 580px;
    }

    /* Form label with icon */
    .form-label-icon {
      font-size: 14px;
      vertical-align: middle;
      margin-right: 4px;
      color: var(--color-accent);
    }

    /* Enhanced date/datetime inputs */
    input[type="date"],
    input[type="datetime-local"] {
      color-scheme: dark;
      cursor: pointer;
    }

    input[type="date"]::-webkit-calendar-picker-indicator,
    input[type="datetime-local"]::-webkit-calendar-picker-indicator {
      filter: invert(0.7) sepia(1) saturate(5) hue-rotate(175deg);
      cursor: pointer;
      padding: 4px;
      border-radius: 4px;
      transition: background var(--transition-fast);
    }

    input[type="date"]::-webkit-calendar-picker-indicator:hover,
    input[type="datetime-local"]::-webkit-calendar-picker-indicator:hover {
      background: rgba(0, 210, 255, 0.12);
    }

    @media (max-width: 768px) {
      .filters-bar .form-select,
      .filters-bar .form-input {
        max-width: 100%;
        flex: 1 1 calc(50% - 4px);
      }
    }
  `],
})
export class AssignmentListComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private router = inject(Router);
  private dialog = inject(ConfirmDialogService);

  items = signal<Assignment[]>([]);
  meta = signal<PaginationMeta | null>(null);
  loading = signal(true);

  // Filter state
  statusFilter = '';
  dateFilter = '';
  crewMemberFilter = '';
  saccoFilter = '';
  currentPage = 1;

  // Dropdown data fetched from API
  crewMembers = signal<CrewMember[]>([]);
  vehicles = signal<Vehicle[]>([]);
  saccos = signal<SACCO[]>([]);

  // Computed autocomplete options
  crewMemberOptions = computed<AutocompleteOption[]>(() =>
    this.crewMembers().map(c => ({
      value: c.id,
      label: c.full_name,
      sublabel: `${c.crew_id} · ${c.role}`,
      badge: c.role,
      searchText: [c.full_name, c.first_name, c.last_name, c.crew_id, c.role, c.kyc_status].join(' '),
    }))
  );

  vehicleOptions = computed<AutocompleteOption[]>(() =>
    this.vehicles().map(v => ({
      value: v.id,
      label: v.registration_no,
      sublabel: `${v.vehicle_type} · Capacity: ${v.capacity}`,
      badge: v.vehicle_type,
      searchText: [v.registration_no, v.vehicle_type, `capacity ${v.capacity}`].join(' '),
    }))
  );

  saccoOptions = computed<AutocompleteOption[]>(() =>
    this.saccos().map(s => ({
      value: s.id,
      label: s.name,
      sublabel: `${s.registration_number} · ${s.county}`,
      badge: s.county,
      searchText: [s.name, s.registration_number, s.county, s.sub_county || '', s.contact_phone, s.contact_email || ''].join(' '),
    }))
  );

  // Create modal
  showCreateModal = signal(false);
  creating = signal(false);
  newAssignment = { crew_member_id: '', vehicle_id: '', sacco_id: '', shift_date: '', shift_start: '', earning_model: 'FIXED', notes: '' };
  fixedAmount = 0;
  commissionRate = 0;

  ngOnInit(): void {
    this.loadDropdownData();
    this.load();
  }

  loadDropdownData(): void {
    // Fetch crew members for dropdown selectors and filters
    this.api.getCrewMembers({ per_page: '200' }).subscribe({
      next: (res) => this.crewMembers.set(res.data),
    });

    // Fetch vehicles
    this.api.getVehicles({ per_page: '200' }).subscribe({
      next: (res) => this.vehicles.set(res.data),
    });

    // Fetch SACCOs
    this.api.getSACCOs({ per_page: '200' }).subscribe({
      next: (res) => this.saccos.set(res.data),
    });
  }

  load(): void {
    this.loading.set(true);
    const params: Record<string, string> = {
      page: this.currentPage.toString(),
      per_page: '20',
    };
    if (this.statusFilter) params['status'] = this.statusFilter;
    if (this.dateFilter) params['shift_date'] = this.dateFilter;
    if (this.crewMemberFilter) params['crew_member_id'] = this.crewMemberFilter;
    if (this.saccoFilter) params['sacco_id'] = this.saccoFilter;

    this.api.getAssignments(params).subscribe({
      next: (res) => { this.items.set(res.data); this.meta.set(res.meta); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  hasActiveFilters(): boolean {
    return !!(this.statusFilter || this.dateFilter || this.crewMemberFilter || this.saccoFilter);
  }

  clearFilters(): void {
    this.statusFilter = '';
    this.dateFilter = '';
    this.crewMemberFilter = '';
    this.saccoFilter = '';
    this.currentPage = 1;
    this.load();
  }

  goToPage(page: number): void {
    this.currentPage = page;
    this.load();
  }

  viewDetail(a: Assignment): void {
    this.router.navigate(['/assignments', a.id]);
  }

  openCreateModal(): void {
    this.newAssignment = { crew_member_id: '', vehicle_id: '', sacco_id: '', shift_date: '', shift_start: '', earning_model: 'FIXED', notes: '' };
    this.fixedAmount = 0;
    this.commissionRate = 0;
    this.showCreateModal.set(true);
  }

  canCreate(): boolean {
    return !!(this.newAssignment.crew_member_id && this.newAssignment.vehicle_id && this.newAssignment.sacco_id && this.newAssignment.shift_date && this.newAssignment.shift_start);
  }

  createAssignment(): void {
    this.creating.set(true);
    const data: Record<string, unknown> = {
      ...this.newAssignment,
      shift_start: this.newAssignment.shift_start ? new Date(this.newAssignment.shift_start).toISOString() : '',
      fixed_amount_cents: Math.round(this.fixedAmount * 100),
      commission_rate: this.commissionRate / 100,
    };
    this.api.createAssignment(data).subscribe({
      next: () => { this.toast.success('Assignment created'); this.showCreateModal.set(false); this.creating.set(false); this.load(); },
      error: () => this.creating.set(false),
    });
  }

  completeAssignment(a: Assignment): void {
    this.dialog.prompt(
      'Complete Assignment',
      `Record the total revenue collected for this shift by ${a.crew_member_name || 'this crew member'}.`,
      {
        confirmText: 'Complete & Credit',
        promptLabel: 'Total Revenue Collected',
        promptPlaceholder: '0.00',
        promptType: 'number',
        promptPrefix: 'KES',
        icon: 'payments',
      }
    ).subscribe(result => {
      if (!result.confirmed) return;
      const amount = parseFloat(result.value || '');
      if (isNaN(amount) || amount <= 0) {
        this.toast.warning('Please enter a valid revenue amount');
        return;
      }
      this.api.completeAssignment(a.id, Math.round(amount * 100)).subscribe({
        next: () => { this.toast.success('Assignment completed & earnings credited'); this.load(); },
      });
    });
  }

  cancelAssignment(a: Assignment): void {
    this.dialog.prompt(
      'Cancel Assignment',
      `Provide a reason for cancelling this assignment for ${a.crew_member_name || 'this crew member'}.`,
      {
        confirmText: 'Cancel Assignment',
        variant: 'danger',
        promptLabel: 'Cancellation Reason',
        promptPlaceholder: 'e.g. Vehicle breakdown, crew unavailable',
        icon: 'event_busy',
      }
    ).subscribe(result => {
      if (!result.confirmed || !result.value?.trim()) {
        if (result.confirmed) this.toast.warning('A cancellation reason is required');
        return;
      }
      this.api.cancelAssignment(a.id, result.value.trim()).subscribe({
        next: () => { this.toast.success('Assignment cancelled'); this.load(); },
      });
    });
  }

  statusBadge(status: string): string {
    switch (status) {
      case 'COMPLETED': return 'badge-success';
      case 'CANCELLED': return 'badge-danger';
      case 'ACTIVE': return 'badge-info';
      case 'SCHEDULED': return 'badge-warning';
      default: return '';
    }
  }
}
