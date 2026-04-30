import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

export interface Crumb {
  label: string;
  route?: string;
}

@Component({
  selector: 'app-breadcrumb',
  standalone: true,
  imports: [CommonModule, RouterLink],
  template: `
    <nav class="breadcrumb" aria-label="Breadcrumb">
      @for (crumb of crumbs(); track crumb.label; let last = $last) {
        @if (crumb.route && !last) {
          <a [routerLink]="crumb.route" class="crumb-link">{{ crumb.label }}</a>
        } @else {
          <span class="crumb-current" [attr.aria-current]="last?'page':null">{{ crumb.label }}</span>
        }
        @if (!last) { <span class="crumb-sep material-icons-round">chevron_right</span> }
      }
    </nav>`,
  styles: [`
    .breadcrumb{display:flex;align-items:center;gap:4px;font-size:0.8125rem;margin-bottom:var(--space-md);}
    .crumb-link{color:var(--color-text-muted);text-decoration:none;transition:color var(--transition-fast);&:hover{color:var(--color-accent);}}
    .crumb-current{color:var(--color-text-secondary);font-weight:500;}
    .crumb-sep{font-size:14px;color:var(--color-text-muted);opacity:0.5;}
  `]
})
export class BreadcrumbComponent {
  crumbs = input<Crumb[]>([]);
}
