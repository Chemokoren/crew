import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../../../../environments/environment';
import {
  RBACRole, PermissionDef, RoleTemplate, UserRoleAssignment,
  RoleComparison, PermissionMatrix, RBACPolicy
} from '../../../../core/models';

interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: { page: number; per_page: number; total: number; total_pages: number };
  grouped?: Record<string, PermissionDef[]>;
}

@Injectable({ providedIn: 'root' })
export class RbacApiService {
  private http = inject(HttpClient);
  private base = `${environment.apiUrl}/rbac`;

  // ── Roles ───────────────────────────────────────────────────────────

  listRoles(params: Record<string, string> = {}): Observable<ApiResponse<RBACRole[]>> {
    return this.http.get<ApiResponse<RBACRole[]>>(`${this.base}/roles`, { params });
  }

  getRole(id: string): Observable<RBACRole> {
    return this.http.get<ApiResponse<RBACRole>>(`${this.base}/roles/${id}`).pipe(map(r => r.data));
  }

  createRole(body: { name: string; description?: string; tenant_id?: string; industry_type?: string; is_template?: boolean }): Observable<RBACRole> {
    return this.http.post<ApiResponse<RBACRole>>(`${this.base}/roles`, body).pipe(map(r => r.data));
  }

  updateRole(id: string, body: { name: string; description?: string; is_active?: boolean }): Observable<RBACRole> {
    return this.http.put<ApiResponse<RBACRole>>(`${this.base}/roles/${id}`, body).pipe(map(r => r.data));
  }

  deleteRole(id: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/roles/${id}`);
  }

  cloneRole(id: string, name: string, tenantId?: string): Observable<RBACRole> {
    return this.http.post<ApiResponse<RBACRole>>(`${this.base}/roles/${id}/clone`, { name, tenant_id: tenantId }).pipe(map(r => r.data));
  }

  toggleRoleActive(id: string, active: boolean): Observable<void> {
    return this.http.post<void>(`${this.base}/roles/${id}/activate`, { active });
  }

  compareRoles(roleAId: string, roleBId: string): Observable<RoleComparison> {
    return this.http.post<ApiResponse<RoleComparison>>(`${this.base}/roles/compare`, { role_a_id: roleAId, role_b_id: roleBId }).pipe(map(r => r.data));
  }

  // ── Permissions ─────────────────────────────────────────────────────

  getRolePermissions(roleId: string): Observable<PermissionDef[]> {
    return this.http.get<ApiResponse<PermissionDef[]>>(`${this.base}/roles/${roleId}/permissions`).pipe(map(r => r.data));
  }

  setRolePermissions(roleId: string, permissionKeys: string[]): Observable<void> {
    return this.http.put<void>(`${this.base}/roles/${roleId}/permissions`, { permission_keys: permissionKeys });
  }

  listPermissions(params: Record<string, string> = {}): Observable<{ data: PermissionDef[]; grouped: Record<string, PermissionDef[]> }> {
    return this.http.get<ApiResponse<PermissionDef[]> & { grouped: Record<string, PermissionDef[]> }>(`${this.base}/permissions`, { params })
      .pipe(map(r => ({ data: r.data, grouped: r.grouped || {} })));
  }

  listPermissionModules(): Observable<string[]> {
    return this.http.get<ApiResponse<string[]>>(`${this.base}/permissions/modules`).pipe(map(r => r.data));
  }

  // ── User Roles ──────────────────────────────────────────────────────

  getUserRoles(userId: string, tenantId?: string): Observable<UserRoleAssignment[]> {
    const params: Record<string, string> = {};
    if (tenantId) params['tenant_id'] = tenantId;
    return this.http.get<ApiResponse<UserRoleAssignment[]>>(`${this.base}/users/${userId}/roles`, { params }).pipe(map(r => r.data));
  }

  assignRoleToUser(userId: string, roleId: string, tenantId?: string, expiresAt?: string): Observable<void> {
    return this.http.post<void>(`${this.base}/users/${userId}/roles`, { role_id: roleId, tenant_id: tenantId, expires_at: expiresAt });
  }

  revokeRoleFromUser(userId: string, roleId: string, tenantId?: string): Observable<void> {
    let params = new HttpParams();
    if (tenantId) params = params.set('tenant_id', tenantId);
    return this.http.delete<void>(`${this.base}/users/${userId}/roles/${roleId}`, { params });
  }

  getUserPermissions(userId: string, tenantId?: string): Observable<string[]> {
    const params: Record<string, string> = {};
    if (tenantId) params['tenant_id'] = tenantId;
    return this.http.get<ApiResponse<string[]>>(`${this.base}/users/${userId}/permissions`, { params }).pipe(map(r => r.data));
  }

  // ── Templates ───────────────────────────────────────────────────────

  listTemplates(industryType?: string): Observable<RoleTemplate[]> {
    const params: Record<string, string> = {};
    if (industryType) params['industry_type'] = industryType;
    return this.http.get<ApiResponse<RoleTemplate[]>>(`${this.base}/templates`, { params }).pipe(map(r => r.data));
  }

  applyTemplate(templateId: string, tenantId: string): Observable<RBACRole> {
    return this.http.post<ApiResponse<RBACRole>>(`${this.base}/templates/${templateId}/apply`, { tenant_id: tenantId }).pipe(map(r => r.data));
  }

  // ── Policies ────────────────────────────────────────────────────────

  listPolicies(tenantId?: string): Observable<ApiResponse<RBACPolicy[]>> {
    const params: Record<string, string> = {};
    if (tenantId) params['tenant_id'] = tenantId;
    return this.http.get<ApiResponse<RBACPolicy[]>>(`${this.base}/policies`, { params });
  }

  createPolicy(body: Partial<RBACPolicy>): Observable<RBACPolicy> {
    return this.http.post<ApiResponse<RBACPolicy>>(`${this.base}/policies`, body).pipe(map(r => r.data));
  }

  // ── Matrix ──────────────────────────────────────────────────────────

  getPermissionMatrix(tenantId?: string, industryType?: string): Observable<PermissionMatrix> {
    const params: Record<string, string> = {};
    if (tenantId) params['tenant_id'] = tenantId;
    if (industryType) params['industry_type'] = industryType;
    return this.http.get<ApiResponse<PermissionMatrix>>(`${this.base}/matrix`, { params }).pipe(map(r => r.data));
  }
}
