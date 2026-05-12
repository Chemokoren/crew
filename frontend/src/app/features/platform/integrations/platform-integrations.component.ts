import { Component } from '@angular/core';
import { PlatformPlaceholderComponent } from '../shared/platform-placeholder.component';

@Component({
  selector: 'app-platform-integrations',
  standalone: true,
  imports: [PlatformPlaceholderComponent],
  template: `
    <app-platform-placeholder
      title="Integrations"
      subtitle="API keys, USSD gateway, payment providers, and webhooks"
      icon="extension"
      description="Monitor and manage all external integrations including USSD, M-Pesa, bank APIs, and webhook endpoints."
      [features]="[
        'USSD gateway management',
        'Payment provider status (M-Pesa, bank APIs)',
        'Webhook monitor & logs',
        'API key management',
        'Service health dashboard'
      ]"
    />
  `
})
export class PlatformIntegrationsComponent {}
