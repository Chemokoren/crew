import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RbacApiService } from './services/rbac-api.service';
import { RBACRole, PermissionDef, RoleTemplate, PermissionMatrix } from '../../../core/models';

type ViewTab = 'roles' | 'matrix' | 'templates';

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
          </div>

          @if (loading()) {
            <div class="rbac-loading"><div class="spinner"></div><span>Loading roles...</span></div>
          } @else {
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
          }
        }

        @case ('matrix') {
          <!-- Permission Matrix -->
          <div class="rbac-toolbar">
            <div class="search-box">
              <span class="material-icons-round">search</span>
              <input type="text" placeholder="Search permissions..." [(ngModel)]="matrixSearch">
            </div>
          </div>

          @if (matrixLoading()) {
            <div class="rbac-loading"><div class="spinner"></div><span>Building matrix...</span></div>
          } @else if (matrix()) {
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
                          </div>
                        </th>
                      }
                    </tr>
                  </thead>
                  <tbody>
                    @for (module of matrix()!.modules; track module) {
                      <tr class="matrix-module-row">
                        <td [attr.colspan]="matrix()!.roles.length + 1" class="matrix-module-cell">
                          <span class="material-icons-round">folder</span> {{ module }}
                        </td>
                      </tr>
                      @for (perm of getModulePerms(module); track perm.key) {
                        @if (matchesSearch(perm)) {
                          <tr class="matrix-perm-row">
                            <td class="matrix-sticky-col matrix-perm-cell">
                              <div class="perm-info">
                                <span class="perm-key">{{ perm.key }}</span>
                                <span class="perm-desc">{{ perm.description }}</span>
                              </div>
                              <span class="risk-dot" [class]="'risk-' + perm.risk_level" [title]="perm.risk_level"></span>
                            </td>
                            @for (role of matrix()!.roles; track role.id) {
                              <td class="matrix-cell" (click)="toggleMatrixPerm(role.id, perm.key)"
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
          }
        }

        @case ('templates') {
          <!-- Templates -->
          @if (loading()) {
            <div class="rbac-loading"><div class="spinner"></div><span>Loading templates...</span></div>
          } @else {
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
                          {{ tmpl.permissions?.length || 0 }} permissions
                        </div>
                      </div>
                    }
                  </div>
                </div>
              }
            </div>
          }
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
  styleUrl: './platform-roles.component.css'
})
export class PlatformRolesComponent implements OnInit {
  private api = inject(RbacApiService);

  // State
  activeTab = signal<ViewTab>('roles');
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

  // Computed
  systemRoleCount = computed(() => this.roles().filter(r => r.is_system).length);
  totalPerms = computed(() => this.allPerms().length);
  templateIndustries = computed(() => [...new Set(this.templates().map(t => t.industry_type))]);
  allPermModules = computed(() => Object.keys(this.allPermsGrouped()));

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
        this.matrixLoading.set(false);
      },
      error: () => this.matrixLoading.set(false)
    });
  }

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
        next: () => { this.showCreateDialog.set(false); this.editingRole.set(null); this.loadRoles(); }
      });
    } else {
      this.api.createRole({ name: this.formName, description: this.formDesc, industry_type: this.formIndustry }).subscribe({
        next: () => { this.showCreateDialog.set(false); this.resetForm(); this.loadRoles(); }
      });
    }
  }

  cloneRoleAction(role: RBACRole): void {
    const name = prompt('Name for cloned role:', `${role.name} (Copy)`);
    if (name) {
      this.api.cloneRole(role.id, name).subscribe({ next: () => this.loadRoles() });
    }
  }

  deleteRoleAction(role: RBACRole): void {
    if (confirm(`Delete role "${role.name}"? This cannot be undone.`)) {
      this.api.deleteRole(role.id).subscribe({ next: () => this.loadRoles() });
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
    this.api.setRolePermissions(role.id, [...this.selectedPermKeys()]).subscribe({
      next: () => { this.showPermDialog.set(false); this.loadRoles(); }
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
    const changes = new Map(this.matrixChanges());
    if (!changes.has(roleId)) changes.set(roleId, new Set());
    const roleChanges = new Set(changes.get(roleId)!);
    if (roleChanges.has(permKey)) roleChanges.delete(permKey); else roleChanges.add(permKey);
    changes.set(roleId, roleChanges);
    this.matrixChanges.set(changes);
  }

  discardMatrixChanges(): void {
    this.matrixChanges.set(new Map());
  }

  saveMatrixChanges(): void {
    const changes = this.matrixChanges();
    const original = this.matrixOriginal();
    const saves: Promise<void>[] = [];

    for (const [roleId, changedPerms] of changes) {
      if (changedPerms.size === 0) continue;
      const origSet = new Set(original[roleId] || []);
      for (const key of changedPerms) {
        if (origSet.has(key)) origSet.delete(key); else origSet.add(key);
      }
      saves.push(new Promise<void>((resolve, reject) => {
        this.api.setRolePermissions(roleId, [...origSet]).subscribe({ next: () => resolve(), error: reject });
      }));
    }

    Promise.all(saves).then(() => { this.loadMatrix(); this.loadRoles(); });
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

  getRiskColor(role: RBACRole): string {
    if (role.is_system) return 'purple';
    if (role.is_template) return 'blue';
    return 'teal';
  }

  private resetForm(): void {
    this.formName = ''; this.formDesc = ''; this.formIndustry = '';
    this.editingRole.set(null);
  }
}
