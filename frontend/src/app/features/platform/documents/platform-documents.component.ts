import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Document } from '../../../core/models';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';

type DTab = 'all' | 'kyc_queue' | 'analytics';

@Component({
  selector: 'app-platform-documents',
  standalone: true,
  imports: [CommonModule, FormsModule, AutocompleteComponent],
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

  readonly typeOptions: AutocompleteOption[] = [
    { value: 'KYC_ID_FRONT', label: 'KYC ID Front', sublabel: 'National ID front side', searchText: 'national id card front identity', badge: 'KYC' },
    { value: 'KYC_ID_BACK', label: 'KYC ID Back', sublabel: 'National ID back side', searchText: 'national id card back identity', badge: 'KYC' },
    { value: 'KYC_SELFIE', label: 'KYC Selfie', sublabel: 'Face verification selfie', searchText: 'selfie face photo', badge: 'KYC' },
    { value: 'SACCO_REGISTRATION', label: 'Org Registration', sublabel: 'Organization registration certificate', searchText: 'company registration business certificate sacco', badge: 'BIZ' },
    { value: 'VEHICLE_LOGBOOK', label: 'Vehicle Logbook', sublabel: 'Motor vehicle logbook', searchText: 'vehicle logbook ownership', badge: 'KYC' },
    { value: 'OTHER', label: 'Other', sublabel: 'Miscellaneous document', searchText: 'other misc document' },
  ];

  readonly statusOptions: AutocompleteOption[] = [
    { value: 'PENDING', label: 'Pending', sublabel: 'Awaiting review', searchText: 'pending waiting review' },
    { value: 'VERIFIED', label: 'Verified', sublabel: 'Approved and verified', searchText: 'verified approved accepted' },
    { value: 'REJECTED', label: 'Rejected', sublabel: 'Review failed', searchText: 'rejected denied failed' },
    { value: 'EXPIRED', label: 'Expired', sublabel: 'Document has expired', searchText: 'expired outdated lapsed' },
  ];

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
    this.api.getDocuments({ status: 'PENDING', document_type: 'KYC_ID_FRONT', per_page: '50' }).subscribe({
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
    switch (t) { 
      case 'KYC_ID_FRONT':
      case 'KYC_ID_BACK': return 'badge'; 
      case 'KYC_SELFIE': return 'face'; 
      case 'SACCO_REGISTRATION': return 'business'; 
      case 'VEHICLE_LOGBOOK': return 'directions_car'; 
      default: return 'description'; 
    }
  }
}
