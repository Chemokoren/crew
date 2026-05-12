import { Component } from '@angular/core';
import { PlatformPlaceholderComponent } from '../shared/platform-placeholder.component';

@Component({
  selector: 'app-platform-roles',
  standalone: true,
  imports: [PlatformPlaceholderComponent],
  template: `
    <app-platform-placeholder
      title="Roles & Permissions"
      subtitle="Default role configurations and permission matrix"
      icon="admin_panel_settings"
      description="Configure system-wide role permissions and default access levels for new organizations."
      [features]="[
        'Visual permission matrix grid',
        'Default configurations for new orgs',
        'Role templates per industry',
        'Custom role creation',
        'Bulk permission updates'
      ]"
    />
  `
})
export class PlatformRolesComponent {}
