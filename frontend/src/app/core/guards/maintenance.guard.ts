import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map, catchError, of } from 'rxjs';
import { ApiService } from '../services/api.service';
import { AuthService } from '../services/auth.service';

/**
 * Guard that blocks ALL routes during maintenance mode.
 * Only SYSTEM_ADMIN users who are already logged in can bypass.
 * If maintenance is active, redirects to /maintenance.
 */
export const maintenanceGuard: CanActivateFn = () => {
  const api = inject(ApiService);
  const auth = inject(AuthService);
  const router = inject(Router);

  // If already logged in as SYSTEM_ADMIN, always allow
  const user = auth.currentUser();
  if (user && user.system_role === 'SYSTEM_ADMIN') {
    return true;
  }

  return api.getSystemStatus().pipe(
    map(r => {
      if (r.data?.maintenance) {
        router.navigate(['/maintenance']);
        return false;
      }
      return true;
    }),
    catchError(() => of(true)) // If status check fails, don't block
  );
};

/**
 * Inverse guard for the /maintenance route itself.
 * Only allows access when maintenance IS active (or always if not logged in).
 * If maintenance is NOT active, redirects to login.
 */
export const maintenancePageGuard: CanActivateFn = () => {
  const api = inject(ApiService);
  const router = inject(Router);

  return api.getSystemStatus().pipe(
    map(r => {
      if (r.data?.maintenance) {
        return true; // Show maintenance page
      }
      router.navigate(['/auth/login']);
      return false;
    }),
    catchError(() => {
      router.navigate(['/auth/login']);
      return of(false);
    })
  );
};
