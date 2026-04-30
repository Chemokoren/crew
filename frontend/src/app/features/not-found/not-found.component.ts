import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-not-found',
  standalone: true,
  imports: [RouterLink],
  template: `
    <div class="not-found">
      <div class="not-found-content animate-fade-in">
        <div class="glitch-code">404</div>
        <h1 class="nf-title">Page Not Found</h1>
        <p class="nf-desc">The page you're looking for doesn't exist or has been moved.</p>
        <a routerLink="/dashboard" class="btn btn-primary btn-lg">
          <span class="material-icons-round">home</span> Back to Dashboard
        </a>
      </div>
    </div>`,
  styles: [`
    .not-found{min-height:100vh;display:flex;align-items:center;justify-content:center;padding:var(--space-xl);background:var(--color-bg-primary);}
    .not-found-content{text-align:center;max-width:440px;}
    .glitch-code{
      font-family:var(--font-heading);font-size:8rem;font-weight:900;line-height:1;
      background:var(--gradient-accent);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;
      letter-spacing:-4px;margin-bottom:var(--space-md);
      text-shadow:0 0 80px rgba(0,210,255,0.3);
    }
    .nf-title{font-size:1.5rem;font-weight:700;color:var(--color-text-primary);margin-bottom:var(--space-sm);}
    .nf-desc{font-size:0.9375rem;color:var(--color-text-muted);margin-bottom:var(--space-xl);line-height:1.6;}
  `]
})
export class NotFoundComponent {}
