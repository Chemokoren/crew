import { Pipe, PipeTransform, inject } from '@angular/core';
import { PermissionService } from '../../core/services/permission.service';

/**
 * Pipe that checks if the current user has a specific permission.
 *
 * Usage:
 *   <button [disabled]="!('workers.create' | can)">Add Worker</button>
 *   <div *ngIf="'wallet.view' | can">Wallet</div>
 */
@Pipe({
  name: 'can',
  standalone: true,
  pure: false // Needs to re-evaluate when permissions change
})
export class CanPipe implements PipeTransform {
  private permissionService = inject(PermissionService);

  transform(permKey: string): boolean {
    return this.permissionService.can(permKey);
  }
}
