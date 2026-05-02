import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { FinancialProfile, ScoreFactor } from '../../../core/models';

@Component({
  selector: 'app-financial-profile',
  standalone: true,
  imports: [CommonModule, RouterModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Financial Profile</h1>
          <p class="page-subtitle">Cross-organization financial identity & credit profile</p>
        </div>
        <a [routerLink]="['/crew', crewId]" class="btn btn-ghost" id="btn-back-crew">
          <span class="material-icons-round">arrow_back</span> Back to Profile
        </a>
      </div>

      @if (loading()) {
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));">
          @for (i of [1,2,3,4]; track i) { <div class="skeleton" style="height:200px;border-radius:var(--radius-lg);"></div> }
        </div>
      } @else if (!profile()) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">account_balance</span>
          <div class="empty-title">Financial profile unavailable</div>
          <div class="empty-description">No financial profile data found for this crew member.</div>
        </div>
      } @else {
        <!-- Identity + Score Banner -->
        <div class="fp-banner glass-card">
          <div class="fp-identity">
            <div class="fp-avatar" [style.background]="avatarGradient()">
              {{ profile()!.full_name.charAt(0) }}
            </div>
            <div class="fp-info">
              <h2 class="fp-name">{{ profile()!.full_name }}</h2>
              <div class="fp-meta">
                <span class="badge badge-outline">ID: {{ profile()!.national_id }}</span>
                <span class="badge" [ngClass]="kycBadge()">{{ profile()!.kyc_status }}</span>
                @if (profile()!.primary_work_type) {
                  <span class="badge badge-accent">{{ profile()!.primary_work_type }}</span>
                }
              </div>
            </div>
          </div>
          <div class="fp-score-block">
            <div class="fp-score-ring" [style.--score-pct]="scorePercent()">
              <div class="fp-score-value">{{ profile()!.composite_score }}</div>
              <div class="fp-score-grade" [style.color]="gradeColor(profile()!.score_grade)">{{ profile()!.score_grade }}</div>
            </div>
            <div class="fp-score-label">Composite Score</div>
          </div>
        </div>

        <!-- Key Metrics -->
        <div class="fp-metrics-grid">
          <div class="glass-card fp-metric">
            <span class="material-icons-round fp-metric-icon" style="color:var(--color-success);">payments</span>
            <div class="fp-metric-value">KES {{ formatCents(profile()!.total_earnings_30d_cents) }}</div>
            <div class="fp-metric-label">Earnings (30d)</div>
            <div class="fp-metric-trend" [ngClass]="trendClass(profile()!.earning_trend)">
              <span class="material-icons-round">{{ trendIcon(profile()!.earning_trend) }}</span>
              {{ profile()!.earning_trend }}
            </div>
          </div>
          <div class="glass-card fp-metric">
            <span class="material-icons-round fp-metric-icon" style="color:var(--color-accent);">account_balance_wallet</span>
            <div class="fp-metric-value">KES {{ formatCents(profile()!.wallet_balance_cents) }}</div>
            <div class="fp-metric-label">Wallet Balance</div>
            <div class="fp-metric-sub">{{ (profile()!.savings_rate * 100).toFixed(0) }}% savings rate</div>
          </div>
          <div class="glass-card fp-metric">
            <span class="material-icons-round fp-metric-icon" style="color:var(--color-warning);">business_center</span>
            <div class="fp-metric-value">{{ profile()!.org_count }}</div>
            <div class="fp-metric-label">Organizations</div>
            <div class="fp-metric-sub">{{ profile()!.cross_org_tenure_months }} months total tenure</div>
          </div>
          <div class="glass-card fp-metric">
            <span class="material-icons-round fp-metric-icon" style="color:#22c55e;">verified</span>
            <div class="fp-metric-value">{{ profile()!.total_loans_completed }}</div>
            <div class="fp-metric-label">Loans Completed</div>
            <div class="fp-metric-sub">{{ (profile()!.on_time_repayment_rate * 100).toFixed(0) }}% on-time</div>
          </div>
        </div>

        <!-- Organization Profiles -->
        @if (profile()!.org_profiles.length) {
          <div class="glass-card fp-section">
            <h3 class="fp-section-title">
              <span class="material-icons-round">apartment</span> Organization History
            </h3>
            <div class="fp-org-grid">
              @for (org of profile()!.org_profiles; track org.org_id) {
                <div class="fp-org-card" [class.active]="org.is_active">
                  <div class="fp-org-header">
                    <div class="fp-org-name">{{ org.org_name }}</div>
                    <span class="badge" [ngClass]="industryBadge(org.industry)">{{ org.industry }}</span>
                  </div>
                  <div class="fp-org-details">
                    <div class="fp-org-row">
                      <span>Role</span><strong>{{ org.role || 'Member' }}</strong>
                    </div>
                    <div class="fp-org-row">
                      <span>Tenure</span><strong>{{ org.tenure_months }} months</strong>
                    </div>
                    <div class="fp-org-row">
                      <span>Earnings (30d)</span><strong>KES {{ formatCents(org.earnings_30d_cents) }}</strong>
                    </div>
                    <div class="fp-org-row">
                      <span>Assignments (30d)</span><strong>{{ org.assignment_count_30d }}</strong>
                    </div>
                  </div>
                  <div class="fp-org-status">
                    <span class="status-dot" [class.active]="org.is_active"></span>
                    {{ org.is_active ? 'Active' : 'Inactive' }}
                  </div>
                </div>
              }
            </div>
          </div>
        }

        <!-- Score Factors -->
        @if (profile()!.factors?.length) {
          <div class="glass-card fp-section">
            <h3 class="fp-section-title">
              <span class="material-icons-round">analytics</span> Score Breakdown
            </h3>
            <div class="factors-grid">
              @for (f of profile()!.factors; track f.name) {
                <div class="factor-item">
                  <div class="factor-header">
                    <span class="factor-name">{{ f.name }}</span>
                    <span class="factor-points" [class.positive]="f.impact === 'POSITIVE'" [class.negative]="f.impact === 'NEGATIVE'">
                      {{ f.points }} / {{ f.max_points }}
                    </span>
                  </div>
                  <div class="factor-bar-track">
                    <div class="factor-bar-fill" [style.width.%]="f.percentage * 100"
                         [class.fill-positive]="f.impact === 'POSITIVE'"
                         [class.fill-negative]="f.impact === 'NEGATIVE'"
                         [class.fill-neutral]="f.impact === 'NEUTRAL'">
                    </div>
                  </div>
                  <div class="factor-desc">{{ f.description }}</div>
                </div>
              }
            </div>
          </div>
        }

        <!-- Products Grid: Loans + Insurance -->
        <div class="fp-products-grid">
          @if (profile()!.available_loan_products?.length) {
            <div class="glass-card fp-section">
              <h3 class="fp-section-title">
                <span class="material-icons-round">credit_card</span> Available Loan Products
              </h3>
              <div class="fp-product-list">
                @for (p of profile()!.available_loan_products; track p.category) {
                  <div class="fp-product-item">
                    <div class="fp-product-icon" [style.background]="productColor(p.category)">
                      <span class="material-icons-round">{{ productIcon(p.category) }}</span>
                    </div>
                    <div class="fp-product-info">
                      <div class="fp-product-label">{{ p.label }}</div>
                      <div class="fp-product-desc">{{ p.description }}</div>
                    </div>
                    <span class="badge badge-outline">{{ p.category }}</span>
                  </div>
                }
              </div>
            </div>
          }

          @if (profile()!.available_insurance?.length) {
            <div class="glass-card fp-section">
              <h3 class="fp-section-title">
                <span class="material-icons-round">health_and_safety</span> Insurance Recommendations
              </h3>
              <div class="fp-product-list">
                @for (ins of profile()!.available_insurance; track ins.type) {
                  <div class="fp-product-item">
                    <div class="fp-product-icon" style="background:rgba(56,189,248,0.12);color:#38bdf8;">
                      <span class="material-icons-round">shield</span>
                    </div>
                    <div class="fp-product-info">
                      <div class="fp-product-label">{{ ins.label }}</div>
                      <div class="fp-product-desc">{{ ins.description }}</div>
                    </div>
                    <span class="badge badge-outline">{{ ins.type }}</span>
                  </div>
                }
              </div>
            </div>
          }
        </div>

        <!-- Suggestions -->
        @if (profile()!.suggestions?.length) {
          <div class="glass-card fp-section">
            <h3 class="fp-section-title">
              <span class="material-icons-round" style="color:var(--color-warning);">lightbulb</span> Improvement Tips
            </h3>
            <ul class="suggestion-list">
              @for (s of profile()!.suggestions; track s) { <li>{{ s }}</li> }
            </ul>
          </div>
        }
      }
    </div>
  `,
  styles: [`
    .fp-banner {
      display: flex; justify-content: space-between; align-items: center;
      padding: var(--space-xl) !important; margin-bottom: var(--space-lg);
      background: linear-gradient(135deg, rgba(99,102,241,0.08) 0%, rgba(56,189,248,0.06) 100%);
    }
    .fp-identity { display: flex; align-items: center; gap: var(--space-md); }
    .fp-avatar {
      width: 64px; height: 64px; border-radius: 50%;
      display: flex; align-items: center; justify-content: center;
      font-size: 1.5rem; font-weight: 800; color: white;
    }
    .fp-name { font-size: 1.25rem; font-weight: 700; margin-bottom: 4px; }
    .fp-meta { display: flex; gap: 6px; flex-wrap: wrap; }

    .fp-score-block { display: flex; flex-direction: column; align-items: center; gap: 4px; }
    .fp-score-ring {
      width: 90px; height: 90px; border-radius: 50%; display: flex; flex-direction: column;
      align-items: center; justify-content: center;
      background: conic-gradient(var(--color-accent) calc(var(--score-pct, 0) * 1%), rgba(255,255,255,0.06) 0%);
      position: relative;
    }
    .fp-score-ring::before {
      content: ''; position: absolute; inset: 6px; border-radius: 50%;
      background: var(--color-surface);
    }
    .fp-score-value { font-size: 1.5rem; font-weight: 800; position: relative; z-index: 1; }
    .fp-score-grade { font-size: 0.65rem; font-weight: 700; letter-spacing: 0.05em; position: relative; z-index: 1; }
    .fp-score-label { font-size: 0.7rem; color: var(--color-text-muted); }

    .fp-metrics-grid {
      display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: var(--space-md); margin-bottom: var(--space-lg);
    }
    .fp-metric { padding: var(--space-lg) !important; text-align: center; }
    .fp-metric-icon { font-size: 1.75rem; margin-bottom: var(--space-xs); }
    .fp-metric-value {
      font-size: 1.25rem; font-weight: 800; color: var(--color-text-primary);
      font-family: var(--font-heading);
    }
    .fp-metric-label { font-size: 0.75rem; color: var(--color-text-muted); margin-top: 2px; }
    .fp-metric-trend {
      display: inline-flex; align-items: center; gap: 2px;
      font-size: 0.7rem; font-weight: 600; padding: 2px 8px; border-radius: 12px; margin-top: 4px;
      .material-icons-round { font-size: 14px; }
    }
    .fp-metric-trend.trend-growing { background: rgba(34,197,94,0.1); color: #22c55e; }
    .fp-metric-trend.trend-stable { background: rgba(251,191,36,0.1); color: #fbbf24; }
    .fp-metric-trend.trend-declining { background: rgba(239,68,68,0.1); color: #ef4444; }
    .fp-metric-sub { font-size: 0.7rem; color: var(--color-text-muted); margin-top: 2px; }

    .fp-section { padding: var(--space-lg) !important; margin-bottom: var(--space-lg); }
    .fp-section-title {
      display: flex; align-items: center; gap: 8px;
      font-size: 1rem; font-weight: 600; margin-bottom: var(--space-md);
      .material-icons-round { font-size: 20px; color: var(--color-accent); }
    }

    .fp-org-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: var(--space-md); }
    .fp-org-card {
      background: rgba(255,255,255,0.02); border: 1px solid var(--color-border);
      border-radius: var(--radius-md); padding: var(--space-md);
      transition: border-color 0.2s;
    }
    .fp-org-card.active { border-color: var(--color-accent); }
    .fp-org-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-sm); }
    .fp-org-name { font-weight: 600; font-size: 0.9rem; }
    .fp-org-details { display: flex; flex-direction: column; gap: 4px; }
    .fp-org-row {
      display: flex; justify-content: space-between; font-size: 0.8rem;
      span { color: var(--color-text-muted); }
      strong { color: var(--color-text-primary); font-weight: 600; }
    }
    .fp-org-status {
      display: flex; align-items: center; gap: 6px;
      margin-top: var(--space-sm); font-size: 0.7rem; color: var(--color-text-muted);
    }
    .status-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-text-muted); }
    .status-dot.active { background: var(--color-success); box-shadow: 0 0 6px rgba(34,197,94,0.5); }

    .fp-products-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-lg); }
    .fp-product-list { display: flex; flex-direction: column; gap: var(--space-sm); }
    .fp-product-item {
      display: flex; align-items: center; gap: var(--space-sm);
      padding: var(--space-sm) var(--space-md); border-radius: var(--radius-md);
      background: rgba(255,255,255,0.02); border: 1px solid var(--color-border);
      transition: transform 0.15s, border-color 0.2s;
      &:hover { transform: translateX(4px); border-color: var(--color-accent); }
    }
    .fp-product-icon {
      width: 36px; height: 36px; border-radius: var(--radius-sm);
      display: flex; align-items: center; justify-content: center; flex-shrink: 0;
      .material-icons-round { font-size: 18px; }
    }
    .fp-product-info { flex: 1; min-width: 0; }
    .fp-product-label { font-size: 0.8125rem; font-weight: 600; }
    .fp-product-desc { font-size: 0.7rem; color: var(--color-text-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

    /* Score factors (reused from credit) */
    .factors-grid { display: flex; flex-direction: column; gap: var(--space-md); }
    .factor-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
    .factor-name { font-size: 0.8125rem; font-weight: 500; color: var(--color-text-secondary); }
    .factor-points { font-size: 0.75rem; font-weight: 700; color: var(--color-text-muted); }
    .factor-points.positive { color: var(--color-success); }
    .factor-points.negative { color: #ef4444; }
    .factor-bar-track { height: 6px; border-radius: 3px; background: rgba(255,255,255,0.06); overflow: hidden; }
    .factor-bar-fill { height: 100%; border-radius: 3px; transition: width 0.6s ease-out; min-width: 2px; }
    .fill-positive { background: var(--color-success); }
    .fill-negative { background: #ef4444; }
    .fill-neutral { background: var(--color-warning); }
    .factor-desc { font-size: 0.7rem; color: var(--color-text-muted); margin-top: 2px; }

    .suggestion-list {
      list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 6px;
      li {
        font-size: 0.8125rem; color: var(--color-text-secondary); padding-left: 24px; position: relative;
        &::before { content: '💡'; position: absolute; left: 0; }
      }
    }

    .badge-accent { background: rgba(99,102,241,0.15); color: #818cf8; }
    .badge-transport { background: rgba(56,189,248,0.12); color: #38bdf8; }
    .badge-construction { background: rgba(251,146,60,0.12); color: #fb923c; }
    .badge-health { background: rgba(34,197,94,0.12); color: #22c55e; }
    .badge-agriculture { background: rgba(163,230,53,0.12); color: #a3e635; }
    .badge-logistics { background: rgba(168,85,247,0.12); color: #a855f7; }

    @media (max-width: 768px) {
      .fp-banner { flex-direction: column; gap: var(--space-md); text-align: center; }
      .fp-identity { flex-direction: column; }
      .fp-products-grid { grid-template-columns: 1fr; }
      .fp-metrics-grid { grid-template-columns: repeat(2, 1fr); }
    }
  `]
})
export class FinancialProfileComponent implements OnInit {
  private api = inject(ApiService);
  private route = inject(ActivatedRoute);
  private toast = inject(ToastService);

  profile = signal<FinancialProfile | null>(null);
  loading = signal(true);
  crewId = '';

  ngOnInit(): void {
    this.crewId = this.route.snapshot.paramMap.get('id') || '';
    if (this.crewId) {
      this.loadProfile();
    } else {
      this.loading.set(false);
    }
  }

  loadProfile(): void {
    this.loading.set(true);
    this.api.getFinancialProfile(this.crewId).subscribe({
      next: r => { this.profile.set(r.data); this.loading.set(false); },
      error: () => { this.loading.set(false); this.toast.error('Failed to load financial profile'); },
    });
  }

  formatCents(cents: number): string {
    return (cents / 100).toLocaleString('en-KE', { minimumFractionDigits: 0, maximumFractionDigits: 0 });
  }

  scorePercent(): string {
    const score = this.profile()?.composite_score || 300;
    return ((score - 300) / 550 * 100).toFixed(0);
  }

  gradeColor(grade: string): string {
    switch (grade) {
      case 'EXCELLENT': return '#00d2ff';
      case 'GOOD': return '#22c55e';
      case 'FAIR': return '#fbbf24';
      case 'POOR': return '#f97316';
      default: return '#ef4444';
    }
  }

  kycBadge(): string {
    return this.profile()?.kyc_status === 'VERIFIED' ? 'badge-success' : 'badge-warning';
  }

  avatarGradient(): string {
    return 'linear-gradient(135deg, #6366f1, #8b5cf6)';
  }

  trendClass(trend: string): string {
    return `trend-${trend.toLowerCase()}`;
  }

  trendIcon(trend: string): string {
    switch (trend) {
      case 'GROWING': return 'trending_up';
      case 'DECLINING': return 'trending_down';
      default: return 'trending_flat';
    }
  }

  industryBadge(industry: string): string {
    return `badge-${industry.toLowerCase()}`;
  }

  productColor(category: string): string {
    switch (category) {
      case 'PERSONAL': return 'rgba(99,102,241,0.12)';
      case 'EMERGENCY': return 'rgba(239,68,68,0.12)';
      case 'EDUCATION': return 'rgba(56,189,248,0.12)';
      case 'BUSINESS': return 'rgba(34,197,94,0.12)';
      case 'ASSET': return 'rgba(251,146,60,0.12)';
      default: return 'rgba(99,102,241,0.12)';
    }
  }

  productIcon(category: string): string {
    switch (category) {
      case 'PERSONAL': return 'person';
      case 'EMERGENCY': return 'emergency';
      case 'EDUCATION': return 'school';
      case 'BUSINESS': return 'business';
      case 'ASSET': return 'build';
      default: return 'credit_card';
    }
  }
}
