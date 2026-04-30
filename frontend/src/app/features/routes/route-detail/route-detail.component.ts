import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { Route as AppRoute, Vehicle } from '../../../core/models';

@Component({
  selector: 'app-route-detail',
  standalone: true,
  imports: [CommonModule, FormsModule, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <button class="btn btn-ghost btn-sm" (click)="goBack()" style="margin-bottom:var(--space-xs);"><span class="material-icons-round" style="font-size:16px;">arrow_back</span> Back to Routes</button>
          <h1 class="page-title">{{ appRoute()?.name || 'Route Details' }}</h1>
          <p class="page-subtitle"><code class="text-accent">{{ appRoute()?.code }}</code> — {{ appRoute()?.origin }} → {{ appRoute()?.destination }}</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-secondary" (click)="openEdit()" id="btn-edit-route"><span class="material-icons-round">edit</span> Edit</button>
        </div>
      </div>

      @if (loading()) {
        @for (i of [1,2]; track i) { <div class="skeleton" style="height:100px;margin:8px 0;border-radius:var(--radius-lg);"></div> }
      } @else if (appRoute(); as r) {
        <!-- Summary cards -->
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));">
          <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">route</span></div><div class="stat-value" style="font-size:0.95rem;">{{ r.origin }}</div><div class="stat-label">Origin</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">place</span></div><div class="stat-value" style="font-size:0.95rem;">{{ r.destination }}</div><div class="stat-label">Destination</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(168,85,247,0.12);color:#a855f7;"><span class="material-icons-round">straighten</span></div><div class="stat-value">{{ r.distance_km ? r.distance_km + ' km' : '—' }}</div><div class="stat-label">Distance</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(255,184,0,0.12);color:#ffb800;"><span class="material-icons-round">directions_bus</span></div><div class="stat-value">{{ assignedVehicles().length }}</div><div class="stat-label">Vehicles</div></div>
        </div>

        <!-- Detail card -->
        <div class="glass-card" style="margin-top:var(--space-lg);">
          <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Route Information</h3>
          <div class="detail-grid">
            <div class="detail-row"><span class="detail-label">Name</span><span class="detail-value" style="font-weight:600;">{{ r.name }}</span></div>
            <div class="detail-row"><span class="detail-label">Code</span><span class="detail-value"><code class="text-accent">{{ r.code }}</code></span></div>
            <div class="detail-row"><span class="detail-label">Origin</span><span class="detail-value">{{ r.origin }}</span></div>
            <div class="detail-row"><span class="detail-label">Destination</span><span class="detail-value">{{ r.destination }}</span></div>
            <div class="detail-row"><span class="detail-label">Distance</span><span class="detail-value">{{ r.distance_km ? r.distance_km + ' km' : 'Not set' }}</span></div>
            <div class="detail-row"><span class="detail-label">Status</span><span class="detail-value"><span class="badge" [ngClass]="r.is_active?'badge-success':'badge-danger'">{{ r.is_active?'Active':'Inactive' }}</span></span></div>
            <div class="detail-row"><span class="detail-label">Created</span><span class="detail-value">{{ r.created_at | relativeTime }}</span></div>
          </div>
        </div>

        <!-- Assigned Vehicles -->
        <div class="glass-card" style="margin-top:var(--space-lg);">
          <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Assigned Vehicles</h3>
          @if (assignedVehicles().length === 0) {
            <div class="empty-state" style="padding:var(--space-lg);"><span class="material-icons-round empty-icon">directions_bus</span><div class="empty-title">No vehicles assigned</div><div class="empty-subtitle">Assign vehicles to this route from the Fleet page</div></div>
          } @else {
            <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Reg No.</th><th>Type</th><th>Capacity</th><th>Status</th></tr></thead>
              <tbody>@for (v of assignedVehicles(); track v.id) {
                <tr style="cursor:pointer;" (click)="goToVehicle(v.id)">
                  <td style="color:var(--color-text-primary);font-weight:600;">{{ v.registration_no }}</td>
                  <td><span class="badge badge-accent">{{ v.vehicle_type }}</span></td>
                  <td>{{ v.capacity }} pax</td>
                  <td><span class="badge" [ngClass]="v.is_active?'badge-success':'badge-danger'">{{ v.is_active?'Active':'Inactive' }}</span></td>
                </tr>
              }</tbody>
            </table></div>
          }
        </div>
      }

      <!-- Edit Modal -->
      @if (showEdit()) {
        <div class="modal-backdrop" (click)="showEdit.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Edit Route</h3><button class="btn-ghost" (click)="showEdit.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <label class="form-label">Name</label><input class="form-input" [(ngModel)]="editForm.name" id="edit-name">
            <label class="form-label" style="margin-top:var(--space-sm);">Code</label><input class="form-input" [(ngModel)]="editForm.code" id="edit-code">
            <label class="form-label" style="margin-top:var(--space-sm);">Origin</label><input class="form-input" [(ngModel)]="editForm.origin" id="edit-origin">
            <label class="form-label" style="margin-top:var(--space-sm);">Destination</label><input class="form-input" [(ngModel)]="editForm.destination" id="edit-dest">
            <label class="form-label" style="margin-top:var(--space-sm);">Distance (km)</label><input class="form-input" type="number" [(ngModel)]="editForm.distance_km" id="edit-distance">
          </div>
          <div class="modal-footer"><button class="btn btn-ghost" (click)="showEdit.set(false)">Cancel</button><button class="btn btn-primary" (click)="submitEdit()" [disabled]="submitting()" id="btn-submit-edit">{{ submitting()?'Saving...':'Save Changes' }}</button></div>
        </div></div>
      }
    </div>
  `,
  styles: [`
    .detail-grid { display: grid; gap: 0; }
    .detail-row { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; border-bottom: 1px solid var(--color-border); &:last-child { border-bottom: none; } }
    .detail-label { font-size: 0.8rem; color: var(--color-text-muted); font-weight: 500; }
    .detail-value { font-size: 0.875rem; color: var(--color-text-secondary); text-align: right; }
    .form-label { display: block; font-size: 0.8rem; font-weight: 500; color: var(--color-text-secondary); margin-bottom: 4px; }
  `]
})
export class RouteDetailComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  appRoute = signal<AppRoute | null>(null);
  assignedVehicles = signal<Vehicle[]>([]);
  loading = signal(true);
  showEdit = signal(false);
  submitting = signal(false);
  routeId = '';

  editForm = { name: '', code: '', origin: '', destination: '', distance_km: 0 };

  ngOnInit(): void {
    this.routeId = this.route.snapshot.paramMap.get('id') || '';
    if (this.routeId) { this.loadRoute(); this.loadVehicles(); }
  }

  goBack(): void { this.router.navigate(['/routes']); }
  goToVehicle(id: string): void { this.router.navigate(['/vehicles', id]); }

  loadRoute(): void {
    this.loading.set(true);
    this.api.getRoute(this.routeId).subscribe({
      next: (r) => {
        this.appRoute.set(r.data);
        this.editForm = {
          name: r.data.name, code: r.data.code,
          origin: r.data.origin, destination: r.data.destination,
          distance_km: r.data.distance_km || 0,
        };
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  loadVehicles(): void {
    // Fetch all vehicles and filter by route_id client-side (backend doesn't support route_id filter)
    this.api.getVehicles({ per_page: '200' }).subscribe({
      next: (r) => {
        this.assignedVehicles.set((r.data || []).filter(v => v.route_id === this.routeId));
      },
    });
  }

  openEdit(): void { this.showEdit.set(true); }

  submitEdit(): void {
    this.submitting.set(true);
    this.api.updateRoute(this.routeId, this.editForm).subscribe({
      next: () => { this.toast.success('Route updated'); this.showEdit.set(false); this.submitting.set(false); this.loadRoute(); },
      error: () => this.submitting.set(false),
    });
  }
}
