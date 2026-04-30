import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { NotificationPreference } from '../../../core/models';

interface PrefToggle {
  key: keyof NotificationPreference;
  label: string;
  description: string;
  icon: string;
  iconBg: string;
  iconColor: string;
}

@Component({
  selector: 'app-notification-preferences',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <button class="btn btn-ghost btn-sm" routerLink="/notifications" style="margin-bottom:var(--space-xs);">
            <span class="material-icons-round" style="font-size:16px;">arrow_back</span> Back to Notifications
          </button>
          <h1 class="page-title">Notification Preferences</h1>
          <p class="page-subtitle">Control how and when you receive notifications</p>
        </div>
      </div>

      @if (loading()) {
        @for (i of [1,2,3,4]; track i) { <div class="skeleton" style="height:72px;margin:6px 0;border-radius:var(--radius-lg);"></div> }
      } @else {
        <div class="prefs-list">
          @for (toggle of toggles; track toggle.key) {
            <div class="pref-card glass-card">
              <div class="pref-left">
                <div class="pref-icon" [style.background]="toggle.iconBg" [style.color]="toggle.iconColor">
                  <span class="material-icons-round">{{ toggle.icon }}</span>
                </div>
                <div class="pref-info">
                  <span class="pref-label">{{ toggle.label }}</span>
                  <span class="pref-desc">{{ toggle.description }}</span>
                </div>
              </div>
              <label class="toggle-switch">
                <input type="checkbox" [checked]="getVal(toggle.key)" (change)="onToggle(toggle.key, $event)" />
                <span class="toggle-slider"></span>
              </label>
            </div>
          }
        </div>

        @if (prefs()?.updated_at) {
          <p class="last-updated">Last updated: {{ prefs()!.updated_at | date:'medium' }}</p>
        }
      }
    </div>
  `,
  styles: [`
    .prefs-list { display: flex; flex-direction: column; gap: var(--space-sm); }
    .pref-card {
      display: flex; align-items: center; justify-content: space-between;
      padding: var(--space-md) var(--space-lg) !important;
    }
    .pref-left { display: flex; align-items: center; gap: var(--space-md); }
    .pref-icon {
      width: 42px; height: 42px; border-radius: var(--radius-md);
      display: flex; align-items: center; justify-content: center; flex-shrink: 0;
      .material-icons-round { font-size: 20px; }
    }
    .pref-info { display: flex; flex-direction: column; gap: 2px; }
    .pref-label { font-size: 0.875rem; font-weight: 600; color: var(--color-text-primary); }
    .pref-desc { font-size: 0.75rem; color: var(--color-text-muted); }
    .last-updated { font-size: 0.75rem; color: var(--color-text-muted); margin-top: var(--space-md); text-align: center; }

    /* Toggle switch */
    .toggle-switch { position: relative; display: inline-block; width: 48px; height: 26px; flex-shrink: 0; }
    .toggle-switch input { opacity: 0; width: 0; height: 0; }
    .toggle-slider {
      position: absolute; cursor: pointer; inset: 0;
      background: rgba(255,255,255,0.08); border-radius: 13px;
      transition: all 0.25s ease;
      &::before {
        content: ''; position: absolute; height: 20px; width: 20px;
        left: 3px; bottom: 3px; background: var(--color-text-muted);
        border-radius: 50%; transition: all 0.25s ease;
      }
    }
    .toggle-switch input:checked + .toggle-slider {
      background: var(--color-accent);
      &::before { transform: translateX(22px); background: #fff; }
    }
  `]
})
export class NotificationPreferencesComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  prefs = signal<NotificationPreference | null>(null);
  loading = signal(true);

  readonly toggles: PrefToggle[] = [
    {
      key: 'sms_opt_in', label: 'SMS Notifications', description: 'Receive alerts via text message',
      icon: 'sms', iconBg: 'rgba(34,197,94,0.12)', iconColor: '#22c55e',
    },
    {
      key: 'push_opt_in', label: 'Push Notifications', description: 'Browser and mobile push alerts',
      icon: 'notifications_active', iconBg: 'rgba(0,210,255,0.12)', iconColor: 'var(--color-accent)',
    },
    {
      key: 'in_app_opt_in', label: 'In-App Notifications', description: 'Show notifications inside the app',
      icon: 'inbox', iconBg: 'rgba(168,85,247,0.12)', iconColor: '#a855f7',
    },
    {
      key: 'marketing_opt_in', label: 'Marketing & Promotions', description: 'Product updates, tips, and offers',
      icon: 'campaign', iconBg: 'rgba(251,191,36,0.12)', iconColor: '#fbbf24',
    },
  ];

  ngOnInit(): void {
    this.api.getNotificationPreferences().subscribe({
      next: r => { this.prefs.set(r.data); this.loading.set(false); },
      error: () => {
        // Default if no prefs exist yet
        this.prefs.set({ sms_opt_in: true, push_opt_in: true, in_app_opt_in: true, marketing_opt_in: false });
        this.loading.set(false);
      },
    });
  }

  getVal(key: keyof NotificationPreference): boolean {
    const p = this.prefs();
    if (!p) return false;
    return !!(p as unknown as Record<string, unknown>)[key];
  }

  onToggle(key: keyof NotificationPreference, event: Event): void {
    const checked = (event.target as HTMLInputElement).checked;
    const updated = { ...this.prefs()!, [key]: checked };
    this.prefs.set(updated);

    this.api.updateNotificationPreferences({
      sms_opt_in: updated.sms_opt_in,
      push_opt_in: updated.push_opt_in,
      in_app_opt_in: updated.in_app_opt_in,
      marketing_opt_in: updated.marketing_opt_in,
    }).subscribe({
      next: () => this.toast.success(`${checked ? 'Enabled' : 'Disabled'}`),
      error: () => {
        // Revert
        this.prefs.set({ ...updated, [key]: !checked });
        this.toast.error('Failed to update preference');
      },
    });
  }
}
