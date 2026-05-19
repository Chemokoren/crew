import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { forkJoin } from 'rxjs';
import { RbacApiService } from './services/rbac-api.service';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { AdminUser, AuditLog, RBACRole, PermissionDef, RoleTemplate, PermissionMatrix, RoleComparison, UserRoleAssignment } from '../../../core/models';

type ViewTab = 'roles' | 'matrix' | 'templates' | 'compare' | 'assignments' | 'audit';
type RoleView = 'grid' | 'table';

@Component({
  selector: 'app-platform-roles',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="rbac-page animate-fade-in">
      <!-- Header -->
      <div class="rbac-header">
        <div class="rbac-header-left">
          <h1 class="rbac-title">
            <span class="material-icons-round rbac-title-icon">admin_panel_settings</span>
            Roles & Permissions
          </h1>
          <p class="rbac-subtitle">Manage roles, permissions, and access control across your organization</p>
        </div>
        <div class="rbac-header-actions">
          <button class="rbac-btn rbac-btn-outline" (click)="refreshData()">
            <span class="material-icons-round">refresh</span> Sync
          </button>
          <button class="rbac-btn rbac-btn-primary" (click)="showCreateDialog.set(true)">
            <span class="material-icons-round">add</span> New Role
          </button>
        </div>
      </div>

      <!-- Stats -->
      <div class="rbac-stats">
        <div class="stat-card glass-card">
          <div class="stat-icon si-purple"><span class="material-icons-round">shield</span></div>
          <div class="stat-info"><span class="stat-value">{{ roles().length }}</span><span class="stat-label">Total Roles</span></div>
        </div>
        <div class="stat-card glass-card">
          <div class="stat-icon si-blue"><span class="material-icons-round">verified_user</span></div>
          <div class="stat-info"><span class="stat-value">{{ systemRoleCount() }}</span><span class="stat-label">System Roles</span></div>
        </div>
        <div class="stat-card glass-card">
          <div class="stat-icon si-teal"><span class="material-icons-round">lock</span></div>
          <div class="stat-info"><span class="stat-value">{{ totalPerms() }}</span><span class="stat-label">Permissions</span></div>
        </div>
        <div class="stat-card glass-card">
          <div class="stat-icon si-amber"><span class="material-icons-round">style</span></div>
          <div class="stat-info"><span class="stat-value">{{ templates().length }}</span><span class="stat-label">Templates</span></div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="rbac-tabs">
        <button [class.active]="activeTab() === 'roles'" (click)="activeTab.set('roles')">
          <span class="material-icons-round">group</span> Roles
        </button>
        <button [class.active]="activeTab() === 'matrix'" (click)="activeTab.set('matrix'); loadMatrix()">
          <span class="material-icons-round">grid_on</span> Permission Matrix
        </button>
        <button [class.active]="activeTab() === 'templates'" (click)="activeTab.set('templates')">
          <span class="material-icons-round">dashboard_customize</span> Industry Templates
        </button>
        <button [class.active]="activeTab() === 'compare'" (click)="activeTab.set('compare')">
          <span class="material-icons-round">compare_arrows</span> Compare
        </button>
        <button [class.active]="activeTab() === 'assignments'" (click)="activeTab.set('assignments'); loadUsers()">
          <span class="material-icons-round">manage_accounts</span> Assignments
        </button>
        <button [class.active]="activeTab() === 'audit'" (click)="activeTab.set('audit'); loadAudit()">
          <span class="material-icons-round">history</span> Activity
        </button>
      </div>

      <!-- Tab Content -->
      @switch (activeTab()) {
        @case ('roles') {
          <!-- Roles List -->
          <div class="rbac-toolbar">
            <div class="search-box">
              <span class="material-icons-round">search</span>
              <input type="text" placeholder="Search roles..." [(ngModel)]="roleSearch" (ngModelChange)="filterRoles()">
            </div>
            <div class="filter-group">
              <select [(ngModel)]="industryFilter" (ngModelChange)="filterRoles()">
                <option value="">All Industries</option>
                <option value="TRANSPORT">Transport</option>
                <option value="CONSTRUCTION">Construction</option>
                <option value="LOGISTICS">Logistics</option>
                <option value="HEALTH">Health</option>
                <option value="AGRICULTURE">Agriculture</option>
                <option value="HOSPITALITY">Hospitality</option>
                <option value="PLATFORM">Platform</option>
              </select>
            </div>
            <div class="segmented-control">
              <button [class.active]="roleView() === 'grid'" (click)="roleView.set('grid')" title="Card view">
                <span class="material-icons-round">grid_view</span>
              </button>
              <button [class.active]="roleView() === 'table'" (click)="roleView.set('table')" title="Table view">
                <span class="material-icons-round">table_rows</span>
              </button>
            </div>
          </div>

          @if (loading()) {
            <div class="rbac-loading"><div class="spinner"></div><span>Loading roles...</span></div>
          } @else if (roleView() === 'grid') {
            <div class="roles-grid">
              @for (role of filteredRoles(); track role.id) {
                <div class="role-card glass-card" [class.inactive]="!role.is_active" [class.system]="role.is_system">
                  <div class="role-card-header">
                    <div class="role-icon-wrap" [class]="'ri-' + getRiskColor(role)">
                      <span class="material-icons-round">{{ role.is_system ? 'security' : 'shield' }}</span>
                    </div>
                    <div class="role-meta">
                      <h3 class="role-name">{{ role.name }}</h3>
                      <span class="role-slug">{{ role.slug }}</span>
                    </div>
                    <div class="role-badges">
                      @if (role.is_system) { <span class="badge badge-system">System</span> }
                      @if (role.is_template) { <span class="badge badge-template">Template</span> }
                      @if (!role.is_active) { <span class="badge badge-inactive">Inactive</span> }
                      @if (role.industry_type) { <span class="badge badge-industry">{{ role.industry_type }}</span> }
                    </div>
                  </div>
                  <p class="role-desc">{{ role.description || 'No description' }}</p>
                  <div class="role-stats-row">
                    <div class="role-stat">
                      <span class="material-icons-round">lock</span>
                      <span>{{ role.permission_count || 0 }} perms</span>
                    </div>
                    <div class="role-stat">
                      <span class="material-icons-round">people</span>
                      <span>{{ role.user_count || 0 }} users</span>
                    </div>
                  </div>
                  <div class="role-actions">
                    <button class="rbac-btn-sm" (click)="editRole(role)" title="Edit">
                      <span class="material-icons-round">edit</span>
                    </button>
                    <button class="rbac-btn-sm" (click)="openPermissions(role)" title="Permissions">
                      <span class="material-icons-round">tune</span>
                    </button>
                    <button class="rbac-btn-sm" (click)="cloneRoleAction(role)" title="Clone">
                      <span class="material-icons-round">content_copy</span>
                    </button>
                    <button class="rbac-btn-sm" (click)="toggleRoleActiveAction(role)" [title]="role.is_active ? 'Archive' : 'Restore'">
                      <span class="material-icons-round">{{ role.is_active ? 'archive' : 'unarchive' }}</span>
                    </button>
                    @if (!role.is_system) {
                      <button class="rbac-btn-sm rbac-btn-danger" (click)="deleteRoleAction(role)" title="Delete">
                        <span class="material-icons-round">delete</span>
                      </button>
                    }
                  </div>
                </div>
              } @empty {
                <div class="empty-state glass-card">
                  <span class="material-icons-round">shield</span>
                  <h3>No roles found</h3>
                  <p>Create your first role or apply an industry template</p>
                </div>
              }
            </div>
          } @else {
            <div class="roles-table-wrap glass-card">
              <table class="roles-table">
                <thead>
                  <tr>
                    <th>Role</th>
                    <th>Industry</th>
                    <th>Status</th>
                    <th>Permissions</th>
                    <th>Users</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  @for (role of filteredRoles(); track role.id) {
                    <tr [class.inactive]="!role.is_active">
                      <td>
                        <div class="role-table-name">
                          <span class="material-icons-round">{{ role.is_system ? 'security' : 'shield' }}</span>
                          <div><strong>{{ role.name }}</strong><small>{{ role.slug }}</small></div>
                        </div>
                      </td>
                      <td>{{ role.industry_type || 'GLOBAL' }}</td>
                      <td>
                        @if (role.is_system) { <span class="badge badge-system">System</span> }
                        @if (role.is_template) { <span class="badge badge-template">Template</span> }
                        @if (role.is_active) { <span class="badge badge-industry">Active</span> } @else { <span class="badge badge-inactive">Archived</span> }
                      </td>
                      <td>{{ role.permission_count || 0 }}</td>
                      <td>{{ role.user_count || 0 }}</td>
                      <td>
                        <div class="role-actions compact">
                          <button class="rbac-btn-sm" (click)="editRole(role)" title="Edit"><span class="material-icons-round">edit</span></button>
                          <button class="rbac-btn-sm" (click)="openPermissions(role)" title="Permissions"><span class="material-icons-round">tune</span></button>
                          <button class="rbac-btn-sm" (click)="cloneRoleAction(role)" title="Clone"><span class="material-icons-round">content_copy</span></button>
                          <button class="rbac-btn-sm" (click)="toggleRoleActiveAction(role)" [title]="role.is_active ? 'Archive' : 'Restore'"><span class="material-icons-round">{{ role.is_active ? 'archive' : 'unarchive' }}</span></button>
                          @if (!role.is_system) {
                            <button class="rbac-btn-sm rbac-btn-danger" (click)="deleteRoleAction(role)" title="Delete"><span class="material-icons-round">delete</span></button>
                          }
                        </div>
                      </td>
                    </tr>
                  } @empty {
                    <tr><td colspan="6"><div class="empty-row">No roles found</div></td></tr>
                  }
                </tbody>
              </table>
            </div>
          }
        }

        @case ('matrix') {
          <!-- Permission Matrix -->
          <div class="rbac-toolbar">
            <div class="search-box">
              <span class="material-icons-round">search</span>
              <input type="text" placeholder="Search permissions..." [(ngModel)]="matrixSearch">
            </div>
            <button class="rbac-btn rbac-btn-outline" (click)="expandAllMatrixModules()">
              <span class="material-icons-round">unfold_more</span> Expand
            </button>
            <button class="rbac-btn rbac-btn-outline" (click)="collapseAllMatrixModules()">
              <span class="material-icons-round">unfold_less</span> Collapse
            </button>
          </div>

          @if (matrixLoading()) {
            <div class="rbac-loading"><div class="spinner"></div><span>Building matrix...</span></div>
          } @else if (matrix()) {
            <div class="matrix-layout">
              <aside class="module-sidebar glass-card">
                @for (module of matrix()!.modules; track module) {
                  <button [class.active]="expandedMatrixModules().has(module)" (click)="toggleMatrixModule(module)">
                    <span class="material-icons-round">{{ expandedMatrixModules().has(module) ? 'folder_open' : 'folder' }}</span>
                    <span>{{ module }}</span>
                    <em>{{ getModulePerms(module).length }}</em>
                    @if (moduleChangeCount(module) > 0) { <strong>{{ moduleChangeCount(module) }}</strong> }
                  </button>
                }
              </aside>

              <div class="matrix-container glass-card">
                <div class="matrix-scroll">
                  <table class="matrix-table">
                    <thead>
                      <tr>
                        <th class="matrix-sticky-col">Permission</th>
                        @for (role of matrix()!.roles; track role.id) {
                          <th class="matrix-role-header">
                            <div class="mrh-inner">
                              <span class="mrh-name">{{ role.name }}</span>
                              <span class="mrh-count">{{ (matrix()!.grants[role.id] || []).length }}</span>
                              <button class="matrix-bulk-btn" (click)="toggleRoleColumn(role.id)" title="Toggle role column">
                                <span class="material-icons-round">{{ isRoleFullyGranted(role.id) ? 'remove_done' : 'done_all' }}</span>
                              </button>
                            </div>
                          </th>
                        }
                      </tr>
                    </thead>
                    <tbody>
                      @for (module of matrix()!.modules; track module) {
                        <tr class="matrix-module-row" [attr.id]="'module-' + module">
                          <td [attr.colspan]="matrix()!.roles.length + 1" class="matrix-module-cell">
                            <button class="matrix-bulk-btn" (click)="toggleModuleForAll(module)" title="Toggle module">
                              <span class="material-icons-round">{{ isModuleFullyGranted(module) ? 'remove_done' : 'done_all' }}</span>
                            </button>
                            <span class="material-icons-round">folder</span> {{ module }}
                          </td>
                        </tr>
                        @if (expandedMatrixModules().has(module)) {
                          @for (perm of getModulePerms(module); track perm.key) {
                            @if (matchesSearch(perm)) {
                              <tr class="matrix-perm-row">
                                <td class="matrix-sticky-col matrix-perm-cell">
                                  <button class="matrix-row-btn" (click)="togglePermissionRow(perm.key)" title="Toggle row">
                                    <span class="material-icons-round">{{ isPermissionGrantedForAll(perm.key) ? 'remove_done' : 'done_all' }}</span>
                                  </button>
                                  <div class="perm-info">
                                    <span class="perm-key">{{ perm.key }}</span>
                                    <span class="perm-desc">{{ perm.description }}</span>
                                  </div>
                                  <span class="risk-dot" [class]="'risk-' + perm.risk_level" [title]="perm.risk_level"></span>
                                </td>
                                @for (role of matrix()!.roles; track role.id; let col = $index) {
                                  <td class="matrix-cell" (click)="toggleMatrixPerm(role.id, perm.key)"
                                      (keydown)="onMatrixKeydown($event, role.id, perm.key)"
                                      tabindex="0" role="checkbox"
                                      [attr.aria-checked]="isGranted(role.id, perm.key)"
                                      [attr.data-row]="permissionRowIndex(perm.key)"
                                      [attr.data-col]="col"
                                      [class.granted]="isGranted(role.id, perm.key)"
                                      [class.modified]="isModified(role.id, perm.key)">
                                    @if (isGranted(role.id, perm.key)) {
                                      <span class="material-icons-round check-icon">check_circle</span>
                                    } @else {
                                      <span class="material-icons-round uncheck-icon">radio_button_unchecked</span>
                                    }
                                  </td>
                                }
                              </tr>
                            }
                          }
                        }
                      }
                    </tbody>
                  </table>
                </div>
                @if (matrixDirty()) {
                  <div class="matrix-save-bar">
                    <span>{{ matrixChangeCount() }} unsaved changes</span>
                    <div>
                      <button class="rbac-btn rbac-btn-outline" (click)="discardMatrixChanges()">Discard</button>
                      <button class="rbac-btn rbac-btn-primary" (click)="saveMatrixChanges()">
                        <span class="material-icons-round">save</span> Save All
                      </button>
                    </div>
                  </div>
                }
              </div>
            </div>
          }
        }

        @case ('templates') {
          <!-- Templates -->
          @if (loading()) {
            <div class="rbac-loading"><div class="spinner"></div><span>Loading templates...</span></div>
          } @else {
            <div class="rbac-toolbar">
              <div class="search-box">
                <span class="material-icons-round">business</span>
                <input type="text" placeholder="Target tenant UUID" [(ngModel)]="templateTenantId">
              </div>
            </div>
            <div class="template-industries">
              @for (industry of templateIndustries(); track industry) {
                <div class="industry-section">
                  <h3 class="industry-title">
                    <span class="material-icons-round">{{ getIndustryIcon(industry) }}</span>
                    {{ industry }}
                  </h3>
                  <div class="template-grid">
                    @for (tmpl of getIndustryTemplates(industry); track tmpl.id) {
                      <div class="template-card glass-card">
                        <div class="tmpl-header">
                          <h4>{{ tmpl.role_name }}</h4>
                          @if (tmpl.is_default) { <span class="badge badge-system">Default</span> }
                        </div>
                        <p class="tmpl-desc">{{ tmpl.description }}</p>
                        <div class="tmpl-perms">
                          <span class="material-icons-round">lock</span>
                          {{ tmpl.permissions.length }} permissions
                        </div>
                        <div class="tmpl-preview">
                          @for (perm of tmpl.permissions.slice(0, 5); track perm) {
                            <span>{{ perm }}</span>
                          }
                          @if (tmpl.permissions.length > 5) { <span>+{{ tmpl.permissions.length - 5 }}</span> }
                        </div>
                        <button class="rbac-btn rbac-btn-outline tmpl-apply" (click)="applyTemplateAction(tmpl)">
                          <span class="material-icons-round">add_task</span> Apply
                        </button>
                      </div>
                    }
                  </div>
                </div>
              }
            </div>
          }
        }

        @case ('compare') {
          <div class="compare-panel glass-card">
            <div class="compare-controls">
              <div class="form-group">
                <label>First Role</label>
                <select [(ngModel)]="compareRoleAId">
                  <option value="">Select role</option>
                  @for (role of roles(); track role.id) { <option [value]="role.id">{{ role.name }}</option> }
                </select>
              </div>
              <div class="form-group">
                <label>Second Role</label>
                <select [(ngModel)]="compareRoleBId">
                  <option value="">Select role</option>
                  @for (role of roles(); track role.id) { <option [value]="role.id">{{ role.name }}</option> }
                </select>
              </div>
              <button class="rbac-btn rbac-btn-primary" (click)="compareRoles()" [disabled]="!compareRoleAId || !compareRoleBId || compareRoleAId === compareRoleBId">
                <span class="material-icons-round">compare_arrows</span> Compare
              </button>
            </div>
            @if (comparison()) {
              <div class="comparison-grid">
                <div class="comparison-col added">
                  <h3>{{ roleName(compareRoleAId) }}</h3>
                  @for (perm of comparison()!.only_in_a; track perm) { <span>{{ perm }}</span> } @empty { <em>No unique permissions</em> }
                </div>
                <div class="comparison-col removed">
                  <h3>{{ roleName(compareRoleBId) }}</h3>
                  @for (perm of comparison()!.only_in_b; track perm) { <span>{{ perm }}</span> } @empty { <em>No unique permissions</em> }
                </div>
                <div class="comparison-col shared">
                  <h3>Shared</h3>
                  @for (perm of comparison()!.shared; track perm) { <span>{{ perm }}</span> } @empty { <em>No shared permissions</em> }
                </div>
              </div>
            }
          </div>
        }

        @case ('assignments') {
          <div class="assignment-panel glass-card">
            <div class="rbac-toolbar">
              <div class="search-box autocomplete-wrap">
                <span class="material-icons-round">search</span>
                <input type="text" placeholder="Search users by phone, email, or role..." [(ngModel)]="userSearch" (ngModelChange)="onUserSearchChange()" (focus)="showUserDropdown.set(true)" autocomplete="off">
                @if (showUserDropdown() && userSearchResults().length > 0) {
                  <div class="autocomplete-dropdown">
                    @for (user of userSearchResults(); track user.id) {
                      <button class="autocomplete-option" (mousedown)="selectUser(user); showUserDropdown.set(false)">
                        <div class="ac-user-info">
                          <strong>{{ user.email || user.phone }}</strong>
                          <span class="ac-user-meta">{{ user.system_role }} · {{ user.phone }}</span>
                        </div>
                        <span class="badge" style="font-size:0.625rem;">{{ user.is_active ? 'Active' : 'Disabled' }}</span>
                      </button>
                    }
                  </div>
                }
                @if (showUserDropdown() && userSearch.length >= 2 && userSearchResults().length === 0 && !userSearchLoading()) {
                  <div class="autocomplete-dropdown">
                    <div class="autocomplete-empty">No users found</div>
                  </div>
                }
                @if (userSearchLoading()) {
                  <div class="autocomplete-dropdown">
                    <div class="autocomplete-empty"><span class="spinner-xs"></span> Searching...</div>
                  </div>
                }
              </div>
              <button class="rbac-btn rbac-btn-outline" (click)="loadUsers()">
                <span class="material-icons-round">refresh</span> Refresh
              </button>
            </div>
            <div class="assignment-grid">
              <div class="user-list">
                @for (user of assignmentUsers(); track user.id) {
                  <button [class.active]="selectedUserId() === user.id" (click)="selectUser(user)">
                    <strong>{{ user.email || user.phone }}</strong>
                    <span>{{ user.system_role }}</span>
                  </button>
                } @empty {
                  <div class="empty-row">Search for a user above to manage their roles</div>
                }
              </div>
              <div class="assignment-detail">
                @if (selectedUser()) {
                  <h3>{{ selectedUser()!.email || selectedUser()!.phone }}</h3>
                  <div class="assignment-form">
                    <select [(ngModel)]="assignRoleId">
                      <option value="">Select role</option>
                      @for (role of roles(); track role.id) { <option [value]="role.id">{{ role.name }}</option> }
                    </select>
                    <input type="text" placeholder="Tenant UUID" [(ngModel)]="assignTenantId">
                    <input type="datetime-local" [(ngModel)]="assignExpiresAt">
                    <button class="rbac-btn rbac-btn-primary" (click)="assignSelectedRole()" [disabled]="!assignRoleId">
                      <span class="material-icons-round">person_add</span> Assign
                    </button>
                  </div>
                  <div class="assigned-roles">
                    <h4>Assigned Roles</h4>
                    @for (ur of selectedUserRoles(); track ur.id) {
                      <div class="assigned-role">
                        <span>{{ ur.role?.name || ur.role_id }}</span>
                        <button class="rbac-btn-sm rbac-btn-danger" (click)="revokeAssignedRole(ur.role_id)" title="Revoke">
                          <span class="material-icons-round">remove_circle</span>
                        </button>
                      </div>
                    } @empty {
                      <div class="empty-row">No active roles</div>
                    }
                  </div>
                  <div class="effective-perms">
                    <h4>Effective Permissions</h4>
                    @for (perm of selectedUserPermissions(); track perm) { <span>{{ perm }}</span> } @empty { <em>No permissions loaded</em> }
                  </div>
                } @else {
                  <div class="empty-state"><span class="material-icons-round">manage_accounts</span><h3>Select a user</h3></div>
                }
              </div>
            </div>
          </div>
        }

        @case ('audit') {
          <div class="audit-panel glass-card">
            <div class="rbac-toolbar">
              <div class="audit-page-info" style="font-size:0.8125rem;color:var(--color-text-muted);">
                @if (auditTotal() > 0) {
                  {{ (auditPage() - 1) * auditPerPage() + 1 }}–{{ mathMin(auditPage() * auditPerPage(), auditTotal()) }} of {{ auditTotal() }}
                }
              </div>
              <button class="rbac-btn rbac-btn-outline" (click)="loadAudit()">
                <span class="material-icons-round">refresh</span> Refresh
              </button>
            </div>
            <div class="audit-list">
              @for (log of rbacAuditLogs(); track log.id) {
                <div class="audit-item">
                  <span class="material-icons-round">history</span>
                  <div>
                    <strong>{{ log.action }}</strong>
                    <small>{{ log.resource }} · {{ log.created_at | date:'medium' }}</small>
                  </div>
                </div>
              } @empty {
                <div class="empty-row">No RBAC activity found</div>
              }
            </div>
            @if (auditTotalPages() > 1) {
              <div class="audit-pagination">
                <button class="rbac-btn-sm" [disabled]="auditPage() <= 1" (click)="auditGoToPage(auditPage() - 1)">
                  <span class="material-icons-round">chevron_left</span>
                </button>
                @for (p of auditVisiblePages(); track p) {
                  <button class="rbac-btn-sm" [class.active]="p === auditPage()" (click)="auditGoToPage(p)">{{ p }}</button>
                }
                <button class="rbac-btn-sm" [disabled]="auditPage() >= auditTotalPages()" (click)="auditGoToPage(auditPage() + 1)">
                  <span class="material-icons-round">chevron_right</span>
                </button>
              </div>
            }
          </div>
        }
      }

      <!-- Create Role Dialog -->
      @if (showCreateDialog()) {
        <div class="dialog-overlay" (click)="showCreateDialog.set(false)">
          <div class="dialog glass-card" (click)="$event.stopPropagation()">
            <div class="dialog-header">
              <h2>{{ editingRole() ? 'Edit Role' : 'Create New Role' }}</h2>
              <button class="dialog-close" (click)="showCreateDialog.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <div class="dialog-body">
              <div class="form-group">
                <label>Role Name</label>
                <input type="text" [(ngModel)]="formName" placeholder="e.g. Shift Manager">
              </div>
              <div class="form-group">
                <label>Description</label>
                <textarea [(ngModel)]="formDesc" rows="3" placeholder="Describe the role's purpose..."></textarea>
              </div>
              <div class="form-group">
                <label>Industry Type</label>
                <select [(ngModel)]="formIndustry">
                  <option value="">None (Global)</option>
                  <option value="TRANSPORT">Transport</option>
                  <option value="CONSTRUCTION">Construction</option>
                  <option value="LOGISTICS">Logistics</option>
                  <option value="HEALTH">Health</option>
                  <option value="AGRICULTURE">Agriculture</option>
                  <option value="HOSPITALITY">Hospitality</option>
                </select>
              </div>
            </div>
            <div class="dialog-footer">
              <button class="rbac-btn rbac-btn-outline" (click)="showCreateDialog.set(false)">Cancel</button>
              <button class="rbac-btn rbac-btn-primary" (click)="saveRole()" [disabled]="!formName">
                {{ editingRole() ? 'Update' : 'Create' }}
              </button>
            </div>
          </div>
        </div>
      }

      <!-- Permissions Dialog -->
      @if (showPermDialog()) {
        <div class="dialog-overlay" (click)="showPermDialog.set(false)">
          <div class="dialog dialog-wide glass-card" (click)="$event.stopPropagation()">
            <div class="dialog-header">
              <h2>Permissions — {{ permDialogRole()?.name }}</h2>
              <button class="dialog-close" (click)="showPermDialog.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <div class="dialog-body">
              <div class="search-box" style="margin-bottom:16px">
                <span class="material-icons-round">search</span>
                <input type="text" placeholder="Search permissions..." [(ngModel)]="permSearch">
              </div>
              <div class="perm-modules-list">
                @for (module of allPermModules(); track module) {
                  <div class="perm-module-section">
                    <div class="perm-module-header" (click)="toggleModule(module)">
                      <span class="material-icons-round">{{ expandedModules().has(module) ? 'expand_more' : 'chevron_right' }}</span>
                      <strong>{{ module }}</strong>
                      <span class="perm-module-count">{{ getModulePermCount(module) }}</span>
                    </div>
                    @if (expandedModules().has(module)) {
                      @for (perm of getFilteredPerms(module); track perm.key) {
                        <label class="perm-toggle">
                          <input type="checkbox" [checked]="selectedPermKeys().has(perm.key)"
                                 (change)="togglePerm(perm.key)">
                          <div class="perm-toggle-info">
                            <span class="perm-key">{{ perm.key }}</span>
                            <span class="perm-desc">{{ perm.description }}</span>
                          </div>
                          <span class="risk-dot" [class]="'risk-' + perm.risk_level"></span>
                        </label>
                      }
                    }
                  </div>
                }
              </div>
            </div>
            <div class="dialog-footer">
              <span class="perm-count-label">{{ selectedPermKeys().size }} permissions selected</span>
              <div>
                <button class="rbac-btn rbac-btn-outline" (click)="showPermDialog.set(false)">Cancel</button>
                <button class="rbac-btn rbac-btn-primary" (click)="savePermissions()">
                  <span class="material-icons-round">save</span> Save Permissions
                </button>
              </div>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styleUrls: ['./platform-roles.component.css', './platform-roles.extras.css']
})
export class PlatformRolesComponent implements OnInit {
  private api = inject(RbacApiService);
  private coreApi = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);

  // State
  activeTab = signal<ViewTab>('roles');
  roleView = signal<RoleView>('grid');
  loading = signal(false);
  roles = signal<RBACRole[]>([]);
  filteredRoles = signal<RBACRole[]>([]);
  templates = signal<RoleTemplate[]>([]);
  allPerms = signal<PermissionDef[]>([]);
  allPermsGrouped = signal<Record<string, PermissionDef[]>>({});

  // Matrix
  matrix = signal<PermissionMatrix | null>(null);
  matrixLoading = signal(false);
  matrixChanges = signal<Map<string, Set<string>>>(new Map());
  matrixOriginal = signal<Record<string, string[]>>({});
  expandedMatrixModules = signal<Set<string>>(new Set());
  matrixSearch = '';

  // Filters
  roleSearch = '';
  industryFilter = '';

  // Dialogs
  showCreateDialog = signal(false);
  editingRole = signal<RBACRole | null>(null);
  formName = '';
  formDesc = '';
  formIndustry = '';

  showPermDialog = signal(false);
  permDialogRole = signal<RBACRole | null>(null);
  selectedPermKeys = signal<Set<string>>(new Set());
  permSearch = '';
  expandedModules = signal<Set<string>>(new Set());

  // Templates
  templateTenantId = '';

  // Comparison
  compareRoleAId = '';
  compareRoleBId = '';
  comparison = signal<RoleComparison | null>(null);

  // User assignment
  users = signal<AdminUser[]>([]);
  selectedUserId = signal('');
  selectedUserRoles = signal<UserRoleAssignment[]>([]);
  selectedUserPermissions = signal<string[]>([]);
  userSearch = '';
  assignRoleId = '';
  assignTenantId = '';
  assignExpiresAt = '';
  showUserDropdown = signal(false);
  userSearchResults = signal<AdminUser[]>([]);
  userSearchLoading = signal(false);
  assignmentUsers = signal<AdminUser[]>([]);
  private userSearchDebounce: any;

  // Audit
  auditLogs = signal<AuditLog[]>([]);
  auditPage = signal(1);
  auditPerPage = signal(20);
  auditTotal = signal(0);
  auditTotalPages = signal(1);

  // Computed
  systemRoleCount = computed(() => this.roles().filter(r => r.is_system).length);
  totalPerms = computed(() => this.allPerms().length);
  templateIndustries = computed(() => [...new Set(this.templates().map(t => t.industry_type))]);
  allPermModules = computed(() => Object.keys(this.allPermsGrouped()));
  selectedUser = computed(() => this.assignmentUsers().find(u => u.id === this.selectedUserId()) || null);
  filteredUsers = computed(() => {
    const search = this.userSearch.trim().toLowerCase();
    if (!search) return this.users();
    return this.users().filter(u =>
      (u.email || '').toLowerCase().includes(search) ||
      u.phone.toLowerCase().includes(search) ||
      u.system_role.toLowerCase().includes(search)
    );
  });
  rbacAuditLogs = computed(() => this.auditLogs());
  auditVisiblePages = computed(() => {
    const total = this.auditTotalPages();
    const current = this.auditPage();
    const pages: number[] = [];
    const start = Math.max(1, current - 2);
    const end = Math.min(total, current + 2);
    for (let i = start; i <= end; i++) pages.push(i);
    return pages;
  });
  matrixPermissionRows = computed(() => this.allPerms().filter(perm =>
    this.expandedMatrixModules().has(perm.module) && this.matchesSearch(perm)
  ));

  matrixDirty = computed(() => {
    const changes = this.matrixChanges();
    for (const [, perms] of changes) { if (perms.size > 0) return true; }
    return false;
  });

  matrixChangeCount = computed(() => {
    let count = 0;
    for (const [, perms] of this.matrixChanges()) { count += perms.size; }
    return count;
  });

  ngOnInit(): void {
    const tenantId = this.auth.currentUser()?.organization_id || '';
    this.templateTenantId = tenantId;
    this.assignTenantId = tenantId;
    this.loadRoles();
    this.loadTemplates();
    this.loadPermissions();
  }

  refreshData(): void {
    this.loadRoles();
    this.loadTemplates();
    this.loadPermissions();
  }

  // ── Data Loading ──────────────────────────────────────────────────

  loadRoles(): void {
    this.loading.set(true);
    const params: Record<string, string> = {};
    this.api.listRoles(params).subscribe({
      next: res => { this.roles.set(res.data || []); this.filterRoles(); this.loading.set(false); },
      error: () => this.loading.set(false)
    });
  }

  loadTemplates(): void {
    this.api.listTemplates().subscribe({ next: t => this.templates.set(t || []) });
  }

  loadPermissions(): void {
    this.api.listPermissions().subscribe({
      next: res => { this.allPerms.set(res.data || []); this.allPermsGrouped.set(res.grouped || {}); }
    });
  }

  loadMatrix(): void {
    this.matrixLoading.set(true);
    this.api.getPermissionMatrix().subscribe({
      next: m => {
        this.matrix.set(m);
        this.matrixOriginal.set({ ...m.grants });
        this.matrixChanges.set(new Map());
        this.expandedMatrixModules.set(new Set(m.modules));
        this.matrixLoading.set(false);
      },
      error: () => this.matrixLoading.set(false)
    });
  }

  loadUsers(): void {
    this.coreApi.getUsers({ per_page: '50' }).subscribe({
      next: res => {
        this.users.set(res.data || []);
        this.assignmentUsers.set(res.data || []);
      }
    });
  }

  onUserSearchChange(): void {
    clearTimeout(this.userSearchDebounce);
    if (this.userSearch.trim().length < 2) {
      this.userSearchResults.set([]);
      this.showUserDropdown.set(false);
      return;
    }
    this.userSearchLoading.set(true);
    this.showUserDropdown.set(true);
    this.userSearchDebounce = setTimeout(() => {
      this.coreApi.getUsers({ search: this.userSearch.trim(), per_page: '10' }).subscribe({
        next: res => {
          this.userSearchResults.set(res.data || []);
          this.userSearchLoading.set(false);
        },
        error: () => this.userSearchLoading.set(false),
      });
    }, 300);
  }

  loadAudit(): void {
    this.coreApi.getAuditLogs({ page: String(this.auditPage()), per_page: String(this.auditPerPage()) }).subscribe({
      next: res => {
        this.auditLogs.set(res.data || []);
        this.auditTotal.set(res.meta?.total || 0);
        this.auditTotalPages.set(res.meta?.total_pages || 1);
      }
    });
  }

  auditGoToPage(page: number): void {
    if (page < 1 || page > this.auditTotalPages()) return;
    this.auditPage.set(page);
    this.loadAudit();
  }

  mathMin(a: number, b: number): number { return Math.min(a, b); }

  // ── Filtering ─────────────────────────────────────────────────────

  filterRoles(): void {
    let list = this.roles();
    if (this.roleSearch) {
      const s = this.roleSearch.toLowerCase();
      list = list.filter(r => r.name.toLowerCase().includes(s) || r.slug.toLowerCase().includes(s));
    }
    if (this.industryFilter) {
      list = list.filter(r => r.industry_type === this.industryFilter);
    }
    this.filteredRoles.set(list);
  }

  matchesSearch(perm: PermissionDef): boolean {
    if (!this.matrixSearch) return true;
    const s = this.matrixSearch.toLowerCase();
    return perm.key.toLowerCase().includes(s) || perm.description.toLowerCase().includes(s);
  }

  // ── Role CRUD ─────────────────────────────────────────────────────

  editRole(role: RBACRole): void {
    this.editingRole.set(role);
    this.formName = role.name;
    this.formDesc = role.description;
    this.formIndustry = role.industry_type;
    this.showCreateDialog.set(true);
  }

  saveRole(): void {
    const editing = this.editingRole();
    if (editing) {
      this.api.updateRole(editing.id, { name: this.formName, description: this.formDesc }).subscribe({
        next: () => { this.toast.success('Role updated'); this.showCreateDialog.set(false); this.editingRole.set(null); this.loadRoles(); }
      });
    } else {
      this.api.createRole({ name: this.formName, description: this.formDesc, industry_type: this.formIndustry }).subscribe({
        next: () => { this.toast.success('Role created'); this.showCreateDialog.set(false); this.resetForm(); this.loadRoles(); }
      });
    }
  }

  cloneRoleAction(role: RBACRole): void {
    const name = prompt('Name for cloned role:', `${role.name} (Copy)`);
    if (name) {
      this.api.cloneRole(role.id, name).subscribe({ next: () => { this.toast.success('Role cloned'); this.loadRoles(); } });
    }
  }

  toggleRoleActiveAction(role: RBACRole): void {
    const active = !role.is_active;
    const label = active ? 'restore' : 'archive';
    if (!confirm(`${label.charAt(0).toUpperCase() + label.slice(1)} role "${role.name}"?`)) return;
    this.api.toggleRoleActive(role.id, active).subscribe({
      next: () => { this.toast.success(`Role ${active ? 'restored' : 'archived'}`); this.loadRoles(); }
    });
  }

  deleteRoleAction(role: RBACRole): void {
    if (confirm(`Delete role "${role.name}"? This cannot be undone.`)) {
      this.api.deleteRole(role.id).subscribe({ next: () => { this.toast.success('Role archived'); this.loadRoles(); } });
    }
  }

  // ── Permissions Dialog ────────────────────────────────────────────

  openPermissions(role: RBACRole): void {
    this.permDialogRole.set(role);
    this.permSearch = '';
    this.expandedModules.set(new Set(this.allPermModules()));
    this.api.getRolePermissions(role.id).subscribe({
      next: perms => {
        this.selectedPermKeys.set(new Set(perms.map(p => p.key)));
        this.showPermDialog.set(true);
      }
    });
  }

  togglePerm(key: string): void {
    const current = new Set(this.selectedPermKeys());
    if (current.has(key)) current.delete(key); else current.add(key);
    this.selectedPermKeys.set(current);
  }

  toggleModule(module: string): void {
    const current = new Set(this.expandedModules());
    if (current.has(module)) current.delete(module); else current.add(module);
    this.expandedModules.set(current);
  }

  savePermissions(): void {
    const role = this.permDialogRole();
    if (!role) return;
    const keys = [...this.selectedPermKeys()];
    if (!this.confirmHighRisk(keys, `Save high-risk permission changes for "${role.name}"?`)) return;
    this.api.setRolePermissions(role.id, keys).subscribe({
      next: () => { this.toast.success('Permissions updated'); this.showPermDialog.set(false); this.loadRoles(); }
    });
  }

  getFilteredPerms(module: string): PermissionDef[] {
    const perms = this.allPermsGrouped()[module] || [];
    if (!this.permSearch) return perms;
    const s = this.permSearch.toLowerCase();
    return perms.filter(p => p.key.includes(s) || p.description.toLowerCase().includes(s));
  }

  getModulePermCount(module: string): number {
    return (this.allPermsGrouped()[module] || []).length;
  }

  // ── Matrix ────────────────────────────────────────────────────────

  getModulePerms(module: string): PermissionDef[] {
    return this.allPermsGrouped()[module] || [];
  }

  isGranted(roleId: string, permKey: string): boolean {
    const changes = this.matrixChanges();
    const changed = changes.get(roleId);
    const original = (this.matrixOriginal()[roleId] || []).includes(permKey);
    if (changed?.has(permKey)) return !original;
    return original;
  }

  isModified(roleId: string, permKey: string): boolean {
    return this.matrixChanges().get(roleId)?.has(permKey) || false;
  }

  toggleMatrixPerm(roleId: string, permKey: string): void {
    this.setGrant(roleId, permKey, !this.isGranted(roleId, permKey));
  }

  setGrant(roleId: string, permKey: string, granted: boolean): void {
    if (this.isGranted(roleId, permKey) === granted) return;
    const changes = new Map(this.matrixChanges());
    if (!changes.has(roleId)) changes.set(roleId, new Set());
    const roleChanges = new Set(changes.get(roleId)!);
    if (roleChanges.has(permKey)) roleChanges.delete(permKey); else roleChanges.add(permKey);
    changes.set(roleId, roleChanges);
    this.matrixChanges.set(changes);
  }

  toggleMatrixModule(module: string): void {
    const current = new Set(this.expandedMatrixModules());
    if (current.has(module)) current.delete(module); else current.add(module);
    this.expandedMatrixModules.set(current);
    setTimeout(() => document.getElementById(`module-${module}`)?.scrollIntoView({ block: 'start', behavior: 'smooth' }), 0);
  }

  expandAllMatrixModules(): void {
    this.expandedMatrixModules.set(new Set(this.matrix()?.modules || []));
  }

  collapseAllMatrixModules(): void {
    this.expandedMatrixModules.set(new Set());
  }

  toggleRoleColumn(roleId: string): void {
    const grant = !this.isRoleFullyGranted(roleId);
    for (const perm of this.allPerms()) this.setGrant(roleId, perm.key, grant);
  }

  toggleModuleForAll(module: string): void {
    const grant = !this.isModuleFullyGranted(module);
    const roles = this.matrix()?.roles || [];
    for (const perm of this.getModulePerms(module)) {
      for (const role of roles) this.setGrant(role.id, perm.key, grant);
    }
  }

  togglePermissionRow(permKey: string): void {
    const grant = !this.isPermissionGrantedForAll(permKey);
    for (const role of this.matrix()?.roles || []) this.setGrant(role.id, permKey, grant);
  }

  isRoleFullyGranted(roleId: string): boolean {
    const perms = this.allPerms();
    return perms.length > 0 && perms.every(perm => this.isGranted(roleId, perm.key));
  }

  isModuleFullyGranted(module: string): boolean {
    const roles = this.matrix()?.roles || [];
    const perms = this.getModulePerms(module);
    return roles.length > 0 && perms.length > 0 && perms.every(perm => roles.every(role => this.isGranted(role.id, perm.key)));
  }

  isPermissionGrantedForAll(permKey: string): boolean {
    const roles = this.matrix()?.roles || [];
    return roles.length > 0 && roles.every(role => this.isGranted(role.id, permKey));
  }

  moduleChangeCount(module: string): number {
    const moduleKeys = new Set(this.getModulePerms(module).map(p => p.key));
    let count = 0;
    for (const [, changed] of this.matrixChanges()) {
      for (const key of changed) if (moduleKeys.has(key)) count++;
    }
    return count;
  }

  permissionRowIndex(permKey: string): number {
    return this.allPerms().filter(perm => this.matchesSearch(perm)).findIndex(perm => perm.key === permKey);
  }

  onMatrixKeydown(event: KeyboardEvent, roleId: string, permKey: string): void {
    if (event.key === ' ' || event.key === 'Enter') {
      event.preventDefault();
      this.toggleMatrixPerm(roleId, permKey);
      return;
    }
    const current = event.currentTarget as HTMLElement;
    const row = Number(current.dataset['row']);
    const col = Number(current.dataset['col']);
    const next = {
      ArrowDown: [row + 1, col],
      ArrowUp: [row - 1, col],
      ArrowRight: [row, col + 1],
      ArrowLeft: [row, col - 1],
    }[event.key] as [number, number] | undefined;
    if (!next) return;
    event.preventDefault();
    document.querySelector<HTMLElement>(`[data-row="${next[0]}"][data-col="${next[1]}"]`)?.focus();
  }

  discardMatrixChanges(): void {
    this.matrixChanges.set(new Map());
  }

  saveMatrixChanges(): void {
    const changes = this.matrixChanges();
    const original = this.matrixOriginal();
    const saves = [];

    for (const [roleId, changedPerms] of changes) {
      if (changedPerms.size === 0) continue;
      const origSet = new Set(original[roleId] || []);
      for (const key of changedPerms) {
        if (origSet.has(key)) origSet.delete(key); else origSet.add(key);
      }
      const keys = [...origSet];
      if (!this.confirmHighRisk(keys, 'Save high-risk permission matrix changes?')) return;
      saves.push(this.api.setRolePermissions(roleId, keys));
    }

    if (saves.length === 0) return;
    forkJoin(saves).subscribe({
      next: () => { this.toast.success('Permission matrix saved'); this.loadMatrix(); this.loadRoles(); }
    });
  }

  // ── Templates ─────────────────────────────────────────────────────

  getIndustryTemplates(industry: string): RoleTemplate[] {
    return this.templates().filter(t => t.industry_type === industry);
  }

  getIndustryIcon(industry: string): string {
    const map: Record<string, string> = {
      TRANSPORT: 'directions_bus', CONSTRUCTION: 'engineering', LOGISTICS: 'local_shipping',
      HEALTH: 'health_and_safety', AGRICULTURE: 'agriculture', HOSPITALITY: 'restaurant',
      PLATFORM: 'cloud', FINANCIAL: 'account_balance'
    };
    return map[industry] || 'business';
  }

  applyTemplateAction(tmpl: RoleTemplate): void {
    let tenantId = this.templateTenantId.trim() || this.auth.currentUser()?.organization_id || '';
    if (!tenantId) tenantId = prompt('Target tenant UUID:') || '';
    if (!tenantId) {
      this.toast.warning('Target tenant is required');
      return;
    }
    if (!this.confirmHighRisk(tmpl.permissions, `Apply template "${tmpl.role_name}" to this tenant?`)) return;
    this.api.applyTemplate(tmpl.id, tenantId).subscribe({
      next: () => { this.toast.success('Template applied'); this.loadRoles(); this.activeTab.set('roles'); }
    });
  }

  // ── Role Comparison ───────────────────────────────────────────────────

  compareRoles(): void {
    if (!this.compareRoleAId || !this.compareRoleBId || this.compareRoleAId === this.compareRoleBId) return;
    this.api.compareRoles(this.compareRoleAId, this.compareRoleBId).subscribe({
      next: comparison => this.comparison.set(comparison)
    });
  }

  roleName(roleId: string): string {
    return this.roles().find(role => role.id === roleId)?.name || 'Role';
  }

  // ── User Assignment ───────────────────────────────────────────────────

  selectUser(user: AdminUser): void {
    this.selectedUserId.set(user.id);
    this.assignTenantId = user.organization_id || this.auth.currentUser()?.organization_id || '';
    this.assignRoleId = '';
    // Add to assignmentUsers if not already there
    const existing = this.assignmentUsers();
    if (!existing.find(u => u.id === user.id)) {
      this.assignmentUsers.set([user, ...existing]);
    }
    this.loadSelectedUserAccess();
  }

  loadSelectedUserAccess(): void {
    const user = this.selectedUser();
    if (!user) return;
    const tenantId = this.assignTenantId || user.organization_id;
    this.api.getUserRoles(user.id, tenantId).subscribe({
      next: roles => this.selectedUserRoles.set(roles || [])
    });
    this.api.getUserPermissions(user.id, tenantId).subscribe({
      next: perms => this.selectedUserPermissions.set(perms || [])
    });
  }

  assignSelectedRole(): void {
    const user = this.selectedUser();
    if (!user || !this.assignRoleId) return;
    const role = this.roles().find(r => r.id === this.assignRoleId);
    if (role && !this.confirmHighRiskForRole(role, `Assign role "${role.name}" to ${user.email || user.phone}?`)) return;
    const expiresAt = this.assignExpiresAt ? new Date(this.assignExpiresAt).toISOString() : undefined;
    this.api.assignRoleToUser(user.id, this.assignRoleId, this.assignTenantId || user.organization_id, expiresAt).subscribe({
      next: () => {
        this.toast.success('Role assigned');
        this.loadSelectedUserAccess();
        this.loadRoles();
      }
    });
  }

  revokeAssignedRole(roleId: string): void {
    const user = this.selectedUser();
    if (!user) return;
    const role = this.roles().find(r => r.id === roleId);
    if (!confirm(`Revoke "${role?.name || roleId}" from ${user.email || user.phone}?`)) return;
    this.api.revokeRoleFromUser(user.id, roleId, this.assignTenantId || user.organization_id).subscribe({
      next: () => {
        this.toast.success('Role revoked');
        this.loadSelectedUserAccess();
        this.loadRoles();
      }
    });
  }

  getRiskColor(role: RBACRole): string {
    if (role.is_system) return 'purple';
    if (role.is_template) return 'blue';
    return 'teal';
  }

  private confirmHighRiskForRole(role: RBACRole, message: string): boolean {
    const keys = this.matrix()?.grants[role.id] || [];
    return this.confirmHighRisk(keys, message);
  }

  private confirmHighRisk(keys: string[], message: string): boolean {
    const highRisk = this.highRiskPermissions(keys);
    if (highRisk.length === 0) return true;
    return confirm(`${message}\n\nHigh-risk permissions: ${highRisk.slice(0, 8).join(', ')}${highRisk.length > 8 ? '...' : ''}`);
  }

  private highRiskPermissions(keys: string[]): string[] {
    const lookup = new Map(this.allPerms().map(perm => [perm.key, perm]));
    return keys.filter(key => {
      const risk = lookup.get(key)?.risk_level;
      return risk === 'high' || risk === 'critical';
    });
  }

  private resetForm(): void {
    this.formName = ''; this.formDesc = ''; this.formIndustry = '';
    this.editingRole.set(null);
  }
}
