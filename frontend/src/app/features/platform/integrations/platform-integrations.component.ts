import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';

interface IntegrationProvider { name: string; slug: string; icon: string; status: 'active' | 'inactive' | 'error'; type: string; description: string; lastPing?: string; }

@Component({
  selector: 'app-platform-integrations',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header"><div>
        <h1 class="page-title grad-title">Integrations</h1>
        <p class="page-subtitle">Monitor external services, webhooks, and API health</p>
      </div></div>

      <!-- Service Health Overview -->
      <div class="health-grid">
        @for (p of providers(); track p.slug) {
          <div class="health-card glass-card" [class]="'status-' + p.status">
            <div class="health-header">
              <div class="health-icon-wrap"><span class="material-icons-round">{{ p.icon }}</span></div>
              <span class="status-dot" [class]="'dot-' + p.status"></span>
            </div>
            <h3>{{ p.name }}</h3>
            <span class="health-type">{{ p.type }}</span>
            <p class="health-desc">{{ p.description }}</p>
            <div class="health-footer">
              <span class="status-badge" [class]="'sb-' + p.status">{{ p.status | titlecase }}</span>
              @if (p.lastPing) { <span class="health-ping">{{ p.lastPing }}</span> }
            </div>
          </div>
        }
      </div>

      <!-- Webhook Logs -->
      <div class="sec-header" style="margin-top:var(--space-xl)">
        <div><h2 class="sec-title"><span class="material-icons-round" style="color:#f59e0b">webhook</span> Webhook Activity</h2>
        <p class="sec-desc">Recent webhook events and callback logs</p></div>
      </div>
      <div class="glass-card log-table">
        <table>
          <thead><tr><th>Time</th><th>Provider</th><th>Event</th><th>Status</th><th>Response</th></tr></thead>
          <tbody>
            @for (log of webhookLogs(); track log.id) {
              <tr>
                <td class="mono">{{ log.time | date:'short' }}</td>
                <td><span class="ch-badge" [style.background]="'rgba(99,102,241,0.12)'" [style.color]="'#6366f1'">{{ log.provider }}</span></td>
                <td>{{ log.event }}</td>
                <td><span class="badge" [ngClass]="log.statusCode < 400 ? 'badge-success' : 'badge-danger'">{{ log.statusCode }}</span></td>
                <td class="mono truncate">{{ log.response }}</td>
              </tr>
            }
            @if (webhookLogs().length === 0) {
              <tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:var(--space-xl)">No webhook events recorded yet</td></tr>
            }
          </tbody>
        </table>
      </div>

      <!-- API Keys -->
      <div class="sec-header" style="margin-top:var(--space-xl)">
        <div><h2 class="sec-title"><span class="material-icons-round" style="color:#8b5cf6">vpn_key</span> API Keys</h2>
        <p class="sec-desc">Service API keys for programmatic access</p></div>
      </div>
      <div class="keys-grid">
        @for (key of apiKeys(); track key.name) {
          <div class="key-card glass-card">
            <div class="key-header"><span class="material-icons-round" style="font-size:18px;color:#8b5cf6">key</span><strong>{{ key.name }}</strong></div>
            <div class="key-value"><code>{{ key.masked }}</code>
              <button class="icon-btn" (click)="copyKey(key.value)" title="Copy"><span class="material-icons-round" style="font-size:16px">content_copy</span></button>
            </div>
            <span class="key-meta">Created {{ key.created | date:'mediumDate' }}</span>
          </div>
        }
        @if (apiKeys().length === 0) {
          <div class="glass-card" style="padding:var(--space-lg);text-align:center;color:var(--color-text-muted)">No API keys configured</div>
        }
      </div>
    </div>
  `,
  styleUrl: './platform-integrations.component.scss',
})
export class PlatformIntegrationsComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  providers = signal<IntegrationProvider[]>([
    { name: 'M-Pesa (JamboPay)', slug: 'jambopay', icon: 'account_balance', status: 'active', type: 'Payment', description: 'STK push, C2B collections, and B2C payouts', lastPing: '2ms' },
    { name: 'SMS Gateway', slug: 'sms', icon: 'sms', status: 'active', type: 'Messaging', description: 'Optimize SMS provider for OTP and notifications', lastPing: '15ms' },
    { name: 'USSD Gateway', slug: 'ussd', icon: 'dialpad', status: 'active', type: 'Channel', description: "Africa's Talking USSD session handler", lastPing: '8ms' },
    { name: 'WhatsApp Cloud API', slug: 'whatsapp', icon: 'chat', status: 'inactive', type: 'Messaging', description: 'Meta WhatsApp Business messaging', lastPing: '-' },
    { name: 'IPRS (KYC)', slug: 'iprs', icon: 'verified_user', status: 'active', type: 'Identity', description: 'National ID verification via IPRS API', lastPing: '120ms' },
    { name: 'PerPay Payroll', slug: 'perpay', icon: 'payments', status: 'active', type: 'Payroll', description: 'External payroll submission and reconciliation', lastPing: '45ms' },
    { name: 'Email (SMTP)', slug: 'email', icon: 'email', status: 'active', type: 'Messaging', description: 'Gmail SMTP for transactional emails', lastPing: '85ms' },
    { name: 'MinIO Storage', slug: 'minio', icon: 'cloud_upload', status: 'active', type: 'Storage', description: 'Object storage for documents and files', lastPing: '5ms' },
  ]);

  webhookLogs = signal<{ id: string; time: string; provider: string; event: string; statusCode: number; response: string }[]>([]);
  apiKeys = signal<{ name: string; value: string; masked: string; created: string }[]>([]);

  ngOnInit() {
    // Load from system settings if configured
    this.api.getSystemSettings('integration.').subscribe({
      next: r => {
        const settings = r.data || [];
        // Update provider status from settings
        const providers = this.providers();
        for (const s of settings) {
          const slug = s.key.replace('integration.', '').replace('_status', '');
          const p = providers.find(p => p.slug === slug);
          if (p) p.status = s.value as any;
        }
        this.providers.set([...providers]);
      },
    });
  }

  copyKey(value: string) {
    navigator.clipboard.writeText(value).then(
      () => this.toast.success('API key copied'),
      () => this.toast.error('Failed to copy')
    );
  }
}
