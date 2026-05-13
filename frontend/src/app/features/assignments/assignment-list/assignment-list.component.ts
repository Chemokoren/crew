import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { ConfirmDialogService } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { TooltipDirective } from '../../../shared/directives/tooltip.directive';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';
import { Assignment, PaginationMeta, CrewMember, Vehicle, Organization, IndustryType, WorkType } from '../../../core/models';

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

        <!-- Filter by Organization (system admins only) -->
        @if (auth.userRole() === 'SYSTEM_ADMIN') {
          <div style="position: relative; z-index: 53; flex: 1; min-width: 200px; max-width: 260px;">
            <app-autocomplete [(ngModel)]="saccoFilter" (ngModelChange)="load()" [options]="saccoOptions()" placeholder="— All Organizations —" id="assignment-sacco-filter"></app-autocomplete>
          </div>
        }

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
              <th>Crew Member</th><th>Vehicle</th><th>Organization</th><th>Route</th><th>Shift Date</th><th>Status</th><th>Earning Model</th><th>Actions</th>
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
                  <td style="color:var(--color-text-secondary);font-size:0.8125rem;">{{ a.organization_name || '—' }}</td>
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
                        <button class="btn btn-sm btn-ghost" (click)="openEditModal(a)" id="edit-{{a.id}}"
                                appTooltip="Edit assignment details" tooltipPosition="top">
                          <span class="material-icons-round" style="font-size:16px;">edit</span>
                        </button>
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
            <div class="modal-header">
              <div>
                <h3>Create Assignment</h3>
                @if (currentOrgName()) {
                  <div style="display:flex;align-items:center;gap:6px;margin-top:4px;">
                    <span class="material-icons-round" style="font-size:14px;color:var(--color-accent);">business</span>
                    <span style="font-size:0.8rem;color:var(--color-text-muted);">{{ currentOrgName() }}</span>
                    <span style="font-size:0.7rem;background:rgba(0,210,255,0.1);color:var(--color-accent);border:1px solid rgba(0,210,255,0.2);border-radius:4px;padding:1px 6px;">Your Organization</span>
                  </div>
                }
              </div>
              <button class="btn btn-ghost btn-icon" (click)="showCreateModal.set(false)"><span class="material-icons-round">close</span></button>
            </div>
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


              <!-- Shift Date -->
              <div class="form-group">
                <label class="form-label">
                  <span class="material-icons-round form-label-icon">calendar_today</span>
                  Shift Date
                </label>
                <input class="form-input" type="date" [(ngModel)]="newAssignment.shift_date" id="create-shift-date" />
              </div>

              <!-- Shift Start -->
              <div class="form-group">
                <label class="form-label">
                  <span class="material-icons-round form-label-icon">schedule</span>
                  Shift Start
                </label>
                <input class="form-input" type="datetime-local" [(ngModel)]="newAssignment.shift_start" id="create-shift-start" step="60" />
              </div>

              <!-- Work Type (Phase F4) -->
              <div class="form-group">
                <label class="form-label">
                  <span class="material-icons-round form-label-icon">work</span>
                  Work Type
                </label>
                <select class="form-select" [(ngModel)]="newAssignment.work_type" id="create-work-type">
                  <option value="SHIFT">Shift</option><option value="DAILY">Daily</option>
                  <option value="HOURLY">Hourly</option><option value="TASK">Task</option>
                  <option value="PROJECT">Project</option><option value="BOOKING">Booking</option>
                </select>
              </div>

              <!-- Vehicle (only for TRANSPORT or unset) -->
              @if (selectedIndustry() === 'TRANSPORT' || !selectedIndustry()) {
                <div class="form-group">
                  <label class="form-label">Vehicle</label>
                  <app-autocomplete
                    [options]="vehicleOptions()"
                    placeholder="Search by registration, type..."
                    inputId="create-vehicle-select"
                    [(ngModel)]="newAssignment.vehicle_id"
                  ></app-autocomplete>
                </div>
              }

              <!-- Work Site (for CONSTRUCTION/LOGISTICS/AGRICULTURE) -->
              @if (selectedIndustry() === 'CONSTRUCTION' || selectedIndustry() === 'LOGISTICS' || selectedIndustry() === 'AGRICULTURE') {
                <div class="form-group">
                  <label class="form-label">
                    <span class="material-icons-round form-label-icon">location_on</span>
                    Work Site / Location
                  </label>
                  <input class="form-input" [(ngModel)]="newAssignment.work_site" placeholder="e.g. Kilimani Road Project, Farm Block A" id="create-work-site" />
                </div>
                <div class="form-group">
                  <label class="form-label">Project Reference</label>
                  <input class="form-input" [(ngModel)]="newAssignment.project_ref" placeholder="e.g. PRJ-2026-001" id="create-project-ref" />
                </div>
              }

              <!-- Hourly Rate (for HOURLY work type) -->
              @if (newAssignment.work_type === 'HOURLY') {
                <div class="form-group">
                  <label class="form-label">Hourly Rate (KES)</label>
                  <input class="form-input" type="number" [(ngModel)]="hourlyRate" placeholder="250" id="create-hourly-rate" />
                </div>
              }

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
      <!-- Edit Modal -->
      @if (showEditModal()) {
        <div class="modal-backdrop" (click)="showEditModal.set(false)">
          <div class="modal-content modal-content-lg" (click)="$event.stopPropagation()">
            <div class="modal-header"><h3>Edit Assignment</h3><button class="btn btn-ghost btn-icon" (click)="showEditModal.set(false)"><span class="material-icons-round">close</span></button></div>
            <div class="modal-body">
              <div class="form-group">
                <label class="form-label"><span class="material-icons-round form-label-icon">calendar_today</span> Shift Date</label>
                <input class="form-input" type="date" [(ngModel)]="editData.shift_date" id="edit-shift-date" />
              </div>
              <div class="form-group">
                <label class="form-label"><span class="material-icons-round form-label-icon">schedule</span> Shift Start</label>
                <input class="form-input" type="datetime-local" [(ngModel)]="editData.shift_start" id="edit-shift-start" step="60" />
              </div>
              <div class="form-group">
                <label class="form-label"><span class="material-icons-round form-label-icon">work</span> Work Type</label>
                <select class="form-select" [(ngModel)]="editData.work_type" id="edit-work-type">
                  <option value="SHIFT">Shift</option><option value="DAILY">Daily</option>
                  <option value="HOURLY">Hourly</option><option value="TASK">Task</option>
                  <option value="PROJECT">Project</option><option value="BOOKING">Booking</option>
                </select>
              </div>
              <div class="form-group">
                <label class="form-label">Earning Model</label>
                <select class="form-select" [(ngModel)]="editData.earning_model" id="edit-earning-model">
                  <option value="FIXED">Fixed</option><option value="COMMISSION">Commission</option><option value="HYBRID">Hybrid</option>
                </select>
              </div>
              @if (editData.earning_model === 'FIXED' || editData.earning_model === 'HYBRID') {
                <div class="form-group"><label class="form-label">Fixed Amount (KES)</label><input class="form-input" type="number" [(ngModel)]="editFixedAmount" id="edit-fixed-amount" /></div>
              }
              @if (editData.earning_model === 'COMMISSION' || editData.earning_model === 'HYBRID') {
                <div class="form-group"><label class="form-label">Commission Rate (%)</label><input class="form-input" type="number" [(ngModel)]="editCommissionRate" step="0.01" id="edit-commission-rate" /></div>
              }
              @if (editData.work_type === 'HOURLY') {
                <div class="form-group"><label class="form-label">Hourly Rate (KES)</label><input class="form-input" type="number" [(ngModel)]="editHourlyRate" id="edit-hourly-rate" /></div>
              }
              <div class="form-group">
                <label class="form-label">Work Site / Location</label>
                <input class="form-input" [(ngModel)]="editData.work_site" id="edit-work-site" />
              </div>
              <div class="form-group">
                <label class="form-label">Project Reference</label>
                <input class="form-input" [(ngModel)]="editData.project_ref" id="edit-project-ref" />
              </div>
              <div class="form-group"><label class="form-label">Notes</label><input class="form-input" [(ngModel)]="editData.notes" id="edit-notes" /></div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="showEditModal.set(false)">Cancel</button>
              <button class="btn btn-primary" (click)="saveEdit()" [disabled]="saving()" id="btn-save-edit">{{ saving() ? 'Saving...' : 'Save Changes' }}</button>
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
  protected auth = inject(AuthService);
  private toast = inject(ToastService);
  private router = inject(Router);
  private dialog = inject(ConfirmDialogService);

  /** The logged-in user's org ID (for auto-selection) */
  userOrgId: string = '';

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
  saccos = signal<Organization[]>([]);

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
  newAssignment = { crew_member_id: '', vehicle_id: '', organization_id: '', shift_date: '', shift_start: '', earning_model: 'FIXED', work_type: 'SHIFT', work_site: '', project_ref: '', notes: '' };
  fixedAmount = 0;
  commissionRate = 0;
  hourlyRate = 0;

  // Edit modal
  showEditModal = signal(false);
  saving = signal(false);
  editAssignmentId = '';
  editData = { shift_date: '', shift_start: '', earning_model: 'FIXED', work_type: 'SHIFT', work_site: '', project_ref: '', notes: '' };
  editFixedAmount = 0;
  editCommissionRate = 0;
  editHourlyRate = 0;

  // Computed name of the logged-in user's org for display in the modal header
  currentOrgName = computed<string>(() => {
    if (!this.userOrgId) return '';
    return this.saccos().find(s => s.id === this.userOrgId)?.name || '';
  });

  // Industry detection from selected Organization (Phase F4)
  selectedIndustry = computed<IndustryType | ''>(() => {
    const saccoId = this.newAssignment.organization_id;
    if (!saccoId) return '';
    const sacco = this.saccos().find(s => s.id === saccoId);
    return sacco?.industry_type || '';
  });

  ngOnInit(): void {
    // Capture the user's org ID for auto-selection
    this.userOrgId = this.auth.currentUser()?.organization_id || '';
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

    // Fetch Organizations
    this.api.getOrganizations({ per_page: '200' }).subscribe({
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
    if (this.saccoFilter) params['organization_id'] = this.saccoFilter;

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
    this.newAssignment = { crew_member_id: '', vehicle_id: '', organization_id: this.userOrgId, shift_date: '', shift_start: '', earning_model: 'FIXED', work_type: 'SHIFT', work_site: '', project_ref: '', notes: '' };
    this.fixedAmount = 0;
    this.commissionRate = 0;
    this.hourlyRate = 0;
    this.showCreateModal.set(true);
  }

  canCreate(): boolean {
    const a = this.newAssignment;
    // org_id is optional in UI since backend auto-injects from JWT
    const baseValid = !!(a.crew_member_id && a.shift_date && a.shift_start);
    const ind = this.selectedIndustry();
    if (ind === 'TRANSPORT' || !ind) {
      return baseValid && !!a.vehicle_id;
    }
    return baseValid;
  }

  createAssignment(): void {
    this.creating.set(true);
    const a = this.newAssignment;
    const data: Record<string, unknown> = {
      ...a,
      shift_start: a.shift_start ? new Date(a.shift_start).toISOString() : '',
      fixed_amount_cents: Math.round(this.fixedAmount * 100),
      commission_rate: this.commissionRate / 100,
      hourly_rate_cents: this.hourlyRate > 0 ? Math.round(this.hourlyRate * 100) : undefined,
    };
    if (!data['vehicle_id']) delete data['vehicle_id'];
    if (!data['work_site']) delete data['work_site'];
    if (!data['project_ref']) delete data['project_ref'];
    this.api.createAssignment(data).subscribe({
      next: () => { this.toast.success('Assignment created'); this.showCreateModal.set(false); this.creating.set(false); this.load(); },
      error: (err: any) => { this.toast.error(err?.error?.message || 'Failed to create assignment'); this.creating.set(false); },
    });
  }

  openEditModal(a: Assignment): void {
    this.editAssignmentId = a.id;
    // Convert ISO date to input-compatible formats
    const shiftDate = a.shift_date ? a.shift_date.substring(0, 10) : '';
    const shiftStart = a.shift_start ? a.shift_start.substring(0, 16) : '';
    this.editData = {
      shift_date: shiftDate,
      shift_start: shiftStart,
      earning_model: a.earning_model || 'FIXED',
      work_type: a.work_type || 'SHIFT',
      work_site: a.work_site || '',
      project_ref: a.project_ref || '',
      notes: a.notes || '',
    };
    this.editFixedAmount = (a.fixed_amount_cents || 0) / 100;
    this.editCommissionRate = (a.commission_rate || 0) * 100;
    this.editHourlyRate = (a.hourly_rate_cents || 0) / 100;
    this.showEditModal.set(true);
  }

  saveEdit(): void {
    this.saving.set(true);
    const d = this.editData;
    const payload: Record<string, unknown> = {
      shift_date: d.shift_date,
      shift_start: d.shift_start ? new Date(d.shift_start).toISOString() : undefined,
      earning_model: d.earning_model,
      work_type: d.work_type,
      work_site: d.work_site || undefined,
      project_ref: d.project_ref || undefined,
      notes: d.notes,
      fixed_amount_cents: Math.round(this.editFixedAmount * 100),
      commission_rate: this.editCommissionRate / 100,
      hourly_rate_cents: this.editHourlyRate > 0 ? Math.round(this.editHourlyRate * 100) : undefined,
    };
    this.api.updateAssignment(this.editAssignmentId, payload).subscribe({
      next: () => {
        this.toast.success('Assignment updated');
        this.showEditModal.set(false);
        this.saving.set(false);
        this.load();
      },
      error: (err: any) => { this.toast.error(err?.error?.message || 'Failed to update assignment'); this.saving.set(false); },
    });
  }

  completeAssignment(a: Assignment): void {
    const labels: Record<string, string> = {
      FIXED: 'Earned Amount',
      COMMISSION: 'Total Revenue Collected',
      HYBRID: 'Total Revenue Collected',
      HOURLY: 'Total Revenue (or leave 0)',
      DAILY_RATE: 'Total Revenue (or leave 0)',
    };
    const promptLabel = labels[a.earning_model] || 'Total Revenue Collected';
    const prefillAmount = a.earning_model === 'FIXED' && a.fixed_amount_cents ? (a.fixed_amount_cents / 100).toString() : '';
    this.dialog.prompt(
      'Complete Assignment',
      `Record the ${promptLabel.toLowerCase()} for this ${a.work_type?.toLowerCase() || 'shift'} by ${a.crew_member_name || 'this crew member'}.`,
      {
        confirmText: 'Complete & Credit',
        promptLabel,
        promptPlaceholder: prefillAmount || '0.00',
        promptType: 'number',
        promptPrefix: 'KES',
        icon: 'payments',
      }
    ).subscribe(result => {
      if (!result.confirmed) return;
      const amount = parseFloat(result.value || prefillAmount || '0');
      if (isNaN(amount) || amount <= 0) {
        this.toast.warning('Please enter a valid amount');
        return;
      }
      this.api.completeAssignment(a.id, Math.round(amount * 100)).subscribe({
        next: () => { this.toast.success('Assignment completed & earnings credited'); this.load(); },
        error: (err: any) => { this.toast.error(err?.error?.message || 'Failed to complete assignment. Please try again.'); },
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
