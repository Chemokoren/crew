import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

/**
 * Placeholder component for platform pages that are planned but not yet implemented.
 * Shows a styled coming-soon state with the page title and description.
 */
@Component({
  selector: 'app-platform-placeholder',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title platform-title">{{ title }}</h1>
          <p class="page-subtitle">{{ subtitle }}</p>
        </div>
      </div>

      <div class="coming-soon glass-card">
        <div class="cs-icon-wrap">
          <span class="material-icons-round">{{ icon }}</span>
        </div>
        <h2 class="cs-title">Coming Soon</h2>
        <p class="cs-desc">{{ description }}</p>
        <div class="cs-features">
          @for (feature of features; track feature) {
            <div class="cs-feature">
              <span class="material-icons-round" style="font-size:16px; color:#8b5cf6;">check_circle</span>
              <span>{{ feature }}</span>
            </div>
          }
        </div>
      </div>
    </div>
  `,
  styles: [`
    .platform-title {
      background: linear-gradient(135deg, #c084fc 0%, #f472b6 100%) !important;
      -webkit-background-clip: text !important;
      -webkit-text-fill-color: transparent !important;
      background-clip: text !important;
    }

    .coming-soon {
      display: flex;
      flex-direction: column;
      align-items: center;
      text-align: center;
      padding: var(--space-2xl) var(--space-xl) !important;
      max-width: 560px;
      margin: var(--space-xl) auto;
    }

    .cs-icon-wrap {
      width: 72px;
      height: 72px;
      border-radius: var(--radius-lg);
      background: linear-gradient(135deg, rgba(139,92,246,0.15), rgba(236,72,153,0.1));
      display: flex;
      align-items: center;
      justify-content: center;
      margin-bottom: var(--space-lg);

      .material-icons-round {
        font-size: 36px;
        color: #8b5cf6;
      }
    }

    .cs-title {
      font-family: var(--font-heading);
      font-size: 1.5rem;
      font-weight: 700;
      background: linear-gradient(135deg, #c084fc, #f472b6);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
      margin-bottom: var(--space-sm);
    }

    .cs-desc {
      font-size: 0.875rem;
      color: var(--color-text-muted);
      margin-bottom: var(--space-lg);
      max-width: 400px;
    }

    .cs-features {
      display: flex;
      flex-direction: column;
      gap: var(--space-sm);
      text-align: left;
      width: 100%;
      max-width: 320px;
    }

    .cs-feature {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      font-size: 0.8125rem;
      color: var(--color-text-secondary);
    }
  `]
})
export class PlatformPlaceholderComponent {
  @Input() title = 'Platform Page';
  @Input() subtitle = '';
  @Input() icon = 'construction';
  @Input() description = 'This page is being built as part of the platform admin redesign.';
  @Input() features: string[] = [];
}
