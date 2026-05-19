import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { NotificationStateService } from '../../../core/services/notification-state.service';
import { ToastService } from '../../../core/services/toast.service';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { Notification } from '../../../core/models';

@Component({
  selector: 'app-notification-list', standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Notifications</h1><p class="page-subtitle">Stay updated on your operations</p></div>
        <div class="page-actions">
          <a routerLink="/notifications/preferences" class="btn btn-secondary" id="btn-prefs">
            <span class="material-icons-round">settings</span> Preferences
          </a>
          @if (unreadCount() > 0) {
            <button class="btn btn-ghost" (click)="markAllRead()" style="color:var(--color-accent);">
              <span class="material-icons-round" style="font-size:18px;">done_all</span> Mark all read
            </button>
          }
        </div>
      </div>

      <!-- Filters (Task 151) -->
      <div class="filters-bar">
        <div class="filter-pills">
          <button class="pill" [class.active]="filterRead() === ''" (click)="setFilterRead('')">All</button>
          <button class="pill" [class.active]="filterRead() === 'unread'" (click)="setFilterRead('unread')">
            Unread
            @if (unreadCount() > 0) { <span class="pill-badge">{{ unreadCount() }}</span> }
          </button>
          <button class="pill" [class.active]="filterRead() === 'read'" (click)="setFilterRead('read')">Read</button>
        </div>
        <select class="form-select filter-select" [ngModel]="filterChannel()" (ngModelChange)="filterChannel.set($event)" id="filter-channel">
          <option value="">All Channels</option>
          <option value="SMS">SMS</option>
          <option value="PUSH">Push</option>
          <option value="IN_APP">In-App</option>
        </select>
      </div>

      @if (loading()) { @for(i of [1,2,3,4];track i){<div class="skeleton" style="height:64px;margin:4px 0;"></div>} }
      @else if (filtered().length === 0) {
        <div class="empty-state"><span class="material-icons-round empty-icon">notifications_none</span>
          <div class="empty-title">{{ hasFilters() ? 'No matching notifications' : 'No notifications' }}</div>
          <div class="empty-description">{{ hasFilters() ? 'Try adjusting your filters.' : "You're all caught up!" }}</div>
        </div>
      } @else {
        <div class="notif-list">
          @for(n of filtered();track n.id){
            <div class="notif-item glass-card" [class.unread]="!n.read_at">
              <div class="notif-icon" [ngClass]="channelClass(n.channel)">
                <span class="material-icons-round">{{ channelIcon(n.channel) }}</span>
              </div>
              <div class="notif-content">
                <span class="notif-title">{{n.title}}</span>
                <span class="notif-body">{{n.body}}</span>
              </div>
              <div class="notif-meta">
                <span class="notif-channel badge" [ngClass]="channelBadge(n.channel)">{{n.channel}}</span>
                <span class="notif-time">{{n.created_at|relativeTime}}</span>
                @if (!n.read_at) {
                  <button class="btn btn-sm btn-ghost notif-mark-btn" (click)="markRead(n)" title="Mark as read">
                    <span class="material-icons-round" style="font-size:16px;">check</span>
                  </button>
                }
              </div>
            </div>
          }
        </div>
      }
    </div>`,
  styles: [`
    .notif-list{display:flex;flex-direction:column;gap:var(--space-sm);}
    .notif-item{display:flex;align-items:flex-start;gap:var(--space-md);padding:var(--space-md)!important;cursor:pointer;transition:all var(--transition-fast);}
    .notif-item:hover{background:rgba(255,255,255,0.02);}
    .notif-item.unread{border-left:3px solid var(--color-accent);}
    .notif-icon{width:36px;height:36px;border-radius:var(--radius-md);display:flex;align-items:center;justify-content:center;flex-shrink:0;.material-icons-round{font-size:18px;}}
    .notif-icon.ch-sms{background:rgba(34,197,94,0.12);color:#22c55e;}
    .notif-icon.ch-push{background:rgba(0,210,255,0.12);color:var(--color-accent);}
    .notif-icon.ch-in_app{background:rgba(168,85,247,0.12);color:#a855f7;}
    .notif-content{flex:1;display:flex;flex-direction:column;gap:2px;}
    .notif-title{font-size:0.875rem;font-weight:600;color:var(--color-text-primary);}
    .notif-body{font-size:0.8125rem;color:var(--color-text-secondary);}
    .notif-meta{display:flex;flex-direction:column;align-items:flex-end;gap:4px;flex-shrink:0;}
    .notif-time{font-size:0.7rem;color:var(--color-text-muted);white-space:nowrap;}
    .notif-channel{font-size:0.6rem !important;}

    .filters-bar{display:flex;gap:var(--space-md);flex-wrap:wrap;margin-bottom:var(--space-lg);align-items:center;}
    .filter-pills{display:flex;gap:4px;background:var(--color-surface-alt);border-radius:var(--radius-md);padding:3px;}
    .pill{
      padding:6px 14px;border:none;background:none;border-radius:var(--radius-sm);
      font-size:0.8125rem;font-weight:500;color:var(--color-text-muted);cursor:pointer;
      transition:all var(--transition-fast);display:flex;align-items:center;gap:4px;
    }
    .pill:hover{color:var(--color-text-primary);}
    .pill.active{background:var(--color-accent);color:#fff;}
    .pill-badge{
      background:rgba(255,255,255,0.2);border-radius:8px;padding:1px 6px;font-size:0.6875rem;font-weight:700;
    }
    .filter-select{min-width:140px;max-width:180px;}
    .notif-mark-btn { margin-top: 4px; padding: 2px 6px; min-height: 24px; color: var(--color-text-muted); }
    .notif-mark-btn:hover { color: var(--color-success); background: rgba(34,197,94,0.1); }
  `]
})
export class NotificationListComponent implements OnInit {
  private api = inject(ApiService);
  private notifState = inject(NotificationStateService);
  private toast = inject(ToastService);

  items = signal<Notification[]>([]);
  loading = signal(true);
  filterRead = signal('');
  filterChannel = signal('');

  unreadCount = computed(() => this.items().filter(n => !n.read_at).length);

  ngOnInit() { this.load(); }

  load(): void {
    this.loading.set(true);
    this.api.getNotifications({ per_page: '50' }).subscribe({
      next: r => { this.items.set(r.data); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  filtered = computed(() => {
    let list = this.items();
    const readFilter = this.filterRead();
    const channelFilter = this.filterChannel();
    
    if (readFilter === 'unread') list = list.filter(n => !n.read_at);
    if (readFilter === 'read') list = list.filter(n => !!n.read_at);
    if (channelFilter) list = list.filter(n => n.channel === channelFilter);
    return list;
  });

  hasFilters(): boolean { return !!(this.filterRead() || this.filterChannel()); }
  setFilterRead(val: string): void { this.filterRead.set(val); }

  markRead(n: Notification): void {
    if (!n.read_at) {
      this.api.markNotificationRead(n.id).subscribe({
        next: () => {
          this.items.update(items => items.map(i => i.id === n.id ? { ...i, read_at: new Date().toISOString() } : i));
          this.notifState.refresh(); // sync badge
        },
      });
    }
  }

  markAllRead(): void {
    const unread = this.items().filter(n => !n.read_at);
    unread.forEach(n => {
      this.api.markNotificationRead(n.id).subscribe();
    });
    this.items.update(items => items.map(n => n.read_at ? n : { ...n, read_at: new Date().toISOString() }));
    this.notifState.refresh();
    this.toast.success(`Marked ${unread.length} as read`);
  }

  channelIcon(ch: string): string {
    return ch === 'SMS' ? 'sms' : ch === 'PUSH' ? 'notifications_active' : 'inbox';
  }
  channelClass(ch: string): string {
    return ch === 'SMS' ? 'ch-sms' : ch === 'PUSH' ? 'ch-push' : 'ch-in_app';
  }
  channelBadge(ch: string): string {
    return ch === 'SMS' ? 'badge-success' : ch === 'PUSH' ? 'badge-info' : 'badge-accent';
  }
}
