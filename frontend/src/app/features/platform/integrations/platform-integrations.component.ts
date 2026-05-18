import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';

interface IntegrationProvider {
  name: string;
  slug: string;
  icon: string;
  status: 'active' | 'inactive' | 'unconfigured';
  type: string;
  description: string;
  enabled: boolean;
  configured: boolean;
  primary: boolean;
}

interface WebhookLog {
  id: string;
  time: string;
  provider: string;
  event: string;
  statusCode: number;
  response: string;
  processed: boolean;
}

@Component({
  selector: 'app-platform-integrations',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header"><div>
        <h1 class="page-title grad-title">Integrations</h1>
        <p class="page-subtitle">Monitor and manage external services, webhooks, and API health</p>
      </div>
      <div class="header-actions">
        <button class="btn btn-outline" (click)="refreshAll()" [disabled]="loading()" id="btn-refresh-integrations">
          <span class="material-icons-round spin-on-load" [class.spinning]="loading()">refresh</span>
          Refresh
        </button>
      </div>
      </div>

      <!-- Loading State -->
      @if (loading()) {
        <div class="loading-bar"><div class="loading-bar-inner"></div></div>
      }

      <!-- Service Health Overview -->
      <div class="health-grid">
        @for (p of providers(); track p.slug) {
          <div class="health-card glass-card" [class]="'status-' + p.status" id="integration-card-{{ p.slug }}">
            <div class="health-header">
              <div class="health-icon-wrap" [class]="'icon-' + p.type.toLowerCase()">
                <span class="material-icons-round">{{ p.icon }}</span>
              </div>
              <div class="health-header-right">
                @if (p.primary) {
                  <span class="primary-badge">PRIMARY</span>
                }
                <span class="status-dot" [class]="'dot-' + p.status" [title]="p.status"></span>
              </div>
            </div>
            <h3>{{ p.name }}</h3>
            <span class="health-type">{{ p.type }}</span>
            <p class="health-desc">{{ p.description }}</p>
            <div class="health-footer">
              <span class="status-badge" [class]="'sb-' + p.status">
                {{ p.status === 'unconfigured' ? 'No Credentials' : (p.status | titlecase) }}
              </span>
              <div class="card-actions">
                @if (p.configured) {
                  <label class="toggle-switch" [title]="p.enabled ? 'Disable' : 'Enable'">
                    <input type="checkbox" [checked]="p.enabled"
                           (change)="toggleProvider(p, $event)" [disabled]="togglingSlug() === p.slug"
                           id="toggle-{{ p.slug }}">
                    <span class="toggle-slider"></span>
                  </label>
                } @else {
                  <button class="btn-configure" title="Configure credentials in .env" disabled>
                    <span class="material-icons-round" style="font-size:16px">settings</span>
                  </button>
                }
              </div>
            </div>
          </div>
        }
      </div>

      <!-- Webhook Logs -->
      <div class="sec-header" style="margin-top:var(--space-xl)">
        <div><h2 class="sec-title"><span class="material-icons-round" style="color:#f59e0b">webhook</span> Webhook Activity</h2>
        <p class="sec-desc">Recent webhook events and callback logs</p></div>
        <button class="btn btn-sm btn-outline" (click)="loadWebhookLogs()" [disabled]="loadingWebhooks()" id="btn-refresh-webhooks">
          <span class="material-icons-round" style="font-size:16px" [class.spinning]="loadingWebhooks()">refresh</span>
          Refresh
        </button>
      </div>
      <div class="glass-card log-table">
        <table>
          <thead><tr><th>Time</th><th>Provider</th><th>Event</th><th>Status</th><th>Response</th></tr></thead>
          <tbody>
            @for (log of webhookLogs(); track log.id) {
              <tr>
                <td class="mono">{{ log.time | date:'short' }}</td>
                <td><span class="ch-badge" [class]="'ch-' + log.provider.toLowerCase()">{{ log.provider }}</span></td>
                <td>{{ log.event }}</td>
                <td>
                  <span class="badge" [ngClass]="log.statusCode < 400 ? 'badge-success' : 'badge-danger'">{{ log.statusCode }}</span>
                  @if (log.processed) {
                    <span class="badge badge-processed" style="margin-left:4px">✓</span>
                  }
                </td>
                <td class="mono truncate">{{ log.response || '—' }}</td>
              </tr>
            }
            @if (webhookLogs().length === 0 && !loadingWebhooks()) {
              <tr><td colspan="5" style="text-align:center;color:var(--color-text-muted);padding:var(--space-xl)">
                <span class="material-icons-round" style="font-size:36px;opacity:0.3;display:block;margin-bottom:8px">webhook</span>
                No webhook events recorded yet
              </td></tr>
            }
          </tbody>
        </table>
      </div>

      <!-- API Keys -->
      <div class="sec-header" style="margin-top:var(--space-xl)">
        <div><h2 class="sec-title"><span class="material-icons-round" style="color:#8b5cf6">vpn_key</span> API Keys</h2>
        <p class="sec-desc">Service API keys for programmatic access</p></div>
        <button class="btn btn-primary-sm" (click)="showGenerateForm.set(true)" id="btn-generate-key">
          <span class="material-icons-round" style="font-size:16px">add</span>
          Generate Key
        </button>
      </div>

      <!-- Generate Key Form -->
      @if (showGenerateForm()) {
        <div class="glass-card key-form" style="margin-bottom:var(--space-lg)">
          <h3 style="font-size:0.875rem;margin:0 0 var(--space-sm)">Generate New API Key</h3>
          <div style="display:flex;gap:var(--space-sm);align-items:center">
            <input type="text" [(ngModel)]="newKeyName" placeholder="Key name (e.g. Mobile App)" class="form-input" id="input-key-name" />
            <button class="btn btn-primary-sm" (click)="generateKey()" [disabled]="!newKeyName || generatingKey()" id="btn-submit-key">
              @if (generatingKey()) {
                <span class="material-icons-round spinning" style="font-size:16px">refresh</span>
              } @else {
                <span class="material-icons-round" style="font-size:16px">vpn_key</span>
              }
              Generate
            </button>
            <button class="btn btn-outline" (click)="showGenerateForm.set(false)">Cancel</button>
          </div>
          @if (newlyGeneratedKey()) {
            <div class="new-key-result">
              <span class="material-icons-round" style="font-size:16px;color:#f59e0b">warning</span>
              <strong>Copy this key now — it won't be shown again in full:</strong>
              <div class="key-value" style="margin-top:8px">
                <code>{{ newlyGeneratedKey() }}</code>
                <button class="icon-btn" (click)="copyKey(newlyGeneratedKey()!)" title="Copy">
                  <span class="material-icons-round" style="font-size:16px">content_copy</span>
                </button>
              </div>
            </div>
          }
        </div>
      }

      <div class="keys-grid">
        @for (key of apiKeys(); track key.name) {
          <div class="key-card glass-card">
            <div class="key-header">
              <div style="display:flex;align-items:center;gap:var(--space-sm)">
                <span class="material-icons-round" style="font-size:18px;color:#8b5cf6">key</span>
                <strong>{{ key.name }}</strong>
              </div>
              <button class="icon-btn revoke-btn" (click)="revokeKey(key)" title="Revoke key">
                <span class="material-icons-round" style="font-size:16px">delete_outline</span>
              </button>
            </div>
            <div class="key-value"><code>{{ key.masked }}</code>
              <button class="icon-btn" (click)="copyKey(key.key)" title="Copy"><span class="material-icons-round" style="font-size:16px">content_copy</span></button>
            </div>
            <span class="key-meta">Created {{ key.created | date:'mediumDate' }}</span>
          </div>
        }
        @if (apiKeys().length === 0 && !showGenerateForm()) {
          <div class="glass-card" style="padding:var(--space-lg);text-align:center;color:var(--color-text-muted)">
            <span class="material-icons-round" style="font-size:36px;opacity:0.3;display:block;margin-bottom:8px">vpn_key</span>
            No API keys — click "Generate Key" to create one
          </div>
        }
      </div>
    </div>
  `,
  styleUrl: './platform-integrations.component.scss',
})
export class PlatformIntegrationsComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  providers = signal<IntegrationProvider[]>([]);
  webhookLogs = signal<WebhookLog[]>([]);
  apiKeys = signal<{ name: string; key: string; masked: string; created: string; slug?: string }[]>([]);

  loading = signal(false);
  loadingWebhooks = signal(false);
  togglingSlug = signal<string | null>(null);
  showGenerateForm = signal(false);
  generatingKey = signal(false);
  newKeyName = '';
  newlyGeneratedKey = signal<string | null>(null);

  ngOnInit() {
    this.refreshAll();
  }

  refreshAll() {
    this.loadIntegrations();
    this.loadWebhookLogs();
    this.loadApiKeys();
  }

  loadIntegrations() {
    this.loading.set(true);
    this.api.getIntegrations().subscribe({
      next: r => {
        const data = r.data || [];
        this.providers.set(data.map((p: any) => ({
          name: p.name,
          slug: p.slug,
          icon: p.icon || 'integration_instructions',
          status: p.status as 'active' | 'inactive' | 'unconfigured',
          type: p.type,
          description: p.description,
          enabled: p.enabled,
          configured: p.configured,
          primary: p.primary,
        })));
        this.loading.set(false);
      },
      error: () => {
        // Fallback to static data if API fails (backward compat)
        this.providers.set([
          { name: 'M-Pesa (JamboPay)', slug: 'jambopay', icon: 'account_balance', status: 'active', type: 'Payment', description: 'STK push, C2B collections, and B2C payouts', enabled: true, configured: true, primary: true },
          { name: 'SMS Gateway', slug: 'sms', icon: 'sms', status: 'active', type: 'Messaging', description: 'Optimize SMS provider for OTP and notifications', enabled: true, configured: true, primary: true },
          { name: 'USSD Gateway', slug: 'africastalking', icon: 'dialpad', status: 'active', type: 'Messaging', description: "Africa's Talking USSD + SMS fallback", enabled: true, configured: true, primary: false },
          { name: 'WhatsApp Cloud API', slug: 'whatsapp', icon: 'chat', status: 'inactive', type: 'Messaging', description: 'Meta WhatsApp Business messaging', enabled: false, configured: false, primary: false },
          { name: 'IPRS (KYC)', slug: 'iprs', icon: 'verified_user', status: 'active', type: 'Identity', description: 'National ID verification via IPRS API', enabled: true, configured: true, primary: true },
          { name: 'PerPay Payroll', slug: 'perpay', icon: 'payments', status: 'active', type: 'Payroll', description: 'External payroll submission and reconciliation', enabled: true, configured: true, primary: true },
          { name: 'Email (SMTP)', slug: 'email', icon: 'email', status: 'active', type: 'Messaging', description: 'Gmail SMTP for transactional emails', enabled: true, configured: true, primary: true },
          { name: 'MinIO Storage', slug: 'minio', icon: 'cloud_upload', status: 'active', type: 'Storage', description: 'Object storage for documents and files', enabled: true, configured: true, primary: true },
        ]);
        this.loading.set(false);
      },
    });
  }

  loadWebhookLogs() {
    this.loadingWebhooks.set(true);
    this.api.getWebhookLogs(50).subscribe({
      next: r => {
        this.webhookLogs.set(r.data || []);
        this.loadingWebhooks.set(false);
      },
      error: () => {
        this.webhookLogs.set([]);
        this.loadingWebhooks.set(false);
      },
    });
  }

  loadApiKeys() {
    this.api.getAPIKeys().subscribe({
      next: r => {
        this.apiKeys.set((r.data || []).map((k: any) => ({
          name: k.name,
          key: k.key,
          masked: k.masked,
          created: k.created,
          slug: k.name ? this.slugify(k.name) : undefined,
        })));
      },
      error: () => this.apiKeys.set([]),
    });
  }

  generateKey() {
    if (!this.newKeyName) return;
    this.generatingKey.set(true);
    this.newlyGeneratedKey.set(null);
    this.api.generateAPIKey(this.newKeyName).subscribe({
      next: r => {
        this.newlyGeneratedKey.set(r.data.key);
        this.toast.success(`API key "${this.newKeyName}" created`);
        this.newKeyName = '';
        this.generatingKey.set(false);
        this.loadApiKeys();
      },
      error: () => {
        this.toast.error('Failed to generate API key');
        this.generatingKey.set(false);
      },
    });
  }

  revokeKey(key: { name: string; slug?: string }) {
    const slug = key.slug || this.slugify(key.name);
    if (!confirm(`Revoke API key "${key.name}"? This action cannot be undone.`)) return;
    this.api.revokeAPIKey(slug).subscribe({
      next: () => {
        this.toast.success(`API key "${key.name}" revoked`);
        this.loadApiKeys();
      },
      error: () => this.toast.error('Failed to revoke API key'),
    });
  }

  private slugify(name: string): string {
    return name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
  }

  toggleProvider(provider: IntegrationProvider, event: Event) {
    const checkbox = event.target as HTMLInputElement;
    const newEnabled = checkbox.checked;

    this.togglingSlug.set(provider.slug);
    this.api.toggleIntegration(provider.slug, newEnabled).subscribe({
      next: () => {
        // Update local state
        const updated = this.providers().map(p =>
          p.slug === provider.slug
            ? { ...p, enabled: newEnabled, status: (newEnabled ? 'active' : 'inactive') as 'active' | 'inactive' | 'unconfigured' }
            : p
        );
        this.providers.set(updated);
        this.toast.success(`${provider.name} ${newEnabled ? 'enabled' : 'disabled'}`);
        this.togglingSlug.set(null);
      },
      error: () => {
        // Revert checkbox
        checkbox.checked = !newEnabled;
        this.toast.error(`Failed to toggle ${provider.name}`);
        this.togglingSlug.set(null);
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
