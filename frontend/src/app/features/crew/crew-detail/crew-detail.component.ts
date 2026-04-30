import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { catchError } from 'rxjs/operators';
import { of } from 'rxjs';
import { environment } from '../../../../environments/environment';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { CrewMember, Wallet } from '../../../core/models';

@Component({
  selector: 'app-crew-detail',
  standalone: true,
  imports: [CommonModule, RouterLink, CurrencyKesPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <a routerLink="/crew" class="back-link">
            <span class="material-icons-round">arrow_back</span> Back to Crew
          </a>
          <h1 class="page-title">{{ crew()?.full_name ?? 'Loading...' }}</h1>
          <p class="page-subtitle">Crew ID: {{ crew()?.crew_id }}</p>
        </div>
        <div class="page-actions">
          @if (crew()?.is_active) {
            <button class="btn btn-danger btn-sm" (click)="deactivate()" id="btn-deactivate-crew">
              <span class="material-icons-round">block</span> Deactivate
            </button>
          }
        </div>
      </div>

      @if (crew(); as c) {
        <div class="detail-grid">
          <div class="glass-card detail-card">
            <h3 class="card-title">Profile Information</h3>
            <div class="detail-rows">
              <div class="detail-row"><span class="detail-label">Full Name</span><span class="detail-value">{{ c.full_name }}</span></div>
              <div class="detail-row"><span class="detail-label">Role</span><span class="badge badge-accent">{{ c.role }}</span></div>
              <div class="detail-row"><span class="detail-label">Crew ID</span><code class="text-accent">{{ c.crew_id }}</code></div>
              <div class="detail-row">
                <span class="detail-label">KYC Status</span>
                <span class="badge" [ngClass]="kycBadge(c.kyc_status)">{{ c.kyc_status }}</span>
              </div>
              <div class="detail-row">
                <span class="detail-label">Status</span>
                <span class="badge" [ngClass]="c.is_active ? 'badge-success' : 'badge-danger'">{{ c.is_active ? 'Active' : 'Inactive' }}</span>
              </div>
              <div class="detail-row"><span class="detail-label">Created</span><span class="detail-value">{{ c.created_at | date:'medium' }}</span></div>
            </div>
          </div>

          <div class="glass-card detail-card">
            <h3 class="card-title">KYC Verification</h3>
            @if (c.kyc_status === 'PENDING') {
              <p class="text-muted" style="margin-bottom: 16px;">Verify this crew member's national ID via IPRS</p>
              <button class="btn btn-primary" (click)="verifyKYC()" [disabled]="verifying()" id="btn-verify-kyc">
                @if (verifying()) { Verifying... } @else { <span class="material-icons-round">verified_user</span> Verify National ID }
              </button>
            } @else if (c.kyc_status === 'VERIFIED') {
              <div class="success-banner">
                <span class="material-icons-round">check_circle</span>
                <span>KYC verified on {{ c.kyc_verified_at | date:'medium' }}</span>
              </div>
            } @else {
              <div class="danger-banner">
                <span class="material-icons-round">cancel</span>
                <span>KYC verification was rejected</span>
              </div>
            }
          </div>
        </div>

        <!-- #70: Quick Links to Wallet, Assignments, and Earnings -->
        <div class="links-section">
          <h3 class="section-title">
            <span class="material-icons-round" style="font-size:20px;color:var(--color-accent);">link</span>
            Quick Links
          </h3>
          <div class="links-grid">
            <a [routerLink]="['/wallets']" [queryParams]="{ crew_member_id: c.id }" class="link-card glass-card" id="link-wallet">
              <div class="link-icon" style="background:var(--color-success-light);color:var(--color-success);">
                <span class="material-icons-round">account_balance_wallet</span>
              </div>
              <div class="link-content">
                <span class="link-title">Wallet</span>
                @if (wallet()) {
                  <span class="link-value">{{ wallet()!.balance_cents | currencyKes }}</span>
                } @else {
                  <span class="link-hint">View balance & transactions</span>
                }
              </div>
              <span class="material-icons-round link-arrow">chevron_right</span>
            </a>
            <a [routerLink]="['/assignments']" [queryParams]="{ crew_member_id: c.id }" class="link-card glass-card" id="link-assignments">
              <div class="link-icon" style="background:rgba(123,97,255,0.12);color:#7b61ff;">
                <span class="material-icons-round">assignment</span>
              </div>
              <div class="link-content">
                <span class="link-title">Assignments</span>
                @if (assignmentCount() >= 0) {
                  <span class="link-value">{{ assignmentCount() }} total</span>
                } @else {
                  <span class="link-hint">View shift assignments</span>
                }
              </div>
              <span class="material-icons-round link-arrow">chevron_right</span>
            </a>
            <a [routerLink]="['/earnings']" [queryParams]="{ crew_member_id: c.id }" class="link-card glass-card" id="link-earnings">
              <div class="link-icon" style="background:rgba(236,72,153,0.12);color:#ec4899;">
                <span class="material-icons-round">trending_up</span>
              </div>
              <div class="link-content">
                <span class="link-title">Earnings</span>
                <span class="link-hint">View earnings reports</span>
              </div>
              <span class="material-icons-round link-arrow">chevron_right</span>
            </a>
          </div>
        </div>
      } @else {
        <div class="skeleton" style="height: 200px;"></div>
      }
    </div>
  `,
  styles: [`
    .back-link { display: inline-flex; align-items: center; gap: 4px; font-size: 0.875rem; color: var(--color-text-muted); margin-bottom: var(--space-sm); .material-icons-round { font-size: 16px; } &:hover { color: var(--color-accent); } }
    .detail-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-lg); }
    @media (max-width: 768px) { .detail-grid, .links-grid { grid-template-columns: 1fr !important; } }
    .card-title { font-size: 1rem; font-weight: 600; margin-bottom: var(--space-md); color: var(--color-text-primary); }
    .detail-rows { display: flex; flex-direction: column; gap: var(--space-sm); }
    .detail-row { display: flex; align-items: center; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid var(--color-border); &:last-child { border-bottom: none; } }
    .detail-label { font-size: 0.8125rem; color: var(--color-text-muted); }
    .detail-value { font-size: 0.875rem; color: var(--color-text-primary); font-weight: 500; }
    .success-banner, .danger-banner { display: flex; align-items: center; gap: var(--space-sm); padding: 12px; border-radius: var(--radius-md); font-size: 0.875rem; }
    .success-banner { background: var(--color-success-light); color: var(--color-success); }
    .danger-banner { background: var(--color-danger-light); color: var(--color-danger); }

    /* #70: Quick links */
    .links-section { margin-top: var(--space-xl); }
    .section-title { display: flex; align-items: center; gap: var(--space-sm); font-family: var(--font-heading); font-size: 1.125rem; font-weight: 600; color: var(--color-text-secondary); margin-bottom: var(--space-md); }
    .links-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--space-md); }
    .link-card {
      display: flex; align-items: center; gap: var(--space-md); padding: var(--space-lg) !important;
      text-decoration: none; cursor: pointer; transition: border-color 200ms;
      &:hover { border-color: var(--color-accent) !important; }
    }
    .link-icon { width: 44px; height: 44px; border-radius: var(--radius-md); display: flex; align-items: center; justify-content: center; flex-shrink: 0; .material-icons-round { font-size: 22px; } }
    .link-content { flex: 1; display: flex; flex-direction: column; }
    .link-title { font-size: 0.875rem; font-weight: 600; color: var(--color-text-primary); }
    .link-value { font-size: 0.8125rem; color: var(--color-accent); font-weight: 600; }
    .link-hint { font-size: 0.75rem; color: var(--color-text-muted); }
    .link-arrow { font-size: 20px; color: var(--color-text-muted); }
  `]
})
export class CrewDetailComponent implements OnInit {
  private api = inject(ApiService);
  private http = inject(HttpClient);
  private route = inject(ActivatedRoute);
  private toast = inject(ToastService);
  private readonly API = environment.apiUrl;
  private readonly silentHeaders = { 'X-Skip-Error-Toast': 'true' };

  crew = signal<CrewMember | null>(null);
  verifying = signal(false);
  wallet = signal<Wallet | null>(null);
  assignmentCount = signal(-1);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.api.getCrewMember(id).subscribe({
        next: (res) => {
          this.crew.set(res.data);
          this.loadRelatedData(res.data.id);
        },
      });
    }
  }

  // #70: Fetch wallet balance and assignment count silently
  private loadRelatedData(crewMemberId: string): void {
    this.http.get<any>(`${this.API}/wallets/${crewMemberId}`, { headers: this.silentHeaders }).pipe(
      catchError(() => of(null))
    ).subscribe(res => { if (res?.data) this.wallet.set(res.data); });

    this.http.get<any>(`${this.API}/assignments`, { headers: this.silentHeaders, params: { crew_member_id: crewMemberId, limit: '1' } }).pipe(
      catchError(() => of(null))
    ).subscribe(res => { if (res) this.assignmentCount.set(res.meta?.total ?? res.data?.length ?? 0); });
  }

  verifyKYC(): void {
    const c = this.crew();
    if (!c) return;
    this.verifying.set(true);
    this.api.verifyNationalID(c.id, '').subscribe({
      next: (res) => { this.crew.set(res.data); this.toast.success('KYC verification initiated'); this.verifying.set(false); },
      error: () => this.verifying.set(false),
    });
  }

  deactivate(): void {
    const c = this.crew();
    if (!c) return;
    if (confirm('Are you sure you want to deactivate this crew member?')) {
      this.api.deactivateCrewMember(c.id).subscribe({
        next: () => { this.toast.success('Crew member deactivated'); this.crew.update(c => c ? { ...c, is_active: false } : null); },
      });
    }
  }

  kycBadge(status: string): string {
    return status === 'VERIFIED' ? 'badge-success' : status === 'REJECTED' ? 'badge-danger' : 'badge-warning';
  }
}
