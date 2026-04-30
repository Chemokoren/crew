import { Component, input } from '@angular/core';

@Component({
  selector: 'app-loading-spinner',
  standalone: true,
  template: `
    <div class="spinner-wrapper" [class.fullpage]="fullpage()" [class.inline]="!fullpage()" [attr.aria-label]="'Loading'" role="status">
      <div class="spinner">
        <svg viewBox="0 0 50 50">
          <circle class="track" cx="25" cy="25" r="20" fill="none" stroke-width="4"></circle>
          <circle class="path" cx="25" cy="25" r="20" fill="none" stroke-width="4" stroke-linecap="round"></circle>
        </svg>
      </div>
      @if (message()) { <span class="spinner-text">{{ message() }}</span> }
    </div>`,
  styles: [`
    .spinner-wrapper{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:var(--space-md);}
    .spinner-wrapper.fullpage{position:fixed;inset:0;z-index:150;background:rgba(10,14,26,0.7);backdrop-filter:blur(4px);}
    .spinner-wrapper.inline{padding:var(--space-xl) 0;}
    .spinner{width:44px;height:44px;}
    .spinner svg{animation:spin 1.4s linear infinite;}
    .track{stroke:rgba(255,255,255,0.06);}
    .path{stroke:url(#grad);stroke-dasharray:80,200;stroke-dashoffset:0;animation:dash 1.4s ease-in-out infinite;}
    .spinner-text{font-size:0.875rem;color:var(--color-text-muted);font-weight:500;}
    @keyframes spin{to{transform:rotate(360deg);}}
    @keyframes dash{0%{stroke-dasharray:1,200;stroke-dashoffset:0;}50%{stroke-dasharray:80,200;stroke-dashoffset:-35px;}100%{stroke-dasharray:80,200;stroke-dashoffset:-125px;}}
  `]
})
export class LoadingSpinnerComponent {
  fullpage = input(false);
  message = input('');
}
