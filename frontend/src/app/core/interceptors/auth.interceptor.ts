import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, switchMap, throwError } from 'rxjs';
import { Router } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { ToastService } from '../services/toast.service';

/** Pages where the interceptor should NOT force redirects or logouts */
function isExemptPage(router: Router): boolean {
  const path = window.location.pathname;
  return path.startsWith('/system-admin') || path.startsWith('/maintenance');
}

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  const toast = inject(ToastService);
  const router = inject(Router);
  const token = authService.accessToken;

  // Skip auth for login/register/refresh/status
  const publicPaths = ['/auth/login', '/auth/register', '/auth/refresh', '/system/status'];
  const isPublic = publicPaths.some(path => req.url.includes(path));

  if (token && !isPublic) {
    req = req.clone({
      setHeaders: { Authorization: `Bearer ${token}` },
    });
  }

  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      // Handle 503 maintenance mode — redirect to maintenance page
      // unless user is on an exempt page (system-admin login, maintenance page itself)
      if (error.status === 503) {
        const code = error.error?.error?.code;
        if (code === 'MAINTENANCE' && !isExemptPage(router)) {
          router.navigate(['/maintenance']);
          return throwError(() => error);
        }
      }

      // Handle 401 — attempt token refresh, then logout on failure
      // Skip on exempt pages to prevent redirect loops during maintenance
      if (error.status === 401 && !isPublic && !req.url.includes('/auth/refresh')) {
        if (isExemptPage(router)) {
          // On system-admin or maintenance page, just silently fail
          return throwError(() => error);
        }
        return authService.refreshAccessToken().pipe(
          switchMap(res => {
            const newReq = req.clone({
              setHeaders: { Authorization: `Bearer ${res.data.access_token}` },
            });
            return next(newReq);
          }),
          catchError(() => {
            toast.error('Session expired. Please log in again.');
            authService.logout();
            return throwError(() => error);
          })
        );
      }
      return throwError(() => error);
    })
  );
};
