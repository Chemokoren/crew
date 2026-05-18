import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import {
  AdminUser, Wallet, WalletTransaction, PayrollRun, AuditLog,
  CrewMember, Organization
} from '../../../core/models';

type SupportTab = 'lookup' | 'wallets' | 'payroll' | 'actions' | 'timeline';

interface SupportTicketSummary {
  openTickets: number;
  stuckWallets: number;
  failedPayrolls: number;
  pendingVerifications: number;
}

@Component({
  selector: 'app-platform-support',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './platform-support.component.html',
  styleUrl: './platform-support.component.scss',
})
export class PlatformSupportComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  activeTab = signal<SupportTab>('lookup');
  loading = signal(false);

  // ── Tab definitions ──
  readonly tabs: { id: SupportTab; label: string; icon: string }[] = [
    { id: 'lookup', label: 'User Lookup', icon: 'person_search' },
    { id: 'wallets', label: 'Wallet Recovery', icon: 'account_balance_wallet' },
    { id: 'payroll', label: 'Payroll Issues', icon: 'payments' },
    { id: 'actions', label: 'Quick Actions', icon: 'flash_on' },
    { id: 'timeline', label: 'Activity Timeline', icon: 'timeline' },
  ];

  // ── Stats ──
  summary = signal<SupportTicketSummary>({
    openTickets: 0,
    stuckWallets: 0,
    failedPayrolls: 0,
    pendingVerifications: 0,
  });

  // ── User Lookup ──
  searchQuery = signal('');
  searchType = signal<'phone' | 'email' | 'name' | 'id'>('phone');
  searchResults = signal<AdminUser[]>([]);
  searchLoading = signal(false);
  selectedUser = signal<AdminUser | null>(null);
  selectedUserWallet = signal<Wallet | null>(null);
  selectedUserTransactions = signal<WalletTransaction[]>([]);
  selectedUserAuditLogs = signal<AuditLog[]>([]);
  selectedUserCrewProfile = signal<CrewMember | null>(null);
  userDetailLoading = signal(false);

  // ── Wallet Recovery ──
  walletSearchQuery = signal('');
  stuckTransactions = signal<WalletTransaction[]>([]);
  walletSearchResults = signal<{ user: AdminUser; wallet: Wallet }[]>([]);
  walletLoading = signal(false);
  walletRecoveryAmount = signal('');
  walletRecoveryReason = signal('');
  selectedWalletUser = signal<AdminUser | null>(null);
  selectedWalletDetail = signal<Wallet | null>(null);
  walletTransactions = signal<WalletTransaction[]>([]);
  walletTxPage = signal(1);
  walletTxTotal = signal(0);

  // ── Payroll Issues ──
  failedPayrolls = signal<PayrollRun[]>([]);
  payrollLoading = signal(false);
  payrollPage = signal(1);
  payrollTotal = signal(0);
  payrollOrgs = signal<Map<string, string>>(new Map());

  // ── Quick Actions ──
  actionUserId = signal('');
  actionPhone = signal('');
  actionSearchResults = signal<AdminUser[]>([]);
  actionSearchLoading = signal(false);
  selectedActionUser = signal<AdminUser | null>(null);
  actionInProgress = signal('');
  resetPasswordValue = signal('');

  // ── Activity Timeline ──
  timelineUserId = signal('');
  timelineSearchQuery = signal('');
  timelineSearchResults = signal<AdminUser[]>([]);
  timelineSearchLoading = signal(false);
  timelineLogs = signal<AuditLog[]>([]);
  timelineLoading = signal(false);
  timelinePage = signal(1);
  timelineTotal = signal(0);
  timelineSelectedUser = signal<AdminUser | null>(null);
  timelineFilterAction = signal('');

  readonly perPage = 15;

  // ── Computed ──
  totalTimelinePages = computed(() => Math.ceil(this.timelineTotal() / this.perPage) || 1);
  totalPayrollPages = computed(() => Math.ceil(this.payrollTotal() / this.perPage) || 1);
  totalWalletTxPages = computed(() => Math.ceil(this.walletTxTotal() / this.perPage) || 1);

  ngOnInit() {
    this.loadSummary();
    this.loadFailedPayrolls();
  }

  switchTab(t: SupportTab) {
    this.activeTab.set(t);
    if (t === 'payroll') this.loadFailedPayrolls();
  }

  // ══════════════════════════════════════════════════════════════
  // Summary stats
  // ══════════════════════════════════════════════════════════════
  loadSummary() {
    // Aggregate stats from existing endpoints
    this.api.getUsers({ per_page: '1' }).subscribe({
      next: r => {
        this.summary.update(s => ({ ...s, openTickets: r.meta?.total || 0 }));
      },
    });
    this.api.getPayrollRuns({ status: 'FAILED', per_page: '1' }).subscribe({
      next: r => {
        this.summary.update(s => ({ ...s, failedPayrolls: r.meta?.total || 0 }));
      },
    });
  }

  // ══════════════════════════════════════════════════════════════
  // User Lookup
  // ══════════════════════════════════════════════════════════════
  searchUsers() {
    const q = this.searchQuery().trim();
    if (!q) return;

    this.searchLoading.set(true);
    this.selectedUser.set(null);

    const params: Record<string, string> = { per_page: '20' };
    const type = this.searchType();
    if (type === 'phone') params['phone'] = q;
    else if (type === 'email') params['email'] = q;
    else if (type === 'name') params['search'] = q;
    else if (type === 'id') params['user_id'] = q;

    this.api.getUsers(params).subscribe({
      next: r => {
        this.searchResults.set(r.data || []);
        this.searchLoading.set(false);
        if ((r.data || []).length === 0) {
          this.toast.info('No users found matching your search');
        }
      },
      error: () => {
        this.searchLoading.set(false);
        this.toast.error('Search failed');
      },
    });
  }

  selectUser(user: AdminUser) {
    this.selectedUser.set(user);
    this.userDetailLoading.set(true);

    // Load wallet & audit logs
    if (user.crew_member_id) {
      this.api.getWalletBalance(user.crew_member_id).subscribe({
        next: r => this.selectedUserWallet.set(r.data),
        error: () => this.selectedUserWallet.set(null),
      });
      this.api.getWalletTransactions(user.crew_member_id, { per_page: '10' }).subscribe({
        next: r => this.selectedUserTransactions.set(r.data || []),
        error: () => this.selectedUserTransactions.set([]),
      });
      this.api.getCrewMember(user.crew_member_id).subscribe({
        next: r => this.selectedUserCrewProfile.set(r.data),
        error: () => this.selectedUserCrewProfile.set(null),
      });
    }

    this.api.getAuditLogs({ user_id: user.id, per_page: '10' }).subscribe({
      next: r => {
        this.selectedUserAuditLogs.set(r.data || []);
        this.userDetailLoading.set(false);
      },
      error: () => this.userDetailLoading.set(false),
    });
  }

  closeUserDetail() {
    this.selectedUser.set(null);
    this.selectedUserWallet.set(null);
    this.selectedUserTransactions.set([]);
    this.selectedUserAuditLogs.set([]);
    this.selectedUserCrewProfile.set(null);
  }

  // ══════════════════════════════════════════════════════════════
  // Wallet Recovery
  // ══════════════════════════════════════════════════════════════
  searchWallets() {
    const q = this.walletSearchQuery().trim();
    if (!q) return;

    this.walletLoading.set(true);
    this.selectedWalletUser.set(null);
    this.selectedWalletDetail.set(null);

    this.api.getUsers({ phone: q, per_page: '10' }).subscribe({
      next: r => {
        const users = r.data || [];
        const results: { user: AdminUser; wallet: Wallet }[] = [];
        let pending = users.filter(u => !!u.crew_member_id).length;

        if (pending === 0) {
          this.walletSearchResults.set([]);
          this.walletLoading.set(false);
          this.toast.info('No wallets found');
          return;
        }

        for (const user of users) {
          if (!user.crew_member_id) continue;
          this.api.getWalletBalance(user.crew_member_id).subscribe({
            next: wr => {
              results.push({ user, wallet: wr.data });
              pending--;
              if (pending <= 0) {
                this.walletSearchResults.set(results);
                this.walletLoading.set(false);
              }
            },
            error: () => {
              pending--;
              if (pending <= 0) {
                this.walletSearchResults.set(results);
                this.walletLoading.set(false);
              }
            },
          });
        }
      },
      error: () => {
        this.walletLoading.set(false);
        this.toast.error('Wallet search failed');
      },
    });
  }

  selectWallet(user: AdminUser) {
    if (!user.crew_member_id) return;
    this.selectedWalletUser.set(user);
    this.walletLoading.set(true);

    this.api.getWalletBalance(user.crew_member_id).subscribe({
      next: r => {
        this.selectedWalletDetail.set(r.data);
        this.loadWalletTransactions(user.crew_member_id!);
      },
      error: () => this.walletLoading.set(false),
    });
  }

  loadWalletTransactions(crewMemberId: string) {
    this.api.getWalletTransactions(crewMemberId, {
      page: String(this.walletTxPage()),
      per_page: String(this.perPage),
    }).subscribe({
      next: r => {
        this.walletTransactions.set(r.data || []);
        this.walletTxTotal.set(r.meta?.total || 0);
        this.walletLoading.set(false);
      },
      error: () => this.walletLoading.set(false),
    });
  }

  walletTxNextPage() {
    if (this.walletTxPage() < this.totalWalletTxPages()) {
      this.walletTxPage.update(p => p + 1);
      const u = this.selectedWalletUser();
      if (u?.crew_member_id) this.loadWalletTransactions(u.crew_member_id);
    }
  }

  walletTxPrevPage() {
    if (this.walletTxPage() > 1) {
      this.walletTxPage.update(p => p - 1);
      const u = this.selectedWalletUser();
      if (u?.crew_member_id) this.loadWalletTransactions(u.crew_member_id);
    }
  }

  creditWalletRecovery() {
    const user = this.selectedWalletUser();
    if (!user?.crew_member_id) return;
    const amount = parseInt(this.walletRecoveryAmount(), 10);
    if (!amount || amount <= 0) {
      this.toast.error('Please enter a valid amount');
      return;
    }
    const reason = this.walletRecoveryReason().trim() || 'Support recovery credit';
    if (!confirm(`You are about to credit KES ${amount.toLocaleString()} to this wallet.\n\nReason: ${reason}\n\nThis action is irreversible. Continue?`)) return;
    const idempotencyKey = `support-recovery-${user.crew_member_id}-${Date.now()}`;

    this.api.creditWallet({
      crew_member_id: user.crew_member_id,
      amount_cents: amount * 100,
      category: 'SUPPORT_RECOVERY',
      reference: idempotencyKey,
      description: reason,
    }, idempotencyKey).subscribe({
      next: () => {
        this.toast.success(`KES ${amount} credited successfully`);
        this.walletRecoveryAmount.set('');
        this.walletRecoveryReason.set('');
        // Refresh wallet
        this.selectWallet(user);
      },
      error: () => this.toast.error('Credit failed — check amount and try again'),
    });
  }

  closeWalletDetail() {
    this.selectedWalletUser.set(null);
    this.selectedWalletDetail.set(null);
    this.walletTransactions.set([]);
    this.walletRecoveryAmount.set('');
    this.walletRecoveryReason.set('');
  }

  // ══════════════════════════════════════════════════════════════
  // Payroll Issues
  // ══════════════════════════════════════════════════════════════
  loadFailedPayrolls() {
    this.payrollLoading.set(true);
    this.api.getPayrollRuns({
      status: 'FAILED',
      page: String(this.payrollPage()),
      per_page: String(this.perPage),
    }).subscribe({
      next: r => {
        const runs = r.data || [];
        this.failedPayrolls.set(runs);
        this.payrollTotal.set(r.meta?.total || 0);
        this.summary.update(s => ({ ...s, failedPayrolls: r.meta?.total || 0 }));

        // Resolve org names
        const orgIds = [...new Set(runs.map(p => p.organization_id))];
        for (const orgId of orgIds) {
          if (!this.payrollOrgs().has(orgId)) {
            this.api.getOrganization(orgId).subscribe({
              next: or => {
                const m = new Map(this.payrollOrgs());
                m.set(orgId, or.data?.name || orgId);
                this.payrollOrgs.set(m);
              },
            });
          }
        }
        this.payrollLoading.set(false);
      },
      error: () => this.payrollLoading.set(false),
    });
  }

  reprocessPayroll(payrollId: string) {
    this.payrollLoading.set(true);
    this.api.processPayrollRun(payrollId).subscribe({
      next: () => {
        this.toast.success('Payroll reprocessed successfully');
        this.loadFailedPayrolls();
      },
      error: () => {
        this.toast.error('Reprocessing failed');
        this.payrollLoading.set(false);
      },
    });
  }

  resubmitPayroll(payrollId: string) {
    this.payrollLoading.set(true);
    this.api.submitPayrollRun(payrollId).subscribe({
      next: () => {
        this.toast.success('Payroll resubmitted for disbursement');
        this.loadFailedPayrolls();
      },
      error: () => {
        this.toast.error('Resubmission failed');
        this.payrollLoading.set(false);
      },
    });
  }

  payrollNextPage() {
    if (this.payrollPage() < this.totalPayrollPages()) {
      this.payrollPage.update(p => p + 1);
      this.loadFailedPayrolls();
    }
  }
  payrollPrevPage() {
    if (this.payrollPage() > 1) {
      this.payrollPage.update(p => p - 1);
      this.loadFailedPayrolls();
    }
  }

  getOrgName(orgId: string): string {
    return this.payrollOrgs().get(orgId) || orgId.substring(0, 8) + '…';
  }

  // ══════════════════════════════════════════════════════════════
  // Quick Actions
  // ══════════════════════════════════════════════════════════════
  searchActionUsers() {
    const q = this.actionPhone().trim();
    if (!q) return;
    this.actionSearchLoading.set(true);
    this.selectedActionUser.set(null);

    this.api.getUsers({ phone: q, per_page: '10' }).subscribe({
      next: r => {
        this.actionSearchResults.set(r.data || []);
        this.actionSearchLoading.set(false);
        if ((r.data || []).length === 0) this.toast.info('No users found');
      },
      error: () => {
        this.actionSearchLoading.set(false);
        this.toast.error('Search failed');
      },
    });
  }

  selectActionUser(user: AdminUser) {
    this.selectedActionUser.set(user);
  }

  resendVerificationCode() {
    const user = this.selectedActionUser();
    if (!user?.crew_member_id) {
      this.toast.error('User has no linked crew profile');
      return;
    }
    this.actionInProgress.set('resend');
    this.api.resendCredentials(user.crew_member_id).subscribe({
      next: () => {
        this.toast.success('Verification code resent successfully');
        this.actionInProgress.set('');
      },
      error: () => {
        this.toast.error('Failed to resend verification code');
        this.actionInProgress.set('');
      },
    });
  }

  enableUserAccount() {
    const user = this.selectedActionUser();
    if (!user) return;
    if (!confirm(`Re-enable account for ${user.phone}?`)) return;
    this.actionInProgress.set('enable');
    this.api.enableAccount(user.id).subscribe({
      next: () => {
        this.toast.success('Account enabled');
        this.actionInProgress.set('');
        // Refresh user in list
        const updated = { ...user, is_active: true };
        this.selectedActionUser.set(updated);
      },
      error: () => {
        this.toast.error('Failed to enable account');
        this.actionInProgress.set('');
      },
    });
  }

  disableUserAccount() {
    const user = this.selectedActionUser();
    if (!user) return;
    if (!confirm(`Are you sure you want to DISABLE the account for ${user.phone}? The user will be unable to log in.`)) return;
    this.actionInProgress.set('disable');
    this.api.disableAccount(user.id).subscribe({
      next: () => {
        this.toast.success('Account disabled');
        this.actionInProgress.set('');
        const updated = { ...user, is_active: false };
        this.selectedActionUser.set(updated);
      },
      error: () => {
        this.toast.error('Failed to disable account');
        this.actionInProgress.set('');
      },
    });
  }

  resetUserPassword() {
    const user = this.selectedActionUser();
    if (!user) return;
    const newPassword = this.resetPasswordValue().trim();
    if (newPassword.length < 8) {
      this.toast.error('Password must be at least 8 characters');
      return;
    }
    this.actionInProgress.set('reset');
    this.api.resetPassword(user.id, newPassword).subscribe({
      next: () => {
        this.toast.success('Password reset successfully');
        this.resetPasswordValue.set('');
        this.actionInProgress.set('');
      },
      error: () => {
        this.toast.error('Password reset failed');
        this.actionInProgress.set('');
      },
    });
  }

  // ══════════════════════════════════════════════════════════════
  // Activity Timeline
  // ══════════════════════════════════════════════════════════════
  searchTimelineUsers() {
    const q = this.timelineSearchQuery().trim();
    if (!q) return;
    this.timelineSearchLoading.set(true);

    this.api.getUsers({ phone: q, per_page: '10' }).subscribe({
      next: r => {
        this.timelineSearchResults.set(r.data || []);
        this.timelineSearchLoading.set(false);
        if ((r.data || []).length === 0) this.toast.info('No users found');
      },
      error: () => {
        this.timelineSearchLoading.set(false);
        this.toast.error('Search failed');
      },
    });
  }

  selectTimelineUser(user: AdminUser) {
    this.timelineSelectedUser.set(user);
    this.timelinePage.set(1);
    this.loadTimeline();
  }

  loadTimeline() {
    const user = this.timelineSelectedUser();
    if (!user) return;

    this.timelineLoading.set(true);
    const params: Record<string, string> = {
      user_id: user.id,
      page: String(this.timelinePage()),
      per_page: String(this.perPage),
    };
    if (this.timelineFilterAction()) {
      params['action'] = this.timelineFilterAction();
    }

    this.api.getAuditLogs(params).subscribe({
      next: r => {
        this.timelineLogs.set(r.data || []);
        this.timelineTotal.set(r.meta?.total || 0);
        this.timelineLoading.set(false);
      },
      error: () => {
        this.timelineLoading.set(false);
        this.toast.error('Failed to load activity timeline');
      },
    });
  }

  filterTimeline() {
    this.timelinePage.set(1);
    this.loadTimeline();
  }

  timelineNextPage() {
    if (this.timelinePage() < this.totalTimelinePages()) {
      this.timelinePage.update(p => p + 1);
      this.loadTimeline();
    }
  }
  timelinePrevPage() {
    if (this.timelinePage() > 1) {
      this.timelinePage.update(p => p - 1);
      this.loadTimeline();
    }
  }

  clearTimelineUser() {
    this.timelineSelectedUser.set(null);
    this.timelineLogs.set([]);
    this.timelineTotal.set(0);
  }

  // ══════════════════════════════════════════════════════════════
  // Helpers
  // ══════════════════════════════════════════════════════════════
  formatKes(cents: number | undefined | null): string {
    if (cents == null) return 'KES 0.00';
    return `KES ${(cents / 100).toLocaleString('en-KE', { minimumFractionDigits: 2 })}`;
  }

  actionColor(action: string): string {
    switch (action) {
      case 'CREATE': return '#10b981';
      case 'UPDATE': return '#6366f1';
      case 'DELETE': return '#ef4444';
      case 'LOGIN': return '#3b82f6';
      case 'LOGOUT': return '#8b5cf6';
      case 'APPROVE': return '#10b981';
      case 'REJECT': return '#ef4444';
      default: return '#64748b';
    }
  }

  statusColor(status: string): string {
    switch (status) {
      case 'COMPLETED': return '#10b981';
      case 'FAILED': return '#ef4444';
      case 'PENDING': return '#f59e0b';
      case 'PROCESSING': return '#6366f1';
      default: return '#64748b';
    }
  }

  roleLabel(role: string): string {
    return role?.replace(/_/g, ' ').toLowerCase().replace(/^\w/, c => c.toUpperCase()) || '—';
  }

  trackById(_: number, item: { id: string }) {
    return item.id;
  }

  onSearchKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') this.searchUsers();
  }

  onWalletSearchKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') this.searchWallets();
  }

  onActionSearchKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') this.searchActionUsers();
  }

  onTimelineSearchKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') this.searchTimelineUsers();
  }
}
