import { Component } from '@angular/core';
import { PlatformPlaceholderComponent } from '../shared/platform-placeholder.component';

@Component({
  selector: 'app-platform-reports',
  standalone: true,
  imports: [PlatformPlaceholderComponent],
  template: `
    <app-platform-placeholder
      title="Reports & Analytics"
      subtitle="Platform-wide business intelligence and growth metrics"
      icon="insights"
      description="Track user growth, financial metrics, org performance, and generate automated reports."
      [features]="[
        'User growth & retention analytics',
        'Financial metrics (GMV, float utilization)',
        'Organization performance ranking',
        'Scheduled weekly/monthly reports',
        'Custom report builder & export'
      ]"
    />
  `
})
export class PlatformReportsComponent {}
