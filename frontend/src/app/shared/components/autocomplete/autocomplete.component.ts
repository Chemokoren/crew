import {
  Component,
  Input,
  Output,
  EventEmitter,
  signal,
  computed,
  OnChanges,
  SimpleChanges,
  HostListener,
  ElementRef,
  inject,
  ChangeDetectionStrategy,
  forwardRef,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, NG_VALUE_ACCESSOR, ControlValueAccessor } from '@angular/forms';

export interface AutocompleteOption {
  /** The actual value (e.g. id) */
  value: string;
  /** Primary display label */
  label: string;
  /** Secondary text shown under the label */
  sublabel?: string;
  /** Small badge/tag shown to the right */
  badge?: string;
  /** All searchable text fields joined for filtering */
  searchText: string;
}

@Component({
  selector: 'app-autocomplete',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => AutocompleteComponent),
      multi: true,
    },
  ],
  template: `
    <div class="autocomplete-wrapper">
      <div class="autocomplete-input-container" [class.is-open]="isOpen()" [class.has-value]="!!selectedOption()">
        <!-- Search icon -->
        <span class="material-icons-round autocomplete-search-icon">search</span>

        <!-- The input -->
        <input
          class="form-input autocomplete-input"
          type="text"
          [placeholder]="placeholder"
          [value]="displayText()"
          (input)="onInput($event)"
          (focus)="onFocus()"
          (keydown)="onKeyDown($event)"
          [attr.id]="inputId"
          autocomplete="off"
        />

        <!-- Clear button when value is selected -->
        @if (selectedOption()) {
          <button class="autocomplete-clear-btn" type="button" (click)="clearSelection($event)" tabindex="-1">
            <span class="material-icons-round">close</span>
          </button>
        }

        <!-- Chevron -->
        <span class="material-icons-round autocomplete-chevron" [class.rotated]="isOpen()">expand_more</span>
      </div>

      <!-- Dropdown -->
      @if (isOpen()) {
        <div class="autocomplete-dropdown">
          @if (filteredOptions().length === 0) {
            <div class="autocomplete-empty">
              <span class="material-icons-round" style="font-size:20px;opacity:0.5;">search_off</span>
              <span>No results found</span>
            </div>
          } @else {
            @for (opt of filteredOptions(); track opt.value; let i = $index) {
              <div
                class="autocomplete-option"
                [class.is-highlighted]="i === highlightedIndex()"
                (mousedown)="selectOption(opt, $event)"
                (mouseenter)="highlightedIndex.set(i)"
              >
                <div class="option-content">
                  <div class="option-label">{{ opt.label }}</div>
                  @if (opt.sublabel) {
                    <div class="option-sublabel">{{ opt.sublabel }}</div>
                  }
                </div>
                @if (opt.badge) {
                  <span class="option-badge">{{ opt.badge }}</span>
                }
              </div>
            }
          }
        </div>
      }
    </div>
  `,
  styles: [`
    .autocomplete-wrapper {
      position: relative;
      width: 100%;
    }

    .autocomplete-input-container {
      position: relative;
      display: flex;
      align-items: center;
    }

    .autocomplete-search-icon {
      position: absolute;
      left: 12px;
      top: 50%;
      transform: translateY(-50%);
      font-size: 18px;
      color: var(--color-text-muted);
      pointer-events: none;
      transition: color var(--transition-fast);
      z-index: 1;
    }

    .autocomplete-input-container.is-open .autocomplete-search-icon,
    .autocomplete-input-container:focus-within .autocomplete-search-icon {
      color: var(--color-accent);
    }

    .autocomplete-input {
      padding-left: 38px !important;
      padding-right: 64px !important;
      cursor: text;
    }

    .autocomplete-input-container.has-value .autocomplete-input {
      color: var(--color-text-primary);
      font-weight: 500;
    }

    .autocomplete-clear-btn {
      position: absolute;
      right: 32px;
      top: 50%;
      transform: translateY(-50%);
      background: rgba(255, 255, 255, 0.06);
      border: none;
      border-radius: 50%;
      width: 20px;
      height: 20px;
      display: flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      color: var(--color-text-muted);
      transition: all var(--transition-fast);
      padding: 0;
      z-index: 1;

      &:hover {
        background: var(--color-danger-light);
        color: var(--color-danger);
      }

      .material-icons-round {
        font-size: 14px;
      }
    }

    .autocomplete-chevron {
      position: absolute;
      right: 10px;
      top: 50%;
      transform: translateY(-50%);
      font-size: 18px;
      color: var(--color-text-muted);
      pointer-events: none;
      transition: transform var(--transition-fast), color var(--transition-fast);

      &.rotated {
        transform: translateY(-50%) rotate(180deg);
        color: var(--color-accent);
      }
    }

    .autocomplete-dropdown {
      position: absolute;
      top: calc(100% + 4px);
      left: 0;
      right: 0;
      background: var(--color-bg-secondary);
      border: 1px solid var(--color-border-hover);
      border-radius: var(--radius-md);
      box-shadow: var(--shadow-lg);
      z-index: 50;
      max-height: 240px;
      overflow-y: auto;
      animation: acDropIn 150ms ease-out;
    }

    @keyframes acDropIn {
      from {
        opacity: 0;
        transform: translateY(-4px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }

    .autocomplete-option {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
      padding: 10px 14px;
      cursor: pointer;
      transition: background var(--transition-fast);
      border-bottom: 1px solid rgba(255,255,255,0.03);

      &:last-child {
        border-bottom: none;
      }

      &:hover,
      &.is-highlighted {
        background: var(--gradient-accent-soft);
      }

      &.is-highlighted {
        background: rgba(0, 210, 255, 0.08);
      }
    }

    .option-content {
      flex: 1;
      min-width: 0;
    }

    .option-label {
      font-size: 0.875rem;
      font-weight: 500;
      color: var(--color-text-primary);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .option-sublabel {
      font-size: 0.75rem;
      color: var(--color-text-muted);
      margin-top: 1px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .option-badge {
      flex-shrink: 0;
      padding: 2px 8px;
      border-radius: var(--radius-full);
      font-size: 0.6875rem;
      font-weight: 500;
      background: rgba(0, 210, 255, 0.12);
      color: var(--color-accent);
      text-transform: uppercase;
      letter-spacing: 0.03em;
    }

    .autocomplete-empty {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 16px;
      color: var(--color-text-muted);
      font-size: 0.8125rem;
      justify-content: center;
    }

    /* Scrollbar inside dropdown */
    .autocomplete-dropdown::-webkit-scrollbar {
      width: 4px;
    }
    .autocomplete-dropdown::-webkit-scrollbar-track {
      background: transparent;
    }
    .autocomplete-dropdown::-webkit-scrollbar-thumb {
      background: rgba(255,255,255,0.1);
      border-radius: 4px;
    }
  `],
})
export class AutocompleteComponent implements OnChanges, ControlValueAccessor {
  @Input() options: AutocompleteOption[] = [];
  @Input() placeholder = 'Search...';
  @Input() inputId = '';

  @Output() selectionChange = new EventEmitter<string>();

  private elRef = inject(ElementRef);

  isOpen = signal(false);
  searchQuery = signal('');
  highlightedIndex = signal(0);
  selectedOption = signal<AutocompleteOption | null>(null);

  private onChange: (val: string) => void = () => {};
  private onTouched: () => void = () => {};

  filteredOptions = computed(() => {
    const query = this.searchQuery().toLowerCase().trim();
    if (!query) return this.options;
    return this.options.filter(o => o.searchText.toLowerCase().includes(query));
  });

  displayText = computed(() => {
    if (this.isOpen()) return this.searchQuery();
    return this.selectedOption()?.label || '';
  });

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['options'] && this.selectedOption()) {
      // Re-validate selection when options change
      const current = this.selectedOption();
      if (current && !this.options.find(o => o.value === current.value)) {
        this.selectedOption.set(null);
      }
    }
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: Event): void {
    if (!this.elRef.nativeElement.contains(event.target)) {
      this.close();
    }
  }

  onFocus(): void {
    this.isOpen.set(true);
    this.searchQuery.set('');
    this.highlightedIndex.set(0);
  }

  onInput(event: Event): void {
    const value = (event.target as HTMLInputElement).value;
    this.searchQuery.set(value);
    this.highlightedIndex.set(0);
    if (!this.isOpen()) this.isOpen.set(true);
  }

  onKeyDown(event: KeyboardEvent): void {
    const opts = this.filteredOptions();
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault();
        this.highlightedIndex.set(Math.min(this.highlightedIndex() + 1, opts.length - 1));
        break;
      case 'ArrowUp':
        event.preventDefault();
        this.highlightedIndex.set(Math.max(this.highlightedIndex() - 1, 0));
        break;
      case 'Enter':
        event.preventDefault();
        if (opts.length > 0 && this.isOpen()) {
          this.selectOption(opts[this.highlightedIndex()]);
        }
        break;
      case 'Escape':
        this.close();
        break;
    }
  }

  selectOption(opt: AutocompleteOption, event?: Event): void {
    event?.preventDefault();
    this.selectedOption.set(opt);
    this.searchQuery.set('');
    this.isOpen.set(false);
    this.onChange(opt.value);
    this.onTouched();
    this.selectionChange.emit(opt.value);
  }

  clearSelection(event: Event): void {
    event.stopPropagation();
    this.selectedOption.set(null);
    this.searchQuery.set('');
    this.onChange('');
    this.onTouched();
    this.selectionChange.emit('');
  }

  close(): void {
    this.isOpen.set(false);
    this.searchQuery.set('');
  }

  // ControlValueAccessor
  writeValue(value: string): void {
    if (value) {
      const opt = this.options.find(o => o.value === value);
      this.selectedOption.set(opt || null);
    } else {
      this.selectedOption.set(null);
    }
  }

  registerOnChange(fn: (val: string) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }
}
