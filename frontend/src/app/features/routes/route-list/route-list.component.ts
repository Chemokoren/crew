import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Route } from '../../../core/models';

@Component({
  selector: 'app-route-list', standalone: true, imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Route Management</h1><p class="page-subtitle">Define transport corridors and routes</p></div>
        <div class="page-actions"><button class="btn btn-primary" (click)="showModal.set(true)" id="btn-add-route"><span class="material-icons-round">add</span> Add Route</button></div>
      </div>
      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:56px;margin:4px 0;"></div>} }
      @else if (items().length === 0) { <div class="empty-state"><span class="material-icons-round empty-icon">route</span><div class="empty-title">No routes defined</div></div> }
      @else {
        <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Code</th><th>Name</th><th>Origin → Destination</th><th>Status</th><th>Actions</th></tr></thead>
          <tbody>@for(r of items();track r.id){<tr>
            <td><code class="text-accent">{{r.code}}</code></td>
            <td style="color:var(--color-text-primary);font-weight:500;">{{r.name}}</td>
            <td>{{r.origin}} → {{r.destination}}</td>
            <td><span class="badge" [ngClass]="r.is_active?'badge-success':'badge-danger'">{{r.is_active?'Active':'Inactive'}}</span></td>
            <td><button class="btn btn-ghost btn-sm" (click)="deleteRoute(r)">Delete</button></td>
          </tr>}</tbody></table></div>
      }
      @if (showModal()) {
        <div class="modal-backdrop" (click)="showModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Add Route</h3><button class="btn btn-ghost btn-icon" (click)="showModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group"><label class="form-label">Name</label><input class="form-input" [(ngModel)]="form.name" placeholder="Nairobi CBD ↔ Ongata Rongai" /></div>
            <div class="form-group"><label class="form-label">Code</label><input class="form-input" [(ngModel)]="form.code" placeholder="NRB-RNG" /></div>
            <div class="form-group"><label class="form-label">Origin</label><input class="form-input" [(ngModel)]="form.origin" placeholder="Nairobi CBD" /></div>
            <div class="form-group"><label class="form-label">Destination</label><input class="form-input" [(ngModel)]="form.destination" placeholder="Ongata Rongai" /></div>
          </div>
          <div class="modal-footer"><button class="btn btn-secondary" (click)="showModal.set(false)">Cancel</button><button class="btn btn-primary" (click)="create()" [disabled]="creating()">{{creating()?'Creating...':'Add Route'}}</button></div>
        </div></div>
      }
    </div>`,
})
export class RouteListComponent implements OnInit {
  private api = inject(ApiService); private toast = inject(ToastService);
  items = signal<Route[]>([]); loading = signal(true); showModal = signal(false); creating = signal(false);
  form = {name:'',code:'',origin:'',destination:''};
  ngOnInit() { this.load(); }
  load() { this.loading.set(true); this.api.getRoutes().subscribe({next:r=>{this.items.set(r.data);this.loading.set(false);},error:()=>this.loading.set(false)}); }
  create() { this.creating.set(true); this.api.createRoute(this.form).subscribe({next:()=>{this.toast.success('Route created');this.showModal.set(false);this.creating.set(false);this.load();},error:()=>this.creating.set(false)}); }
  deleteRoute(r:Route) { if(confirm(`Delete ${r.name}?`)){this.api.deleteRoute(r.id).subscribe({next:()=>{this.toast.success('Route deleted');this.load();}}); } }
}
