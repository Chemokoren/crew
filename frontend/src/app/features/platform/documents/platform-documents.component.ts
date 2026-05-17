import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Document } from '../../../core/models';

type DTab = 'all' | 'kyc_queue' | 'analytics';

@Component({
  selector: 'app-platform-documents',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './platform-documents.component.html',
  styleUrl: './platform-documents.component.scss',
})
export class PlatformDocumentsComponent implements OnInit {
  private api = inject(ApiService);
  private toast = inject(ToastService);

  activeTab = signal<DTab>('all');
  loading = signal(true);
  documents = signal<Document[]>([]);
  totalDocs = signal(0);
  page = signal(1);
  perPage = 20;

  // Filters
  filterType = signal('');
  filterStatus = signal('');
  filterSearch = signal('');

  // KYC queue
  kycPending = signal<Document[]>([]);

  readonly tabs: { id: DTab; label: string; icon: string }[] = [
    { id: 'all', label: 'All Documents', icon: 'folder' },
    { id: 'kyc_queue', label: 'KYC Queue', icon: 'verified_user' },
    { id: 'analytics', label: 'Analytics', icon: 'pie_chart' },
  ];

  readonly docTypes = ['NATIONAL_ID', 'KRA_PIN', 'PASSPORT', 'DRIVERS_LICENSE', 'COMPANY_REG', 'CONTRACT', 'OTHER'];
  readonly statuses = ['PENDING', 'VERIFIED', 'REJECTED', 'EXPIRED'];

  ngOnInit() { this.loadDocuments(); }

  switchTab(t: DTab) {
    this.activeTab.set(t);
    if (t === 'kyc_queue') this.loadKycQueue();
  }

  loadDocuments() {
    this.loading.set(true);
    const params: Record<string, string> = { page: String(this.page()), per_page: String(this.perPage) };
    if (this.filterType()) params['document_type'] = this.filterType();
    if (this.filterStatus()) params['status'] = this.filterStatus();

    this.api.getDocuments(params).subscribe({
      next: r => { this.documents.set(r.data || []); this.totalDocs.set(r.meta?.total || 0); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  loadKycQueue() {
    this.api.getDocuments({ status: 'PENDING', document_type: 'NATIONAL_ID', per_page: '50' }).subscribe({
      next: r => this.kycPending.set(r.data || []),
    });
  }

  applyFilters() { this.page.set(1); this.loadDocuments(); }
  clearFilters() { this.filterType.set(''); this.filterStatus.set(''); this.filterSearch.set(''); this.page.set(1); this.loadDocuments(); }

  nextPage() { if (this.page() * this.perPage < this.totalDocs()) { this.page.set(this.page() + 1); this.loadDocuments(); } }
  prevPage() { if (this.page() > 1) { this.page.set(this.page() - 1); this.loadDocuments(); } }
  get totalPages(): number { return Math.ceil(this.totalDocs() / this.perPage); }

  verifyDoc(doc: Document) {
    this.api.verifyDocument(doc.id).subscribe({
      next: () => { this.toast.success('Document verified'); this.loadDocuments(); this.loadKycQueue(); },
      error: () => this.toast.error('Verification failed'),
    });
  }

  rejectDoc(doc: Document) {
    this.api.rejectDocument(doc.id).subscribe({
      next: () => { this.toast.success('Document rejected'); this.loadDocuments(); this.loadKycQueue(); },
      error: () => this.toast.error('Rejection failed'),
    });
  }

  statusColor(s: string): string {
    switch (s) { case 'VERIFIED': return '#10b981'; case 'REJECTED': return '#ef4444'; case 'PENDING': return '#f59e0b'; default: return '#6366f1'; }
  }

  typeIcon(t: string): string {
    switch (t) { case 'NATIONAL_ID': return 'badge'; case 'KRA_PIN': return 'receipt'; case 'PASSPORT': return 'flight'; default: return 'description'; }
  }
}
