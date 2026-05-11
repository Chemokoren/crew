import { Component, inject, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../../core/services/api.service';
import { AuthService } from '../../../core/services/auth.service';
import { ToastService } from '../../../core/services/toast.service';
import { ConfirmDialogService } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { Document, DocumentType, CrewMember } from '../../../core/models';
import { AutocompleteComponent, AutocompleteOption } from '../../../shared/components/autocomplete/autocomplete.component';

@Component({
  selector: 'app-document-list', standalone: true,
  imports: [CommonModule, FormsModule, RelativeTimePipe, AutocompleteComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div><h1 class="page-title">Documents</h1><p class="page-subtitle">Upload, download, and manage files</p></div>
        <button class="btn btn-primary" (click)="showUploadModal.set(true)" id="btn-upload-doc">
          <span class="material-icons-round">upload_file</span> Upload Document
        </button>
      </div>

      <!-- Filters -->
      <div class="filters-bar">
        <select class="form-select filter-select" [(ngModel)]="filterType" (ngModelChange)="applyFilter()" id="filter-doc-type">
          <option value="">All Types</option>
          @for (t of docTypes; track t.value) { <option [value]="t.value">{{ t.label }}</option> }
        </select>

        <div style="width:240px;position:relative;z-index:10;">
          <app-autocomplete
            [ngModel]="filterCrewMemberId()"
            (ngModelChange)="filterCrewMemberId.set($event); applyFilter()"
            [options]="crewOptions()"
            placeholder="Filter by owner..."
            inputId="filter-doc-owner"
          ></app-autocomplete>
        </div>

        @if (filterType || filterCrewMemberId()) {
          <button class="btn btn-ghost btn-sm" (click)="filterType=''; filterCrewMemberId.set(''); applyFilter()" style="color:var(--color-text-muted);">
            <span class="material-icons-round" style="font-size:16px;">close</span> Clear Filters
          </button>
        }
      </div>

      <!-- Stats -->
      <div class="stats-grid" style="grid-template-columns:repeat(auto-fit,minmax(140px,1fr));margin-bottom:var(--space-lg);">
        <div class="stat-card"><div class="stat-icon" style="background:rgba(0,210,255,0.12);color:var(--color-accent);"><span class="material-icons-round">folder</span></div><div class="stat-value">{{ items().length }}</div><div class="stat-label">Total Files</div></div>
        <div class="stat-card"><div class="stat-icon" style="background:rgba(168,85,247,0.12);color:#a855f7;"><span class="material-icons-round">storage</span></div><div class="stat-value">{{ totalSize() }}</div><div class="stat-label">Total Size</div></div>
        <div class="stat-card"><div class="stat-icon" style="background:var(--color-success-light);color:var(--color-success);"><span class="material-icons-round">category</span></div><div class="stat-value">{{ uniqueTypes() }}</div><div class="stat-label">Types</div></div>
      </div>

      @if (loading()) { @for(i of [1,2,3];track i){<div class="skeleton" style="height:56px;margin:4px 0;"></div>} }
      @else if (filtered().length === 0) {
        <div class="empty-state"><span class="material-icons-round empty-icon">folder_open</span>
          <div class="empty-title">{{ filterType ? 'No matching documents' : 'No documents uploaded' }}</div>
          <div class="empty-description">Upload KYC documents, vehicle logbooks, and other files.</div>
        </div>
      } @else {
         <div class="data-table-wrapper"><table class="data-table"><thead><tr>
          <th>File</th><th>Type</th><th>Owner</th><th>Size</th><th>Uploaded</th><th>Actions</th>
        </tr></thead><tbody>
          <!-- KYC grouped rows: one per employee -->
          @for(g of kycGroups();track g.crewMemberId){
            <tr class="kyc-group-row">
              <td>
                <div style="display:flex;align-items:center;gap:var(--space-sm);">
                  <span class="material-icons-round" style="color:#6366f1;font-size:20px;">badge</span>
                  <div>
                    <span style="font-weight:600;color:var(--color-text-primary);">KYC Documents</span>
                    <div style="font-size:0.72rem;color:var(--color-text-muted);">{{ g.docs.length }} file(s): {{ g.docTypeLabels }}</div>
                  </div>
                </div>
              </td>
              <td>
                <span class="badge" [class.badge-success]="g.kycStatus === 'VERIFIED'" [class.badge-warning]="g.kycStatus === 'PENDING'" [class.badge-danger]="g.kycStatus === 'REJECTED'" [class.badge-accent]="!g.kycStatus">{{ g.kycStatus || 'PENDING' }}</span>
              </td>
              <td style="font-weight:500;color:var(--color-text-primary);">{{ g.ownerName }}</td>
              <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{ formatSize(g.totalSize) }}</td>
              <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{ g.latestUpload | relativeTime }}</td>
              <td><div style="display:flex;gap:4px;flex-wrap:wrap;">
                @for (doc of g.docs; track doc.id) {
                  <button class="btn btn-sm btn-secondary" (click)="download(doc)" [disabled]="downloading()" [title]="doc.document_type">
                    <span class="material-icons-round" style="font-size:14px;">download</span> {{ doc.document_type === 'KYC_ID_FRONT' ? 'Front' : doc.document_type === 'KYC_ID_BACK' ? 'Back' : 'Selfie' }}
                  </button>
                }
                @if (g.kycStatus !== 'VERIFIED') {
                  <button class="btn btn-sm btn-success" (click)="verifyKYC(g.docs[0])" [disabled]="verifyingKYC()" title="Approve KYC">
                    <span class="material-icons-round" style="font-size:14px;">verified</span> Verify
                  </button>
                  <button class="btn btn-sm btn-warning" (click)="rejectKYC(g.docs[0])" [disabled]="verifyingKYC()" title="Reject KYC">
                    <span class="material-icons-round" style="font-size:14px;">cancel</span> Reject
                  </button>
                } @else {
                  <span class="badge badge-success" style="padding:6px 12px;"><span class="material-icons-round" style="font-size:14px;">check_circle</span> Verified</span>
                  <button class="btn btn-sm btn-outline-danger" (click)="unverifyKYC(g.docs[0])" [disabled]="verifyingKYC()" title="Revoke verification">
                    <span class="material-icons-round" style="font-size:14px;">remove_circle</span> Unverify
                  </button>
                }
              </div></td>
            </tr>
          }
          <!-- Non-KYC documents: individual rows -->
          @for(d of nonKYCDocs();track d.id){<tr>
            <td>
              <div style="display:flex;align-items:center;gap:var(--space-sm);">
                <span class="material-icons-round file-icon" [style.color]="mimeColor(d.mime_type)">{{ mimeIcon(d.mime_type) }}</span>
                <span style="font-weight:500;color:var(--color-text-primary);">{{ d.file_name }}</span>
              </div>
            </td>
            <td><span class="badge badge-accent">{{ d.document_type }}</span></td>
            <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{ ownerName(d) }}</td>
            <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{ formatSize(d.file_size) }}</td>
            <td style="font-size:0.8125rem;color:var(--color-text-muted);">{{ d.created_at | relativeTime }}</td>
            <td><div style="display:flex;gap:4px;flex-wrap:wrap;">
              <button class="btn btn-sm btn-secondary" (click)="download(d)" [disabled]="downloading()">
                <span class="material-icons-round" style="font-size:14px;">download</span> Download
              </button>
              <button class="btn btn-sm btn-danger" (click)="deleteDoc(d)">
                <span class="material-icons-round" style="font-size:14px;">delete</span>
              </button>
            </div></td>
          </tr>}</tbody></table></div>
      }

      <!-- Upload Modal (Task 163) -->
      @if (showUploadModal()) {
        <div class="modal-backdrop" (click)="showUploadModal.set(false)"><div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header"><h3>Upload Document</h3><button class="btn btn-ghost btn-icon" (click)="showUploadModal.set(false)"><span class="material-icons-round">close</span></button></div>
          <div class="modal-body">
            <div class="form-group"><label class="form-label">Document Type *</label>
              <select class="form-select" [(ngModel)]="uploadForm.document_type" required>
                @for (t of docTypes; track t.value) { <option [value]="t.value">{{ t.label }}</option> }
              </select>
            </div>
            <div class="form-group"><label class="form-label">Crew Member (optional)</label>
              <app-autocomplete
                [(ngModel)]="uploadForm.crew_member_id"
                [options]="crewOptions()"
                placeholder="Search crew members..."
              ></app-autocomplete>
            </div>
            <div class="form-group"><label class="form-label">File *</label>
              <div class="drop-zone" [class.has-file]="selectedFile" (click)="fileInput.click()" (dragover)="$event.preventDefault()" (drop)="onDrop($event)">
                <input #fileInput type="file" hidden (change)="onFileSelect($event)" />
                @if (selectedFile) {
                  <span class="material-icons-round" style="color:var(--color-success);font-size:32px;">check_circle</span>
                  <span class="drop-text">{{ selectedFile.name }} ({{ formatSize(selectedFile.size) }})</span>
                } @else {
                  <span class="material-icons-round" style="color:var(--color-text-muted);font-size:32px;">cloud_upload</span>
                  <span class="drop-text">Click or drag file here</span>
                }
              </div>
            </div>
            @if (uploadProgress() > 0 && uploadProgress() < 100) {
              <div class="progress-track"><div class="progress-fill" [style.width.%]="uploadProgress()"></div></div>
            }
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" (click)="showUploadModal.set(false)">Cancel</button>
            <button class="btn btn-primary" (click)="submitUpload()" [disabled]="uploading()||!selectedFile">{{ uploading() ? 'Uploading...' : 'Upload' }}</button>
          </div>
        </div></div>
      }
    </div>`,
  styles: [`
    .filters-bar{display:flex;gap:var(--space-sm);flex-wrap:wrap;margin-bottom:var(--space-lg);align-items:center;}
    .filter-select{min-width:180px;max-width:240px;}
    .file-icon{font-size:20px;}
    .drop-zone{
      border:2px dashed var(--color-border);border-radius:var(--radius-md);
      padding:var(--space-xl) var(--space-lg);display:flex;flex-direction:column;
      align-items:center;gap:var(--space-sm);cursor:pointer;transition:all var(--transition-fast);
    }
    .drop-zone:hover,.drop-zone.has-file{border-color:var(--color-accent);background:rgba(0,210,255,0.04);}
    .drop-text{font-size:0.8125rem;color:var(--color-text-muted);}
    .progress-track{height:6px;border-radius:3px;background:rgba(255,255,255,0.06);overflow:hidden;margin-top:var(--space-sm);}
    .progress-fill{height:100%;border-radius:3px;background:var(--gradient-accent);transition:width 0.3s;}
    /* KYC group row highlight */
    .kyc-group-row { background: rgba(99,102,241,0.03); }
    .kyc-group-row:hover { background: rgba(99,102,241,0.07); }
    .badge-success { background: rgba(34,197,94,0.12); color: #22c55e; }
    .badge-warning { background: rgba(245,158,11,0.12); color: #f59e0b; }
    .badge-danger { background: rgba(239,68,68,0.12); color: #ef4444; }
    .btn-outline-danger {
      background: transparent; color: #ef4444; border: 1px solid rgba(239,68,68,0.4);
      transition: all var(--transition-fast);
    }
    .btn-outline-danger:hover:not(:disabled) { background: rgba(239,68,68,0.1); border-color: #ef4444; }
  `]
})
export class DocumentListComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmDialogService);

  items = signal<Document[]>([]);
  crewMembers = signal<CrewMember[]>([]);
  loading = signal(true);
  showUploadModal = signal(false);
  uploading = signal(false);
  downloading = signal(false);
  verifyingKYC = signal(false);
  uploadProgress = signal(0);
  filterType = '';
  filterCrewMemberId = signal('');
  selectedFile: File | null = null;

  uploadForm = { document_type: 'OTHER' as DocumentType, crew_member_id: '' };

  crewOptions = computed<AutocompleteOption[]>(() => {
    return this.crewMembers().map(c => ({
      value: c.id,
      label: `${c.first_name} ${c.last_name}`,
      sublabel: `ID: ${c.crew_id || ''}`,
      badge: c.role,
      searchText: `${c.first_name} ${c.last_name} ${c.crew_id || ''}`
    }));
  });

  /** Map crew member IDs to names for quick lookup in the Owner column */
  private crewNameMap = computed<Record<string, string>>(() => {
    const map: Record<string, string> = {};
    for (const c of this.crewMembers()) {
      map[c.id] = `${c.first_name} ${c.last_name}`;
    }
    return map;
  });

  readonly docTypes = [
    { value: 'KYC_ID_FRONT', label: 'KYC ID Front' },
    { value: 'KYC_ID_BACK', label: 'KYC ID Back' },
    { value: 'KYC_SELFIE', label: 'KYC Selfie' },
    { value: 'SACCO_REGISTRATION', label: 'Organization Registration' },
    { value: 'VEHICLE_LOGBOOK', label: 'Vehicle Logbook' },
    { value: 'OTHER', label: 'Other' },
  ];

  ngOnInit() {
    this.load();
    this.api.getCrewMembers({ per_page: '200' }).subscribe({ next: r => this.crewMembers.set(r.data) });
  }

  load(): void {
    this.loading.set(true);
    const params: Record<string, string> = { per_page: '200' };
    if (this.filterType) params['document_type'] = this.filterType;
    if (this.filterCrewMemberId()) params['crew_member_id'] = this.filterCrewMemberId();

    this.api.getDocuments(params).subscribe({
      next: r => { this.items.set(r.data); this.loading.set(false); },
      error: () => this.loading.set(false),
    });
  }

  filtered = computed(() => {
    const t = this.filterType;
    const crewId = this.filterCrewMemberId();
    let list = this.items();
    if (t) list = list.filter(d => d.document_type === t);
    if (crewId) list = list.filter(d => d.crew_member_id === crewId);
    return list;
  });

  /** Group KYC documents by employee — one row per employee */
  kycGroups = computed(() => {
    const kycDocs = this.filtered().filter(d => this.isKYCDoc(d) && d.crew_member_id);
    const grouped = new Map<string, Document[]>();
    for (const doc of kycDocs) {
      const key = doc.crew_member_id!;
      if (!grouped.has(key)) grouped.set(key, []);
      grouped.get(key)!.push(doc);
    }
    const crewMap = this.crewNameMap();
    const crewList = this.crewMembers();
    return Array.from(grouped.entries()).map(([crewMemberId, docs]) => {
      const crew = crewList.find(c => c.id === crewMemberId);
      // Deduplicate: keep only the latest doc per type
      const latestByType = new Map<string, Document>();
      for (const d of docs) {
        const existing = latestByType.get(d.document_type);
        if (!existing || new Date(d.created_at) > new Date(existing.created_at)) {
          latestByType.set(d.document_type, d);
        }
      }
      const uniqueDocs = Array.from(latestByType.values());
      return {
        crewMemberId,
        ownerName: crewMap[crewMemberId] || crewMemberId.slice(0, 8) + '…',
        kycStatus: crew?.kyc_status || 'PENDING',
        docs: uniqueDocs,
        docTypeLabels: uniqueDocs.map(d => d.document_type.replace('KYC_', '').replace('_', ' ')).join(', '),
        totalSize: uniqueDocs.reduce((sum, d) => sum + d.file_size, 0),
        latestUpload: uniqueDocs.reduce((latest, d) => new Date(d.created_at) > new Date(latest) ? d.created_at : latest, uniqueDocs[0].created_at),
      };
    });
  });

  /** Non-KYC documents — shown as individual rows */
  nonKYCDocs = computed(() => {
    return this.filtered().filter(d => !this.isKYCDoc(d));
  });

  totalSize = computed(() => this.formatSize(this.items().reduce((s, d) => s + d.file_size, 0)));
  uniqueTypes = computed(() => new Set(this.items().map(d => d.document_type)).size);

  applyFilter(): void { this.load(); }

  // --- KYC Verification ---
  isKYCDoc(d: Document): boolean {
    return d.document_type === 'KYC_ID_FRONT' || d.document_type === 'KYC_ID_BACK' || d.document_type === 'KYC_SELFIE';
  }

  ownerName(d: Document): string {
    if (!d.crew_member_id) return '—';
    return this.crewNameMap()[d.crew_member_id] || d.crew_member_id.slice(0, 8) + '…';
  }

  verifyKYC(d: Document): void {
    if (!d.crew_member_id) return;
    this.confirm.confirm('Approve KYC', 'Verify this employee\'s identity documents? This will grant them full system access.', {
      confirmText: 'Approve', icon: 'verified_user',
    }).subscribe(r => {
      if (!r.confirmed) return;
      this.verifyingKYC.set(true);
      this.api.updateKYC(d.crew_member_id!, { kyc_status: 'VERIFIED' }).subscribe({
        next: () => {
          this.verifyingKYC.set(false);
          this.toast.success('KYC verified successfully');
          this.load();
          // Reload crew members to get updated kyc_status
          this.api.getCrewMembers({ per_page: '200' }).subscribe({ next: r => this.crewMembers.set(r.data) });
        },
        error: () => { this.verifyingKYC.set(false); this.toast.error('KYC verification failed'); },
      });
    });
  }

  rejectKYC(d: Document): void {
    if (!d.crew_member_id) return;
    this.confirm.confirm('Reject KYC', 'Reject this employee\'s identity documents? They will need to re-submit.', {
      confirmText: 'Reject', variant: 'danger', icon: 'cancel',
    }).subscribe(r => {
      if (!r.confirmed) return;
      this.verifyingKYC.set(true);
      this.api.updateKYC(d.crew_member_id!, { kyc_status: 'REJECTED' }).subscribe({
        next: () => {
          this.verifyingKYC.set(false);
          this.toast.warning('KYC rejected — employee will be notified');
          this.load();
          this.api.getCrewMembers({ per_page: '200' }).subscribe({ next: r => this.crewMembers.set(r.data) });
        },
        error: () => { this.verifyingKYC.set(false); this.toast.error('KYC rejection failed'); },
      });
    });
  }

  unverifyKYC(d: Document): void {
    if (!d.crew_member_id) return;
    this.confirm.prompt(
      'Revoke KYC Verification',
      'This will unverify the employee and revoke their verified status. They will be notified with the reason you provide.',
      {
        confirmText: 'Unverify', variant: 'danger', icon: 'remove_circle',
        promptLabel: 'Reason for un-verification *',
        promptPlaceholder: 'e.g. Expired ID, Fraudulent documents, Updated information required...',
      }
    ).subscribe(r => {
      if (!r.confirmed) return;
      const reason = (r.value || '').trim();
      if (!reason) {
        this.toast.error('A reason is required to unverify a user');
        return;
      }
      this.verifyingKYC.set(true);
      this.api.updateKYC(d.crew_member_id!, { kyc_status: 'PENDING', reason }).subscribe({
        next: () => {
          this.verifyingKYC.set(false);
          this.toast.warning('KYC verification revoked — employee has been notified');
          this.load();
          this.api.getCrewMembers({ per_page: '200' }).subscribe({ next: r => this.crewMembers.set(r.data) });
        },
        error: () => { this.verifyingKYC.set(false); this.toast.error('Failed to revoke KYC verification'); },
      });
    });
  }

  // --- Upload (Task 163) ---
  onFileSelect(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.selectedFile = input.files?.[0] || null;
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    this.selectedFile = event.dataTransfer?.files?.[0] || null;
  }

  submitUpload(): void {
    if (!this.selectedFile) return;
    this.uploading.set(true);
    this.uploadProgress.set(10);

    const fd = new FormData();
    fd.append('file', this.selectedFile);
    fd.append('document_type', this.uploadForm.document_type);
    if (this.uploadForm.crew_member_id) fd.append('crew_member_id', this.uploadForm.crew_member_id);

    this.uploadProgress.set(50);
    this.api.uploadDocument(fd).subscribe({
      next: () => {
        this.uploadProgress.set(100);
        this.toast.success('Document uploaded');
        this.showUploadModal.set(false);
        this.uploading.set(false);
        this.selectedFile = null;
        this.uploadForm = { document_type: 'OTHER', crew_member_id: '' };
        this.uploadProgress.set(0);
        this.load();
      },
      error: () => { this.uploading.set(false); this.uploadProgress.set(0); },
    });
  }

  // --- Download (Task 164) ---
  download(d: Document): void {
    this.downloading.set(true);
    this.api.downloadDocument(d.id).subscribe({
      next: r => {
        this.downloading.set(false);
        const url = r.data?.download_url;
        if (url) {
          window.open(url, '_blank');
        } else {
          this.toast.error('Download URL not available');
        }
      },
      error: () => { this.downloading.set(false); this.toast.error('Download failed — storage may be unavailable'); },
    });
  }

  // --- Delete (Task 165) ---
  deleteDoc(d: Document): void {
    this.confirm.danger('Delete Document', `Delete "${d.file_name}"? This action cannot be undone.`).subscribe(r => {
      if (!r.confirmed) return;
      this.api.deleteDocument(d.id).subscribe({
        next: () => { this.toast.success('Document deleted'); this.load(); },
      });
    });
  }

  // --- Helpers ---
  formatSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  mimeIcon(mime: string): string {
    if (mime?.startsWith('image/')) return 'image';
    if (mime?.includes('pdf')) return 'picture_as_pdf';
    if (mime?.includes('spreadsheet') || mime?.includes('csv') || mime?.includes('excel')) return 'table_chart';
    return 'description';
  }

  mimeColor(mime: string): string {
    if (mime?.startsWith('image/')) return '#22c55e';
    if (mime?.includes('pdf')) return '#ef4444';
    if (mime?.includes('spreadsheet') || mime?.includes('csv')) return '#22c55e';
    return 'var(--color-accent)';
  }
}
