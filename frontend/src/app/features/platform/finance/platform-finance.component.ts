import { Component } from '@angular/core';
import { PlatformPlaceholderComponent } from '../shared/platform-placeholder.component';

@Component({
  selector: 'app-platform-finance',
  standalone: true,
  imports: [PlatformPlaceholderComponent],
  template: `
    <app-platform-placeholder
      title="Float Oversight"
      subtitle="Cross-organization financial monitoring and approvals"
      icon="account_balance"
      description="Monitor float balances, approve pending top-ups, and track high-value transactions across all organizations."
      [features]="[
        'Platform-wide float dashboard',
        'Pending top-up approvals queue',
        'High-value transaction alerts',
        'Balance reconciliation tools',
        'Revenue & commission tracking'
      ]"
    />
  `
})
export class PlatformFinanceComponent {}
