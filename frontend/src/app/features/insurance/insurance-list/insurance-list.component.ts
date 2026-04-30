import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../../core/services/api.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { InsurancePolicy } from '../../../core/models';

@Component({
  selector: 'app-insurance-list', standalone: true,
  imports: [CommonModule, CurrencyKesPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Insurance</h1><p class="page-subtitle">Manage insurance policies and premium deductions</p></div>
      </div>
      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:56px;margin:4px 0;"></div>} }
      @else if (items().length === 0) {
        <div class="empty-state"><span class="material-icons-round empty-icon">health_and_safety</span>
          <div class="empty-title">No insurance policies</div>
        </div>
      } @else {
        <div class="data-table-wrapper"><table class="data-table"><thead><tr>
          <th>Provider</th><th>Type</th><th>Premium</th><th>Frequency</th><th>Period</th><th>Status</th>
        </tr></thead><tbody>
          @for(p of items();track p.id){<tr>
            <td style="font-weight:500;color:var(--color-text-primary);">{{p.provider}}</td>
            <td><span class="badge badge-accent">{{p.policy_type}}</span></td>
            <td>{{p.premium_cents|currencyKes}}</td>
            <td>{{p.frequency}}</td>
            <td style="font-size:0.8125rem;">{{p.start_date|date:'mediumDate'}} — {{p.end_date|date:'mediumDate'}}</td>
            <td><span class="badge" [ngClass]="p.status==='ACTIVE'?'badge-success':'badge-danger'">{{p.status}}</span></td>
          </tr>}</tbody></table></div>
      }
    </div>`,
})
export class InsuranceListComponent implements OnInit {
  private api = inject(ApiService);
  items = signal<InsurancePolicy[]>([]); loading = signal(true);
  ngOnInit() { this.load(); }
  load() { this.loading.set(true); this.api.getInsurancePolicies().subscribe({next:r=>{this.items.set(r.data);this.loading.set(false);},error:()=>this.loading.set(false)}); }
}
