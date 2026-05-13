import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { PermissionService } from '../services/permission.service';

/**
 * Route guard that checks if the current user has ALL of the specified permissions.
 * Usage in routes: canActivate: [permissionGuard('roles.view', 'roles.manage_permissions')]
 */
export function permissionGuard(...permKeys: string[]): CanActivateFn {
  return () => {
    const permissionService = inject(PermissionService);
    const router = inject(Router);

    if (permissionService.canAll(...permKeys)) {
      return true;
    }

    // Redirect to dashboard if permission denied
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

    if (permissionService.canAny(...permKeys)) {
      return true;
    }

    return router.createUrlTree(['/dashboard']);
  };
}
