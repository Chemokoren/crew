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
  private currentView: 'primary' | 'else' | null = null;
  private keys: string[] = [];

  @Input()
  set hasPermission(permKey: string | string[]) {
    this.keys = Array.isArray(permKey) ? permKey : [permKey];
    this.render();
  }

  @Input()
  set hasPermissionElse(template: TemplateRef<unknown> | undefined) {
    this.elseTemplate = template;
    this.render();
  }

  private elseTemplate?: TemplateRef<unknown>;

  constructor() {
    effect(() => {
      this.permissionService.permissions();
      this.render();
    });
  }

  private render(): void {
    const allowed = this.keys.length > 0 && this.permissionService.canAll(...this.keys);
    const nextView = allowed ? 'primary' : (this.elseTemplate ? 'else' : null);

    if (this.currentView === nextView) {
      return;
    }

    this.viewContainer.clear();
    if (nextView === 'primary') {
      this.viewContainer.createEmbeddedView(this.templateRef);
    } else if (nextView === 'else' && this.elseTemplate) {
      this.viewContainer.createEmbeddedView(this.elseTemplate);
    }
    this.currentView = nextView;
  }
}
