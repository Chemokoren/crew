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

  /** Show a danger confirmation (e.g. delete). */
  danger(title: string, message: string): Subject<DialogResult> {
    return this.confirm(title, message, { variant: 'danger', confirmText: 'Delete' });
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
            <span class="material-icons-round">{{svc.config().variant==='danger'?'warning':'help_outline'}}</span>
          </div>
          <h3 class="dialog-title">{{svc.config().title}}</h3>
          <p class="dialog-message">{{svc.config().message}}</p>
          @if (svc.config().prompt) {
            <div class="form-group" style="margin-top:var(--space-md);width:100%;">
              @if (svc.config().promptLabel) { <label class="form-label">{{svc.config().promptLabel}}</label> }
              <input class="form-input" [placeholder]="svc.config().promptPlaceholder||''" [ngModel]="svc.inputValue()" (ngModelChange)="svc.inputValue.set($event)" (keydown.enter)="svc._resolve(true)" autofocus />
            </div>
          }
          <div class="dialog-actions">
            <button class="btn btn-secondary" (click)="svc._resolve(false)">{{svc.config().cancelText||'Cancel'}}</button>
            <button class="btn" [ngClass]="svc.config().variant==='danger'?'btn-danger':'btn-primary'" (click)="svc._resolve(true)">
              {{svc.config().confirmText||'Confirm'}}
            </button>
          </div>
        </div>
      </div>
    }`,
  styles: [`
    .dialog-box{max-width:400px;display:flex;flex-direction:column;align-items:center;text-align:center;padding:var(--space-xl) var(--space-lg)!important;}
    .dialog-icon{width:52px;height:52px;border-radius:50%;display:flex;align-items:center;justify-content:center;margin-bottom:var(--space-md);background:rgba(0,210,255,0.12);color:var(--color-accent);
      .material-icons-round{font-size:26px;}
    }
    .dialog-icon.danger{background:var(--color-danger-light);color:var(--color-danger);}
    .dialog-title{font-size:1.125rem;font-weight:700;color:var(--color-text-primary);margin-bottom:var(--space-xs);}
    .dialog-message{font-size:0.875rem;color:var(--color-text-muted);line-height:1.5;}
    .dialog-actions{display:flex;gap:var(--space-sm);margin-top:var(--space-lg);width:100%;justify-content:center;}
    .dialog-actions .btn{min-width:100px;}
  `]
})
export class ConfirmDialogComponent {
  constructor(public svc: ConfirmDialogService) {}
}
