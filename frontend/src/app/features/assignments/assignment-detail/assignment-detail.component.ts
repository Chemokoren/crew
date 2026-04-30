import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Assignment } from '../../../core/models';

@Component({
  selector: 'app-assignment-detail',
  standalone: true,
  imports: [CommonModule, RouterLink],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <div style="display:flex;align-items:center;gap:8px;margin-bottom:4px;">
            <a routerLink="/assignments" class="btn btn-ghost btn-icon" id="btn-back-assignments">
              <span class="material-icons-round">arrow_back</span>
            </a>
            <h1 class="page-title">Assignment Details</h1>
          </div>
          <p class="page-subtitle">Full shift assignment information</p>
        </div>
        @if (assignment()) {
          <div class="page-actions" style="display:flex;gap:8px;">
            @if (assignment()!.status === 'ACTIVE' || assignment()!.status === 'SCHEDULED') {
              <button class="btn btn-primary" (click)="completeAssignment()" id="btn-complete">
                <span class="material-icons-round">check_circle</span> Complete
              </button>
              <button class="btn btn-danger" (click)="cancelAssignment()" id="btn-cancel">
                <span class="material-icons-round">cancel</span> Cancel
              </button>
            }
          </div>
        }
      </div>

      @if (loading()) {
        <div class="detail-grid">
          @for (i of [1,2,3,4,5,6]; track i) { <div class="skeleton" style="height:80px;"></div> }
        </div>
      } @else if (!assignment()) {
        <div class="empty-state">
          <span class="material-icons-round empty-icon">search_off</span>
          <div class="empty-title">Assignment not found</div>
          <div class="empty-description">This assignment may have been removed or the ID is invalid.</div>
          <a routerLink="/assignments" class="btn btn-primary" style="margin-top:16px;">Back to Assignments</a>
        </div>
      } @else {
        <!-- Status Banner -->
        <div class="status-banner" [ngClass]="statusBannerClass(assignment()!.status)">
          <span class="material-icons-round">{{ statusIcon(assignment()!.status) }}</span>
          <div>
            <strong>{{ assignment()!.status }}</strong>
            <span style="opacity:0.8;margin-left:8px;">Shift on {{ assignment()!.shift_date | date:'fullDate' }}</span>
          </div>
        </div>

        <div class="detail-grid">
          <!-- Crew Member Card -->
          <div class="detail-card">
            <div class="detail-card-header">
              <span class="material-icons-round detail-card-icon" style="color:var(--color-accent);">person</span>
              <h3>Crew Member</h3>
            </div>
            <div class="detail-field">
              <span class="detail-label">Name</span>
              <span class="detail-value highlight">{{ assignment()!.crew_member_name || '—' }}</span>
            </div>
            <div class="detail-field">
              <span class="detail-label">Member ID</span>
              <span class="detail-value mono">{{ assignment()!.crew_member_id }}</span>
            </div>
          </div>

          <!-- Vehicle Card -->
          <div class="detail-card">
            <div class="detail-card-header">
              <span class="material-icons-round detail-card-icon" style="color:var(--color-warning);">directions_bus</span>
              <h3>Vehicle</h3>
            </div>
            <div class="detail-field">
              <span class="detail-label">Registration</span>
              <span class="detail-value highlight">{{ assignment()!.vehicle_registration_no || '—' }}</span>
            </div>
            <div class="detail-field">
              <span class="detail-label">Vehicle ID</span>
              <span class="detail-value mono">{{ assignment()!.vehicle_id }}</span>
            </div>
          </div>

          <!-- SACCO Card -->
          <div class="detail-card">
            <div class="detail-card-header">
              <span class="material-icons-round detail-card-icon" style="color:var(--color-success);">business</span>
              <h3>SACCO</h3>
            </div>
            <div class="detail-field">
              <span class="detail-label">Organization</span>
              <span class="detail-value highlight">{{ assignment()!.sacco_name || '—' }}</span>
            </div>
            <div class="detail-field">
              <span class="detail-label">SACCO ID</span>
              <span class="detail-value mono">{{ assignment()!.sacco_id }}</span>
            </div>
          </div>

          <!-- Route Card -->
          <div class="detail-card">
            <div class="detail-card-header">
              <span class="material-icons-round detail-card-icon" style="color:var(--color-info);">route</span>
              <h3>Route</h3>
            </div>
            <div class="detail-field">
              <span class="detail-label">Route Name</span>
              <span class="detail-value highlight">{{ assignment()!.route_name || 'Not assigned' }}</span>
            </div>
            @if (assignment()!.route_id) {
              <div class="detail-field">
                <span class="detail-label">Route ID</span>
                <span class="detail-value mono">{{ assignment()!.route_id }}</span>
              </div>
            }
          </div>

          <!-- Shift Schedule Card -->
          <div class="detail-card">
            <div class="detail-card-header">
              <span class="material-icons-round detail-card-icon" style="color:var(--color-accent-secondary);">schedule</span>
              <h3>Shift Schedule</h3>
            </div>
            <div class="detail-field">
              <span class="detail-label">Shift Date</span>
              <span class="detail-value">{{ assignment()!.shift_date | date:'fullDate' }}</span>
            </div>
            <div class="detail-field">
              <span class="detail-label">Start Time</span>
              <span class="detail-value">{{ assignment()!.shift_start | date:'shortTime' }}</span>
            </div>
            <div class="detail-field">
              <span class="detail-label">End Time</span>
              <span class="detail-value">{{ assignment()!.shift_end ? (assignment()!.shift_end | date:'shortTime') : 'Ongoing' }}</span>
            </div>
          </div>

          <!-- Earnings Card -->
          <div class="detail-card">
            <div class="detail-card-header">
              <span class="material-icons-round detail-card-icon" style="color:var(--color-success);">payments</span>
              <h3>Earnings Configuration</h3>
            </div>
            <div class="detail-field">
              <span class="detail-label">Earning Model</span>
              <span class="detail-value"><span class="badge badge-accent">{{ assignment()!.earning_model }}</span></span>
            </div>
            @if (assignment()!.earning_model === 'FIXED' || assignment()!.earning_model === 'HYBRID') {
              <div class="detail-field">
                <span class="detail-label">Fixed Amount</span>
                <span class="detail-value highlight">KES {{ ((assignment()!.fixed_amount_cents || 0) / 100).toFixed(2) }}</span>
              </div>
            }
            @if (assignment()!.earning_model === 'COMMISSION' || assignment()!.earning_model === 'HYBRID') {
              <div class="detail-field">
                <span class="detail-label">Commission Rate</span>
                <span class="detail-value highlight">{{ ((assignment()!.commission_rate || 0) * 100).toFixed(1) }}%</span>
              </div>
            }
            @if (assignment()!.hybrid_base_cents) {
              <div class="detail-field">
                <span class="detail-label">Hybrid Base</span>
                <span class="detail-value">KES {{ (assignment()!.hybrid_base_cents! / 100).toFixed(2) }}</span>
              </div>
            }
            @if (assignment()!.commission_basis) {
              <div class="detail-field">
                <span class="detail-label">Commission Basis</span>
                <span class="detail-value">{{ assignment()!.commission_basis }}</span>
              </div>
            }
          </div>
        </div>

        <!-- Notes Section -->
        @if (assignment()!.notes) {
          <div class="detail-card" style="margin-top:var(--space-lg);">
            <div class="detail-card-header">
              <span class="material-icons-round detail-card-icon" style="color:var(--color-text-muted);">notes</span>
              <h3>Notes</h3>
            </div>
            <p style="color:var(--color-text-secondary);line-height:1.6;margin:0;">{{ assignment()!.notes }}</p>
          </div>
        }

        <!-- Meta -->
        <div class="detail-meta">
          <span>Created: {{ assignment()!.created_at | date:'medium' }}</span>
        </div>
      }
    </div>
  `,
  styles: [`
    .status-banner {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 14px 20px;
      border-radius: var(--radius-lg);
      margin-bottom: var(--space-lg);
      font-weight: 600;
      font-size: 0.9375rem;
    }
    .status-banner .material-icons-round { font-size: 22px; }

    .status-active { background: linear-gradient(135deg, rgba(59,130,246,0.15), rgba(59,130,246,0.05)); color: var(--color-info); border: 1px solid rgba(59,130,246,0.2); }
    .status-scheduled { background: linear-gradient(135deg, rgba(168,85,247,0.15), rgba(168,85,247,0.05)); color: #a855f7; border: 1px solid rgba(168,85,247,0.2); }
    .status-completed { background: linear-gradient(135deg, rgba(16,185,129,0.15), rgba(16,185,129,0.05)); color: var(--color-success); border: 1px solid rgba(16,185,129,0.2); }
    .status-cancelled { background: linear-gradient(135deg, rgba(239,68,68,0.15), rgba(239,68,68,0.05)); color: var(--color-danger); border: 1px solid rgba(239,68,68,0.2); }

    .detail-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: var(--space-lg);
    }

    .detail-card {
      background: var(--color-bg-card);
      border: 1px solid var(--color-border);
      border-radius: var(--radius-lg);
      padding: var(--space-lg);
      transition: border-color var(--transition-fast);
    }
    .detail-card:hover { border-color: var(--color-border-hover); }

    .detail-card-header {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: var(--space-md);
      padding-bottom: var(--space-sm);
      border-bottom: 1px solid var(--color-border);
    }
    .detail-card-header h3 {
      font-size: 0.9375rem;
      font-weight: 600;
      color: var(--color-text-primary);
      margin: 0;
    }
    .detail-card-icon { font-size: 20px; }

    .detail-field {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 0;
    }
    .detail-field + .detail-field { border-top: 1px solid rgba(255,255,255,0.03); }

    .detail-label {
      font-size: 0.8125rem;
      color: var(--color-text-muted);
      font-weight: 500;
    }
    .detail-value {
      font-size: 0.875rem;
      color: var(--color-text-primary);
      font-weight: 600;
      text-align: right;
    }
    .detail-value.highlight { color: var(--color-accent); }
    .detail-value.mono {
      font-family: var(--font-mono, 'JetBrains Mono', monospace);
      font-size: 0.75rem;
      color: var(--color-text-muted);
      word-break: break-all;
      max-width: 200px;
    }

    .detail-meta {
      display: flex;
      gap: var(--space-lg);
      margin-top: var(--space-lg);
      padding-top: var(--space-md);
      border-top: 1px solid var(--color-border);
      color: var(--color-text-muted);
      font-size: 0.8125rem;
    }

    @media (max-width: 768px) {
      .detail-grid { grid-template-columns: 1fr; }
      .detail-value.mono { max-width: 140px; }
    }
  `]
})
export class AssignmentDetailComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  assignment = signal<Assignment | null>(null);
  loading = signal(true);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.loadAssignment(id);
    }
  }

  loadAssignment(id: string): void {
    this.loading.set(true);
    this.api.getAssignment(id).subscribe({
      next: (res) => { this.assignment.set(res.data); this.loading.set(false); },
      error: () => { this.loading.set(false); },
    });
  }

  completeAssignment(): void {
    const a = this.assignment();
    if (!a) return;
    const revenue = prompt('Enter total revenue collected (KES):');
    if (revenue) {
      this.api.completeAssignment(a.id, Math.round(parseFloat(revenue) * 100)).subscribe({
        next: () => { this.toast.success('Assignment completed & earnings credited'); this.loadAssignment(a.id); },
      });
    }
  }

  cancelAssignment(): void {
    const a = this.assignment();
    if (!a) return;
    const reason = prompt('Reason for cancellation:');
    if (reason) {
      this.api.cancelAssignment(a.id, reason).subscribe({
        next: () => { this.toast.success('Assignment cancelled'); this.loadAssignment(a.id); },
      });
    }
  }

  statusBannerClass(status: string): string {
    switch (status) {
      case 'ACTIVE': return 'status-active';
      case 'SCHEDULED': return 'status-scheduled';
      case 'COMPLETED': return 'status-completed';
      case 'CANCELLED': return 'status-cancelled';
      default: return '';
    }
  }

  statusIcon(status: string): string {
    switch (status) {
      case 'ACTIVE': return 'play_circle';
      case 'SCHEDULED': return 'event';
      case 'COMPLETED': return 'check_circle';
      case 'CANCELLED': return 'cancel';
      default: return 'info';
    }
  }
}
