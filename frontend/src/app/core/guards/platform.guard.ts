import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

/**
 * Guard that restricts access to /platform/* routes to platform staff roles only.
 */
export const platformGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (auth.isPlatformUser()) {
    return true;
  }

  router.navigate(['/dashboard']);
  return false;
};
