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
    @if (isAuthRoute()) {
      <router-outlet />
    } @else if (isPlatformRoute()) {
      <div class="app-layout platform-layout">
        <app-platform-sidebar [(mobileOpen)]="sidebarMobileOpen" />
        <app-platform-topbar (menuToggle)="toggleMobileSidebar()" />
        <app-announcement-banner />
        <main class="main-content" id="main-content">
          <router-outlet />
        </main>
      </div>
    } @else {
      <div class="app-layout">
        <app-sidebar [(mobileOpen)]="sidebarMobileOpen" />
        <app-topbar (menuToggle)="toggleMobileSidebar()" />
        <app-announcement-banner />
        <main class="main-content" id="main-content">
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

    @media (max-width: 768px) {
      .main-content {
        margin-left: 0;
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
    if (this.auth.accessToken) {
      this.auth.fetchProfile().subscribe({
        next: () => this.orgCtx.load(),
        error: () => { /* handled by interceptor */ },
      });
    }
  }

  isAuthRoute(): boolean {
    const url = this.currentUrl || this.router.url;
    return url.startsWith('/auth');
  }

  isPlatformRoute(): boolean {
    const url = this.currentUrl || this.router.url;
    return url.startsWith('/platform');
  }

  toggleMobileSidebar(): void {
    this.sidebarMobileOpen.update(v => !v);
  }
}
