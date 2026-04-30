import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { Vehicle, SACCO, Route as AppRoute } from '../../../core/models';

@Component({
  selector: 'app-vehicle-detail',
  standalone: true,
  imports: [CommonModule, FormsModule, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <button class="btn btn-ghost btn-sm" (click)="goBack()" style="margin-bottom:var(--space-xs);"><span class="material-icons-round" style="font-size:16px;">arrow_back</span> Back to Fleet</button>
          <h1 class="page-title">{{ vehicle()?.registration_no || 'Vehicle Details' }}</h1>
          <p class="page-subtitle">{{ vehicle()?.vehicle_type }} — {{ vehicle()?.capacity }} pax capacity</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-secondary" (click)="openEdit()" id="btn-edit-vehicle"><span class="material-icons-round">edit</span> Edit</button>
        </div>
      </div>

      @if (loading()) {
        @for (i of [1,2]; track i) { <div class="skeleton" style="height:100px;margin:8px 0;border-radius:var(--radius-lg);"></div> }
      } @else if (vehicle(); as v) {
        <!-- Summary cards -->
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));">
          <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">directions_bus</span></div><div class="stat-value" style="font-size:1rem;">{{ v.registration_no }}</div><div class="stat-label">Registration</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(168,85,247,0.12);color:#a855f7;"><span class="material-icons-round">category</span></div><div class="stat-value">{{ v.vehicle_type }}</div><div class="stat-label">Type</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(255,184,0,0.12);color:#ffb800;"><span class="material-icons-round">business</span></div><div class="stat-value" style="font-size:0.9rem;">{{ saccoName() }}</div><div class="stat-label">SACCO</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">route</span></div><div class="stat-value" style="font-size:0.9rem;">{{ routeName() }}</div><div class="stat-label">Route</div></div>
        </div>

        <!-- Detail card -->
        <div class="glass-card" style="margin-top:var(--space-lg);">
          <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Vehicle Information</h3>
          <div class="detail-grid">
            <div class="detail-row"><span class="detail-label">Registration No.</span><span class="detail-value" style="font-weight:600;color:var(--color-text-primary);">{{ v.registration_no }}</span></div>
            <div class="detail-row"><span class="detail-label">Vehicle Type</span><span class="detail-value"><span class="badge badge-accent">{{ v.vehicle_type }}</span></span></div>
            <div class="detail-row"><span class="detail-label">Capacity</span><span class="detail-value">{{ v.capacity }} passengers</span></div>
            <div class="detail-row"><span class="detail-label">SACCO</span><span class="detail-value">{{ saccoName() }}</span></div>
            <div class="detail-row"><span class="detail-label">Assigned Route</span><span class="detail-value">{{ routeName() }}</span></div>
            <div class="detail-row"><span class="detail-label">Status</span><span class="detail-value"><span class="badge" [ngClass]="v.is_active?'badge-success':'badge-danger'">{{ v.is_active?'Active':'Inactive' }}</span></span></div>
            <div class="detail-row"><span class="detail-label">Registered</span><span class="detail-value">{{ v.created_at | relativeTime }}</span></div>
          </div>
        </div>
      }

      <!-- Edit Modal -->
      @if (showEdit()) {
        <div class="modal-backdrop" (click)="showEdit.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Edit Vehicle</h3><button class="btn-ghost" (click)="showEdit.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <label class="form-label">Registration No.</label><input class="form-input" [(ngModel)]="editForm.registration_no" id="edit-reg">
            <label class="form-label" style="margin-top:var(--space-sm);">Type</label>
            <select class="form-select" [(ngModel)]="editForm.vehicle_type" id="edit-type"><option value="MATATU">Matatu</option><option value="BODA">Boda Boda</option><option value="TUK_TUK">Tuk Tuk</option></select>
            <label class="form-label" style="margin-top:var(--space-sm);">Capacity</label><input class="form-input" type="number" [(ngModel)]="editForm.capacity" id="edit-capacity">
            <label class="form-label" style="margin-top:var(--space-sm);">Route</label>
            <select class="form-select" [(ngModel)]="editForm.route_id" id="edit-route">
              <option value="">— No route —</option>
              @for (r of routes(); track r.id) { <option [value]="r.id">{{ r.name }} ({{ r.code }})</option> }
            </select>
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
export class VehicleDetailComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  vehicle = signal<Vehicle | null>(null);
  saccoName = signal('—');
  routeName = signal('—');
  routes = signal<AppRoute[]>([]);
  loading = signal(true);
  showEdit = signal(false);
  submitting = signal(false);
  vehicleId = '';

  editForm = { registration_no: '', vehicle_type: 'MATATU', capacity: 14, route_id: '' };

  ngOnInit(): void {
    this.vehicleId = this.route.snapshot.paramMap.get('id') || '';
    if (this.vehicleId) {
      this.loadVehicle();
      this.api.getRoutes({ per_page: '200' }).subscribe({ next: r => this.routes.set(r.data) });
    }
  }

  goBack(): void { this.router.navigate(['/vehicles']); }

  loadVehicle(): void {
    this.loading.set(true);
    this.api.getVehicle(this.vehicleId).subscribe({
      next: (r) => {
        const v = r.data;
        this.vehicle.set(v);
        this.editForm = {
          registration_no: v.registration_no,
          vehicle_type: v.vehicle_type,
          capacity: v.capacity,
          route_id: v.route_id || '',
        };
        // Resolve SACCO name
        if (v.sacco_id) {
          this.api.getSACCO(v.sacco_id).subscribe({
            next: sr => this.saccoName.set(sr.data.name),
            error: () => this.saccoName.set(v.sacco_id.slice(0, 8) + '...'),
          });
        }
        // Resolve route name
        if (v.route_id) {
          this.api.getRoute(v.route_id).subscribe({
            next: rr => this.routeName.set(rr.data.name),
            error: () => this.routeName.set('—'),
          });
        }
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  openEdit(): void { this.showEdit.set(true); }

  submitEdit(): void {
    this.submitting.set(true);
    const data: Record<string, unknown> = {
      registration_no: this.editForm.registration_no,
      vehicle_type: this.editForm.vehicle_type,
      capacity: this.editForm.capacity,
    };
    if (this.editForm.route_id) data['route_id'] = this.editForm.route_id;
    this.api.updateVehicle(this.vehicleId, data).subscribe({
      next: () => { this.toast.success('Vehicle updated'); this.showEdit.set(false); this.submitting.set(false); this.loadVehicle(); },
      error: () => this.submitting.set(false),
    });
  }
}
