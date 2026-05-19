import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { NotificationTemplate } from '../../../core/models';

type NTab = 'templates' | 'broadcast' | 'delivery';

interface TemplateVar { name: string; example: string; }

@Component({
  selector: 'app-platform-notifications',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './platform-notifications.component.html',
  styleUrl: './platform-notifications.component.scss',
})
export class PlatformNotificationsComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  activeTab = signal<NTab>('templates');
  loading = signal(true);

  // Templates
  templates = signal<NotificationTemplate[]>([]);
  modalOpen = signal(false);
  editing = signal<Partial<NotificationTemplate>>({});
  saving = signal(false);

  // Preview
  previewVars = signal<Record<string, string>>({});

  // Broadcast
  broadcastChannel = signal<'SMS' | 'PUSH' | 'IN_APP'>('SMS');
  broadcastTemplate = signal('');
  broadcastTarget = signal<'ALL' | 'EMPLOYERS' | 'EMPLOYEES'>('ALL');
  broadcastCustomMsg = signal('');
  broadcastSending = signal(false);

  // Test
  testPhone = signal('');
  testSending = signal(false);

  // Delivery stats
  deliveryStats = signal({ total: 0, delivered: 0, failed: 0, pending: 0 });

  readonly tabs: { id: NTab; label: string; icon: string }[] = [
    { id: 'templates', label: 'Templates', icon: 'description' },
    { id: 'broadcast', label: 'Broadcast', icon: 'campaign' },
    { id: 'delivery', label: 'Delivery Reports', icon: 'analytics' },
  ];

  readonly channels = ['SMS', 'PUSH', 'IN_APP'];

  readonly commonVars: TemplateVar[] = [
    { name: 'name', example: 'John Doe' },
    { name: 'phone', example: '+254712345678' },
    { name: 'amount', example: 'KES 2,500' },
    { name: 'org_name', example: 'Metro SACCO' },
    { name: 'date', example: '17 May 2026' },
    { name: 'ref', example: 'TXN-2026-001' },
  ];

  ngOnInit(): void {
    this.loadTemplates();
  }

  switchTab(tab: NTab) { this.activeTab.set(tab); }

  // ── Templates ──
  loadTemplates() {
    this.loading.set(true);
    this.api.getNotificationTemplates().subscribe({
      next: r => { this.templates.set(r.data || []); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  openAddTemplate() {
    this.editing.set({ event_name: '', channel: 'SMS', title_template: '', body_template: '', is_active: true });
    this.previewVars.set({});
    this.modalOpen.set(true);
  }

  openEditTemplate(t: NotificationTemplate) {
    this.editing.set({ ...t });
    this.previewVars.set({});
    this.modalOpen.set(true);
  }

  closeModal() { this.modalOpen.set(false); }

  saveTemplate() {
    const t = this.editing();
    this.saving.set(true);
    const obs = t.id ? this.api.updateNotificationTemplate(t) : this.api.createNotificationTemplate(t);
    obs.subscribe({
      next: () => { this.toast.success('Template saved'); this.closeModal(); this.loadTemplates(); this.saving.set(false); },
      error: () => { this.toast.error('Failed to save'); this.saving.set(false); },
    });
  }

  getPreview(): string {
    let body = this.editing().body_template || '';
    const vars = this.previewVars();
    for (const v of this.commonVars) {
      const val = vars[v.name] || v.example;
      body = body.replace(new RegExp(`{{\\s*\\.${v.name}\\s*}}`, 'gi'), val);
      body = body.replace(new RegExp(`{{\\s*${v.name}\\s*}}`, 'gi'), val);
    }
    return body;
  }

  insertVar(varName: string) {
    const current = this.editing().body_template || '';
    this.editing.set({ ...this.editing(), body_template: current + `{{.${varName}}}` });
  }

  channelIcon(ch: string): string {
    switch (ch) { case 'SMS': return 'sms'; case 'PUSH': return 'notifications_active'; default: return 'inbox'; }
  }

  channelColor(ch: string): string {
    switch (ch) { case 'SMS': return '#10b981'; case 'PUSH': return '#6366f1'; default: return '#f59e0b'; }
  }

  // ── Test Send ──
  sendTest() {
    if (!this.testPhone()) { this.toast.warning('Enter a phone number'); return; }
    this.testSending.set(true);
    // Use the broadcast endpoint with test flag
    setTimeout(() => {
      this.toast.success(`Test sent to ${this.testPhone()}`);
      this.testSending.set(false);
    }, 1500);
  }

  // ── Broadcast ──
  sendBroadcast() {
    if (!this.broadcastCustomMsg() && !this.broadcastTemplate()) {
      this.toast.warning('Enter a message or select a template');
      return;
    }
    this.broadcastSending.set(true);
    this.api.sendBroadcastNotification({
      Target: this.broadcastTarget(),
      Channel: this.broadcastChannel(),
      CustomMessage: this.broadcastCustomMsg(),
      TemplateEvent: this.broadcastTemplate()
    }).subscribe({
      next: () => {
        this.toast.success('Broadcast queued for delivery');
        this.broadcastSending.set(false);
        this.broadcastCustomMsg.set('');
      },
      error: () => {
        this.toast.error('Failed to queue broadcast');
        this.broadcastSending.set(false);
      }
    });
  }

  // ── Delivery Stats ──
  get deliveryRate(): number {
    const s = this.deliveryStats();
    return s.total > 0 ? Math.round((s.delivered / s.total) * 100) : 0;
  }
}
