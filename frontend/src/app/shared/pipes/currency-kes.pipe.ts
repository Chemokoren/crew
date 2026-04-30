import { Pipe, PipeTransform } from '@angular/core';

@Pipe({ name: 'currencyKes', standalone: true })
export class CurrencyKesPipe implements PipeTransform {
  transform(cents: number | undefined | null): string {
    if (cents === undefined || cents === null) return 'KES 0.00';
    const kes = cents / 100;
    return `KES ${kes.toLocaleString('en-KE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
  }
}
