import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { PermissionService } from '../services/permission.service';
import { ToastService } from '../services/toast.service';

/**
 * Route guard that checks if the current user has ALL of the specified permissions.
 * Usage in routes: canActivate: [permissionGuard('roles.view', 'roles.manage_permissions')]
 */
export function permissionGuard(...permKeys: string[]): CanActivateFn {
  return () => {
    const permissionService = inject(PermissionService);
    const router = inject(Router);
    const toast = inject(ToastService);

    if (permissionService.canAll(...permKeys)) {
      return true;
    }

    toast.warning('You do not have permission to access that page.');
    return router.createUrlTree(['/dashboard']);
  };
}

/**
 * Route guard that checks if the current user has ANY of the specified permissions.
 * Usage in routes: canActivate: [anyPermissionGuard('roles.view', 'platform.manage_roles')]
 */
export function anyPermissionGuard(...permKeys: string[]): CanActivateFn {
  return () => {
    const permissionService = inject(PermissionService);
    const router = inject(Router);
    const toast = inject(ToastService);

    if (permissionService.canAny(...permKeys)) {
      return true;
    }

    toast.warning('You do not have permission to access that page.');
    return router.createUrlTree(['/dashboard']);
  };
}
