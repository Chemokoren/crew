import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import {
  CreditScore, DetailedScoreResult, ScoreFactor,
  CreditScoreHistory, LoanTier, CrewMember
} from '../../../core/models';

@Component({
  selector: 'app-credit-score',
  standalone: true,
  imports: [CommonModule, FormsModule, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">Credit Score</h1>
          <p class="page-subtitle">View your creditworthiness score, factor breakdown, and loan eligibility</p>
        </div>
        <div class="page-actions">
          @if (isAdmin()) {
            <div class="crew-selector">
              <select class="form-select" [(ngModel)]="selectedCrewId" (ngModelChange)="onCrewChange()" id="select-crew-credit">
                <option value="">— Select Crew Member —</option>
                @for (c of crewMembers(); track c.id) { <option [value]="c.id">{{ c.first_name }} {{ c.last_name }}</option> }
              </select>
            </div>
            <button class="btn btn-secondary" (click)="calculateScore()" [disabled]="calculating() || !selectedCrewId" id="btn-calculate-score">
              <span class="material-icons-round">refresh</span>
              {{ calculating() ? 'Calculating...' : 'Recalculate' }}
            </button>
          }
        </div>
      </div>

      @if (loading()) {
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));">
          @for (i of [1,2,3]; track i) { <div class="skeleton" style="height:300px;border-radius:var(--radius-lg);"></div> }
        </div>
      } @else if (!score()) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">credit_score</span>
          <div class="empty-title">No credit score available</div>
          <div class="empty-description">
            @if (isAdmin()) { Select a crew member and calculate their score. }
            @else { Your score will be calculated based on your earnings history and activity. }
          </div>
        </div>
      } @else {

        <!-- Top Row: Gauge + Tier -->
        <div class="score-top-grid">

          <!-- Score Gauge (Task 133) -->
          <div class="glass-card gauge-card">
            <div class="gauge-container">
              <svg viewBox="0 0 200 140" class="gauge-svg">
                <defs>
                  <linearGradient id="gaugeGrad" x1="0" y1="0" x2="1" y2="0">
                    <stop offset="0%" stop-color="#ef4444" />
                    <stop offset="35%" stop-color="#fbbf24" />
                    <stop offset="65%" stop-color="#22c55e" />
                    <stop offset="100%" stop-color="#00d2ff" />
                  </linearGradient>
                </defs>
                <!-- Background arc -->
                <path d="M 20 130 A 80 80 0 0 1 180 130" fill="none" stroke="rgba(255,255,255,0.06)" stroke-width="14" stroke-linecap="round" />
                <!-- Score arc -->
                <path [attr.d]="'M 20 130 A 80 80 0 0 1 180 130'" fill="none" stroke="url(#gaugeGrad)" stroke-width="14" stroke-linecap="round"
                  [attr.stroke-dasharray]="arcLength()" [attr.stroke-dashoffset]="arcOffset()" style="transition: stroke-dashoffset 1s ease-out;" />
                <!-- Score text -->
                <text x="100" y="105" text-anchor="middle" fill="var(--color-text-primary)" font-size="36" font-weight="800" font-family="var(--font-heading)">{{ score()!.score }}</text>
                <text x="100" y="128" text-anchor="middle" [attr.fill]="gradeColor(score()!.grade)" font-size="13" font-weight="700" letter-spacing="0.08em">{{ score()!.grade }}</text>
              </svg>
            </div>
            <div class="gauge-footer">
              <span class="gauge-range">300</span>
              <span class="gauge-label">out of 850</span>
              <span class="gauge-range">850</span>
            </div>
            @if (score()!.computed_at) {
              <div class="gauge-updated">Last computed {{ score()!.computed_at | relativeTime }}</div>
            }
          </div>

          <!-- Loan Tier (Task 137) -->
          <div class="glass-card tier-card">
            <div class="tier-header">
              <div class="tier-icon" [style.background]="tierIconBg()" [style.color]="tierIconColor()">
                <span class="material-icons-round">{{ tierIcon() }}</span>
              </div>
              <span class="badge" [ngClass]="tierBadgeClass()">{{ tier()?.grade || 'N/A' }}</span>
            </div>
            <h3 class="tier-title">Your Loan Tier</h3>
            @if (tier(); as t) {
              <div class="tier-details">
                <div class="tier-row"><span class="tier-label">Max Loan</span><span class="tier-value highlight">KES {{ t.max_loan_kes | number:'1.0-0' }}</span></div>
                <div class="tier-row"><span class="tier-label">Interest Rate</span><span class="tier-value">{{ t.interest_rate }}%</span></div>
                <div class="tier-row"><span class="tier-label">Max Tenure</span><span class="tier-value">{{ t.max_tenure_days }} days</span></div>
                <div class="tier-row"><span class="tier-label">Cooldown</span><span class="tier-value">{{ t.cooldown_days }} days</span></div>
              </div>
              <p class="tier-desc">{{ t.description }}</p>
            } @else {
              <div class="empty-state" style="padding:var(--space-md);">
                <div class="empty-subtitle">Score too low for loan eligibility (min 400)</div>
              </div>
            }
          </div>
        </div>

        <!-- Score History Chart (Task 135) -->
        @if (history().length > 1) {
          <div class="glass-card" style="margin-top:var(--space-lg);">
            <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Score History</h3>
            <div class="chart-container">
              <svg [attr.viewBox]="'0 0 ' + chartWidth + ' ' + chartHeight" class="history-chart" preserveAspectRatio="none">
                <defs>
                  <linearGradient id="chartFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stop-color="var(--color-accent)" stop-opacity="0.25" />
                    <stop offset="100%" stop-color="var(--color-accent)" stop-opacity="0" />
                  </linearGradient>
                </defs>
                <!-- Grid lines -->
                @for (y of chartGridY; track y) {
                  <line [attr.x1]="40" [attr.y1]="y" [attr.x2]="chartWidth" [attr.y2]="y" stroke="rgba(255,255,255,0.05)" stroke-width="1"/>
                }
                <!-- Area fill -->
                <path [attr.d]="chartAreaPath()" fill="url(#chartFill)" />
                <!-- Line -->
                <polyline [attr.points]="chartLinePath()" fill="none" stroke="var(--color-accent)" stroke-width="2" stroke-linejoin="round" />
                <!-- Dots -->
                @for (p of chartPoints(); track p.x) {
                  <circle [attr.cx]="p.x" [attr.cy]="p.y" r="3" fill="var(--color-accent)" stroke="var(--color-surface)" stroke-width="1.5" />
                }
                <!-- Y-axis labels -->
                @for (label of chartYLabels; track label.y) {
                  <text [attr.x]="35" [attr.y]="label.y + 4" text-anchor="end" fill="var(--color-text-muted)" font-size="10">{{ label.text }}</text>
                }
                <!-- X-axis labels -->
                @for (label of chartXLabels(); track label.x) {
                  <text [attr.x]="label.x" [attr.y]="chartHeight - 2" text-anchor="middle" fill="var(--color-text-muted)" font-size="9">{{ label.text }}</text>
                }
              </svg>
            </div>
          </div>
        }

        <!-- Factor Breakdown (Task 134) -->
        @if (detailed(); as d) {
          <div class="glass-card" style="margin-top:var(--space-lg);">
            <h3 style="font-size:1rem;font-weight:600;margin-bottom:var(--space-md);">Score Breakdown</h3>
            <div class="factors-grid">
              @for (f of d.factors; track f.name) {
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
            @if (d.suggestions && d.suggestions.length) {
              <div class="suggestions-section">
                <h4 style="font-size:0.875rem;font-weight:600;margin-bottom:var(--space-sm);color:var(--color-text-primary);">
                  <span class="material-icons-round" style="font-size:16px;vertical-align:middle;color:var(--color-warning);">lightbulb</span>
                  Tips to Improve
                </h4>
                <ul class="suggestion-list">
                  @for (s of d.suggestions; track s) { <li>{{ s }}</li> }
                </ul>
              </div>
            }
          </div>
        }
      }
    </div>
  `,
  styles: [`
    .score-top-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-lg); }
    .gauge-card { display: flex; flex-direction: column; align-items: center; padding: var(--space-xl) !important; }
    .gauge-container { width: 220px; height: 160px; }
    .gauge-svg { width: 100%; height: 100%; }
    .gauge-footer { display: flex; justify-content: space-between; align-items: center; width: 200px; margin-top: -8px; }
    .gauge-range { font-size: 0.7rem; color: var(--color-text-muted); font-weight: 500; }
    .gauge-label { font-size: 0.75rem; color: var(--color-text-muted); }
    .gauge-updated { font-size: 0.7rem; color: var(--color-text-muted); margin-top: var(--space-sm); }

    .tier-card { padding: var(--space-lg) !important; display: flex; flex-direction: column; }
    .tier-header { display: flex; justify-content: space-between; align-items: center; }
    .tier-icon {
      width: 44px; height: 44px; border-radius: var(--radius-md);
      display: flex; align-items: center; justify-content: center;
    }
    .tier-title { font-size: 1rem; font-weight: 600; margin: var(--space-sm) 0; }
    .tier-details { display: flex; flex-direction: column; gap: 8px; }
    .tier-row { display: flex; justify-content: space-between; align-items: center; padding: 6px 0; border-bottom: 1px solid var(--color-border); }
    .tier-row:last-child { border-bottom: none; }
    .tier-label { font-size: 0.8rem; color: var(--color-text-muted); }
    .tier-value { font-size: 0.875rem; font-weight: 600; color: var(--color-text-primary); }
    .tier-value.highlight {
      background: var(--gradient-accent); -webkit-background-clip: text;
      -webkit-text-fill-color: transparent; background-clip: text; font-size: 1.1rem; font-weight: 800;
    }
    .tier-desc { font-size: 0.75rem; color: var(--color-text-muted); margin-top: auto; padding-top: var(--space-sm); }

    .chart-container { height: 180px; }
    .history-chart { width: 100%; height: 100%; }

    .factors-grid { display: flex; flex-direction: column; gap: var(--space-md); }
    .factor-item { }
    .factor-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
    .factor-name { font-size: 0.8125rem; font-weight: 500; color: var(--color-text-secondary); }
    .factor-points { font-size: 0.75rem; font-weight: 700; color: var(--color-text-muted); }
    .factor-points.positive { color: var(--color-success); }
    .factor-points.negative { color: #ef4444; }
    .factor-bar-track {
      height: 6px; border-radius: 3px; background: rgba(255,255,255,0.06); overflow: hidden;
    }
    .factor-bar-fill { height: 100%; border-radius: 3px; transition: width 0.6s ease-out; min-width: 2px; }
    .fill-positive { background: var(--color-success); }
    .fill-negative { background: #ef4444; }
    .fill-neutral { background: var(--color-warning); }
    .factor-desc { font-size: 0.7rem; color: var(--color-text-muted); margin-top: 2px; }

    .suggestions-section { margin-top: var(--space-lg); padding-top: var(--space-md); border-top: 1px solid var(--color-border); }
    .suggestion-list {
      list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 6px;
      li {
        font-size: 0.8125rem; color: var(--color-text-secondary); padding-left: 20px; position: relative;
        &::before { content: '💡'; position: absolute; left: 0; }
      }
    }

    .crew-selector { min-width: 200px; }

    @media (max-width: 768px) {
      .score-top-grid { grid-template-columns: 1fr; }
      .gauge-container { width: 180px; height: 130px; }
    }
  `]
})
export class CreditScoreComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);

  score = signal<CreditScore | null>(null);
  detailed = signal<DetailedScoreResult | null>(null);
  history = signal<CreditScoreHistory[]>([]);
  tier = signal<LoanTier | null>(null);
  crewMembers = signal<CrewMember[]>([]);
  loading = signal(true);
  calculating = signal(false);
  selectedCrewId = '';

  // Chart dimensions
  readonly chartWidth = 600;
  readonly chartHeight = 180;
  readonly chartPadding = { top: 15, right: 15, bottom: 20, left: 45 };
  readonly chartGridY = [30, 60, 90, 120, 150];
  readonly chartYLabels = [
    { y: 30, text: '850' }, { y: 60, text: '700' }, { y: 90, text: '550' },
    { y: 120, text: '400' }, { y: 150, text: '300' }
  ];

  ngOnInit(): void {
    const user = this.auth.currentUser();
    if (this.isAdmin()) {
      this.api.getCrewMembers({ per_page: '200' }).subscribe({
        next: r => { this.crewMembers.set(r.data); this.loading.set(false); },
        error: () => this.loading.set(false),
      });
    } else if (user?.crew_member_id) {
      this.selectedCrewId = user.crew_member_id;
      this.loadAll();
    } else {
      this.loading.set(false);
    }
  }

  isAdmin(): boolean { return this.auth.isAdmin(); }

  onCrewChange(): void {
    if (this.selectedCrewId) this.loadAll();
    else { this.score.set(null); this.detailed.set(null); this.history.set([]); this.tier.set(null); }
  }

  loadAll(): void {
    this.loading.set(true);
    const id = this.selectedCrewId;

    this.api.getCreditScore(id).subscribe({
      next: r => { this.score.set(r.data); this.loading.set(false); },
      error: () => this.loading.set(false),
    });

    this.api.getDetailedScore(id).subscribe({
      next: r => this.detailed.set(r.data),
      error: () => {},
    });

    this.api.getScoreHistory(id, 30).subscribe({
      next: r => this.history.set(r.data || []),
      error: () => {},
    });

    this.api.getLoanTier(id).subscribe({
      next: r => this.tier.set(r.data),
      error: () => this.tier.set(null),
    });
  }

  calculateScore(): void {
    if (!this.selectedCrewId) return;
    this.calculating.set(true);
    this.api.calculateScore(this.selectedCrewId).subscribe({
      next: () => {
        this.toast.success('Score recalculated');
        this.calculating.set(false);
        this.loadAll();
      },
      error: () => this.calculating.set(false),
    });
  }

  // --- Gauge arc math ---
  private readonly totalArc = 251.33; // Approx arc length for the SVG path (semi-circle r=80)

  arcLength(): string { return `${this.totalArc} ${this.totalArc}`; }

  arcOffset(): number {
    const s = this.score();
    if (!s) return this.totalArc;
    const pct = Math.max(0, Math.min(1, (s.score - 300) / 550));
    return this.totalArc * (1 - pct);
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

  // --- Loan Tier styling ---
  tierIcon(): string {
    const g = this.tier()?.grade || '';
    return g === 'EXCELLENT' ? 'diamond' : g === 'GOOD' ? 'stars' : g === 'FAIR' ? 'trending_up' : 'trending_flat';
  }
  tierIconBg(): string { return `${this.gradeColor(this.tier()?.grade || '')}20`; }
  tierIconColor(): string { return this.gradeColor(this.tier()?.grade || ''); }
  tierBadgeClass(): string {
    const g = this.tier()?.grade || '';
    return g === 'EXCELLENT' || g === 'GOOD' ? 'badge-success' : g === 'FAIR' ? 'badge-warning' : 'badge-danger';
  }

  // --- History chart ---
  chartPoints = computed(() => {
    const h = this.history();
    if (h.length < 2) return [];
    const sorted = [...h].sort((a, b) => new Date(a.computed_at).getTime() - new Date(b.computed_at).getTime());
    const xStart = this.chartPadding.left;
    const xEnd = this.chartWidth - this.chartPadding.right;
    const yStart = this.chartPadding.top;
    const yEnd = this.chartHeight - this.chartPadding.bottom;
    const xStep = (xEnd - xStart) / Math.max(1, sorted.length - 1);
    return sorted.map((s, i) => ({
      x: xStart + i * xStep,
      y: yEnd - ((s.score - 300) / 550) * (yEnd - yStart),
    }));
  });

  chartLinePath(): string {
    return this.chartPoints().map(p => `${p.x},${p.y}`).join(' ');
  }

  chartAreaPath(): string {
    const pts = this.chartPoints();
    if (pts.length < 2) return '';
    const yEnd = this.chartHeight - this.chartPadding.bottom;
    let d = `M ${pts[0].x} ${yEnd}`;
    for (const p of pts) d += ` L ${p.x} ${p.y}`;
    d += ` L ${pts[pts.length - 1].x} ${yEnd} Z`;
    return d;
  }

  chartXLabels = computed(() => {
    const h = this.history();
    if (h.length < 2) return [];
    const sorted = [...h].sort((a, b) => new Date(a.computed_at).getTime() - new Date(b.computed_at).getTime());
    const pts = this.chartPoints();
    const step = Math.max(1, Math.floor(sorted.length / 6));
    return sorted
      .filter((_, i) => i % step === 0 || i === sorted.length - 1)
      .map((s, idx) => ({
        x: pts[sorted.indexOf(s)]?.x || 0,
        text: new Date(s.computed_at).toLocaleDateString('en', { month: 'short', day: 'numeric' }),
      }));
  });
}
