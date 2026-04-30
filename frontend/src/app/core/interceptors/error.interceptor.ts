import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, throwError } from 'rxjs';
import { ToastService } from '../services/toast.service';

export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const toast = inject(ToastService);

  // Allow callers to suppress the global error toast for specific requests
  // by setting a custom header. The header is stripped before sending.
  const skipToast = req.headers.has('X-Skip-Error-Toast');
  const cleanedReq = skipToast ? req.clone({ headers: req.headers.delete('X-Skip-Error-Toast') }) : req;

  return next(cleanedReq).pipe(
    catchError((error: HttpErrorResponse) => {
      let message = 'An unexpected error occurred';

      if (error.status === 0) {
        message = 'Unable to connect to server. Please check your connection.';
      } else if (error.error?.message) {
        message = error.error.message;
      } else if (error.status === 403) {
        message = 'You do not have permission to perform this action.';
      } else if (error.status === 404) {
        message = 'The requested resource was not found.';
      } else if (error.status === 409) {
        message = error.error?.message || 'A conflict occurred. Please try again.';
      } else if (error.status === 429) {
        message = 'Too many requests. Please slow down.';
      } else if (error.status >= 500) {
        message = 'Server error. Please try again later.';
      }

      // Don't show toast for auth-related 401s (handled by auth interceptor)
      // or for requests that opted out via X-Skip-Error-Toast
      if (error.status !== 401 && !skipToast) {
        toast.error(message);
      }

      return throwError(() => error);
    })
  );
};
