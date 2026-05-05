import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';
import { CrewMember, Organization, Vehicle } from '../../../core/models';

@Component({
  selector: 'app-bulk-assignment',
  standalone: true,
  imports: [CommonModule, FormsModule, AutocompleteComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <button class="btn btn-ghost btn-sm" style="margin-bottom:var(--space-xs);" (click)="goBack()">
            <span class="material-icons-round" style="font-size:16px;">arrow_back</span> Back to Assignments
          </button>
          <h1 class="page-title">Bulk Assignments</h1>
          <p class="page-subtitle">Generate recurring assignments for multiple workers across a date range</p>
        </div>
      </div>

      <div class="bulk-layout">
        <!-- Step 1: Select Workers -->
        <div class="glass-card bulk-section">
          <div class="bulk-step-header">
            <span class="bulk-step-badge">1</span>
            <div>
              <h3 class="bulk-step-title">Select Workers</h3>
              <p class="bulk-step-desc">Choose crew members to assign</p>
            </div>
          </div>

          <div style="display:flex;gap:var(--space-sm);align-items:center;flex-wrap:wrap;margin-bottom:var(--space-md);">
            <div style="flex:1;min-width:250px;">
              <app-autocomplete [options]="crewOptions()" placeholder="Search workers..." [(ngModel)]="tempCrewId" inputId="bulk-crew-search"></app-autocomplete>
            </div>
            <button class="btn btn-primary btn-sm" (click)="addWorker()" [disabled]="!tempCrewId" id="btn-add-worker">
              <span class="material-icons-round" style="font-size:16px;">add</span> Add
            </button>
            <button class="btn btn-secondary btn-sm" (click)="addAllWorkers()" id="btn-add-all">Add All</button>
          </div>

          @if (selectedWorkers().length > 0) {
            <div class="selected-chips">
              @for (w of selectedWorkers(); track w.id) {
                <div class="chip">
                  <span>{{ w.full_name }}</span>
                  <button class="chip-remove" (click)="removeWorker(w.id)">
                    <span class="material-icons-round" style="font-size:14px;">close</span>
                  </button>
                </div>
              }
            </div>
            <div class="bulk-count">{{ selectedWorkers().length }} worker{{ selectedWorkers().length > 1 ? 's' : '' }} selected</div>
          }
        </div>

        <!-- Step 2: Configure Schedule -->
        <div class="glass-card bulk-section">
          <div class="bulk-step-header">
            <span class="bulk-step-badge">2</span>
            <div>
              <h3 class="bulk-step-title">Schedule</h3>
              <p class="bulk-step-desc">Set date range, work type, and earning model</p>
            </div>
          </div>

          <div class="form-grid">
            <div class="form-group">
              <label class="form-label">Organization</label>
              <app-autocomplete [options]="saccoOptions()" [ngModel]="organizationId()" (ngModelChange)="organizationId.set($event)" placeholder="Select Organization..." inputId="bulk-sacco"></app-autocomplete>
            </div>
            <div class="form-group">
              <label class="form-label">Work Type</label>
              <select class="form-select" [(ngModel)]="config.work_type" id="bulk-work-type">
                <option value="SHIFT">Shift</option><option value="DAILY">Daily</option>
                <option value="HOURLY">Hourly</option><option value="TASK">Task</option>
                <option value="PROJECT">Project</option><option value="BOOKING">Booking</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">
                <span class="material-icons-round form-label-icon">calendar_today</span> Start Date
              </label>
              <input class="form-input" type="date" [ngModel]="startDate()" (ngModelChange)="startDate.set($event)" id="bulk-start-date" />
            </div>
            <div class="form-group">
              <label class="form-label">
                <span class="material-icons-round form-label-icon">event</span> End Date
              </label>
              <input class="form-input" type="date" [ngModel]="endDate()" (ngModelChange)="endDate.set($event)" id="bulk-end-date" />
            </div>
            <div class="form-group">
              <label class="form-label">Shift Start Time</label>
              <input class="form-input" type="time" [(ngModel)]="config.shift_time" id="bulk-shift-time" />
            </div>
            <div class="form-group">
              <label class="form-label">Earning Model</label>
              <select class="form-select" [(ngModel)]="config.earning_model" id="bulk-earning">
                <option value="FIXED">Fixed</option><option value="COMMISSION">Commission</option><option value="HYBRID">Hybrid</option>
              </select>
            </div>
            @if (config.earning_model === 'FIXED' || config.earning_model === 'HYBRID') {
              <div class="form-group">
                <label class="form-label">Fixed Amount (KES)</label>
                <input class="form-input" type="number" [(ngModel)]="config.fixed_amount" placeholder="500" id="bulk-fixed" />
              </div>
            }
            @if (config.earning_model === 'COMMISSION' || config.earning_model === 'HYBRID') {
              <div class="form-group">
                <label class="form-label">Commission Rate (%)</label>
                <input class="form-input" type="number" [(ngModel)]="config.commission_rate" step="0.01" placeholder="10" id="bulk-comm" />
              </div>
            }
            <div class="form-group" style="grid-column:1/-1;">
              <label class="form-label">
                <input type="checkbox" [ngModel]="skipWeekends()" (ngModelChange)="skipWeekends.set($event)" id="bulk-skip-weekends" style="margin-right:6px;">
                Skip weekends (Sat & Sun)
              </label>
            </div>
          </div>
        </div>

        <!-- Step 3: Preview & Submit -->
        <div class="glass-card bulk-section">
          <div class="bulk-step-header">
            <span class="bulk-step-badge">3</span>
            <div>
              <h3 class="bulk-step-title">Preview & Submit</h3>
              <p class="bulk-step-desc">Review before generating assignments</p>
            </div>
          </div>

          <div class="preview-stats">
            <div class="preview-stat">
              <span class="preview-stat-value">{{ selectedWorkers().length }}</span>
              <span class="preview-stat-label">Workers</span>
            </div>
            <div class="preview-stat">
              <span class="preview-stat-value">{{ dateCount() }}</span>
              <span class="preview-stat-label">Days</span>
            </div>
            <div class="preview-stat">
              <span class="preview-stat-value">{{ totalAssignments() }}</span>
              <span class="preview-stat-label">Total Assignments</span>
            </div>
          </div>

          <button class="btn btn-primary" style="width:100%;margin-top:var(--space-md);"
                  (click)="submit()" [disabled]="submitting() || !canSubmit()" id="btn-submit-bulk">
            @if (submitting()) {
              <span class="material-icons-round spin" style="font-size:18px;">sync</span> Generating...
            } @else {
              <span class="material-icons-round" style="font-size:18px;">rocket_launch</span>
              Generate {{ totalAssignments() }} Assignments
            }
          </button>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .bulk-layout { display: flex; flex-direction: column; gap: var(--space-lg); }
    .bulk-section { padding: var(--space-lg) !important; }
    .bulk-step-header { display: flex; align-items: flex-start; gap: var(--space-md); margin-bottom: var(--space-lg); }
    .bulk-step-badge {
      width: 32px; height: 32px; border-radius: 50%; background: var(--gradient-accent);
      display: flex; align-items: center; justify-content: center;
      font-size: 0.875rem; font-weight: 700; color: var(--color-text-inverse); flex-shrink: 0;
    }
    .bulk-step-title { font-size: 1rem; font-weight: 700; margin-bottom: 2px; }
    .bulk-step-desc { font-size: 0.8rem; color: var(--color-text-muted); }

    .selected-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: var(--space-sm); }
    .chip {
      display: inline-flex; align-items: center; gap: 6px;
      padding: 4px 10px; border-radius: var(--radius-full);
      background: rgba(99,102,241,0.12); color: #818cf8;
      font-size: 0.8rem; font-weight: 500;
    }
    .chip-remove { background: none; border: none; color: inherit; cursor: pointer; padding: 0; display: flex; opacity: 0.7; }
    .chip-remove:hover { opacity: 1; }
    .bulk-count { font-size: 0.75rem; color: var(--color-text-muted); }

    .form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); }
    .form-label-icon { font-size: 14px; vertical-align: middle; margin-right: 4px; color: var(--color-accent); }

    .preview-stats { display: flex; gap: var(--space-lg); justify-content: center; }
    .preview-stat { display: flex; flex-direction: column; align-items: center; }
    .preview-stat-value { font-size: 1.75rem; font-weight: 800; color: var(--color-accent); font-family: var(--font-heading); }
    .preview-stat-label { font-size: 0.75rem; color: var(--color-text-muted); }

    .spin { animation: spin 1s linear infinite; }
    @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

    input[type="date"], input[type="time"] { color-scheme: dark; cursor: pointer; }
    input[type="date"]::-webkit-calendar-picker-indicator,
    input[type="time"]::-webkit-calendar-picker-indicator {
      filter: invert(0.7) sepia(1) saturate(5) hue-rotate(175deg); cursor: pointer;
    }

    @media (max-width: 768px) {
      .form-grid { grid-template-columns: 1fr; }
      .preview-stats { flex-direction: column; align-items: center; }
    }
  `]
})
export class BulkAssignmentComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private router = inject(Router);

  crewMembers = signal<CrewMember[]>([]);
  saccos = signal<Organization[]>([]);
  selectedWorkers = signal<CrewMember[]>([]);
  submitting = signal(false);
  tempCrewId = '';

  /* Reactive fields used in computed() must be signals */
  organizationId = signal('');
  startDate = signal('');
  endDate = signal('');
  skipWeekends = signal(true);

  /* Non-reactive config fields – only read on submit */
  config = {
    work_type: 'SHIFT', shift_time: '07:00',
    earning_model: 'FIXED', fixed_amount: 0, commission_rate: 0,
  };

  crewOptions = computed<AutocompleteOption[]>(() =>
    this.crewMembers()
      .filter(c => !this.selectedWorkers().find(w => w.id === c.id))
      .map(c => ({ value: c.id, label: c.full_name, sublabel: `${c.crew_id} · ${c.role}`, searchText: `${c.full_name} ${c.crew_id} ${c.role}` }))
  );

  saccoOptions = computed<AutocompleteOption[]>(() =>
    this.saccos().map(s => ({ value: s.id, label: s.name, sublabel: s.county, searchText: `${s.name} ${s.county}` }))
  );

  dateCount = computed(() => {
    const sd = this.startDate();
    const ed = this.endDate();
    const skip = this.skipWeekends();
    if (!sd || !ed) return 0;
    const start = new Date(sd);
    const end = new Date(ed);
    let count = 0;
    for (const d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
      if (skip && (d.getDay() === 0 || d.getDay() === 6)) continue;
      count++;
    }
    return count;
  });

  totalAssignments = computed(() => this.selectedWorkers().length * this.dateCount());

  ngOnInit(): void {
    this.api.getCrewMembers({ per_page: '200' }).subscribe({ next: r => this.crewMembers.set(r.data) });
    this.api.getOrganizations({ per_page: '200' }).subscribe({ next: r => this.saccos.set(r.data) });
  }

  goBack(): void { this.router.navigate(['/assignments']); }

  addWorker(): void {
    if (!this.tempCrewId) return;
    const crew = this.crewMembers().find(c => c.id === this.tempCrewId);
    if (crew && !this.selectedWorkers().find(w => w.id === crew.id)) {
      this.selectedWorkers.update(list => [...list, crew]);
    }
    this.tempCrewId = '';
  }

  addAllWorkers(): void {
    this.selectedWorkers.set([...this.crewMembers()]);
  }

  removeWorker(id: string): void {
    this.selectedWorkers.update(list => list.filter(w => w.id !== id));
  }

  canSubmit = computed(() =>
    this.selectedWorkers().length > 0 && !!this.organizationId() && this.dateCount() > 0
  );

  submit(): void {
    this.submitting.set(true);
    const assignments: Record<string, unknown>[] = [];
    const start = new Date(this.startDate());
    const end = new Date(this.endDate());
    const skip = this.skipWeekends();

    for (const d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
      if (skip && (d.getDay() === 0 || d.getDay() === 6)) continue;
      const dateStr = d.toISOString().slice(0, 10);
      for (const w of this.selectedWorkers()) {
        assignments.push({
          crew_member_id: w.id,
          organization_id: this.organizationId(),
          work_type: this.config.work_type,
          shift_date: dateStr,
          shift_start: new Date(`${dateStr}T${this.config.shift_time}:00`).toISOString(),
          earning_model: this.config.earning_model,
          fixed_amount_cents: Math.round(this.config.fixed_amount * 100),
          commission_rate: this.config.commission_rate / 100,
        });
      }
    }

    this.api.bulkCreateAssignments({ assignments }).subscribe({
      next: () => {
        this.toast.success(`${assignments.length} assignments created successfully`);
        this.submitting.set(false);
        this.router.navigate(['/assignments']);
      },
      error: (err) => {
        const msg = err?.error?.message || err?.error?.error || 'Failed to create assignments. Please check your inputs.';
        this.toast.error(msg);
        this.submitting.set(false);
      },
    });
  }
}
