import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Assignment, PaginationMeta, CrewMember, Vehicle, SACCO } from '../../../core/models';

@Component({
  selector: 'app-assignment-list',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Assignments</h1>
          <p class="page-subtitle">Track crew shifts, vehicles, and earnings</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-primary" (click)="openCreateModal()" id="btn-create-assignment">
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
        <select class="form-select" [(ngModel)]="crewMemberFilter" (ngModelChange)="load()" id="assignment-crew-filter">
          <option value="">All Crew Members</option>
          @for (c of crewMembers(); track c.id) {
            <option [value]="c.id">{{ c.full_name }} ({{ c.crew_id }})</option>
          }
        </select>

        <!-- Filter by SACCO -->
        <select class="form-select" [(ngModel)]="saccoFilter" (ngModelChange)="load()" id="assignment-sacco-filter">
          <option value="">All SACCOs</option>
          @for (s of saccos(); track s.id) {
            <option [value]="s.id">{{ s.name }}</option>
          }
        </select>

        @if (hasActiveFilters()) {
          <button class="btn btn-ghost btn-sm" (click)="clearFilters()" id="btn-clear-filters" style="white-space:nowrap;">
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
                      <button class="btn btn-sm btn-ghost" (click)="viewDetail(a)" title="View details" id="view-{{a.id}}">
                        <span class="material-icons-round" style="font-size:16px;">visibility</span>
                      </button>
                      @if (a.status === 'ACTIVE' || a.status === 'SCHEDULED') {
                        <button class="btn btn-sm btn-primary" (click)="completeAssignment(a)" id="complete-{{a.id}}">Complete</button>
                        <button class="btn btn-sm btn-danger" (click)="cancelAssignment(a)" id="cancel-{{a.id}}">Cancel</button>
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
              <button class="btn btn-sm btn-ghost" [disabled]="meta()!.page <= 1" (click)="goToPage(meta()!.page - 1)">← Prev</button>
              <button class="btn btn-sm btn-ghost" [disabled]="meta()!.page >= meta()!.total_pages" (click)="goToPage(meta()!.page + 1)">Next →</button>
            </div>
          </div>
        }
      }

      <!-- Create Modal with Dropdown Selectors -->
      @if (showCreateModal()) {
        <div class="modal-backdrop" (click)="showCreateModal.set(false)">
          <div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header"><h3>Create Assignment</h3><button class="btn btn-ghost btn-icon" (click)="showCreateModal.set(false)"><span class="material-icons-round">close</span></button></div>
            <div class="modal-body">
              <!-- Crew Member Dropdown -->
              <div class="form-group">
                <label class="form-label">Crew Member</label>
                <select class="form-select" [(ngModel)]="newAssignment.crew_member_id" id="create-crew-select">
                  <option value="">Select a crew member...</option>
                  @for (c of crewMembers(); track c.id) {
                    <option [value]="c.id">{{ c.full_name }} — {{ c.crew_id }} ({{ c.role }})</option>
                  }
                </select>
              </div>

              <!-- Vehicle Dropdown -->
              <div class="form-group">
                <label class="form-label">Vehicle</label>
                <select class="form-select" [(ngModel)]="newAssignment.vehicle_id" id="create-vehicle-select">
                  <option value="">Select a vehicle...</option>
                  @for (v of vehicles(); track v.id) {
                    <option [value]="v.id">{{ v.registration_no }} — {{ v.vehicle_type }} (Cap: {{ v.capacity }})</option>
                  }
                </select>
              </div>

              <!-- SACCO Dropdown -->
              <div class="form-group">
                <label class="form-label">SACCO</label>
                <select class="form-select" [(ngModel)]="newAssignment.sacco_id" id="create-sacco-select">
                  <option value="">Select a SACCO...</option>
                  @for (s of saccos(); track s.id) {
                    <option [value]="s.id">{{ s.name }}</option>
                  }
                </select>
              </div>

              <div class="form-group"><label class="form-label">Shift Date</label><input class="form-input" type="date" [(ngModel)]="newAssignment.shift_date" /></div>
              <div class="form-group"><label class="form-label">Shift Start</label><input class="form-input" type="datetime-local" [(ngModel)]="newAssignment.shift_start" /></div>
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
    const revenue = prompt('Enter total revenue collected (KES):');
    if (revenue) {
      this.api.completeAssignment(a.id, Math.round(parseFloat(revenue) * 100)).subscribe({
        next: () => { this.toast.success('Assignment completed & earnings credited'); this.load(); },
      });
    }
  }

  cancelAssignment(a: Assignment): void {
    const reason = prompt('Reason for cancellation:');
    if (reason) {
      this.api.cancelAssignment(a.id, reason).subscribe({
        next: () => { this.toast.success('Assignment cancelled'); this.load(); },
      });
    }
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
