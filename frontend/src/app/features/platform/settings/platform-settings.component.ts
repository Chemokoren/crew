import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { StatutoryRate, SystemSetting, SystemAnnouncement } from '../../../core/models';

type SettingsTab = 'rates' | 'flags' | 'defaults' | 'announcements' | 'maintenance';

interface FeatureFlag {
  key: string;
  label: string;
  description: string;
  category: string;
  enabled: boolean;
}

@Component({
  selector: 'app-platform-settings',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './platform-settings.component.html',
  styleUrl: './platform-settings.component.scss',
})
export class PlatformSettingsComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  activeTab = signal<SettingsTab>('rates');
  loading = signal(true);

  // Statutory Rates
  rates = signal<StatutoryRate[]>([]);
  rateModalOpen = signal(false);
  editingRate = signal<Partial<StatutoryRate>>({});
  rateSaving = signal(false);

  // Feature Flags
  featureFlags = signal<FeatureFlag[]>([]);
  flagsSaving = signal(false);
  flagsDirty = signal(false);

  // System Settings (defaults)
  defaultSettings = signal<SystemSetting[]>([]);
  defaultsSaving = signal(false);

  // Default tenant config form
  defaultKycRequired = signal(false);
  defaultKycMode = signal('UPLOAD');
  defaultStatutory = signal(false);
  defaultTopupMode = signal('HYBRID');

  // Announcements
  announcements = signal<SystemAnnouncement[]>([]);
  announcementModalOpen = signal(false);
  editingAnnouncement = signal<Partial<SystemAnnouncement>>({});
  announcementSaving = signal(false);

  // Maintenance
  maintenanceActive = signal(false);
  maintenanceMessage = signal('System is undergoing scheduled maintenance. Please try again later.');
  maintenanceStart = signal('');
  maintenanceEnd = signal('');
  maintenanceSaving = signal(false);

  tabs: { id: SettingsTab; label: string; icon: string }[] = [
    { id: 'rates', label: 'Statutory Rates', icon: 'account_balance' },
    { id: 'flags', label: 'Feature Flags', icon: 'toggle_on' },
    { id: 'defaults', label: 'Tenant Defaults', icon: 'tune' },
    { id: 'announcements', label: 'Announcements', icon: 'campaign' },
    { id: 'maintenance', label: 'Maintenance', icon: 'engineering' },
  ];

  readonly defaultFlags: Omit<FeatureFlag, 'enabled'>[] = [
    { key: 'feature.mobile_money_enabled', label: 'Mobile Money Top-ups', description: 'Enable M-Pesa, Airtel, T-Kash wallet top-ups', category: 'Payments' },
    { key: 'feature.bank_topup_enabled', label: 'Bank Transfer Top-ups', description: 'Enable bank transfer wallet top-ups', category: 'Payments' },
    { key: 'feature.card_topup_enabled', label: 'Card Payments', description: 'Enable Visa/Mastercard wallet top-ups', category: 'Payments' },
    { key: 'feature.loans_enabled', label: 'Loan Module', description: 'Enable loan applications and management', category: 'Modules' },
    { key: 'feature.insurance_enabled', label: 'Insurance Module', description: 'Enable insurance policy management', category: 'Modules' },
    { key: 'feature.payroll_enabled', label: 'Payroll Module', description: 'Enable payroll processing and schedules', category: 'Modules' },
    { key: 'feature.credit_scoring_enabled', label: 'Credit Scoring', description: 'Enable credit score computation engine', category: 'Modules' },
    { key: 'feature.ussd_enabled', label: 'USSD Gateway', description: 'Enable USSD session handling for feature phones', category: 'Channels' },
    { key: 'feature.kyc_required_global', label: 'Global KYC Enforcement', description: 'Require KYC verification for all tenants', category: 'Compliance' },
    { key: 'feature.whatsapp_enabled', label: 'WhatsApp Notifications', description: 'Enable WhatsApp message channel', category: 'Channels' },
  ];

  ngOnInit(): void {
    this.loadRates();
    this.loadFlags();
    this.loadDefaults();
    this.loadAnnouncements();
    this.loadMaintenance();
  }

  switchTab(tab: SettingsTab) {
    this.activeTab.set(tab);
  }

  // ── Statutory Rates ──
  loadRates() {
    this.loading.set(true);
    this.api.getStatutoryRates().subscribe({
      next: r => { this.rates.set(r.data || []); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  openAddRate() {
    this.editingRate.set({ name: '', rate: 0, rate_type: 'PERCENTAGE', effective_from: new Date().toISOString().split('T')[0], is_active: true });
    this.rateModalOpen.set(true);
  }

  openEditRate(rate: StatutoryRate) {
    this.editingRate.set({ ...rate, effective_from: rate.effective_from?.split('T')[0] });
    this.rateModalOpen.set(true);
  }

  closeRateModal() { this.rateModalOpen.set(false); }

  saveRate() {
    const rate = this.editingRate();
    this.rateSaving.set(true);
    const obs = rate.id
      ? this.api.updateStatutoryRate(rate.id, rate)
      : this.api.createStatutoryRate(rate);
    obs.subscribe({
      next: () => { this.toast.success('Rate saved'); this.closeRateModal(); this.loadRates(); this.rateSaving.set(false); },
      error: () => { this.toast.error('Failed to save rate'); this.rateSaving.set(false); },
    });
  }

  toggleRateActive(rate: StatutoryRate) {
    this.api.updateStatutoryRate(rate.id, { ...rate, is_active: !rate.is_active }).subscribe({
      next: () => { this.toast.success(`Rate ${rate.is_active ? 'deactivated' : 'activated'}`); this.loadRates(); },
      error: () => this.toast.error('Failed to update rate'),
    });
  }

  // ── Feature Flags ──
  loadFlags() {
    this.api.getSystemSettings('feature.').subscribe({
      next: r => {
        const saved = r.data || [];
        const flags = this.defaultFlags.map(f => {
          const existing = saved.find(s => s.key === f.key);
          return { ...f, enabled: existing ? existing.value === 'true' : true };
        });
        this.featureFlags.set(flags);
        this.flagsDirty.set(false);
      },
      error: () => {
        this.featureFlags.set(this.defaultFlags.map(f => ({ ...f, enabled: true })));
      },
    });
  }

  toggleFlag(flag: FeatureFlag) {
    const flags = this.featureFlags();
    const idx = flags.findIndex(f => f.key === flag.key);
    if (idx >= 0) {
      flags[idx] = { ...flags[idx], enabled: !flags[idx].enabled };
      this.featureFlags.set([...flags]);
      this.flagsDirty.set(true);
    }
  }

  saveFlags() {
    this.flagsSaving.set(true);
    const settings = this.featureFlags().map(f => ({
      key: f.key, value: String(f.enabled), value_type: 'bool' as const,
      category: 'feature', label: f.label,
    }));
    this.api.bulkUpsertSystemSettings(settings).subscribe({
      next: () => { this.toast.success('Feature flags saved'); this.flagsSaving.set(false); this.flagsDirty.set(false); },
      error: () => { this.toast.error('Failed to save flags'); this.flagsSaving.set(false); },
    });
  }

  revertFlags() { this.loadFlags(); }

  getFlagCategories(): string[] {
    return [...new Set(this.featureFlags().map(f => f.category))];
  }

  getFlagsByCategory(cat: string): FeatureFlag[] {
    return this.featureFlags().filter(f => f.category === cat);
  }

  // ── Default Tenant Config ──
  loadDefaults() {
    this.api.getSystemSettings('defaults.').subscribe({
      next: r => {
        const s = r.data || [];
        this.defaultKycRequired.set(s.find(x => x.key === 'defaults.kyc_required')?.value === 'true');
        this.defaultKycMode.set(s.find(x => x.key === 'defaults.kyc_mode')?.value || 'UPLOAD');
        this.defaultStatutory.set(s.find(x => x.key === 'defaults.statutory_deductions')?.value === 'true');
        this.defaultTopupMode.set(s.find(x => x.key === 'defaults.topup_verification_mode')?.value || 'HYBRID');
      },
    });
  }

  saveDefaults() {
    this.defaultsSaving.set(true);
    const settings = [
      { key: 'defaults.kyc_required', value: String(this.defaultKycRequired()), value_type: 'bool' as const, category: 'defaults', label: 'KYC Required' },
      { key: 'defaults.kyc_mode', value: this.defaultKycMode(), value_type: 'string' as const, category: 'defaults', label: 'KYC Verification Mode' },
      { key: 'defaults.statutory_deductions', value: String(this.defaultStatutory()), value_type: 'bool' as const, category: 'defaults', label: 'Statutory Deductions' },
      { key: 'defaults.topup_verification_mode', value: this.defaultTopupMode(), value_type: 'string' as const, category: 'defaults', label: 'Top-up Verification Mode' },
    ];
    this.api.bulkUpsertSystemSettings(settings).subscribe({
      next: () => { this.toast.success('Defaults saved'); this.defaultsSaving.set(false); },
      error: () => { this.toast.error('Failed to save defaults'); this.defaultsSaving.set(false); },
    });
  }

  // ── Announcements ──
  loadAnnouncements() {
    this.api.getAnnouncements().subscribe({
      next: r => this.announcements.set(r.data || []),
    });
  }

  openAddAnnouncement() {
    this.editingAnnouncement.set({ title: '', body: '', severity: 'INFO', is_active: true });
    this.announcementModalOpen.set(true);
  }

  openEditAnnouncement(a: SystemAnnouncement) {
    this.editingAnnouncement.set({
      ...a,
      start_at: a.start_at ? new Date(a.start_at).toISOString().slice(0, 16) : undefined,
      end_at: a.end_at ? new Date(a.end_at).toISOString().slice(0, 16) : undefined,
    });
    this.announcementModalOpen.set(true);
  }

  closeAnnouncementModal() { this.announcementModalOpen.set(false); }

  saveAnnouncement() {
    const a = { ...this.editingAnnouncement() };
    // Convert datetime-local values to full RFC3339 timestamps for Go backend
    if (a.start_at && !a.start_at.includes('Z') && a.start_at.length <= 16) {
      a.start_at = new Date(a.start_at).toISOString();
    }
    if (a.end_at && !a.end_at.includes('Z') && a.end_at.length <= 16) {
      a.end_at = new Date(a.end_at).toISOString();
    }
    this.announcementSaving.set(true);
    const obs = a.id
      ? this.api.updateAnnouncement(a.id, a)
      : this.api.createAnnouncement(a);
    obs.subscribe({
      next: () => { this.toast.success('Announcement saved'); this.closeAnnouncementModal(); this.loadAnnouncements(); this.announcementSaving.set(false); },
      error: () => { this.toast.error('Failed to save'); this.announcementSaving.set(false); },
    });
  }

  deleteAnnouncement(a: SystemAnnouncement) {
    if (!confirm('Delete this announcement?')) return;
    this.api.deleteAnnouncement(a.id).subscribe({
      next: () => { this.toast.success('Deleted'); this.loadAnnouncements(); },
      error: () => this.toast.error('Failed to delete'),
    });
  }

  toggleAnnouncementActive(a: SystemAnnouncement) {
    this.api.updateAnnouncement(a.id, { ...a, is_active: !a.is_active }).subscribe({
      next: () => { this.toast.success(`Announcement ${a.is_active ? 'disabled' : 'enabled'}`); this.loadAnnouncements(); },
      error: () => this.toast.error('Update failed'),
    });
  }

  severityColor(s: string): string {
    switch (s) { case 'CRITICAL': return '#ef4444'; case 'WARNING': return '#f59e0b'; default: return '#6366f1'; }
  }

  severityBg(s: string): string {
    switch (s) { case 'CRITICAL': return 'rgba(239,68,68,0.12)'; case 'WARNING': return 'rgba(245,158,11,0.12)'; default: return 'rgba(99,102,241,0.12)'; }
  }

  // ── Maintenance Mode ──
  loadMaintenance() {
    this.api.getSystemSettings('maintenance.').subscribe({
      next: r => {
        const s = r.data || [];
        this.maintenanceActive.set(s.find(x => x.key === 'maintenance.active')?.value === 'true');
        this.maintenanceMessage.set(s.find(x => x.key === 'maintenance.message')?.value || this.maintenanceMessage());
        this.maintenanceStart.set(s.find(x => x.key === 'maintenance.start')?.value || '');
        this.maintenanceEnd.set(s.find(x => x.key === 'maintenance.end')?.value || '');
      },
    });
  }

  saveMaintenance() {
    this.maintenanceSaving.set(true);
    const settings = [
      { key: 'maintenance.active', value: String(this.maintenanceActive()), value_type: 'bool' as const, category: 'maintenance', label: 'Maintenance Active' },
      { key: 'maintenance.message', value: this.maintenanceMessage(), value_type: 'string' as const, category: 'maintenance', label: 'Maintenance Message' },
      { key: 'maintenance.start', value: this.maintenanceStart(), value_type: 'string' as const, category: 'maintenance', label: 'Start Time' },
      { key: 'maintenance.end', value: this.maintenanceEnd(), value_type: 'string' as const, category: 'maintenance', label: 'End Time' },
    ];
    this.api.bulkUpsertSystemSettings(settings).subscribe({
      next: () => { this.toast.success('Maintenance settings saved'); this.maintenanceSaving.set(false); },
      error: () => { this.toast.error('Failed to save'); this.maintenanceSaving.set(false); },
    });
  }

  toggleMaintenance() {
    if (!this.maintenanceActive() && !confirm('Enable maintenance mode? Users will see a maintenance banner.')) return;
    this.maintenanceActive.set(!this.maintenanceActive());
    this.saveMaintenance();
  }
}
