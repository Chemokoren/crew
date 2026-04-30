import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { Notification } from '../../../core/models';

@Component({
  selector: 'app-notification-list', standalone: true,
  imports: [CommonModule, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Notifications</h1><p class="page-subtitle">Stay updated on your operations</p></div>
      </div>
      @if (loading()) { @for(i of [1,2,3,4];track i){<div class="skeleton" style="height:64px;margin:4px 0;"></div>} }
      @else if (items().length === 0) {
        <div class="empty-state"><span class="material-icons-round empty-icon">notifications_none</span>
          <div class="empty-title">No notifications</div>
          <div class="empty-description">You're all caught up!</div>
        </div>
      } @else {
        <div class="notif-list">
          @for(n of items();track n.id){
            <div class="notif-item glass-card" [class.unread]="!n.read_at" (click)="markRead(n)">
              <div class="notif-icon"><span class="material-icons-round">{{n.channel==='SMS'?'sms':'notifications'}}</span></div>
              <div class="notif-content">
                <span class="notif-title">{{n.title}}</span>
                <span class="notif-body">{{n.body}}</span>
              </div>
              <span class="notif-time">{{n.created_at|relativeTime}}</span>
            </div>
          }
        </div>
      }
    </div>`,
  styles: [`
    .notif-list{display:flex;flex-direction:column;gap:var(--space-sm);}
    .notif-item{display:flex;align-items:flex-start;gap:var(--space-md);padding:var(--space-md)!important;cursor:pointer;transition:all var(--transition-fast);}
    .notif-item.unread{border-left:3px solid var(--color-accent);}
    .notif-icon{width:36px;height:36px;border-radius:var(--radius-md);background:rgba(0,210,255,0.12);color:var(--color-accent);display:flex;align-items:center;justify-content:center;flex-shrink:0;.material-icons-round{font-size:18px;}}
    .notif-content{flex:1;display:flex;flex-direction:column;gap:2px;}
    .notif-title{font-size:0.875rem;font-weight:600;color:var(--color-text-primary);}
    .notif-body{font-size:0.8125rem;color:var(--color-text-secondary);}
    .notif-time{font-size:0.75rem;color:var(--color-text-muted);white-space:nowrap;}
  `]
})
export class NotificationListComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  items = signal<Notification[]>([]); loading = signal(true);
  ngOnInit() { this.load(); }
  load() { this.loading.set(true); this.api.getNotifications({per_page:'50'}).subscribe({next:r=>{this.items.set(r.data);this.loading.set(false);},error:()=>this.loading.set(false)}); }
  markRead(n: Notification) {
    if (!n.read_at) {
      this.api.markNotificationRead(n.id).subscribe({next:()=>{
        this.items.update(items=>items.map(i=>i.id===n.id?{...i,read_at:new Date().toISOString()}:i));
      }});
    }
  }
}
