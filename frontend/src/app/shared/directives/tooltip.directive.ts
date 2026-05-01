import { Directive, ElementRef, HostListener, Input, OnDestroy, Renderer2 } from '@angular/core';

/**
 * Reusable tooltip directive.
 *
 * Usage:
 *   <button appTooltip="Click to save">Save</button>
 *   <button appTooltip="Delete record" tooltipPosition="left">🗑</button>
 *
 * Positions: 'top' (default), 'bottom', 'left', 'right'
 */
@Directive({
  selector: '[appTooltip]',
  standalone: true,
})
export class TooltipDirective implements OnDestroy {
  @Input('appTooltip') text = '';
  @Input() tooltipPosition: 'top' | 'bottom' | 'left' | 'right' = 'top';

  private tooltipEl: HTMLElement | null = null;
  private showTimeout: ReturnType<typeof setTimeout> | null = null;

  constructor(private el: ElementRef<HTMLElement>, private renderer: Renderer2) {}

  @HostListener('mouseenter')
  onMouseEnter(): void {
    if (!this.text) return;
    this.showTimeout = setTimeout(() => this.show(), 300);
  }

  @HostListener('mouseleave')
  onMouseLeave(): void {
    this.hide();
  }

  @HostListener('click')
  onClick(): void {
    this.hide();
  }

  private show(): void {
    if (this.tooltipEl) return;

    // Create tooltip element
    this.tooltipEl = this.renderer.createElement('div') as HTMLElement;
    this.renderer.addClass(this.tooltipEl, 'app-tooltip');
    this.renderer.addClass(this.tooltipEl, `app-tooltip--${this.tooltipPosition}`);
    this.tooltipEl.textContent = this.text;
    this.renderer.appendChild(document.body, this.tooltipEl);

    // Position it relative to the host element
    this.positionTooltip();

    // Trigger the entrance animation
    requestAnimationFrame(() => {
      if (this.tooltipEl) this.renderer.addClass(this.tooltipEl, 'app-tooltip--visible');
    });
  }

  private positionTooltip(): void {
    if (!this.tooltipEl) return;

    const hostRect = this.el.nativeElement.getBoundingClientRect();
    const tooltipRect = this.tooltipEl.getBoundingClientRect();
    const gap = 8;

    let top = 0;
    let left = 0;

    switch (this.tooltipPosition) {
      case 'top':
        top = hostRect.top - tooltipRect.height - gap;
        left = hostRect.left + hostRect.width / 2 - tooltipRect.width / 2;
        break;
      case 'bottom':
        top = hostRect.bottom + gap;
        left = hostRect.left + hostRect.width / 2 - tooltipRect.width / 2;
        break;
      case 'left':
        top = hostRect.top + hostRect.height / 2 - tooltipRect.height / 2;
        left = hostRect.left - tooltipRect.width - gap;
        break;
      case 'right':
        top = hostRect.top + hostRect.height / 2 - tooltipRect.height / 2;
        left = hostRect.right + gap;
        break;
    }

    // Keep within viewport
    left = Math.max(8, Math.min(left, window.innerWidth - tooltipRect.width - 8));
    top = Math.max(8, top);

    this.renderer.setStyle(this.tooltipEl, 'top', `${top + window.scrollY}px`);
    this.renderer.setStyle(this.tooltipEl, 'left', `${left + window.scrollX}px`);
  }

  private hide(): void {
    if (this.showTimeout) {
      clearTimeout(this.showTimeout);
      this.showTimeout = null;
    }
    if (this.tooltipEl) {
      const el = this.tooltipEl;
      this.renderer.removeClass(el, 'app-tooltip--visible');
      setTimeout(() => {
        if (el.parentNode) el.parentNode.removeChild(el);
      }, 150);
      this.tooltipEl = null;
    }
  }

  ngOnDestroy(): void {
    this.hide();
  }
}
