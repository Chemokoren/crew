import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, Router, NavigationEnd } from '@angular/router';
import { filter } from 'rxjs';
import { AuthService } from './core/services/auth.service';
import { SidebarComponent } from './shared/components/sidebar/sidebar.component';
import { TopbarComponent } from './shared/components/topbar/topbar.component';
import { ToastComponent } from './shared/components/toast/toast.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet, SidebarComponent, TopbarComponent, ToastComponent],
  template: `
    @if (isAuthRoute()) {
      <router-outlet />
    } @else {
      <div class="app-layout">
        <app-sidebar [(mobileOpen)]="sidebarMobileOpen" />
        <app-topbar (menuToggle)="toggleMobileSidebar()" />
        <main class="main-content">
          <router-outlet />
        </main>
      </div>
    }
    <app-toast />
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
        // If the token is invalid/expired, the error interceptor handles 401 → logout.
        error: () => { /* handled by interceptor */ },
      });
    }
  }

  isAuthRoute(): boolean {
    const url = this.currentUrl || this.router.url;
    return url.startsWith('/auth');
  }

  toggleMobileSidebar(): void {
    this.sidebarMobileOpen.update(v => !v);
  }
}
