import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Organization, SACCOFloat, SACCOFloatTransaction } from '../../../core/models';
import { Subject } from 'rxjs';
import { debounceTime, distinctUntilChanged } from 'rxjs/operators';

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

  // Pagination & Search
  searchQuery = signal('');
  currentPage = signal(1);
  perPage = signal(20);
  totalOrgs = signal(0);
  private searchSubject = new Subject<string>();

  // Drill-down
  selectedOrg = signal<Organization | null>(null);
  orgTransactions = signal<SACCOFloatTransaction[]>([]);
  loadingTxns = signal(false);
  
  // Transaction Pagination
  txnPage = signal(1);
  txnPerPage = signal(20);
  totalTxns = signal(0);
  totalTxnPages = computed(() => Math.ceil(this.totalTxns() / this.txnPerPage()));

  totalPages = computed(() => Math.ceil(this.totalOrgs() / this.perPage()));

  ngOnInit() {
    this.searchSubject.pipe(
      debounceTime(300),
      distinctUntilChanged()
    ).subscribe(q => {
      this.searchQuery.set(q);
      this.currentPage.set(1);
      this.loadOrgs();
    });
    this.loadOrgs();
  }

  onSearch(event: any) {
    this.searchSubject.next(event.target.value);
  }

  setPage(p: number) {
    if (p < 1 || p > this.totalPages()) return;
    this.currentPage.set(p);
    this.loadOrgs();
  }

  changePerPage(event: any) {
    this.perPage.set(parseInt(event.target.value, 10));
    this.currentPage.set(1);
    this.loadOrgs();
  }

  selectOrg(org: Organization) {
    this.selectedOrg.set(org);
    this.activeTab.set('transactions');
    this.txnPage.set(1);
    this.loadTransactions(org.id);
  }

  setTxnPage(p: number) {
    if (p < 1 || p > this.totalTxnPages()) return;
    this.txnPage.set(p);
    const org = this.selectedOrg();
    if (org) this.loadTransactions(org.id);
  }

  changeTxnPerPage(event: any) {
    this.txnPerPage.set(parseInt(event.target.value, 10));
    this.txnPage.set(1);
    const org = this.selectedOrg();
    if (org) this.loadTransactions(org.id);
  }

  loadTransactions(orgId: string) {
    this.loadingTxns.set(true);
    this.api.getFloatTransactions(orgId, { 
      page: this.txnPage().toString(),
      per_page: this.txnPerPage().toString() 
    }).subscribe({
      next: r => {
        this.orgTransactions.set(r.data || []);
        this.totalTxns.set(r.meta?.total || 0);
        this.loadingTxns.set(false);
      },
      error: () => this.loadingTxns.set(false)
    });
  }

  switchTab(t: FTab) { this.activeTab.set(t); }

  loadOrgs() {
    this.loading.set(true);
    this.api.getOrganizations({
      page: this.currentPage().toString(),
      per_page: this.perPage().toString(),
      search: this.searchQuery()
    }).subscribe({
      next: r => {
        const orgs = r.data || [];
        this.orgs.set(orgs);
        this.totalOrgs.set(r.meta?.total || 0);
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
