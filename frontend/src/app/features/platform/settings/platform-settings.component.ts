import { Component } from '@angular/core';
import { PlatformPlaceholderComponent } from '../shared/platform-placeholder.component';

@Component({
  selector: 'app-platform-settings',
  standalone: true,
  imports: [PlatformPlaceholderComponent],
  template: `
    <app-platform-placeholder
      title="System Settings"
      subtitle="Global platform configuration, feature flags, and defaults"
      icon="settings"
      description="Manage statutory rates, feature toggles, default tenant configurations, and system-wide announcements."
      [features]="[
        'Statutory rates management (NSSF, SHA, Housing)',
        'Global feature flags & toggles',
        'Default tenant configuration templates',
        'System-wide announcements',
        'Maintenance mode controls'
      ]"
    />
  `
})
export class PlatformSettingsComponent {}
