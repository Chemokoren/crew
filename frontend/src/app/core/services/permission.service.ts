import { Injectable, signal, computed, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';

const STORAGE_KEY = 'amy_user_permissions';

/**
 * PermissionService provides a signal-based permission store.
 * - Loads user permissions on login from the backend.
 * - Caches in localStorage for offline awareness.
 * - Exposes synchronous `can()`, `canAny()`, `canAll()` checks.
 *
 * All permissions flow through the dynamic RBAC system — the backend
 * merges system-role permissions into the response so no frontend
 * role-based overrides are needed.
 */
@Injectable({ providedIn: 'root' })
export class PermissionService {
  private http = inject(HttpClient);
  private apiUrl = `${environment.apiUrl}/rbac`;

  /** Current user's permission keys */
  readonly permissions = signal<Set<string>>(new Set());

  /** Whether permissions have been loaded at least once */
  readonly loaded = signal(false);

  /** Loading state */
  readonly loading = signal(false);

  /** Total permission count */
  readonly count = computed(() => this.permissions().size);

  constructor() {
    this.loadFromStorage();
  }

  /** Check if user has a specific permission */
  can(permKey: string): boolean {
    return this.permissions().has(permKey);
  }

  /** Check if user has ANY of the specified permissions */
  canAny(...permKeys: string[]): boolean {
    const perms = this.permissions();
    return permKeys.some(k => perms.has(k));
  }

  /** Check if user has ALL of the specified permissions */
  canAll(...permKeys: string[]): boolean {
    const perms = this.permissions();
    return permKeys.every(k => perms.has(k));
  }

  /** Load permissions for the current user from the backend */
  loadForUser(userId: string, tenantId?: string): void {
    this.loading.set(true);
    const params: Record<string, string> = {};
    if (tenantId) params['tenant_id'] = tenantId;

    this.http.get<{ success: boolean; data: string[] }>(
      `${this.apiUrl}/users/${userId}/permissions`, { params }
    ).subscribe({
      next: (res) => {
        if (res.success && res.data) {
          this.setPermissions(res.data);
        }
        this.loading.set(false);
        this.loaded.set(true);
      },
      error: () => {
        this.loading.set(false);
        this.loaded.set(true);
      }
    });
  }

  /** Set permissions directly (used during login) */
  setPermissions(keys: string[]): void {
    this.permissions.set(new Set(keys));
    this.saveToStorage(keys);
    this.loaded.set(true);
  }

  /** Clear all permissions (used during logout) */
  clear(): void {
    this.permissions.set(new Set());
    this.loaded.set(false);
    localStorage.removeItem(STORAGE_KEY);
  }

  /** Load cached permissions from localStorage */
  private loadFromStorage(): void {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const keys: string[] = JSON.parse(stored);
        this.permissions.set(new Set(keys));
        this.loaded.set(true);
      }
    } catch {
      // Ignore corrupt storage
    }
  }

  /** Persist permissions to localStorage */
  private saveToStorage(keys: string[]): void {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(keys));
    } catch {
      // Ignore storage errors
    }
  }
}
