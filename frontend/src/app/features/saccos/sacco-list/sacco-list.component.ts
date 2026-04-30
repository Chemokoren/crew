import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { SACCO } from '../../../core/models';

@Component({
  selector: 'app-sacco-list', standalone: true, imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">SACCO Management</h1><p class="page-subtitle">Manage transport cooperatives and their operations</p></div>
        <div class="page-actions"><button class="btn btn-primary" (click)="showModal.set(true)" id="btn-add-sacco"><span class="material-icons-round">add</span> Add SACCO</button></div>
      </div>
      <div class="filters-bar">
        <div class="search-input-wrapper"><span class="material-icons-round search-icon">search</span><input class="form-input" placeholder="Search SACCOs..." [(ngModel)]="search" (ngModelChange)="load()" id="sacco-search" /></div>
      </div>
      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:56px;margin:4px 0;"></div>} }
      @else if (items().length === 0) { <div class="empty-state"><span class="material-icons-round empty-icon">business</span><div class="empty-title">No SACCOs found</div></div> }
      @else {
        <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Name</th><th>Reg No.</th><th>County</th><th>Contact</th><th>Status</th><th>Actions</th></tr></thead>
          <tbody>@for(s of items();track s.id){<tr style="cursor:pointer;" (click)="viewSACCO(s)">
            <td style="color:var(--color-text-primary);font-weight:500;">{{s.name}}</td>
            <td><code class="text-accent">{{s.registration_number}}</code></td>
            <td>{{s.county}}</td><td>{{s.contact_phone}}</td>
            <td><span class="badge" [ngClass]="s.is_active?'badge-success':'badge-danger'">{{s.is_active?'Active':'Inactive'}}</span></td>
            <td style="display:flex;gap:var(--space-xs);">
              <button class="btn btn-ghost btn-sm" (click)="viewSACCO(s);$event.stopPropagation()" id="view-sacco-{{s.id}}">View</button>
              <button class="btn btn-ghost btn-sm" style="color:var(--color-danger);" (click)="deleteSACCO(s);$event.stopPropagation()" id="delete-sacco-{{s.id}}">Delete</button>
            </td>
          </tr>}</tbody></table></div>
      }
      @if (showModal()) {
        <div class="modal-backdrop" (click)="showModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Add SACCO</h3><button class="btn btn-ghost btn-icon" (click)="showModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group"><label class="form-label">Name</label><input class="form-input" [(ngModel)]="form.name" placeholder="Matatu SACCO" /></div>
            <div class="form-group"><label class="form-label">Registration Number</label><input class="form-input" [(ngModel)]="form.registration_number" placeholder="CS/2024/001" /></div>
            <div class="form-group"><label class="form-label">County</label><input class="form-input" [(ngModel)]="form.county" placeholder="Nairobi" /></div>
            <div class="form-group"><label class="form-label">Contact Phone</label><input class="form-input" [(ngModel)]="form.contact_phone" placeholder="+254..." /></div>
            <div class="form-group"><label class="form-label">Contact Email</label><input class="form-input" type="email" [(ngModel)]="form.contact_email" placeholder="admin@sacco.co.ke" /></div>
          </div>
          <div class="modal-footer"><button class="btn btn-secondary" (click)="showModal.set(false)">Cancel</button><button class="btn btn-primary" (click)="create()" [disabled]="creating()">{{creating()?'Creating...':'Create SACCO'}}</button></div>
        </div></div>
      }
    </div>`,
})
export class SaccoListComponent implements OnInit {
  private api = inject(ApiService); private toast = inject(ToastService); private router = inject(Router);
  items = signal<SACCO[]>([]); loading = signal(true); showModal = signal(false); creating = signal(false); search = '';
  form = { name:'', registration_number:'', county:'', contact_phone:'', contact_email:'' };
  ngOnInit() { this.load(); }
  load() { this.loading.set(true); this.api.getSACCOs(this.search?{search:this.search}:undefined).subscribe({next:r=>{this.items.set(r.data);this.loading.set(false);},error:()=>this.loading.set(false)}); }
  create() { this.creating.set(true); this.api.createSACCO(this.form).subscribe({next:()=>{this.toast.success('SACCO created');this.showModal.set(false);this.creating.set(false);this.load();},error:()=>this.creating.set(false)}); }
  deleteSACCO(s:SACCO) { if(confirm(`Delete ${s.name}?`)){this.api.deleteSACCO(s.id).subscribe({next:()=>{this.toast.success('SACCO deleted');this.load();}}); } }
  viewSACCO(s:SACCO) { this.router.navigate(['/saccos', s.id]); }
}
