import { Component, inject, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { ServiceCodeRoute, ShortcodeRequest, ABTest, IndustryType, MNOProvider } from '../../../core/models';

type UssdSubTab = 'routes' | 'mno' | 'ab';

// Seed data for demo — will be replaced by API calls when backend is ready
const SEED_ROUTES: ServiceCodeRoute[] = [
  { id: '1', service_code: '*384*123#', industry_type: 'TRANSPORT', is_active: true, roles: ['DRIVER','CONDUCTOR','RIDER','BOOKING_AGENT','DISPATCHER'], created_at: '2026-01-15T00:00:00Z', updated_at: '2026-05-01T00:00:00Z' },
  { id: '2', service_code: '*384*200#', industry_type: 'CONSTRUCTION', is_active: true, roles: ['MASON','CARPENTER','PLUMBER','ELECTRICIAN','GENERAL_LABORER'], created_at: '2026-02-01T00:00:00Z', updated_at: '2026-05-01T00:00:00Z' },
  { id: '3', service_code: '*384*300#', industry_type: 'HEALTH', is_active: true, roles: ['CHV','CHP','NURSE'], created_at: '2026-03-01T00:00:00Z', updated_at: '2026-05-01T00:00:00Z' },
  { id: '4', service_code: '*384*400#', industry_type: 'LOGISTICS', is_active: true, roles: ['DELIVERY_RIDER','DRIVER','LOADER','DISPATCHER'], created_at: '2026-03-15T00:00:00Z', updated_at: '2026-05-01T00:00:00Z' },
  { id: '5', service_code: '*384*500#', industry_type: 'AGRICULTURE', is_active: true, roles: ['PICKER','FIELD_WORKER','SORTER'], created_at: '2026-04-01T00:00:00Z', updated_at: '2026-05-01T00:00:00Z' },
  { id: '6', service_code: '*384*600#', industry_type: 'HOSPITALITY', is_active: true, roles: ['WAITER','COOK','HOUSEKEEPER'], created_at: '2026-04-15T00:00:00Z', updated_at: '2026-05-01T00:00:00Z' },
];

const SEED_MNO: ShortcodeRequest[] = [
  { id: '1', service_code: '*384*123#', mno: 'SAFARICOM', status: 'ACTIVE', submitted_at: '2026-01-10T00:00:00Z', provisioned_at: '2026-01-14T00:00:00Z' },
  { id: '2', service_code: '*384*123#', mno: 'AIRTEL', status: 'ACTIVE', submitted_at: '2026-01-10T00:00:00Z', provisioned_at: '2026-01-16T00:00:00Z' },
  { id: '3', service_code: '*384*200#', mno: 'SAFARICOM', status: 'PROVISIONED', submitted_at: '2026-02-01T00:00:00Z', provisioned_at: '2026-02-05T00:00:00Z' },
  { id: '4', service_code: '*384*300#', mno: 'SAFARICOM', status: 'PENDING', submitted_at: '2026-05-05T00:00:00Z' },
  { id: '5', service_code: '*384*400#', mno: 'TELKOM', status: 'REJECTED', submitted_at: '2026-04-20T00:00:00Z', rejected_reason: 'Duplicate shortcode range' },
];

const SEED_AB: ABTest[] = [
  { id: '1', name: 'Transport role ordering', service_code: '*384*123#', variant_a_label: 'Default order', variant_b_label: 'Rider-first', variant_a_roles: ['DRIVER','CONDUCTOR','RIDER'], variant_b_roles: ['RIDER','DRIVER','CONDUCTOR'], traffic_split_pct: 50, status: 'RUNNING', impressions_a: 1247, impressions_b: 1253, conversions_a: 834, conversions_b: 921, started_at: '2026-05-01T00:00:00Z', created_at: '2026-04-28T00:00:00Z' },
  { id: '2', name: 'Health simplified menu', service_code: '*384*300#', variant_a_label: 'Full roles', variant_b_label: 'CHV only', variant_a_roles: ['CHV','CHP','NURSE'], variant_b_roles: ['CHV'], traffic_split_pct: 30, status: 'DRAFT', impressions_a: 0, impressions_b: 0, conversions_a: 0, conversions_b: 0, created_at: '2026-05-04T00:00:00Z' },
];

@Component({
  selector: 'app-ussd-management', standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './ussd-management.component.html',
  styleUrl: './ussd-management.component.css',
})
export class UssdManagementComponent {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  subTab = signal<UssdSubTab>('routes');
  cacheRefreshing = signal(false);
  cacheRefreshResult = signal<'success' | 'error' | null>(null);

  // Service code routes
  routes = signal<ServiceCodeRoute[]>(SEED_ROUTES);
  showRouteModal = signal(false);
  editingRoute = signal(false);
  savingRoute = signal(false);
  routeForm: Partial<ServiceCodeRoute> & { rolesText: string } = { service_code: '', industry_type: 'TRANSPORT', is_active: true, rolesText: '', organization_id: '' };

  // MNO provisioning
  mnoRequests = signal<ShortcodeRequest[]>(SEED_MNO);
  showMnoModal = signal(false);
  savingMno = signal(false);
  mnoForm = { service_code: '', mno: 'SAFARICOM' as MNOProvider, callback_url: '' };

  // A/B testing
  abTests = signal<ABTest[]>(SEED_AB);
  showAbModal = signal(false);
  savingAb = signal(false);
  abForm = { name: '', service_code: '', variant_a_label: 'Control', variant_b_label: 'Variant B', variant_a_roles: '', variant_b_roles: '', traffic_split_pct: 50 };

  readonly industries: IndustryType[] = ['TRANSPORT', 'CONSTRUCTION', 'HEALTH', 'LOGISTICS', 'AGRICULTURE', 'HOSPITALITY'];
  readonly subTabs: { key: UssdSubTab; label: string; icon: string }[] = [
    { key: 'routes', label: 'Service Codes', icon: 'route' },
    { key: 'mno', label: 'MNO Provisioning', icon: 'cell_tower' },
    { key: 'ab', label: 'A/B Tests', icon: 'science' },
  ];

  // --- Cache ---
  refreshCache() {
    this.cacheRefreshing.set(true);
    this.cacheRefreshResult.set(null);
    this.api.refreshUSSDRoleCache().subscribe({
      next: () => { this.cacheRefreshing.set(false); this.cacheRefreshResult.set('success'); this.toast.success('Cache refresh triggered'); setTimeout(() => this.cacheRefreshResult.set(null), 8000); },
      error: () => { this.cacheRefreshing.set(false); this.cacheRefreshResult.set('error'); this.toast.error('Failed to refresh cache'); setTimeout(() => this.cacheRefreshResult.set(null), 8000); },
    });
  }

  // --- Routes ---
  openRouteModal(r?: ServiceCodeRoute) {
    if (r) {
      this.routeForm = { ...r, rolesText: r.roles.join(', ') };
      this.editingRoute.set(true);
    } else {
      this.routeForm = { service_code: '', industry_type: 'TRANSPORT', is_active: true, rolesText: '', organization_id: '' };
      this.editingRoute.set(false);
    }
    this.showRouteModal.set(true);
  }

  saveRoute() {
    this.savingRoute.set(true);
    const roles = this.routeForm.rolesText.split(',').map(s => s.trim().toUpperCase()).filter(Boolean);
    const route: ServiceCodeRoute = {
      id: this.routeForm.id || crypto.randomUUID(),
      service_code: this.routeForm.service_code || '',
      industry_type: (this.routeForm.industry_type || 'TRANSPORT') as IndustryType,
      organization_id: this.routeForm.organization_id || undefined,
      is_active: this.routeForm.is_active !== false,
      roles,
      created_at: this.routeForm.created_at || new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    // TODO: Replace with API call when backend is ready
    const current = this.routes();
    if (this.editingRoute()) {
      this.routes.set(current.map(r => r.id === route.id ? route : r));
    } else {
      this.routes.set([...current, route]);
    }
    this.toast.success(`Route ${this.editingRoute() ? 'updated' : 'created'}`);
    this.showRouteModal.set(false);
    this.savingRoute.set(false);
  }

  toggleRoute(r: ServiceCodeRoute) {
    this.routes.set(this.routes().map(x => x.id === r.id ? { ...x, is_active: !x.is_active, updated_at: new Date().toISOString() } : x));
    this.toast.success(`Route ${r.is_active ? 'disabled' : 'enabled'}`);
  }

  deleteRoute(r: ServiceCodeRoute) {
    if (!confirm(`Delete route ${r.service_code}?`)) return;
    this.routes.set(this.routes().filter(x => x.id !== r.id));
    this.toast.success('Route deleted');
  }

  // --- MNO ---
  openMnoModal() {
    this.mnoForm = { service_code: '', mno: 'SAFARICOM', callback_url: '' };
    this.showMnoModal.set(true);
  }

  submitMno() {
    this.savingMno.set(true);
    const req: ShortcodeRequest = {
      id: crypto.randomUUID(),
      service_code: this.mnoForm.service_code,
      mno: this.mnoForm.mno,
      status: 'PENDING',
      submitted_at: new Date().toISOString(),
      callback_url: this.mnoForm.callback_url || undefined,
    };
    this.mnoRequests.set([req, ...this.mnoRequests()]);
    this.toast.success(`Provisioning request submitted to ${this.mnoForm.mno}`);
    this.showMnoModal.set(false);
    this.savingMno.set(false);
  }

  getMnoStatusClass(s: string): string {
    return s === 'ACTIVE' ? 'badge-success' : s === 'PROVISIONED' ? 'badge-info' : s === 'PENDING' ? 'badge-warning' : s === 'REJECTED' ? 'badge-danger' : 'badge-neutral';
  }

  // --- A/B ---
  openAbModal() {
    this.abForm = { name: '', service_code: '', variant_a_label: 'Control', variant_b_label: 'Variant B', variant_a_roles: '', variant_b_roles: '', traffic_split_pct: 50 };
    this.showAbModal.set(true);
  }

  saveAbTest() {
    this.savingAb.set(true);
    const test: ABTest = {
      id: crypto.randomUUID(),
      name: this.abForm.name,
      service_code: this.abForm.service_code,
      variant_a_label: this.abForm.variant_a_label,
      variant_b_label: this.abForm.variant_b_label,
      variant_a_roles: this.abForm.variant_a_roles.split(',').map(s => s.trim()).filter(Boolean),
      variant_b_roles: this.abForm.variant_b_roles.split(',').map(s => s.trim()).filter(Boolean),
      traffic_split_pct: this.abForm.traffic_split_pct,
      status: 'DRAFT',
      impressions_a: 0, impressions_b: 0, conversions_a: 0, conversions_b: 0,
      created_at: new Date().toISOString(),
    };
    this.abTests.set([test, ...this.abTests()]);
    this.toast.success('A/B test created');
    this.showAbModal.set(false);
    this.savingAb.set(false);
  }

  toggleAbTest(t: ABTest) {
    const next = t.status === 'RUNNING' ? 'PAUSED' : 'RUNNING';
    this.abTests.set(this.abTests().map(x => x.id === t.id ? { ...x, status: next, started_at: next === 'RUNNING' ? (x.started_at || new Date().toISOString()) : x.started_at } : x));
    this.toast.success(`Test ${next === 'RUNNING' ? 'started' : 'paused'}`);
  }

  endAbTest(t: ABTest) {
    if (!confirm(`End A/B test "${t.name}"?`)) return;
    this.abTests.set(this.abTests().map(x => x.id === t.id ? { ...x, status: 'COMPLETED' as const, ended_at: new Date().toISOString() } : x));
    this.toast.success('Test completed');
  }

  conversionRate(impressions: number, conversions: number): string {
    return impressions > 0 ? (conversions / impressions * 100).toFixed(1) : '0.0';
  }

  getAbStatusClass(s: string): string {
    return s === 'RUNNING' ? 'badge-success' : s === 'PAUSED' ? 'badge-warning' : s === 'COMPLETED' ? 'badge-neutral' : 'badge-accent';
  }
}
