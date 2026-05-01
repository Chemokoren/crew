import { Component, Injectable, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subject } from 'rxjs';

/** Result from confirm/prompt dialog */
export interface DialogResult {
  confirmed: boolean;
  value?: string;
}

interface DialogConfig {
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'default' | 'danger';
  /** If true, shows a text input (prompt mode) */
  prompt?: boolean;
  promptLabel?: string;
  promptPlaceholder?: string;
  /** Input type: 'text' | 'number' | 'tel' etc. Default: 'text' */
  promptType?: string;
  /** Prefix shown inside the input (e.g. 'KES') */
  promptPrefix?: string;
  /** Suffix shown inside the input (e.g. '%') */
  promptSuffix?: string;
  /** Icon name from Material Icons (overrides default question/warning) */
  icon?: string;
}

@Injectable({ providedIn: 'root' })
export class ConfirmDialogService {
  readonly visible = signal(false);
  readonly config = signal<DialogConfig>({ title: '', message: '' });
  readonly inputValue = signal('');
  private result$ = new Subject<DialogResult>();

  /** Show a confirm dialog. Returns observable that emits once. */
  confirm(title: string, message: string, opts?: Partial<DialogConfig>): Subject<DialogResult> {
    this.config.set({ title, message, variant: 'default', ...opts });
    this.inputValue.set('');
    this.visible.set(true);
    this.result$ = new Subject<DialogResult>();
    return this.result$;
  }

  /** Show a prompt dialog with text input. */
  prompt(title: string, message: string, opts?: Partial<DialogConfig>): Subject<DialogResult> {
    return this.confirm(title, message, { ...opts, prompt: true });
  }

  /** Show a danger confirmation (e.g. delete, cancel). */
  danger(title: string, message: string, opts?: Partial<DialogConfig>): Subject<DialogResult> {
    return this.confirm(title, message, { variant: 'danger', confirmText: 'Delete', ...opts });
  }

  /** @internal */
  _resolve(confirmed: boolean): void {
    this.result$.next({ confirmed, value: this.inputValue() });
    this.result$.complete();
    this.visible.set(false);
  }
}

@Component({
  selector: 'app-confirm-dialog',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    @if (svc.visible()) {
      <div class="modal-backdrop" (click)="svc._resolve(false)" role="alertdialog" aria-modal="true" [attr.aria-label]="svc.config().title">
        <div class="modal-content dialog-box" (click)="$event.stopPropagation()">
          <div class="dialog-icon" [class.danger]="svc.config().variant==='danger'">
            <span class="material-icons-round">{{ dialogIcon() }}</span>
          </div>
          <h3 class="dialog-title">{{ svc.config().title }}</h3>
          <p class="dialog-message">{{ svc.config().message }}</p>
          @if (svc.config().prompt) {
            <div class="form-group" style="margin-top:var(--space-md);width:100%;">
              @if (svc.config().promptLabel) { <label class="form-label">{{ svc.config().promptLabel }}</label> }
              <div class="input-wrapper" [class.has-prefix]="!!svc.config().promptPrefix" [class.has-suffix]="!!svc.config().promptSuffix">
                @if (svc.config().promptPrefix) {
                  <span class="input-addon prefix">{{ svc.config().promptPrefix }}</span>
                }
                <input
                  class="form-input"
                  [type]="svc.config().promptType || 'text'"
                  [placeholder]="svc.config().promptPlaceholder || ''"
                  [ngModel]="svc.inputValue()"
                  (ngModelChange)="svc.inputValue.set($event)"
                  (keydown.enter)="svc._resolve(true)"
                  autofocus
                />
                @if (svc.config().promptSuffix) {
                  <span class="input-addon suffix">{{ svc.config().promptSuffix }}</span>
                }
              </div>
            </div>
          }
          <div class="dialog-actions">
            <button class="btn btn-secondary" (click)="svc._resolve(false)">{{ svc.config().cancelText || 'Cancel' }}</button>
            <button class="btn" [ngClass]="svc.config().variant==='danger' ? 'btn-danger' : 'btn-primary'" (click)="svc._resolve(true)">
              {{ svc.config().confirmText || 'Confirm' }}
            </button>
          </div>
        </div>
      </div>
    }`,
  styles: [`
    .dialog-box {
      max-width: 440px; display: flex; flex-direction: column; align-items: center;
      text-align: center; padding: var(--space-xl) var(--space-lg) !important;
    }
    .dialog-icon {
      width: 52px; height: 52px; border-radius: 50%; display: flex;
      align-items: center; justify-content: center; margin-bottom: var(--space-md);
      background: rgba(0, 210, 255, 0.12); color: var(--color-accent);
      .material-icons-round { font-size: 26px; }
    }
    .dialog-icon.danger { background: var(--color-danger-light); color: var(--color-danger); }
    .dialog-title { font-size: 1.125rem; font-weight: 700; color: var(--color-text-primary); margin-bottom: var(--space-xs); }
    .dialog-message { font-size: 0.875rem; color: var(--color-text-muted); line-height: 1.5; }
    .dialog-actions {
      display: flex; gap: var(--space-sm); margin-top: var(--space-lg);
      width: 100%; justify-content: center;
    }
    .dialog-actions .btn { min-width: 100px; }

    /* Input with prefix/suffix addons */
    .input-wrapper {
      position: relative; display: flex; align-items: stretch;
      border-radius: var(--radius-md); overflow: hidden;
    }
    .input-wrapper .form-input {
      flex: 1; border-radius: 0; min-width: 0;
    }
    .input-wrapper:not(.has-prefix) .form-input { border-top-left-radius: var(--radius-md); border-bottom-left-radius: var(--radius-md); }
    .input-wrapper:not(.has-suffix) .form-input { border-top-right-radius: var(--radius-md); border-bottom-right-radius: var(--radius-md); }

    .input-addon {
      display: flex; align-items: center; padding: 0 12px;
      background: rgba(255, 255, 255, 0.04); border: 1px solid var(--color-border);
      color: var(--color-text-muted); font-size: 0.8125rem; font-weight: 600;
      white-space: nowrap; user-select: none;
    }
    .input-addon.prefix { border-right: none; border-radius: var(--radius-md) 0 0 var(--radius-md); }
    .input-addon.suffix { border-left: none; border-radius: 0 var(--radius-md) var(--radius-md) 0; }
  `]
})
export class ConfirmDialogComponent {
  constructor(public svc: ConfirmDialogService) {}

  dialogIcon(): string {
    const cfg = this.svc.config();
    if (cfg.icon) return cfg.icon;
    return cfg.variant === 'danger' ? 'warning' : 'help_outline';
  }
}
