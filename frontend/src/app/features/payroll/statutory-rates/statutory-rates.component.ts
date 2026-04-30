import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../../core/services/api.service';
import { StatutoryRate } from '../../../core/models';

@Component({
  selector: 'app-statutory-rates',
  standalone: true,
  imports: [CommonModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Statutory Rates</h1>
          <p class="page-subtitle">Kenya statutory deduction rates — SHA, NSSF, Housing Levy</p>
        </div>
      </div>

      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:80px;margin:8px 0;border-radius:var(--radius-lg);"></div>} }
      @else if (rates().length === 0) {
        <div class="empty-state"><span class="material-icons-round empty-icon">gavel</span><div class="empty-title">No statutory rates configured</div><div class="empty-subtitle">Contact the system administrator to seed statutory rates.</div></div>
      } @else {

        <!-- Rate Cards -->
        <div class="rates-grid">
          @for (r of rates(); track r.id) {
            <div class="rate-card glass-card">
              <div class="rate-header">
                <div class="rate-icon" [style.background]="iconBg(r.name)" [style.color]="iconColor(r.name)">
                  <span class="material-icons-round">{{ icon(r.name) }}</span>
                </div>
                <span class="badge" [ngClass]="r.is_active ? 'badge-success' : 'badge-danger'">{{ r.is_active ? 'Active' : 'Inactive' }}</span>
              </div>
              <div class="rate-name">{{ r.name }}</div>
              <div class="rate-value">{{ formatRate(r) }}</div>
              <div class="rate-meta">
                <span><span class="material-icons-round" style="font-size:14px;">category</span> {{ r.rate_type }}</span>
                <span><span class="material-icons-round" style="font-size:14px;">event</span> Since {{ r.effective_from | date:'mediumDate' }}</span>
              </div>
            </div>
          }
        </div>

        <!-- Table View -->
        <div class="glass-card" style="margin-top:var(--space-lg);">
          <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">All Rates</h3>
          <div class="data-table-wrapper"><table class="data-table">
            <thead><tr><th>Name</th><th>Rate</th><th>Type</th><th>Effective From</th><th>Status</th></tr></thead>
            <tbody>
              @for (r of rates(); track r.id) {
                <tr>
                  <td style="font-weight:500;color:var(--color-text-primary);">{{ r.name }}</td>
                  <td style="font-weight:600;">{{ formatRate(r) }}</td>
                  <td><span class="badge badge-accent">{{ r.rate_type }}</span></td>
                  <td>{{ r.effective_from | date:'mediumDate' }}</td>
                  <td><span class="badge" [ngClass]="r.is_active ? 'badge-success' : 'badge-danger'">{{ r.is_active ? 'Active' : 'Inactive' }}</span></td>
                </tr>
              }
            </tbody>
          </table></div>
        </div>
      }
    </div>
  `,
  styles: [`
    .rates-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: var(--space-md); }
    .rate-card { padding: var(--space-lg) !important; display: flex; flex-direction: column; gap: var(--space-sm); }
    .rate-header { display: flex; justify-content: space-between; align-items: center; }
    .rate-icon {
      width: 44px; height: 44px; border-radius: var(--radius-md);
      display: flex; align-items: center; justify-content: center;
      .material-icons-round { font-size: 22px; }
    }
    .rate-name { font-size: 0.875rem; color: var(--color-text-muted); font-weight: 500; margin-top: var(--space-xs); }
    .rate-value {
      font-family: var(--font-heading); font-size: 1.75rem; font-weight: 800;
      background: var(--gradient-accent); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text;
    }
    .rate-meta {
      display: flex; gap: var(--space-md); margin-top: var(--space-xs);
      font-size: 0.7rem; color: var(--color-text-muted);
      span { display: inline-flex; align-items: center; gap: 4px; }
    }
  `]
})
export class StatutoryRatesComponent implements OnInit {
  private api = inject(ApiService);
  rates = signal<StatutoryRate[]>([]);
  loading = signal(true);

  ngOnInit(): void {
    this.api.getStatutoryRates().subscribe({
      next: r => { this.rates.set(r.data || []); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  formatRate(r: StatutoryRate): string {
    if (r.rate_type === 'PERCENTAGE') return (r.rate * 100).toFixed(2) + '%';
    if (r.rate_type === 'FIXED') return 'KES ' + r.rate.toLocaleString();
    return r.rate.toString();
  }

  icon(name: string): string {
    const n = name.toLowerCase();
    if (n.includes('sha') || n.includes('health')) return 'local_hospital';
    if (n.includes('nssf') || n.includes('social')) return 'security';
    if (n.includes('housing') || n.includes('levy')) return 'home';
    return 'gavel';
  }

  iconBg(name: string): string {
    const n = name.toLowerCase();
    if (n.includes('sha') || n.includes('health')) return 'rgba(239,68,68,0.12)';
    if (n.includes('nssf') || n.includes('social')) return 'rgba(168,85,247,0.12)';
    if (n.includes('housing') || n.includes('levy')) return 'rgba(251,191,36,0.12)';
    return 'rgba(0,210,255,0.12)';
  }

  iconColor(name: string): string {
    const n = name.toLowerCase();
    if (n.includes('sha') || n.includes('health')) return '#ef4444';
    if (n.includes('nssf') || n.includes('social')) return '#a855f7';
    if (n.includes('housing') || n.includes('levy')) return '#fbbf24';
    return 'var(--color-accent)';
  }
}
