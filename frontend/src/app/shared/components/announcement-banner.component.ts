import { Component, inject, OnInit, OnDestroy, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';
import { SystemAnnouncement } from '../../core/models';

@Component({
  selector: 'app-announcement-banner',
  standalone: true,
  imports: [CommonModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @for (a of announcements(); track a.id) {
      @if (!dismissed().has(a.id)) {
        <div class="announcement-banner" [class]="'severity-' + a.severity.toLowerCase()">
          <div class="banner-content">
            <span class="material-icons-round banner-icon">{{ severityIcon(a.severity) }}</span>
            <div class="banner-text">
              <strong>{{ a.title }}</strong>
              <span class="banner-body">{{ a.body }}</span>
            </div>
          </div>
          <button class="banner-dismiss" (click)="dismiss(a.id)" title="Dismiss">
            <span class="material-icons-round">close</span>
          </button>
        </div>
      }
    }
  `,
  styles: [`
    :host {
      display: block;
      margin-left: var(--sidebar-width);
      margin-top: var(--topbar-height);
      position: sticky;
      top: var(--topbar-height);
      z-index: 50;
    }
    @media (max-width: 768px) {
      :host { margin-left: 0; }
    }

    .announcement-banner {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 10px 20px;
      font-size: 0.8125rem;
      animation: slideDown 0.3s ease-out;
    }

    .severity-info {
      background: linear-gradient(135deg, rgba(99,102,241,0.12), rgba(139,92,246,0.08));
      color: #6366f1;
      border-bottom: 1px solid rgba(99,102,241,0.15);
    }
    .severity-warning {
      background: linear-gradient(135deg, rgba(245,158,11,0.15), rgba(234,88,12,0.08));
      color: #d97706;
      border-bottom: 1px solid rgba(245,158,11,0.15);
    }
    .severity-critical {
      background: linear-gradient(135deg, rgba(239,68,68,0.15), rgba(220,38,38,0.08));
      color: #dc2626;
      border-bottom: 1px solid rgba(239,68,68,0.15);
    }

    .banner-content {
      display: flex;
      align-items: center;
      gap: 10px;
      flex: 1;
      min-width: 0;
    }

    .banner-icon { font-size: 20px; flex-shrink: 0; }

    .banner-text {
      display: flex;
      align-items: center;
      gap: 8px;
      flex-wrap: wrap;
      strong { font-weight: 700; }
    }

    .banner-body { opacity: 0.85; }

    .banner-dismiss {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 28px;
      height: 28px;
      border-radius: 6px;
      border: none;
      background: transparent;
      cursor: pointer;
      color: inherit;
      opacity: 0.6;
      transition: all 0.15s;
      flex-shrink: 0;
      .material-icons-round { font-size: 18px; }
      &:hover { opacity: 1; background: rgba(0,0,0,0.08); }
    }

    @keyframes slideDown {
      from { opacity: 0; transform: translateY(-8px); }
      to { opacity: 1; transform: translateY(0); }
    }
  `]
})
export class AnnouncementBannerComponent implements OnInit, OnDestroy {
  private api = inject(ApiService);
  private pollInterval: ReturnType<typeof setInterval> | null = null;

  announcements = signal<SystemAnnouncement[]>([]);
  dismissed = signal<Set<string>>(new Set());

  ngOnInit(): void {
    // Restore dismissed state from session
    try {
      const stored = sessionStorage.getItem('dismissed_announcements');
      if (stored) this.dismissed.set(new Set(JSON.parse(stored)));
    } catch {}
    this.loadAnnouncements();
    // Poll every 5 minutes for new announcements
    this.pollInterval = setInterval(() => this.loadAnnouncements(), 5 * 60 * 1000);
  }

  ngOnDestroy(): void {
    if (this.pollInterval) clearInterval(this.pollInterval);
  }

  loadAnnouncements(): void {
    this.api.getActiveAnnouncements().subscribe({
      next: r => this.announcements.set(r.data || []),
    });
  }

  dismiss(id: string): void {
    const d = new Set(this.dismissed());
    d.add(id);
    this.dismissed.set(d);
    // Persist for session
    try { sessionStorage.setItem('dismissed_announcements', JSON.stringify([...d])); } catch {}
  }

  severityIcon(severity: string): string {
    switch (severity) {
      case 'CRITICAL': return 'error';
      case 'WARNING': return 'warning';
      default: return 'info';
    }
  }
}
