import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { CurrencyKesPipe } from '../../shared/pipes/currency-kes.pipe';
import { Earning } from '../../core/models';

@Component({
  selector: 'app-earnings', standalone: true, imports: [CommonModule, FormsModule, CurrencyKesPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Earnings</h1><p class="page-subtitle">Track your earnings across shifts</p></div>
      </div>
      <div class="filters-bar">
        <input class="form-input" type="date" [(ngModel)]="dateFrom" (ngModelChange)="load()" style="max-width:180px;" />
        <input class="form-input" type="date" [(ngModel)]="dateTo" (ngModelChange)="load()" style="max-width:180px;" />
      </div>
      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:48px;margin:4px 0;"></div>} }
      @else if (items().length === 0) { <div class="empty-state"><span class="material-icons-round empty-icon">trending_up</span><div class="empty-title">No earnings recorded</div></div> }
      @else {
        <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Date</th><th>Type</th><th>Amount</th></tr></thead>
          <tbody>@for(e of items();track e.id){<tr>
            <td>{{e.created_at|date:'medium'}}</td>
            <td><span class="badge badge-accent">{{e.earning_type}}</span></td>
            <td style="font-weight:600;color:var(--color-success);">{{e.amount_cents|currencyKes}}</td>
          </tr>}</tbody></table></div>
      }
    </div>`,
})
export class EarningsComponent implements OnInit {
  private api = inject(ApiService);
  items = signal<Earning[]>([]); loading = signal(true); dateFrom = ''; dateTo = '';
  ngOnInit() { this.load(); }
  load() {
    this.loading.set(true);
    const p: Record<string,string> = {per_page:'50'};
    if(this.dateFrom) p['date_from']=this.dateFrom;
    if(this.dateTo) p['date_to']=this.dateTo;
    this.api.getEarnings(p).subscribe({next:r=>{this.items.set(r.data);this.loading.set(false);},error:()=>this.loading.set(false)});
  }
}
