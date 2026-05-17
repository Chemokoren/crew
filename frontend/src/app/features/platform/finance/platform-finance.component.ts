import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Organization, SACCOFloat, SACCOFloatTransaction } from '../../../core/models';

type FTab = 'overview' | 'approvals' | 'transactions';

@Component({
  selector: 'app-platform-finance',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './platform-finance.component.html',
  styleUrl: './platform-finance.component.scss',
})
export class PlatformFinanceComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  activeTab = signal<FTab>('overview');
  loading = signal(true);

  orgs = signal<Organization[]>([]);
  floats = signal<Map<string, SACCOFloat>>(new Map());
  pendingTopups = signal<SACCOFloatTransaction[]>([]);
  recentTxns = signal<SACCOFloatTransaction[]>([]);
  totalFloat = signal(0);
  totalPending = signal(0);

  readonly tabs: { id: FTab; label: string; icon: string }[] = [
    { id: 'overview', label: 'Float Overview', icon: 'account_balance_wallet' },
    { id: 'approvals', label: 'Pending Approvals', icon: 'pending_actions' },
    { id: 'transactions', label: 'Transactions', icon: 'receipt_long' },
  ];

  ngOnInit() { this.loadOrgs(); }

  switchTab(t: FTab) { this.activeTab.set(t); }

  loadOrgs() {
    this.loading.set(true);
    this.api.getOrganizations({ per_page: '100' }).subscribe({
      next: r => {
        const orgs = r.data || [];
        this.orgs.set(orgs);
        let total = 0;
        for (const org of orgs) {
          this.api.getSACCOFloat(org.id).subscribe({
            next: fr => {
              const f = fr.data;
              if (f) {
                const m = this.floats();
                m.set(org.id, f);
                this.floats.set(new Map(m));
                total += f.balance_cents || 0;
                this.totalFloat.set(total);
              }
            },
          });
        }
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  getFloat(orgId: string): SACCOFloat | undefined {
    return this.floats().get(orgId);
  }

  formatKes(cents: number | undefined): string {
    if (!cents) return 'KES 0.00';
    return `KES ${(cents / 100).toLocaleString('en-KE', { minimumFractionDigits: 2 })}`;
  }

  approveTopup(txId: string, orgId: string) {
    this.api.confirmTopUp(orgId, txId).subscribe({
      next: () => { this.toast.success('Top-up approved'); this.loadOrgs(); },
      error: () => this.toast.error('Approval failed'),
    });
  }

  rejectTopup(txId: string, orgId: string) {
    this.api.rejectTopUp(orgId, txId).subscribe({
      next: () => { this.toast.success('Top-up rejected'); this.loadOrgs(); },
      error: () => this.toast.error('Rejection failed'),
    });
  }
}
