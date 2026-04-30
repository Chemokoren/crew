import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { CurrencyKesPipe } from '../../../shared/pipes/currency-kes.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { SACCO, SACCOFloat, SACCOMembership, SACCOFloatTransaction, CrewMember, PaginationMeta } from '../../../core/models';

@Component({
  selector: 'app-sacco-detail',
  standalone: true,
  imports: [CommonModule, FormsModule, CurrencyKesPipe, RelativeTimePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <!-- Header -->
      <div class="page-header">
        <div>
          <button class="btn btn-ghost btn-sm" (click)="goBack()" style="margin-bottom:var(--space-xs);">
            <span class="material-icons-round" style="font-size:16px;">arrow_back</span> Back to SACCOs
          </button>
          <h1 class="page-title">{{ sacco()?.name || 'SACCO Details' }}</h1>
          <p class="page-subtitle">{{ sacco()?.registration_number }}</p>
        </div>
        <div class="page-actions">
          <button class="btn btn-secondary" (click)="openModal('edit')" id="btn-edit-sacco">
            <span class="material-icons-round">edit</span> Edit
          </button>
        </div>
      </div>

      @if (loading()) {
        @for (i of [1,2,3]; track i) { <div class="skeleton" style="height:100px;margin:8px 0;border-radius:var(--radius-lg);"></div> }
      } @else if (sacco(); as s) {
        <!-- Info cards -->
        <div class="stats-grid" style="grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));">
          <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">business</span></div><div class="stat-value">{{ s.county }}</div><div class="stat-label">County</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">phone</span></div><div class="stat-value" style="font-size:0.95rem;">{{ s.contact_phone }}</div><div class="stat-label">Contact Phone</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(255,184,0,0.12);color:#ffb800;"><span class="material-icons-round">groups</span></div><div class="stat-value">{{ memberCount() }}</div><div class="stat-label">Members</div></div>
          <div class="stat-card"><div class="stat-icon" style="background:rgba(168,85,247,0.12);color:#a855f7;"><span class="material-icons-round">account_balance</span></div><div class="stat-value">{{ (saccoFloat()?.balance_cents || 0) | currencyKes }}</div><div class="stat-label">Float Balance</div></div>
        </div>

        <!-- Tab Nav -->
        <div class="tab-nav" style="margin-top:var(--space-lg);">
          <button class="tab-item" [class.active]="activeTab === 'info'" (click)="activeTab='info'">Details</button>
          <button class="tab-item" [class.active]="activeTab === 'members'" (click)="activeTab='members'">Members</button>
          <button class="tab-item" [class.active]="activeTab === 'float'" (click)="activeTab='float';loadFloatTxs()">Float</button>
        </div>

        <!-- Tab: Details + Edit -->
        @if (activeTab === 'info') {
          <div class="glass-card" style="margin-top:var(--space-md);">
            <div class="detail-grid">
              <div class="detail-row"><span class="detail-label">Name</span><span class="detail-value">{{ s.name }}</span></div>
              <div class="detail-row"><span class="detail-label">Registration No.</span><span class="detail-value"><code class="text-accent">{{ s.registration_number }}</code></span></div>
              <div class="detail-row"><span class="detail-label">County</span><span class="detail-value">{{ s.county }}</span></div>
              <div class="detail-row"><span class="detail-label">Sub-County</span><span class="detail-value">{{ s.sub_county || '—' }}</span></div>
              <div class="detail-row"><span class="detail-label">Contact Phone</span><span class="detail-value">{{ s.contact_phone }}</span></div>
              <div class="detail-row"><span class="detail-label">Contact Email</span><span class="detail-value">{{ s.contact_email || '—' }}</span></div>
              <div class="detail-row"><span class="detail-label">Currency</span><span class="detail-value">{{ s.currency }}</span></div>
              <div class="detail-row"><span class="detail-label">Status</span><span class="detail-value"><span class="badge" [ngClass]="s.is_active?'badge-success':'badge-danger'">{{ s.is_active?'Active':'Inactive' }}</span></span></div>
              <div class="detail-row"><span class="detail-label">Created</span><span class="detail-value">{{ s.created_at | relativeTime }}</span></div>
            </div>
          </div>
        }

        <!-- Tab: Members -->
        @if (activeTab === 'members') {
          <div class="glass-card" style="margin-top:var(--space-md);">
            <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:var(--space-md);">
              <h3 style="font-size:1rem;font-weight:600;margin:0;">Members</h3>
              <button class="btn btn-primary btn-sm" (click)="openModal('addMember')" id="btn-add-member">
                <span class="material-icons-round" style="font-size:16px;">person_add</span> Add Member
              </button>
            </div>
            @if (members().length === 0) {
              <div class="empty-state" style="padding:var(--space-lg);"><span class="material-icons-round empty-icon">group_off</span><div class="empty-title">No members yet</div></div>
            } @else {
              <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Crew Member ID</th><th>Role</th><th>Joined</th><th>Actions</th></tr></thead>
                <tbody>@for (m of members(); track m.id) {
                  <tr>
                    <td><code class="text-accent">{{ m.crew_member_id | slice:0:8 }}...</code></td>
                    <td><span class="badge badge-info">{{ m.role }}</span></td>
                    <td>{{ m.joined_at | relativeTime }}</td>
                    <td><button class="btn btn-ghost btn-sm" style="color:var(--color-danger);" (click)="removeMember(m)" id="remove-member-{{m.id}}"><span class="material-icons-round" style="font-size:16px;">person_remove</span></button></td>
                  </tr>
                }</tbody>
              </table></div>
            }
          </div>
        }

        <!-- Tab: Float -->
        @if (activeTab === 'float') {
          <div class="glass-card" style="margin-top:var(--space-md);">
            <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:var(--space-md);flex-wrap:wrap;gap:var(--space-sm);">
              <h3 style="font-size:1rem;font-weight:600;margin:0;">Float Account</h3>
              <div style="display:flex;gap:var(--space-sm);">
                <button class="btn btn-primary btn-sm" (click)="openModal('creditFloat')" id="btn-credit-float"><span class="material-icons-round" style="font-size:16px;">add_circle</span> Credit</button>
                <button class="btn btn-secondary btn-sm" (click)="openModal('debitFloat')" id="btn-debit-float"><span class="material-icons-round" style="font-size:16px;">remove_circle</span> Debit</button>
              </div>
            </div>

            @if (saccoFloat(); as f) {
              <div style="background:var(--color-bg-tertiary);border-radius:var(--radius-md);padding:var(--space-md);margin-bottom:var(--space-md);display:flex;gap:var(--space-xl);flex-wrap:wrap;">
                <div><span style="font-size:0.75rem;color:var(--color-text-muted);">Balance</span><div style="font-size:1.5rem;font-weight:700;color:var(--color-accent);">{{ f.balance_cents | currencyKes }}</div></div>
                <div><span style="font-size:0.75rem;color:var(--color-text-muted);">Currency</span><div style="font-size:1rem;font-weight:500;">{{ f.currency }}</div></div>
              </div>
            }

            <h4 style="font-size:0.85rem;font-weight:600;margin-bottom:var(--space-sm);">Float Transactions</h4>
            @if (floatTxs().length === 0) {
              <div class="empty-state" style="padding:var(--space-lg);"><span class="material-icons-round empty-icon">receipt_long</span><div class="empty-title">No float transactions</div></div>
            } @else {
              <div class="data-table-wrapper"><table class="data-table"><thead><tr><th>Type</th><th>Amount</th><th>Balance After</th><th>Reference</th><th>Date</th></tr></thead>
                <tbody>@for (tx of floatTxs(); track tx.id) {
                  <tr>
                    <td><span class="badge" [ngClass]="tx.transaction_type==='CREDIT'?'badge-success':'badge-danger'">{{ tx.transaction_type }}</span></td>
                    <td [style.color]="tx.transaction_type==='CREDIT'?'var(--color-success)':'var(--color-danger)'" style="font-weight:600;">{{ tx.transaction_type==='CREDIT'?'+':'-' }}{{ tx.amount_cents | currencyKes }}</td>
                    <td>{{ tx.balance_after_cents | currencyKes }}</td>
                    <td>{{ tx.reference || '—' }}</td>
                    <td>{{ tx.created_at | relativeTime }}</td>
                  </tr>
                }</tbody>
              </table></div>

              @if (floatTxMeta(); as m) {
                @if (m.total_pages > 1) {
                  <div class="pagination" style="margin-top:var(--space-md);">
                    <button class="page-btn" [disabled]="floatPage===1" (click)="floatPage=floatPage-1;loadFloatTxs()">← Prev</button>
                    <span style="font-size:0.8rem;color:var(--color-text-muted);">Page {{ floatPage }} of {{ m.total_pages }}</span>
                    <button class="page-btn" [disabled]="floatPage>=m.total_pages" (click)="floatPage=floatPage+1;loadFloatTxs()">Next →</button>
                  </div>
                }
              }
            }
          </div>
        }
      }

      <!-- ========== MODALS ========== -->

      <!-- Edit SACCO Modal -->
      @if (showModal() === 'edit') {
        <div class="modal-backdrop" (click)="closeModal()"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Edit SACCO</h3><button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <label class="form-label">Name</label><input class="form-input" [(ngModel)]="editForm.name" id="edit-name">
            <label class="form-label" style="margin-top:var(--space-sm);">County</label><input class="form-input" [(ngModel)]="editForm.county" id="edit-county">
            <label class="form-label" style="margin-top:var(--space-sm);">Sub-County</label><input class="form-input" [(ngModel)]="editForm.sub_county" id="edit-subcounty">
            <label class="form-label" style="margin-top:var(--space-sm);">Contact Phone</label><input class="form-input" [(ngModel)]="editForm.contact_phone" id="edit-phone">
            <label class="form-label" style="margin-top:var(--space-sm);">Contact Email</label><input class="form-input" type="email" [(ngModel)]="editForm.contact_email" id="edit-email">
          </div>
          <div class="modal-footer"><button class="btn btn-ghost" (click)="closeModal()">Cancel</button><button class="btn btn-primary" (click)="submitEdit()" [disabled]="submitting()" id="btn-submit-edit">{{ submitting()?'Saving...':'Save Changes' }}</button></div>
        </div></div>
      }

      <!-- Add Member Modal -->
      @if (showModal() === 'addMember') {
        <div class="modal-backdrop" (click)="closeModal()"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Add Member</h3><button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <label class="form-label">Crew Member</label>
            <select class="form-select" [(ngModel)]="memberCrewId" id="modal-member-crew">
              <option value="">— Select —</option>
              @for (c of crewList(); track c.id) { <option [value]="c.id">{{ c.first_name }} {{ c.last_name }} ({{ c.crew_id }})</option> }
            </select>
            <label class="form-label" style="margin-top:var(--space-sm);">Role in SACCO</label>
            <select class="form-select" [(ngModel)]="memberRole" id="modal-member-role">
              <option value="MEMBER">Member</option>
              <option value="CHAIRMAN">Chairman</option>
              <option value="TREASURER">Treasurer</option>
              <option value="SECRETARY">Secretary</option>
            </select>
          </div>
          <div class="modal-footer"><button class="btn btn-ghost" (click)="closeModal()">Cancel</button><button class="btn btn-primary" (click)="submitAddMember()" [disabled]="submitting()" id="btn-submit-member">{{ submitting()?'Adding...':'Add Member' }}</button></div>
        </div></div>
      }

      <!-- Credit Float Modal -->
      @if (showModal() === 'creditFloat') {
        <div class="modal-backdrop" (click)="closeModal()"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Credit Float</h3><button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <label class="form-label">Amount (KES)</label><input type="number" class="form-input" [(ngModel)]="floatAmount" min="1" placeholder="e.g. 50000" id="modal-float-credit-amount">
            <label class="form-label" style="margin-top:var(--space-sm);">Reference</label><input class="form-input" [(ngModel)]="floatReference" placeholder="e.g. Bank deposit" id="modal-float-credit-ref">
          </div>
          <div class="modal-footer"><button class="btn btn-ghost" (click)="closeModal()">Cancel</button><button class="btn btn-primary" (click)="submitCreditFloat()" [disabled]="submitting()" id="btn-submit-credit-float">{{ submitting()?'Processing...':'Credit Float' }}</button></div>
        </div></div>
      }

      <!-- Debit Float Modal -->
      @if (showModal() === 'debitFloat') {
        <div class="modal-backdrop" (click)="closeModal()"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Debit Float</h3><button class="btn-ghost" (click)="closeModal()"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <label class="form-label">Amount (KES)</label><input type="number" class="form-input" [(ngModel)]="floatAmount" min="1" placeholder="e.g. 10000" id="modal-float-debit-amount">
            <label class="form-label" style="margin-top:var(--space-sm);">Reference</label><input class="form-input" [(ngModel)]="floatReference" placeholder="e.g. Operational expense" id="modal-float-debit-ref">
          </div>
          <div class="modal-footer"><button class="btn btn-ghost" (click)="closeModal()">Cancel</button><button class="btn btn-danger" (click)="submitDebitFloat()" [disabled]="submitting()" id="btn-submit-debit-float">{{ submitting()?'Processing...':'Debit Float' }}</button></div>
        </div></div>
      }
    </div>
  `,
  styles: [`
    .detail-grid { display: grid; gap: var(--space-sm); }
    .detail-row { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; border-bottom: 1px solid var(--color-border); &:last-child { border-bottom: none; } }
    .detail-label { font-size: 0.8rem; color: var(--color-text-muted); font-weight: 500; }
    .detail-value { font-size: 0.875rem; color: var(--color-text-primary); font-weight: 500; text-align: right; }
    .form-label { display: block; font-size: 0.8rem; font-weight: 500; color: var(--color-text-secondary); margin-bottom: 4px; }
  `]
})
export class SaccoDetailComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  sacco = signal<SACCO | null>(null);
  saccoFloat = signal<SACCOFloat | null>(null);
  members = signal<SACCOMembership[]>([]);
  memberCount = signal(0);
  floatTxs = signal<SACCOFloatTransaction[]>([]);
  floatTxMeta = signal<PaginationMeta | null>(null);
  crewList = signal<CrewMember[]>([]);
  loading = signal(true);
  showModal = signal<string | null>(null);
  submitting = signal(false);

  activeTab = 'info';
  saccoId = '';
  floatPage = 1;

  // Edit form
  editForm = { name: '', county: '', sub_county: '', contact_phone: '', contact_email: '' };

  // Member form
  memberCrewId = '';
  memberRole = 'MEMBER';

  // Float form
  floatAmount = 0;
  floatReference = '';

  ngOnInit(): void {
    this.saccoId = this.route.snapshot.paramMap.get('id') || '';
    if (this.saccoId) {
      this.loadSACCO();
      this.loadMembers();
      this.loadFloat();
      this.api.getCrewMembers({ per_page: '200' }).subscribe({ next: (r) => this.crewList.set(r.data) });
    }
  }

  goBack(): void { this.router.navigate(['/saccos']); }

  loadSACCO(): void {
    this.loading.set(true);
    this.api.getSACCO(this.saccoId).subscribe({
      next: (r) => {
        this.sacco.set(r.data);
        this.editForm = {
          name: r.data.name, county: r.data.county,
          sub_county: r.data.sub_county || '', contact_phone: r.data.contact_phone,
          contact_email: r.data.contact_email || '',
        };
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  loadMembers(): void {
    this.api.getSACCOMembers(this.saccoId, { per_page: '100' }).subscribe({
      next: (r) => { this.members.set(r.data || []); this.memberCount.set(r.meta?.total || (r.data?.length || 0)); },
    });
  }

  loadFloat(): void {
    this.api.getSACCOFloat(this.saccoId).subscribe({
      next: (r) => this.saccoFloat.set(r.data),
      error: () => {},
    });
  }

  loadFloatTxs(): void {
    this.api.getFloatTransactions(this.saccoId, { page: String(this.floatPage), per_page: '20' }).subscribe({
      next: (r) => { this.floatTxs.set(r.data || []); this.floatTxMeta.set(r.meta); },
    });
  }

  // --- Modals ---
  openModal(type: string): void { this.floatAmount = 0; this.floatReference = ''; this.memberCrewId = ''; this.memberRole = 'MEMBER'; this.showModal.set(type); }
  closeModal(): void { this.showModal.set(null); }

  submitEdit(): void {
    this.submitting.set(true);
    this.api.updateSACCO(this.saccoId, this.editForm).subscribe({
      next: () => { this.toast.success('SACCO updated'); this.closeModal(); this.submitting.set(false); this.loadSACCO(); },
      error: () => this.submitting.set(false),
    });
  }

  submitAddMember(): void {
    if (!this.memberCrewId) { this.toast.error('Select a crew member'); return; }
    this.submitting.set(true);
    this.api.addSACCOMember(this.saccoId, { crew_member_id: this.memberCrewId, role: this.memberRole }).subscribe({
      next: () => { this.toast.success('Member added'); this.closeModal(); this.submitting.set(false); this.loadMembers(); },
      error: () => this.submitting.set(false),
    });
  }

  removeMember(m: SACCOMembership): void {
    if (!confirm('Remove this member from the SACCO?')) return;
    this.api.removeSACCOMember(this.saccoId, m.id).subscribe({
      next: () => { this.toast.success('Member removed'); this.loadMembers(); },
    });
  }

  submitCreditFloat(): void {
    if (this.floatAmount <= 0) { this.toast.error('Enter a valid amount'); return; }
    this.submitting.set(true);
    this.api.creditSACCOFloat(this.saccoId, {
      amount_cents: Math.round(this.floatAmount * 100),
      idempotency_key: crypto.randomUUID(),
      reference: this.floatReference,
    }).subscribe({
      next: () => { this.toast.success('Float credited'); this.closeModal(); this.submitting.set(false); this.loadFloat(); this.loadFloatTxs(); },
      error: () => this.submitting.set(false),
    });
  }

  submitDebitFloat(): void {
    if (this.floatAmount <= 0) { this.toast.error('Enter a valid amount'); return; }
    this.submitting.set(true);
    this.api.debitSACCOFloat(this.saccoId, {
      amount_cents: Math.round(this.floatAmount * 100),
      idempotency_key: crypto.randomUUID(),
      reference: this.floatReference,
    }).subscribe({
      next: () => { this.toast.success('Float debited'); this.closeModal(); this.submitting.set(false); this.loadFloat(); this.loadFloatTxs(); },
      error: () => this.submitting.set(false),
    });
  }
}
