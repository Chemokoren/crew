import { Directive, Input, TemplateRef, ViewContainerRef, inject, effect } from '@angular/core';
import { PermissionService } from '../../core/services/permission.service';

/**
 * Structural directive that conditionally renders elements based on user permissions.
 *
 * Usage:
 *   <button *hasPermission="'workers.create'">Add Worker</button>
 *   <div *hasPermission="'wallet.withdraw'; else noAccess">Wallet Actions</div>
 */
@Directive({
  selector: '[hasPermission]',
  standalone: true
})
export class HasPermissionDirective {
  private templateRef = inject(TemplateRef<unknown>);
  private viewContainer = inject(ViewContainerRef);
  private permissionService = inject(PermissionService);
  private hasView = false;

  @Input()
  set hasPermission(permKey: string | string[]) {
    const keys = Array.isArray(permKey) ? permKey : [permKey];

    effect(() => {
      // Re-evaluate when permissions signal changes
      const perms = this.permissionService.permissions();
      const allowed = keys.every(k => perms.has(k));

      if (allowed && !this.hasView) {
        this.viewContainer.createEmbeddedView(this.templateRef);
        this.hasView = true;
      } else if (!allowed && this.hasView) {
        this.viewContainer.clear();
        this.hasView = false;
      }
    });
  }

  @Input() hasPermissionElse?: TemplateRef<unknown>;
}
