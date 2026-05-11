import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { ToastService } from '../services/toast.service';

/**
 * Blocks employees with non-verified KYC from accessing protected routes.
 * Allows access only to /profile and /notifications.
 * Redirects to /profile with a warning toast.
 */
export const kycGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const toast = inject(ToastService);

  if (auth.isKycBlocked()) {
    toast.warning('Your identity verification is incomplete. Please complete your KYC to access all features.');
    router.navigate(['/profile']);
    return false;
  }

  return true;
};
