import { Component, inject, signal, ChangeDetectionStrategy, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { ToastService } from '../../core/services/toast.service';

@Component({
  selector: 'app-maintenance',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="maintenance-page">
      <!-- Animated background -->
      <div class="bg-grid"></div>
      <div class="bg-glow bg-glow-1"></div>
      <div class="bg-glow bg-glow-2"></div>
      <div class="bg-glow bg-glow-3"></div>

      <!-- Floating particles -->
      <div class="particles">
        @for (p of particles; track p) {
          <div class="particle" [style.--delay]="p.delay" [style.--x]="p.x" [style.--size]="p.size"></div>
        }
      </div>

      <div class="maintenance-container">
        <!-- Logo & Branding -->
        <div class="brand-section" (click)="onLogoClick()">
          <div class="logo-ring">
            <div class="logo-ring-inner">
              <img src="/logo.png" alt="AMY MIS" class="logo-img" />
            </div>
            <svg class="ring-spinner" viewBox="0 0 100 100">
              <circle cx="50" cy="50" r="46" fill="none" stroke="url(#ring-grad)" stroke-width="2"
                      stroke-dasharray="80 210" stroke-linecap="round" />
              <defs>
                <linearGradient id="ring-grad" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stop-color="#00d2ff" />
                  <stop offset="100%" stop-color="#7b61ff" />
                </linearGradient>
              </defs>
            </svg>
          </div>
        </div>

        <!-- Main Content -->
        <div class="content-section">
          <div class="status-badge">
            <span class="pulse-dot"></span>
            <span>System Maintenance</span>
          </div>

          <h1 class="hero-title">
            We'll Be Back<br/>
            <span class="gradient-text">Shortly</span>
          </h1>

          <p class="hero-desc">
            {{ maintenanceMessage() }}
          </p>

          <!-- Gear Animation -->
          <div class="gear-container">
            <div class="gear gear-1">
              <span class="material-icons-round">settings</span>
            </div>
            <div class="gear gear-2">
              <span class="material-icons-round">settings</span>
            </div>
            <div class="gear gear-3">
              <span class="material-icons-round">build</span>
            </div>
          </div>

          <p class="sub-desc">
            Our engineering team is performing scheduled maintenance to improve
            system performance, security, and reliability. All services will be
            restored as soon as possible.
          </p>
        </div>

        <!-- Contact Cards -->
        <div class="contact-section">
          <h3 class="contact-title">
            <span class="material-icons-round">support_agent</span>
            Need urgent assistance?
          </h3>
          <div class="contact-grid">
            <a href="tel:+254780058775" class="contact-card">
              <div class="contact-icon phone-icon">
                <span class="material-icons-round">call</span>
              </div>
              <div class="contact-info">
                <span class="contact-label">Call Us</span>
                <span class="contact-value">+254 780 058 775</span>
              </div>
            </a>
            <a href="mailto:support@amymis.co.ke" class="contact-card">
              <div class="contact-icon email-icon">
                <span class="material-icons-round">email</span>
              </div>
              <div class="contact-info">
                <span class="contact-label">Email Support</span>
                <span class="contact-value">support&#64;amymis.co.ke</span>
              </div>
            </a>
            <a href="https://wa.me/254780058775" target="_blank" class="contact-card">
              <div class="contact-icon whatsapp-icon">
                <span class="material-icons-round">chat</span>
              </div>
              <div class="contact-info">
                <span class="contact-label">WhatsApp</span>
                <span class="contact-value">+254 780 058 775</span>
              </div>
            </a>
          </div>
        </div>

        <!-- Footer -->
        <div class="maintenance-footer">
          <p>&copy; {{ currentYear }} AMY MIS — Workforce Financial Operating System</p>
        </div>
      </div>

      <!-- Hidden admin login panel -->
      @if (showAdminLogin()) {
        <div class="admin-overlay" (click)="showAdminLogin.set(false)">
          <div class="admin-panel glass-panel" (click)="$event.stopPropagation()">
            <div class="admin-header">
              <span class="material-icons-round admin-shield">admin_panel_settings</span>
              <h3>System Administrator Access</h3>
              <p>This login is restricted to authorized system administrators only.</p>
            </div>
            <form (ngSubmit)="adminLogin()" class="admin-form">
              <div class="admin-field">
                <label for="admin-phone">Phone Number</label>
                <input id="admin-phone" type="tel" [(ngModel)]="adminPhone" name="phone"
                       placeholder="+254..." autocomplete="tel" />
              </div>
              <div class="admin-field">
                <label for="admin-password">Password</label>
                <input id="admin-password" type="password" [(ngModel)]="adminPassword" name="password"
                       placeholder="••••••••" autocomplete="current-password" />
              </div>
              <button type="submit" class="admin-login-btn" [disabled]="adminLoading()">
                @if (adminLoading()) {
                  <span class="spinner-sm"></span> Authenticating...
                } @else {
                  <span class="material-icons-round">lock_open</span> Sign In
                }
              </button>
            </form>
            <button class="admin-close" (click)="showAdminLogin.set(false)">
              <span class="material-icons-round">close</span>
            </button>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .maintenance-page {
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      background: #0a0e1a;
      position: relative;
      overflow: hidden;
      color: #e2e8f0;
      font-family: var(--font-body, 'Inter', sans-serif);
    }

    /* Animated grid background */
    .bg-grid {
      position: absolute;
      inset: 0;
      background-image:
        linear-gradient(rgba(0, 210, 255, 0.03) 1px, transparent 1px),
        linear-gradient(90deg, rgba(0, 210, 255, 0.03) 1px, transparent 1px);
      background-size: 60px 60px;
      animation: gridShift 20s linear infinite;
    }
    @keyframes gridShift {
      to { transform: translate(60px, 60px); }
    }

    /* Glowing orbs */
    .bg-glow {
      position: absolute;
      border-radius: 50%;
      filter: blur(100px);
      opacity: 0.4;
      animation: glowFloat 8s ease-in-out infinite alternate;
    }
    .bg-glow-1 {
      width: 400px; height: 400px;
      background: radial-gradient(circle, rgba(0, 210, 255, 0.3), transparent);
      top: -100px; left: -100px;
    }
    .bg-glow-2 {
      width: 300px; height: 300px;
      background: radial-gradient(circle, rgba(123, 97, 255, 0.25), transparent);
      bottom: -50px; right: -50px;
      animation-delay: -4s;
    }
    .bg-glow-3 {
      width: 250px; height: 250px;
      background: radial-gradient(circle, rgba(0, 210, 255, 0.15), transparent);
      top: 50%; left: 50%;
      transform: translate(-50%, -50%);
      animation-delay: -2s;
    }
    @keyframes glowFloat {
      from { transform: translate(0, 0) scale(1); }
      to { transform: translate(20px, -20px) scale(1.1); }
    }

    /* Floating particles */
    .particles { position: absolute; inset: 0; pointer-events: none; }
    .particle {
      position: absolute;
      width: calc(var(--size, 3) * 1px);
      height: calc(var(--size, 3) * 1px);
      background: rgba(0, 210, 255, 0.5);
      border-radius: 50%;
      left: calc(var(--x, 50) * 1%);
      bottom: -10px;
      animation: particleRise 12s linear infinite;
      animation-delay: calc(var(--delay, 0) * 1s);
    }
    @keyframes particleRise {
      to { transform: translateY(-100vh); opacity: 0; }
    }

    /* Container */
    .maintenance-container {
      position: relative;
      z-index: 1;
      width: 100%;
      max-width: 640px;
      padding: 40px 24px;
      text-align: center;
      animation: containerIn 0.8s ease-out;
    }
    @keyframes containerIn {
      from { opacity: 0; transform: translateY(30px); }
      to { opacity: 1; transform: translateY(0); }
    }

    /* Logo */
    .brand-section {
      margin-bottom: 32px;
      cursor: pointer;
      -webkit-user-select: none;
      user-select: none;
    }
    .logo-ring {
      width: 120px; height: 120px;
      margin: 0 auto;
      position: relative;
    }
    .logo-ring-inner {
      width: 100px; height: 100px;
      position: absolute;
      top: 10px; left: 10px;
      border-radius: 50%;
      background: rgba(255, 255, 255, 0.04);
      border: 1px solid rgba(255, 255, 255, 0.08);
      display: flex; align-items: center; justify-content: center;
      overflow: hidden;
    }
    .logo-img { width: 64px; height: 64px; object-fit: contain; }
    .ring-spinner {
      position: absolute; inset: 0;
      width: 100%; height: 100%;
      animation: ringRotate 4s linear infinite;
    }
    @keyframes ringRotate { to { transform: rotate(360deg); } }

    /* Status badge */
    .status-badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 6px 16px;
      border-radius: 100px;
      background: rgba(239, 68, 68, 0.12);
      border: 1px solid rgba(239, 68, 68, 0.25);
      color: #f87171;
      font-size: 0.8125rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      margin-bottom: 24px;
    }
    .pulse-dot {
      width: 8px; height: 8px;
      background: #ef4444;
      border-radius: 50%;
      animation: pulse 2s ease-in-out infinite;
    }
    @keyframes pulse {
      0%, 100% { opacity: 1; transform: scale(1); }
      50% { opacity: 0.5; transform: scale(1.4); }
    }

    /* Hero */
    .hero-title {
      font-size: 2.5rem;
      font-weight: 800;
      line-height: 1.1;
      margin-bottom: 16px;
      font-family: var(--font-heading, 'Inter', sans-serif);
      color: #f1f5f9;
    }
    .gradient-text {
      background: linear-gradient(135deg, #00d2ff, #7b61ff, #ff6b9d);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }
    .hero-desc {
      font-size: 1.0625rem;
      color: #94a3b8;
      line-height: 1.6;
      max-width: 480px;
      margin: 0 auto 28px;
    }
    .sub-desc {
      font-size: 0.875rem;
      color: #64748b;
      line-height: 1.6;
      max-width: 480px;
      margin: 0 auto 40px;
    }

    /* Gear animation */
    .gear-container {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0;
      margin: 28px auto 20px;
      height: 70px;
    }
    .gear {
      display: flex; align-items: center; justify-content: center;
      .material-icons-round { color: rgba(0, 210, 255, 0.35); }
    }
    .gear-1 {
      animation: gearCW 4s linear infinite;
      .material-icons-round { font-size: 48px; }
    }
    .gear-2 {
      animation: gearCCW 4s linear infinite;
      margin-left: -6px;
      .material-icons-round { font-size: 36px; color: rgba(123, 97, 255, 0.35); }
    }
    .gear-3 {
      animation: gearCW 3s linear infinite;
      margin-left: -4px;
      .material-icons-round { font-size: 28px; color: rgba(255, 107, 157, 0.3); }
    }
    @keyframes gearCW { to { transform: rotate(360deg); } }
    @keyframes gearCCW { to { transform: rotate(-360deg); } }

    /* Contact section */
    .contact-section {
      background: rgba(255, 255, 255, 0.03);
      border: 1px solid rgba(255, 255, 255, 0.06);
      border-radius: 16px;
      padding: 28px 24px;
      margin-bottom: 32px;
    }
    .contact-title {
      display: flex; align-items: center; justify-content: center;
      gap: 8px;
      font-size: 1rem; font-weight: 600;
      color: #cbd5e1;
      margin-bottom: 20px;
      .material-icons-round { font-size: 22px; color: #00d2ff; }
    }
    .contact-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
      gap: 12px;
    }
    .contact-card {
      display: flex; align-items: center; gap: 12px;
      padding: 14px 16px;
      border-radius: 12px;
      background: rgba(255, 255, 255, 0.03);
      border: 1px solid rgba(255, 255, 255, 0.06);
      text-decoration: none; color: inherit;
      transition: all 0.25s ease;
      &:hover {
        background: rgba(0, 210, 255, 0.06);
        border-color: rgba(0, 210, 255, 0.15);
        transform: translateY(-2px);
      }
    }
    .contact-icon {
      width: 40px; height: 40px;
      border-radius: 10px;
      display: flex; align-items: center; justify-content: center;
      flex-shrink: 0;
      .material-icons-round { font-size: 20px; }
    }
    .phone-icon { background: rgba(0, 210, 255, 0.1); .material-icons-round { color: #00d2ff; } }
    .email-icon { background: rgba(123, 97, 255, 0.1); .material-icons-round { color: #7b61ff; } }
    .whatsapp-icon { background: rgba(37, 211, 102, 0.1); .material-icons-round { color: #25d366; } }
    .contact-info { display: flex; flex-direction: column; text-align: left; }
    .contact-label { font-size: 0.6875rem; color: #64748b; text-transform: uppercase; letter-spacing: 0.5px; }
    .contact-value { font-size: 0.8125rem; font-weight: 600; color: #e2e8f0; }

    /* Footer */
    .maintenance-footer {
      font-size: 0.75rem;
      color: #475569;
      p { margin: 0; }
    }

    /* Admin overlay */
    .admin-overlay {
      position: fixed; inset: 0; z-index: 1000;
      background: rgba(0, 0, 0, 0.7);
      backdrop-filter: blur(8px);
      display: flex; align-items: center; justify-content: center;
      animation: fadeIn 0.2s ease;
    }
    @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }

    .glass-panel {
      background: rgba(15, 23, 42, 0.95);
      border: 1px solid rgba(255, 255, 255, 0.08);
      border-radius: 20px;
      padding: 36px 32px;
      width: 100%;
      max-width: 400px;
      position: relative;
      animation: panelIn 0.3s ease-out;
    }
    @keyframes panelIn {
      from { opacity: 0; transform: scale(0.95) translateY(10px); }
      to { opacity: 1; transform: scale(1) translateY(0); }
    }

    .admin-header {
      text-align: center;
      margin-bottom: 28px;
      h3 { font-size: 1.125rem; font-weight: 700; color: #f1f5f9; margin: 12px 0 6px; }
      p { font-size: 0.8125rem; color: #64748b; margin: 0; }
    }
    .admin-shield {
      font-size: 40px;
      color: #f59e0b;
    }

    .admin-form {
      display: flex; flex-direction: column; gap: 16px;
    }
    .admin-field {
      display: flex; flex-direction: column; gap: 6px;
      label {
        font-size: 0.8125rem; font-weight: 500; color: #94a3b8;
      }
      input {
        background: rgba(255, 255, 255, 0.04);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 10px;
        padding: 12px 14px;
        font-size: 0.9375rem;
        color: #e2e8f0;
        outline: none;
        transition: border-color 0.2s;
        &:focus {
          border-color: rgba(0, 210, 255, 0.4);
          box-shadow: 0 0 0 3px rgba(0, 210, 255, 0.08);
        }
        &::placeholder { color: #475569; }
      }
    }

    .admin-login-btn {
      display: flex; align-items: center; justify-content: center; gap: 8px;
      width: 100%;
      padding: 13px;
      border: none;
      border-radius: 10px;
      background: linear-gradient(135deg, #f59e0b, #d97706);
      color: #0a0e1a;
      font-size: 0.9375rem;
      font-weight: 700;
      cursor: pointer;
      margin-top: 4px;
      transition: all 0.2s;
      .material-icons-round { font-size: 18px; }
      &:hover:not(:disabled) { filter: brightness(1.1); transform: translateY(-1px); }
      &:disabled { opacity: 0.6; cursor: not-allowed; }
    }

    .spinner-sm {
      display: inline-block; width: 16px; height: 16px;
      border: 2px solid rgba(0, 0, 0, 0.2); border-top-color: currentColor;
      border-radius: 50%; animation: spin 600ms linear infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }

    .admin-close {
      position: absolute; top: 12px; right: 12px;
      background: none; border: none;
      color: #64748b; cursor: pointer; padding: 4px;
      border-radius: 8px;
      transition: all 0.2s;
      &:hover { color: #e2e8f0; background: rgba(255,255,255,0.06); }
    }

    @media (max-width: 640px) {
      .hero-title { font-size: 1.875rem; }
      .contact-grid { grid-template-columns: 1fr; }
      .glass-panel { margin: 0 16px; padding: 28px 20px; }
    }
  `]
})
export class MaintenanceComponent implements OnInit, OnDestroy {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private router = inject(Router);
  private toast = inject(ToastService);

  maintenanceMessage = signal('Our system is currently undergoing scheduled maintenance to improve your experience. We apologize for the inconvenience and appreciate your patience.');
  showAdminLogin = signal(false);
  adminLoading = signal(false);
  adminPhone = '';
  adminPassword = '';
  currentYear = new Date().getFullYear();

  // Triple-click counter for hidden admin login
  private clickCount = 0;
  private clickTimer: any;

  // Floating particles config
  particles = Array.from({ length: 15 }, (_, i) => ({
    delay: (Math.random() * 10).toFixed(1),
    x: (Math.random() * 100).toFixed(0),
    size: (2 + Math.random() * 4).toFixed(0),
  }));

  private statusInterval: any;

  ngOnInit(): void {
    // Fetch maintenance message
    this.api.getSystemStatus().subscribe({
      next: r => {
        if (r.data?.message) {
          this.maintenanceMessage.set(r.data.message);
        }
        // If maintenance was disabled while we're on this page, redirect away
        if (!r.data?.maintenance) {
          this.router.navigate(['/auth/login']);
        }
      },
    });

    // Poll every 30s to auto-redirect when maintenance ends
    this.statusInterval = setInterval(() => {
      this.api.getSystemStatus().subscribe({
        next: r => {
          if (!r.data?.maintenance) {
            this.router.navigate(['/auth/login']);
          }
        },
      });
    }, 30000);
  }

  ngOnDestroy(): void {
    if (this.statusInterval) clearInterval(this.statusInterval);
  }

  onLogoClick(): void {
    this.clickCount++;
    clearTimeout(this.clickTimer);
    this.clickTimer = setTimeout(() => (this.clickCount = 0), 800);
    if (this.clickCount >= 3) {
      this.clickCount = 0;
      this.showAdminLogin.set(true);
    }
  }

  adminLogin(): void {
    if (!this.adminPhone || !this.adminPassword) {
      this.toast.warning('Please enter phone and password');
      return;
    }
    this.adminLoading.set(true);
    this.auth.login(this.adminPhone, this.adminPassword).subscribe({
      next: () => {
        const user = this.auth.currentUser();
        if (user?.system_role === 'SYSTEM_ADMIN') {
          this.toast.success('Welcome, Administrator');
          this.router.navigate(['/platform/command-center']);
        } else {
          this.toast.error('Access denied. Only System Administrators can access the platform during maintenance.');
          this.auth.logout();
          this.showAdminLogin.set(false);
        }
        this.adminLoading.set(false);
      },
      error: (err) => {
        this.adminLoading.set(false);
        const msg = err.error?.error?.message || err.error?.message || 'Invalid credentials';
        this.toast.error(msg);
      },
    });
  }
}
