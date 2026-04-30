import { Directive, input, output, HostListener, HostBinding } from '@angular/core';

export interface SortEvent {
  column: string;
  direction: 'asc' | 'desc';
}

/**
 * Directive for sortable table columns.
 * Usage: <th appSortable="field_name" (sorted)="onSort($event)">Label</th>
 */
@Directive({
  selector: '[appSortable]',
  standalone: true,
})
export class SortableDirective {
  appSortable = input.required<string>();
  currentSort = input('');
  currentDir = input<'asc' | 'desc'>('asc');
  sorted = output<SortEvent>();

  @HostBinding('style.cursor') cursor = 'pointer';
  @HostBinding('style.user-select') userSelect = 'none';
  @HostBinding('attr.aria-sort') get ariaSort(): string | null {
    if (this.currentSort() !== this.appSortable()) return null;
    return this.currentDir() === 'asc' ? 'ascending' : 'descending';
  }

  @HostListener('click')
  onClick(): void {
    const col = this.appSortable();
    let dir: 'asc' | 'desc' = 'asc';
    if (this.currentSort() === col) {
      dir = this.currentDir() === 'asc' ? 'desc' : 'asc';
    }
    this.sorted.emit({ column: col, direction: dir });
  }
}

/**
 * Generic client-side sort helper.
 * Usage: sortData(items, 'field', 'asc')
 */
export function sortData<T>(items: T[], column: string, direction: 'asc' | 'desc'): T[] {
  return [...items].sort((a, b) => {
    const aVal = (a as Record<string, unknown>)[column];
    const bVal = (b as Record<string, unknown>)[column];
    if (aVal == null && bVal == null) return 0;
    if (aVal == null) return 1;
    if (bVal == null) return -1;
    const cmp = String(aVal).localeCompare(String(bVal), undefined, { numeric: true, sensitivity: 'base' });
    return direction === 'asc' ? cmp : -cmp;
  });
}
