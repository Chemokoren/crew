import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, Router, NavigationEnd } from '@angular/router';
import { filter } from 'rxjs';
import { AuthService } from './core/services/auth.service';
import { OrgContextService } from './core/services/org-context.service';
import { SidebarComponent } from './shared/components/sidebar/sidebar.component';
import { TopbarComponent } from './shared/components/topbar/topbar.component';
import { PlatformSidebarComponent } from './shared/components/platform-sidebar/platform-sidebar.component';
import { PlatformTopbarComponent } from './shared/components/platform-topbar/platform-topbar.component';
import { ToastComponent } from './shared/components/toast/toast.component';
import { ConfirmDialogComponent } from './shared/components/confirm-dialog/confirm-dialog.component';
import { AnnouncementBannerComponent } from './shared/components/announcement-banner.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    CommonModule, RouterOutlet,
    SidebarComponent, TopbarComponent,
    PlatformSidebarComponent, PlatformTopbarComponent,
    ToastComponent, ConfirmDialogComponent, AnnouncementBannerComponent,
  ],
  template: `
    @if (isMaintenanceRoute()) {
      <router-outlet />
    } @else if (isAuthRoute()) {
      <router-outlet />
    } @else if (isPlatformRoute()) {
      <div class="app-layout platform-layout" [class.sidebar-collapsed]="sidebarCollapsed()">
        <app-platform-sidebar [(mobileOpen)]="sidebarMobileOpen" [(collapsed)]="sidebarCollapsed" />
        <app-platform-topbar (menuToggle)="toggleMobileSidebar()" />
        <main class="main-content" id="main-content">
          <app-announcement-banner />
          <router-outlet />
        </main>
      </div>
    } @else {
      <div class="app-layout" [class.sidebar-collapsed]="sidebarCollapsed()">
        <app-sidebar [(mobileOpen)]="sidebarMobileOpen" [(collapsed)]="sidebarCollapsed" />
        <app-topbar (menuToggle)="toggleMobileSidebar()" />
        <main class="main-content" id="main-content">
          <app-announcement-banner />
          <router-outlet />
        </main>
      </div>
    }
    <app-toast />
    <app-confirm-dialog />
  `,
  styles: [`
    .app-layout {
      min-height: 100vh;
    }

    .main-content {
      margin-left: var(--sidebar-width);
      margin-top: var(--topbar-height);
      padding: var(--space-xl);
      min-height: calc(100vh - var(--topbar-height));
      transition: margin-left var(--transition-base);
      animation: fadeIn 300ms ease-out;
    }

    .sidebar-collapsed .main-content {
      margin-left: var(--sidebar-collapsed-width);
    }

    :host ::ng-deep .sidebar-collapsed .topbar,
    :host ::ng-deep .sidebar-collapsed .platform-topbar {
      left: var(--sidebar-collapsed-width) !important;
    }

    @media (max-width: 768px) {
      .main-content {
        margin-left: 0 !important;
        padding: var(--space-md);
      }
    }
  `]
})
export class AppComponent implements OnInit {
  private router = inject(Router);
  private auth = inject(AuthService);
  private orgCtx = inject(OrgContextService);

  sidebarMobileOpen = signal(false);
  sidebarCollapsed = signal(false);
  private currentUrl = '';

  constructor() {
    // Close mobile sidebar on route change
    this.router.events.pipe(
      filter(event => event instanceof NavigationEnd)
    ).subscribe((event) => {
      this.currentUrl = (event as NavigationEnd).urlAfterRedirects || (event as NavigationEnd).url;
      this.sidebarMobileOpen.set(false);
    });
  }

  ngOnInit(): void {
    // Refresh user profile from server on app init to keep stale localStorage in sync.
    // Only fires if the user has a token (i.e., was previously authenticated).
    // Skip on maintenance/system-admin pages to avoid 503 redirect loops.
    const path = window.location.pathname;
    if (this.auth.accessToken && !path.startsWith('/maintenance') && !path.startsWith('/system-admin')) {
      this.auth.fetchProfile().subscribe({
        next: () => this.orgCtx.load(),
        error: () => { /* handled by interceptor */ },
      });
    }
  }

  isAuthRoute(): boolean {
    const url = this.currentUrl || this.router.url;
    return url.startsWith('/auth') || url.startsWith('/system-admin');
  }

  isMaintenanceRoute(): boolean {
    const url = this.currentUrl || this.router.url;
    return url.startsWith('/maintenance');
  }

  isPlatformRoute(): boolean {
    const url = this.currentUrl || this.router.url;
    return url.startsWith('/platform');
  }

  toggleMobileSidebar(): void {
    this.sidebarMobileOpen.update(v => !v);
  }
}
