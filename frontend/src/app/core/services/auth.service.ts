import { Injectable, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, tap, catchError, throwError, BehaviorSubject } from 'rxjs';
import { environment } from '../../../environments/environment';
import { User, AuthResponse, TokenPair, SystemRole, ApiResponse } from '../models';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly API = environment.apiUrl;
  private tokenRefreshInProgress = false;

  private readonly currentUserSignal = signal<User | null>(null);
  readonly currentUser = this.currentUserSignal.asReadonly();
  readonly isAuthenticated = computed(() => !!this.currentUserSignal());
  readonly userRole = computed(() => this.currentUserSignal()?.system_role ?? null);

  private refreshTokenSubject = new BehaviorSubject<string | null>(null);

  constructor(private http: HttpClient, private router: Router) {
    this.loadUserFromStorage();
  }

  private loadUserFromStorage(): void {
    const userJson = localStorage.getItem('amy_user');
    if (userJson) {
      try {
        this.currentUserSignal.set(JSON.parse(userJson));
      } catch {
        this.clearAuth();
      }
    }
  }

  get accessToken(): string | null {
    return localStorage.getItem('amy_access_token');
  }

  get refreshToken(): string | null {
    return localStorage.getItem('amy_refresh_token');
  }

  login(phone: string, password: string): Observable<ApiResponse<AuthResponse>> {
    return this.http.post<ApiResponse<AuthResponse>>(`${this.API}/auth/login`, { phone, password }).pipe(
      tap(res => this.handleAuthSuccess(res.data)),
      catchError(err => throwError(() => err))
    );
  }

  register(payload: {
    phone: string;
    email?: string;
    password: string;
    role: SystemRole;
    first_name?: string;
    last_name?: string;
    national_id?: string;
    crew_role?: string;
  }): Observable<ApiResponse<AuthResponse>> {
    return this.http.post<ApiResponse<AuthResponse>>(`${this.API}/auth/register`, payload).pipe(
      tap(res => this.handleAuthSuccess(res.data)),
      catchError(err => throwError(() => err))
    );
  }

  refreshAccessToken(): Observable<ApiResponse<TokenPair>> {
    const refreshToken = this.refreshToken;
    if (!refreshToken) {
      this.logout();
      return throwError(() => new Error('No refresh token'));
    }

    return this.http.post<ApiResponse<TokenPair>>(`${this.API}/auth/refresh`, {
      refresh_token: refreshToken,
    }).pipe(
      tap(res => {
        localStorage.setItem('amy_access_token', res.data.access_token);
        localStorage.setItem('amy_refresh_token', res.data.refresh_token);
      }),
      catchError(err => {
        this.logout();
        return throwError(() => err);
      })
    );
  }

  fetchProfile(): Observable<ApiResponse<User>> {
    return this.http.get<ApiResponse<User>>(`${this.API}/auth/me`).pipe(
      tap(res => {
        this.currentUserSignal.set(res.data);
        localStorage.setItem('amy_user', JSON.stringify(res.data));
      })
    );
  }

  changePassword(oldPassword: string, newPassword: string): Observable<unknown> {
    return this.http.post(`${this.API}/auth/change-password`, {
      old_password: oldPassword,
      new_password: newPassword,
    });
  }

  logout(): void {
    this.clearAuth();
    this.router.navigate(['/auth/login']);
  }

  hasRole(...roles: SystemRole[]): boolean {
    const user = this.currentUserSignal();
    return user ? roles.includes(user.system_role) : false;
  }

  isAdmin(): boolean {
    return this.hasRole('SYSTEM_ADMIN');
  }

  isSaccoAdmin(): boolean {
    return this.hasRole('SACCO_ADMIN');
  }

  private handleAuthSuccess(data: AuthResponse): void {
    localStorage.setItem('amy_access_token', data.tokens.access_token);
    localStorage.setItem('amy_refresh_token', data.tokens.refresh_token);
    localStorage.setItem('amy_user', JSON.stringify(data.user));
    this.currentUserSignal.set(data.user);
  }

  private clearAuth(): void {
    localStorage.removeItem('amy_access_token');
    localStorage.removeItem('amy_refresh_token');
    localStorage.removeItem('amy_user');
    this.currentUserSignal.set(null);
  }
}
