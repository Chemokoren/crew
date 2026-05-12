import { Component } from '@angular/core';
import { PlatformPlaceholderComponent } from '../shared/platform-placeholder.component';

@Component({
  selector: 'app-platform-documents',
  standalone: true,
  imports: [PlatformPlaceholderComponent],
  template: `
    <app-platform-placeholder
      title="Documents"
      subtitle="Cross-organization document oversight and verification"
      icon="folder"
      description="Review and manage documents across all organizations — KYC submissions, registrations, and compliance files."
      [features]="[
        'Cross-org document search',
        'KYC document verification queue',
        'Document type analytics',
        'Bulk approval workflows',
        'Storage usage monitoring'
      ]"
    />
  `
})
export class PlatformDocumentsComponent {}
