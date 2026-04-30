import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { PayrollRun, SACCO } from '../../../core/models';

@Component({
  selector: 'app-payroll-list', standalone: true, imports: [CommonModule, FormsModule, CurrencyKesPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Payroll & Compliance</h1><p class="page-subtitle">Process payroll runs with statutory deductions (SHA, NSSF, Housing Levy)</p></div>
        <div class="page-actions"><button class="btn btn-ghost" (click)="viewRates()" style="color:var(--color-text-muted);"><span class="material-icons-round">gavel</span> Statutory Rates</button><button class="btn btn-primary" (click)="showModal.set(true)" id="btn-create-payroll"><span class="material-icons-round">add</span> New Payroll Run</button></div>
      </div>
      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:56px;margin:4px 0;"></div>} }
      @else if (items().length === 0) { <div class="empty-state"><span class="material-icons-round empty-icon">receipt_long</span><div class="empty-title">No payroll runs</div><div class="empty-description">Create your first payroll run to process statutory deductions and compliance.</div></div> }
      @else {
        <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Period</th><th>Status</th><th>Gross</th><th>Deductions</th><th>Net</th><th>Entries</th><th>Actions</th></tr></thead>
          <tbody>@for(p of items();track p.id){<tr>
            <td style="color:var(--color-text-primary);font-weight:500;">{{p.period_start|date:'mediumDate'}} — {{p.period_end|date:'mediumDate'}}</td>
            <td><span class="badge" [ngClass]="statusBadge(p.status)">{{p.status}}</span></td>
            <td>{{p.total_gross_cents|currencyKes}}</td>
            <td class="text-danger">{{p.total_deductions_cents|currencyKes}}</td>
            <td style="font-weight:600;">{{p.total_net_cents|currencyKes}}</td>
            <td>{{p.entry_count}}</td>
            <td>
              <div style="display:flex;gap:4px;flex-wrap:wrap;">
                <button class="btn btn-sm btn-ghost" (click)="viewRun(p, $event)" style="color:var(--color-accent);">View</button>
                @if(p.status==='DRAFT'){<button class="btn btn-sm btn-secondary" (click)="processRun(p, $event)">Process</button>}
                @if(p.status==='PROCESSED'){<button class="btn btn-primary btn-sm" (click)="approveRun(p, $event)">Approve</button>}
                @if(p.status==='APPROVED'){<button class="btn btn-primary btn-sm" (click)="submitRun(p, $event)">Submit</button>}
              </div>
            </td>
          </tr>}</tbody></table></div>
      }
      @if (showModal()) {
        <div class="modal-backdrop" (click)="showModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Create Payroll Run</h3><button class="btn btn-ghost btn-icon" (click)="showModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group"><label class="form-label">SACCO</label>
              <select class="form-select" [(ngModel)]="form.sacco_id" id="select-sacco">
                <option value="">— Select SACCO —</option>
                @for (s of saccos(); track s.id) { <option [value]="s.id">{{ s.name }}</option> }
              </select>
            </div>
            <div class="form-group"><label class="form-label">Period Start</label><input class="form-input" type="date" [(ngModel)]="form.period_start" /></div>
            <div class="form-group"><label class="form-label">Period End</label><input class="form-input" type="date" [(ngModel)]="form.period_end" /></div>
          </div>
          <div class="modal-footer"><button class="btn btn-secondary" (click)="showModal.set(false)">Cancel</button><button class="btn btn-primary" (click)="create()" [disabled]="creating()">{{creating()?'Creating...':'Create Run'}}</button></div>
        </div></div>
      }
    </div>`,
  styles: [`.text-danger { color: #ef4444; }`],
})
export class PayrollListComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private router = inject(Router);
  items = signal<PayrollRun[]>([]); loading = signal(true); showModal = signal(false); creating = signal(false);
  saccos = signal<SACCO[]>([]);
  form = { sacco_id: '', period_start: '', period_end: '' };

  ngOnInit() {
    this.load();
    this.api.getSACCOs().subscribe({ next: r => this.saccos.set(r.data) });
  }

  load() { this.loading.set(true); this.api.getPayrollRuns().subscribe({ next: r => { this.items.set(r.data); this.loading.set(false); }, error: () => this.loading.set(false) }); }

  create() {
    this.creating.set(true);
    this.api.createPayrollRun(this.form).subscribe({
      next: () => { this.toast.success('Payroll run created'); this.showModal.set(false); this.creating.set(false); this.form = { sacco_id: '', period_start: '', period_end: '' }; this.load(); },
      error: () => this.creating.set(false),
    });
  }

  viewRun(p: PayrollRun, e: Event) { e.stopPropagation(); this.router.navigate(['/payroll', p.id]); }
  viewRates() { this.router.navigate(['/statutory-rates']); }
  processRun(p: PayrollRun, e: Event) { e.stopPropagation(); this.api.processPayrollRun(p.id).subscribe({ next: () => { this.toast.success('Payroll processed'); this.load(); } }); }
  approveRun(p: PayrollRun, e: Event) { e.stopPropagation(); this.api.approvePayrollRun(p.id).subscribe({ next: () => { this.toast.success('Payroll approved'); this.load(); } }); }
  submitRun(p: PayrollRun, e: Event) { e.stopPropagation(); this.api.submitPayrollRun(p.id).subscribe({ next: () => { this.toast.success('Payroll submitted to PerPay'); this.load(); } }); }
  statusBadge(s: string) { return s === 'COMPLETED' || s === 'APPROVED' ? 'badge-success' : s === 'FAILED' ? 'badge-danger' : s === 'SUBMITTED' ? 'badge-info' : 'badge-warning'; }
}
