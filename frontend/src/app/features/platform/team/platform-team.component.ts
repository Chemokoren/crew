import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { AdminUser } from '../../../core/models';

@Component({
  selector: 'app-platform-team',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './platform-team.component.html',
  styleUrl: './platform-team.component.scss',
})
export class PlatformTeamComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  loading = signal(true);
  members = signal<AdminUser[]>([]);
  totalMembers = signal(0);
  page = signal(1);
  perPage = 20;

  // Invite modal
  inviteOpen = signal(false);
  inviteData = signal({ first_name: '', last_name: '', phone: '', email: '', role: 'SYSTEM_ADMIN', password: '' });
  inviteSaving = signal(false);

  // Filter
  filterRole = signal('');

  readonly roles = [
    { value: 'SYSTEM_ADMIN', label: 'System Admin', color: '#ef4444' },
    { value: 'SACCO_ADMIN', label: 'Organization Admin', color: '#f59e0b' },
    { value: 'CREW', label: 'Crew Member', color: '#3b82f6' },
    { value: 'LENDER', label: 'Lender', color: '#10b981' },
    { value: 'INSURER', label: 'Insurer', color: '#8b5cf6' },
  ];

  ngOnInit() { this.loadMembers(); }

  loadMembers() {
    this.loading.set(true);
    const params: Record<string, string> = { page: String(this.page()), per_page: String(this.perPage) };
    if (this.filterRole()) params['role'] = this.filterRole();

    this.api.getUsers(params).subscribe({
      next: r => { this.members.set(r.data || []); this.totalMembers.set(r.meta?.total || 0); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  openInvite() {
    this.inviteData.set({ first_name: '', last_name: '', phone: '', email: '', role: 'SYSTEM_ADMIN', password: '' });
    this.inviteOpen.set(true);
  }

  closeInvite() { this.inviteOpen.set(false); }

  sendInvite() {
    const d = this.inviteData();
    if (!d.first_name || !d.phone || !d.password) { this.toast.warning('Fill in required fields'); return; }
    this.inviteSaving.set(true);
    this.api.registerTeamMember(d).subscribe({
      next: () => { this.toast.success('Team member added'); this.closeInvite(); this.loadMembers(); this.inviteSaving.set(false); },
      error: () => { this.toast.error('Failed to add member'); this.inviteSaving.set(false); },
    });
  }

  toggleActive(m: AdminUser) {
    const obs = m.is_active ? this.api.disableAccount(m.id) : this.api.enableAccount(m.id);
    obs.subscribe({
      next: () => { this.toast.success(`User ${m.is_active ? 'disabled' : 'enabled'}`); this.loadMembers(); },
      error: () => this.toast.error('Failed to update'),
    });
  }

  resetPwd(m: AdminUser) {
    if (!confirm(`Reset password for ${m.phone}?`)) return;
    this.api.resetPassword(m.id, '').subscribe({
      next: (r: any) => this.toast.success(`New password: ${r.data?.new_password || 'sent'}`),
      error: () => this.toast.error('Reset failed'),
    });
  }

  nextPage() { if (this.page() * this.perPage < this.totalMembers()) { this.page.set(this.page() + 1); this.loadMembers(); } }
  prevPage() { if (this.page() > 1) { this.page.set(this.page() - 1); this.loadMembers(); } }
  get totalPages(): number { return Math.ceil(this.totalMembers() / this.perPage); }

  roleInfo(role: string) { return this.roles.find(r => r.value === role) || { label: role, color: '#6366f1' }; }

  getInitials(m: AdminUser): string {
    // Use phone last 2 digits as fallback
    return m.phone?.slice(-2) || '??';
  }
}
