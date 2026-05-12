import { Component } from '@angular/core';
import { PlatformPlaceholderComponent } from '../shared/platform-placeholder.component';

@Component({
  selector: 'app-platform-notifications',
  standalone: true,
  imports: [PlatformPlaceholderComponent],
  template: `
    <app-platform-placeholder
      title="Notification Templates"
      subtitle="Manage system-wide notification channels and templates"
      icon="notifications"
      description="Create and manage SMS, Push, and In-App notification templates with variable preview and delivery tracking."
      [features]="[
        'Visual template editor with variable preview',
        'Multi-channel management (SMS, Push, In-App)',
        'Test mode — send test notifications',
        'Delivery reports & analytics',
        'Bulk notification broadcasting'
      ]"
    />
  `
})
export class PlatformNotificationsComponent {}
