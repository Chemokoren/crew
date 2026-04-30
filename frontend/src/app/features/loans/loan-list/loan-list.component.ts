import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { LoanApplication } from '../../../core/models';

@Component({
  selector: 'app-loan-list', standalone: true,
  imports: [CommonModule, FormsModule, CurrencyKesPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Loans</h1><p class="page-subtitle">Apply, track, and manage loans</p></div>
      </div>
      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:56px;margin:4px 0;"></div>} }
      @else if (items().length === 0) {
        <div class="empty-state"><span class="material-icons-round empty-icon">savings</span>
          <div class="empty-title">No loan applications</div>
          <div class="empty-description">Apply for a loan based on your earnings history and credit score.</div>
        </div>
      } @else {
        <div class="data-table-wrapper"><table class="data-table"><thead><tr>
          <th>Category</th><th>Amount</th><th>Tenure</th><th>Status</th><th>Repaid</th><th>Created</th><th>Actions</th>
        </tr></thead><tbody>
          @for(l of items();track l.id){<tr>
            <td><span class="badge badge-accent">{{l.category}}</span></td>
            <td style="font-weight:600;">{{l.amount_cents|currencyKes}}</td>
            <td>{{l.tenure_days}} days</td>
            <td><span class="badge" [ngClass]="statusBadge(l.status)">{{l.status}}</span></td>
            <td>{{l.total_repaid_cents|currencyKes}}</td>
            <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{l.created_at|date:'mediumDate'}}</td>
            <td>
              @if(l.status==='DISBURSED'){<button class="btn btn-sm btn-primary" (click)="repay(l)">Repay</button>}
            </td>
          </tr>}</tbody></table></div>
      }
    </div>`,
})
export class LoanListComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  items = signal<LoanApplication[]>([]); loading = signal(true);
  ngOnInit() { this.load(); }
  load() { this.loading.set(true); this.api.getLoans().subscribe({next:r=>{this.items.set(r.data);this.loading.set(false);},error:()=>this.loading.set(false)}); }
  repay(l: LoanApplication) {
    const amt = prompt('Repayment amount (KES):');
    if (amt) { this.api.repayLoan(l.id, Math.round(parseFloat(amt)*100)).subscribe({next:()=>{this.toast.success('Repayment processed');this.load();}}); }
  }
  statusBadge(s:string) { return s==='REPAID'||s==='APPROVED'?'badge-success':s==='REJECTED'||s==='DEFAULTED'?'badge-danger':s==='DISBURSED'?'badge-info':'badge-warning'; }
}
