import { Component } from '@angular/core';
import { PlatformPlaceholderComponent } from '../shared/platform-placeholder.component';

@Component({
  selector: 'app-platform-support',
  standalone: true,
  imports: [PlatformPlaceholderComponent],
  template: `
    <app-platform-placeholder
      title="Support Center"
      subtitle="Help organizations and users resolve issues"
      icon="support_agent"
      description="A centralized support dashboard for handling user issues, stuck wallets, and operational escalations."
      [features]="[
        'User lookup & quick resolution tools',
        'Stuck wallet & transaction recovery',
        'Failed payroll reprocessing',
        'Verification code resending',
        'Activity timeline per user'
      ]"
    />
  `
})
export class PlatformSupportComponent {}
