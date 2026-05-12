import { Component } from '@angular/core';
import { PlatformPlaceholderComponent } from '../shared/platform-placeholder.component';

@Component({
  selector: 'app-platform-team',
  standalone: true,
  imports: [PlatformPlaceholderComponent],
  template: `
    <app-platform-placeholder
      title="Platform Team"
      subtitle="Manage platform staff — support, finance, auditors"
      icon="group_work"
      description="Invite and manage platform team members with specialized roles for support, finance, and audit functions."
      [features]="[
        'Platform staff roster & roles',
        'Invite new team members',
        'Role-based access assignment',
        'Activity log per team member',
        'On-call schedule management'
      ]"
    />
  `
})
export class PlatformTeamComponent {}
