import { Injectable, signal } from '@angular/core';

export interface ToastMessage {
  id: number;
  type: 'success' | 'error' | 'warning' | 'info';
  message: string;
  duration: number;
}

@Injectable({ providedIn: 'root' })
export class ToastService {
  private counter = 0;
  readonly toasts = signal<ToastMessage[]>([]);

  success(message: string, duration = 4000): void {
    this.addToast('success', message, duration);
  }

  error(message: string, duration = 6000): void {
    this.addToast('error', message, duration);
  }

  warning(message: string, duration = 5000): void {
    this.addToast('warning', message, duration);
  }

  info(message: string, duration = 4000): void {
    this.addToast('info', message, duration);
  }

  dismiss(id: number): void {
    this.toasts.update(toasts => toasts.filter(t => t.id !== id));
  }

  private addToast(type: ToastMessage['type'], message: string, duration: number): void {
    const id = ++this.counter;
    const toast: ToastMessage = { id, type, message, duration };
    this.toasts.update(toasts => [...toasts, toast]);
    setTimeout(() => this.dismiss(id), duration);
  }
}
