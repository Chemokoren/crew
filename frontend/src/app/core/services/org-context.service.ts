import { Injectable, signal, computed, inject } from '@angular/core';
import { ApiService } from './api.service';
import { AuthService } from './auth.service';
import { IndustryType, IndustryTemplate } from '../models';
import { getIndustryTemplate, INDUSTRY_ICONS } from '../config/industry-templates';

/**
 * Reactive service that tracks the authenticated user's organization
 * industry type and exposes it for sidebar/menu adaptation.
 *
 * Loaded once on login (or app init) and refreshed on industry change.
 */
@Injectable({ providedIn: 'root' })
export class OrgContextService {
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);

  /** Current org industry type — defaults to GENERAL until loaded. */
  readonly industryType = signal<IndustryType>('GENERAL');

  /** Whether the org context has been loaded. */
  readonly loaded = signal(false);

  /** Full industry template for the current org — reactively derived. */
  readonly template = computed<IndustryTemplate>(() => getIndustryTemplate(this.industryType()));

  /** Singular worker label (e.g. "Worker", "Staff", "Crew Member"). */
  readonly workerLabel = computed(() => this.template().ui_labels['worker'] || 'Worker');

  /** Plural workers label (e.g. "Workers", "Staff", "Crew Members"). */
  readonly workersLabel = computed(() => {
    const singular = this.workerLabel();
    // Simple pluralisation — handles most industry labels
    if (singular.endsWith('ff') || singular.endsWith('f')) return singular;
    if (singular.endsWith('s')) return singular;
    return singular + 's';
  });

  /** Material icon for the current industry. */
  readonly industryIcon = computed(() => INDUSTRY_ICONS[this.industryType()] || 'business');

  /**
   * Loads the organization for the current user.
   * Call once after login / app bootstrap.
   */
  load(): void {
    const user = this.auth.currentUser();
    if (!user?.organization_id) {
      // System admin without org — show all menu items
      this.industryType.set('GENERAL');
      this.loaded.set(true);
      return;
    }

    this.api.getOrganization(user.organization_id).subscribe({
      next: r => {
        this.industryType.set(r.data?.industry_type || 'GENERAL');
        this.loaded.set(true);
      },
      error: () => {
        this.industryType.set('GENERAL');
        this.loaded.set(true);
      },
    });
  }

  /**
   * Update after industry change (called from tenant settings).
   */
  setIndustry(industry: IndustryType): void {
    this.industryType.set(industry);
  }

  /**
   * Get a UI label for the current industry.
   */
  label(key: string): string {
    const tmpl = getIndustryTemplate(this.industryType());
    return tmpl.ui_labels[key] || key;
  }

  /**
   * Returns true if a sidebar feature is relevant to the current industry.
   */
  isFeatureVisible(feature: string): boolean {
    const industry = this.industryType();
    const role = this.auth.userRole();

    // System admins always see everything
    if (role === 'SYSTEM_ADMIN') return true;

    switch (feature) {
      case 'vehicles':
        // Only transport and logistics use vehicles
        return ['TRANSPORT', 'LOGISTICS'].includes(industry);
      case 'routes':
        // Only transport uses routes
        return industry === 'TRANSPORT';
      case 'work-sites':
        // Non-transport industries use work sites
        return !['TRANSPORT'].includes(industry) || industry === 'GENERAL';
      case 'facilitators':
        // Transport and logistics have facilitators (booking agents, dispatchers)
        return ['TRANSPORT', 'LOGISTICS', 'GENERAL', 'CUSTOM'].includes(industry);
      default:
        return true;
    }
  }
}
