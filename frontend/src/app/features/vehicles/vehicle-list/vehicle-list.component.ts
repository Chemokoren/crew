import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Vehicle } from '../../../core/models';

@Component({
  selector: 'app-vehicle-list', standalone: true, imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Fleet Management</h1><p class="page-subtitle">Manage matatus, boda bodas, and tuk tuks</p></div>
        <div class="page-actions"><button class="btn btn-primary" (click)="showModal.set(true)" id="btn-add-vehicle"><span class="material-icons-round">add</span> Add Vehicle</button></div>
      </div>
      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:56px;margin:4px 0;"></div>} }
      @else if (items().length === 0) { <div class="empty-state"><span class="material-icons-round empty-icon">directions_bus</span><div class="empty-title">No vehicles registered</div></div> }
      @else {
        <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Reg No.</th><th>Type</th><th>Capacity</th><th>Status</th><th>Actions</th></tr></thead>
          <tbody>@for(v of items();track v.id){<tr>
            <td style="color:var(--color-text-primary);font-weight:600;">{{v.registration_no}}</td>
            <td><span class="badge badge-accent">{{v.vehicle_type}}</span></td>
            <td>{{v.capacity}} pax</td>
            <td><span class="badge" [ngClass]="v.is_active?'badge-success':'badge-danger'">{{v.is_active?'Active':'Inactive'}}</span></td>
            <td><button class="btn btn-ghost btn-sm" (click)="deleteVehicle(v)" id="del-vehicle-{{v.id}}">Delete</button></td>
          </tr>}</tbody></table></div>
      }
      @if (showModal()) {
        <div class="modal-backdrop" (click)="showModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Add Vehicle</h3><button class="btn btn-ghost btn-icon" (click)="showModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group"><label class="form-label">Registration Number</label><input class="form-input" [(ngModel)]="form.registration_no" placeholder="KAA 123A" /></div>
            <div class="form-group"><label class="form-label">SACCO ID</label><input class="form-input" [(ngModel)]="form.sacco_id" placeholder="UUID" /></div>
            <div class="form-group"><label class="form-label">Type</label><select class="form-select" [(ngModel)]="form.vehicle_type"><option value="MATATU">Matatu</option><option value="BODA">Boda Boda</option><option value="TUK_TUK">Tuk Tuk</option></select></div>
            <div class="form-group"><label class="form-label">Capacity</label><input class="form-input" type="number" [(ngModel)]="form.capacity" placeholder="14" /></div>
          </div>
          <div class="modal-footer"><button class="btn btn-secondary" (click)="showModal.set(false)">Cancel</button><button class="btn btn-primary" (click)="create()" [disabled]="creating()">{{creating()?'Creating...':'Add Vehicle'}}</button></div>
        </div></div>
      }
    </div>`,
})
export class VehicleListComponent implements OnInit {
  private api = inject(ApiService); private toast = inject(ToastService);
  items = signal<Vehicle[]>([]); loading = signal(true); showModal = signal(false); creating = signal(false);
  form = {registration_no:'',sacco_id:'',vehicle_type:'MATATU',capacity:14};
  ngOnInit() { this.load(); }
  load() { this.loading.set(true); this.api.getVehicles().subscribe({next:r=>{this.items.set(r.data);this.loading.set(false);},error:()=>this.loading.set(false)}); }
  create() { this.creating.set(true); this.api.createVehicle(this.form as any).subscribe({next:()=>{this.toast.success('Vehicle added');this.showModal.set(false);this.creating.set(false);this.load();},error:()=>this.creating.set(false)}); }
  deleteVehicle(v:Vehicle) { if(confirm(`Delete ${v.registration_no}?`)){this.api.deleteVehicle(v.id).subscribe({next:()=>{this.toast.success('Vehicle deleted');this.load();}}); } }
}
