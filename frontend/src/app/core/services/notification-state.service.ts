import { Injectable, inject, signal, computed } from '@angular/core';
import { ApiService } from './api.service';
import { Notification } from '../models';

/**
 * Shared state service for notification unread count.
 * Consumed by sidebar and topbar for badge display (Task 150).
 * Polls every 60s and also refreshes on-demand.
 */
@Injectable({ providedIn: 'root' })
export class NotificationStateService {
  private api = inject(ApiService);
  private notifications = signal<Notification[]>([]);
  private intervalId: ReturnType<typeof setInterval> | null = null;

  /** Computed unread count for badges */
  readonly unreadCount = computed(() => this.notifications().filter(n => !n.read_at).length);

  /** Start polling (called once from app shell) */
  init(): void {
    this.refresh();
    if (!this.intervalId) {
      this.intervalId = setInterval(() => this.refresh(), 60_000);
    }
  }

  /** Manual refresh — call after marking a notification as read */
  refresh(): void {
    this.api.getNotifications({ per_page: '50' }).subscribe({
      next: r => this.notifications.set(r.data),
      error: () => {},
    });
  }

  /** Mark a notification read locally + on server */
  markRead(id: string): void {
    this.api.markNotificationRead(id).subscribe({
      next: () => {
        this.notifications.update(items =>
          items.map(n => n.id === id ? { ...n, read_at: new Date().toISOString() } : n)
        );
      },
    });
  }

  destroy(): void {
    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
  }
}
