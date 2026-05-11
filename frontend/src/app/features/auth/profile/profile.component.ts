import { Component, inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AuthService } from '../../../core/services/auth.service';
import { ApiService } from '../../../core/services/api.service';
import { ToastService } from '../../../core/services/toast.service';
import { User, CrewProfile } from '../../../core/models';

@Component({
  selector: 'app-profile',
  standalone: true,
  imports: [CommonModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="animate-fade-in">
      <div class="page-header">
        <div>
          <h1 class="page-title">My Profile</h1>
          <p class="page-subtitle">Manage your account, job specialization &amp; identity verification</p>
        </div>
      </div>

      @if (loading()) {
        <div class="profile-grid">
          <div class="skeleton" style="height: 320px;"></div>
          <div class="skeleton" style="height: 320px;"></div>
        </div>
      } @else if (user()) {
        <!-- KYC Banner -->
        @if (user()!.crew_profile && user()!.crew_profile!.kyc_status !== 'VERIFIED') {
          <div class="kyc-banner" [class.kyc-pending]="user()!.crew_profile!.kyc_status === 'PENDING'" [class.kyc-rejected]="user()!.crew_profile!.kyc_status === 'REJECTED'">
            <span class="material-icons-round kyc-banner-icon">
              {{ user()!.crew_profile!.kyc_status === 'REJECTED' ? 'gpp_bad' : 'verified_user' }}
            </span>
            <div class="kyc-banner-text">
              <strong>{{ user()!.crew_profile!.kyc_status === 'REJECTED' ? 'Identity Verification Rejected' : 'Identity Verification Required' }}</strong>
              <span>{{ user()!.crew_profile!.kyc_status === 'REJECTED'
                ? 'Your ID verification was rejected. Please re-submit with correct details.'
                : 'Verify your National ID to unlock withdrawals, transfers and loan applications.' }}</span>
            </div>
            @if (user()!.kyc_restrictions && user()!.kyc_restrictions!.length) {
              <div class="kyc-restricted-list">
                <span class="kyc-restricted-label">Restricted:</span>
                @for (action of user()!.kyc_restrictions!; track action) {
                  <span class="badge badge-danger badge-sm">{{ formatAction(action) }}</span>
                }
              </div>
            }
          </div>
        }

        <div class="profile-grid">
          <!-- Profile Card -->
          <div class="glass-card profile-card">
            <div class="profile-header">
              <div class="profile-avatar">{{ userInitials() }}</div>
              <div class="profile-identity">
                <span class="profile-name">{{ user()!.crew_profile ? user()!.crew_profile!.full_name : user()!.phone }}</span>
                <span class="badge badge-accent">{{ formatRole(user()!.system_role) }}</span>
              </div>
            </div>

            <div class="detail-rows">
              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">badge</span> User ID
                </span>
                <code class="detail-value text-accent">{{ user()!.id | slice:0:8 }}…</code>
              </div>

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">phone</span> Phone
                </span>
                <span class="detail-value">{{ user()!.phone }}</span>
              </div>

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">mail</span> Email
                </span>
                @if (editingEmail()) {
                  <div class="inline-edit">
                    <input class="form-input" [(ngModel)]="emailValue" placeholder="john@example.com" style="max-width: 220px;" id="profile-email-input" />
                    <button class="btn btn-primary btn-sm" (click)="saveEmail()" [disabled]="savingEmail()" id="profile-email-save">
                      @if (savingEmail()) { <span class="spinner-sm"></span> } @else { Save }
                    </button>
                    <button class="btn btn-ghost btn-sm" (click)="cancelEmailEdit()">Cancel</button>
                  </div>
                } @else {
                  <div class="inline-edit">
                    <span class="detail-value">{{ user()!.email || '—' }}</span>
                    <button class="btn btn-ghost btn-sm" (click)="startEmailEdit()" id="profile-email-edit">
                      <span class="material-icons-round" style="font-size:16px;">edit</span>
                    </button>
                  </div>
                }
              </div>

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">security</span> Role
                </span>
                <span class="detail-value">{{ formatRole(user()!.system_role) }}</span>
              </div>

              @if (user()!.crew_member_id) {
                <div class="detail-row">
                  <span class="detail-label">
                    <span class="material-icons-round detail-icon">groups</span> Member ID
                  </span>
                  <code class="detail-value text-accent">{{ user()!.crew_member_id! | slice:0:8 }}…</code>
                </div>
              }

              @if (user()!.organization_id) {
                <div class="detail-row">
                  <span class="detail-label">
                    <span class="material-icons-round detail-icon">business</span> Organization ID
                  </span>
                  <code class="detail-value text-accent">{{ user()!.organization_id! | slice:0:8 }}…</code>
                </div>
              }

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">calendar_today</span> Joined
                </span>
                <span class="detail-value">{{ user()!.created_at | date:'mediumDate' }}</span>
              </div>

              @if (user()!.last_login_at) {
                <div class="detail-row">
                  <span class="detail-label">
                    <span class="material-icons-round detail-icon">login</span> Last Login
                  </span>
                  <span class="detail-value">{{ user()!.last_login_at | date:'medium' }}</span>
                </div>
              }

              <div class="detail-row">
                <span class="detail-label">
                  <span class="material-icons-round detail-icon">circle</span> Status
                </span>
                <span class="badge" [ngClass]="user()!.is_active ? 'badge-success' : 'badge-danger'">
                  {{ user()!.is_active ? 'Active' : 'Inactive' }}
                </span>
              </div>
            </div>
          </div>

          <!-- Right Column -->
          <div class="right-column">
            <!-- Job / Specialization Card -->
            @if (user()!.crew_profile) {
              <div class="glass-card job-card">
                <h3 class="card-title">
                  <span class="material-icons-round" style="font-size: 20px; color: var(--color-accent);">work</span>
                  Job / Specialization
                </h3>
                <div class="detail-rows">
                  <div class="detail-row">
                    <span class="detail-label">Role</span>
                    @if (editingJob()) {
                      <select class="form-input form-select-sm" [(ngModel)]="jobRole" id="profile-job-role">
                        <option value="DRIVER">Driver</option>
                        <option value="CONDUCTOR">Conductor</option>
                        <option value="RIDER">Rider</option>
                        <option value="OTHER">Other</option>
                      </select>
                    } @else {
                      <span class="detail-value">{{ formatRole(user()!.crew_profile!.role) }}</span>
                    }
                  </div>
                  <div class="detail-row">
                    <span class="detail-label">Job Title</span>
                    @if (editingJob()) {
                      <input class="form-input" [(ngModel)]="jobTitle" placeholder="e.g. Plumber, Electrician" style="max-width: 200px;" id="profile-job-title" />
                    } @else {
                      <span class="detail-value">{{ user()!.crew_profile!.job_title || '—' }}</span>
                    }
                  </div>
                </div>
                <div class="card-actions">
                  @if (editingJob()) {
                    <button class="btn btn-primary btn-sm" (click)="saveJob()" [disabled]="savingJob()" id="profile-job-save">
                      @if (savingJob()) { <span class="spinner-sm"></span> } @else { Save }
                    </button>
                    <button class="btn btn-ghost btn-sm" (click)="editingJob.set(false)">Cancel</button>
                  } @else {
                    <button class="btn btn-secondary btn-sm" (click)="startJobEdit()" id="profile-job-edit">
                      <span class="material-icons-round" style="font-size:16px;">edit</span> Edit
                    </button>
                  }
                </div>
              </div>
            }

            <!-- KYC Verification Card -->
            @if (user()!.crew_profile) {
              <div class="glass-card kyc-card">
                <h3 class="card-title">
                  <span class="material-icons-round" style="font-size: 20px;" [style.color]="kycColor()">verified_user</span>
                  Identity Verification (KYC)
                </h3>
                <div class="kyc-status-row">
                  <span class="kyc-status-badge" [class.verified]="user()!.crew_profile!.kyc_status === 'VERIFIED'" [class.rejected]="user()!.crew_profile!.kyc_status === 'REJECTED'" [class.pending]="user()!.crew_profile!.kyc_status === 'PENDING'">
                    <span class="material-icons-round" style="font-size: 16px;">
                      {{ user()!.crew_profile!.kyc_status === 'VERIFIED' ? 'check_circle' : user()!.crew_profile!.kyc_status === 'REJECTED' ? 'cancel' : 'pending' }}
                    </span>
                    {{ user()!.crew_profile!.kyc_status }}
                  </span>
                  @if (user()!.crew_profile!.kyc_verified_at) {
                    <span class="detail-value" style="font-size: 0.75rem;">Verified {{ user()!.crew_profile!.kyc_verified_at | date:'mediumDate' }}</span>
                  }
                </div>

                @if (user()!.crew_profile!.kyc_status !== 'VERIFIED') {
                  <!-- KYC Mode Toggle -->
                  <div class="kyc-mode-toggle">
                    <button class="kyc-mode-btn" [class.active]="kycActiveMode() === 'UPLOAD'" (click)="kycActiveMode.set('UPLOAD')" id="kyc-mode-upload">
                      <span class="material-icons-round" style="font-size:16px;">cloud_upload</span> Upload ID
                    </button>
                    <button class="kyc-mode-btn" [class.active]="kycActiveMode() === 'MANUAL'" (click)="kycActiveMode.set('MANUAL')" id="kyc-mode-manual">
                      <span class="material-icons-round" style="font-size:16px;">edit_note</span> Enter ID Details
                    </button>
                  </div>

                  <!-- UPLOAD Mode -->
                  @if (kycActiveMode() === 'UPLOAD') {
                    <div class="kyc-form">
                      <p class="kyc-hint">Upload clear photos of the front and back of your Kenyan National ID.</p>
                      <div class="form-group">
                        <label class="form-label" for="kyc-upload-national-id">National ID Number</label>
                        <input class="form-input" id="kyc-upload-national-id" [(ngModel)]="kycNationalId" placeholder="e.g. 12345678" maxlength="10" />
                      </div>
                      <div class="kyc-upload-grid">
                        <div class="kyc-upload-zone" (click)="idFrontInput.click()" (dragover)="$event.preventDefault()" (drop)="onFileDrop($event, 'front')">
                          <input #idFrontInput type="file" accept="image/*" hidden (change)="onFileSelected($event, 'front')" id="kyc-id-front-input" />
                          @if (idFrontPreview()) {
                            <img [src]="idFrontPreview()!" alt="ID Front" class="kyc-upload-preview" />
                            <span class="kyc-upload-change">Change</span>
                          } @else {
                            <span class="material-icons-round kyc-upload-icon">add_a_photo</span>
                            <span class="kyc-upload-label">Front of ID</span>
                            <span class="kyc-upload-sublabel">Click or drag to upload</span>
                          }
                        </div>
                        <div class="kyc-upload-zone" (click)="idBackInput.click()" (dragover)="$event.preventDefault()" (drop)="onFileDrop($event, 'back')">
                          <input #idBackInput type="file" accept="image/*" hidden (change)="onFileSelected($event, 'back')" id="kyc-id-back-input" />
                          @if (idBackPreview()) {
                            <img [src]="idBackPreview()!" alt="ID Back" class="kyc-upload-preview" />
                            <span class="kyc-upload-change">Change</span>
                          } @else {
                            <span class="material-icons-round kyc-upload-icon">add_a_photo</span>
                            <span class="kyc-upload-label">Back of ID</span>
                            <span class="kyc-upload-sublabel">Click or drag to upload</span>
                          }
                        </div>
                      </div>
                      <button class="btn btn-primary" (click)="submitKYCUpload()" [disabled]="submittingKYC() || !kycNationalId || !idFrontFile()" id="btn-submit-kyc-upload">
                        @if (submittingKYC()) { <span class="spinner-sm"></span> Uploading… } @else {
                          <span class="material-icons-round" style="font-size:16px;">cloud_upload</span> Upload &amp; Verify
                        }
                      </button>
                    </div>
                  }

                  <!-- MANUAL Mode -->
                  @if (kycActiveMode() === 'MANUAL') {
                    <div class="kyc-form">
                      <p class="kyc-hint">Enter your Kenyan National ID number and serial number for IPRS verification.</p>
                      <div class="form-group">
                        <label class="form-label" for="kyc-national-id">National ID Number</label>
                        <input class="form-input" id="kyc-national-id" [(ngModel)]="kycNationalId" placeholder="e.g. 12345678" maxlength="10" />
                      </div>
                      <div class="form-group">
                        <label class="form-label" for="kyc-serial">ID Serial Number</label>
                        <input class="form-input" id="kyc-serial" [(ngModel)]="kycSerialNumber" placeholder="Serial number on your ID" />
                      </div>
                      <button class="btn btn-primary" (click)="submitKYC()" [disabled]="submittingKYC() || !kycNationalId || !kycSerialNumber" id="btn-submit-kyc">
                        @if (submittingKYC()) { <span class="spinner-sm"></span> Verifying… } @else {
                          <span class="material-icons-round" style="font-size:16px;">fact_check</span> Verify Identity
                        }
                      </button>
                    </div>
                  }
                }
              </div>
            }

            <!-- Security Card -->
            <div class="glass-card security-card">
              <h3 class="card-title">
                <span class="material-icons-round" style="font-size: 20px; color: var(--color-accent);">shield</span>
                Security
              </h3>
              <div class="security-section">
                <div class="security-item">
                  <div class="security-info">
                    <span class="security-label">Password</span>
                    <span class="security-description">Change your account password to keep your account secure.</span>
                  </div>
                  <button class="btn btn-secondary btn-sm" (click)="showPasswordModal.set(true)" id="btn-change-password">
                    <span class="material-icons-round" style="font-size:16px;">lock</span> Change Password
                  </button>
                </div>
                <div class="security-divider"></div>
                <div class="security-item">
                  <div class="security-info">
                    <span class="security-label">Session</span>
                    <span class="security-description">Sign out of your account on this device.</span>
                  </div>
                  <button class="btn btn-danger btn-sm" (click)="auth.logout()" id="btn-logout-profile">
                    <span class="material-icons-round" style="font-size:16px;">logout</span> Sign Out
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      }

      <!-- Change Password Modal -->
      @if (showPasswordModal()) {
        <div class="modal-backdrop" (click)="showPasswordModal.set(false)">
          <div class="modal-content" (click)="$event.stopPropagation()">
            <div class="modal-header">
              <h3>Change Password</h3>
              <button class="btn btn-ghost btn-icon" (click)="showPasswordModal.set(false)">
                <span class="material-icons-round">close</span>
              </button>
            </div>
            <form class="modal-body" (ngSubmit)="changePassword()" id="change-password-form">
              <div class="form-group">
                <label class="form-label" for="cp-current">Current Password</label>
                <input class="form-input" id="cp-current" type="password" placeholder="Enter current password"
                       [(ngModel)]="currentPassword" name="currentPassword" required autocomplete="current-password" />
              </div>
              <div class="form-group">
                <label class="form-label" for="cp-new">New Password</label>
                <input class="form-input" id="cp-new" type="password" placeholder="Min 8 characters"
                       [(ngModel)]="newPassword" name="newPassword" required minlength="8" autocomplete="new-password" />
              </div>
              <div class="form-group">
                <label class="form-label" for="cp-confirm">Confirm New Password</label>
                <input class="form-input" id="cp-confirm" type="password" placeholder="Re-enter new password"
                       [(ngModel)]="confirmPassword" name="confirmPassword" required autocomplete="new-password" />
                @if (confirmPassword && newPassword !== confirmPassword) {
                  <span class="form-error">Passwords do not match</span>
                }
              </div>
            </form>
            <div class="modal-footer">
              <button class="btn btn-secondary" (click)="showPasswordModal.set(false)">Cancel</button>
              <button class="btn btn-primary" (click)="changePassword()" [disabled]="changingPassword()" id="submit-change-password">
                @if (changingPassword()) {
                  <span class="spinner-sm"></span> Changing...
                } @else {
                  <span class="material-icons-round" style="font-size:16px;">lock</span> Update Password
                }
              </button>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .profile-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: var(--space-lg);
    }
    @media (max-width: 900px) {
      .profile-grid { grid-template-columns: 1fr; }
    }
    .right-column { display: flex; flex-direction: column; gap: var(--space-lg); }
    .profile-card, .security-card, .job-card, .kyc-card { padding: var(--space-lg) !important; }
    .profile-header {
      display: flex; align-items: center; gap: var(--space-lg);
      padding-bottom: var(--space-lg); margin-bottom: var(--space-md);
      border-bottom: 1px solid var(--color-border);
    }
    .profile-avatar {
      width: 64px; height: 64px; border-radius: var(--radius-lg);
      background: var(--gradient-accent); display: flex; align-items: center;
      justify-content: center; font-size: 1.25rem; font-weight: 800;
      color: var(--color-text-inverse); flex-shrink: 0;
    }
    .profile-identity { display: flex; flex-direction: column; gap: 4px; }
    .profile-name {
      font-family: var(--font-heading); font-size: 1.25rem;
      font-weight: 700; color: var(--color-text-primary);
    }
    .detail-rows { display: flex; flex-direction: column; gap: 0; }
    .detail-row {
      display: flex; align-items: center; justify-content: space-between;
      padding: 10px 0; border-bottom: 1px solid var(--color-border); gap: var(--space-md);
      &:last-child { border-bottom: none; }
    }
    .detail-label {
      font-size: 0.8125rem; color: var(--color-text-muted);
      display: flex; align-items: center; gap: 6px; white-space: nowrap;
    }
    .detail-icon { font-size: 16px; }
    .detail-value {
      font-size: 0.875rem; color: var(--color-text-primary);
      font-weight: 500; text-align: right;
    }
    .inline-edit { display: flex; align-items: center; gap: var(--space-xs); }
    .card-title {
      display: flex; align-items: center; gap: var(--space-sm);
      font-size: 1rem; font-weight: 600; margin-bottom: var(--space-lg);
      color: var(--color-text-primary);
    }
    .card-actions { margin-top: var(--space-md); display: flex; gap: var(--space-sm); }
    .form-select-sm {
      max-width: 160px; padding: 4px 8px; font-size: 0.8125rem;
      border-radius: var(--radius-sm);
    }
    /* KYC Banner */
    .kyc-banner {
      display: flex; align-items: flex-start; gap: var(--space-md);
      padding: var(--space-md) var(--space-lg); border-radius: var(--radius-md);
      margin-bottom: var(--space-lg); flex-wrap: wrap;
      background: rgba(255, 193, 7, 0.08); border: 1px solid rgba(255, 193, 7, 0.3);
    }
    .kyc-banner.kyc-rejected {
      background: rgba(244, 67, 54, 0.08); border-color: rgba(244, 67, 54, 0.3);
    }
    .kyc-banner-icon { font-size: 28px; color: #ffc107; }
    .kyc-banner.kyc-rejected .kyc-banner-icon { color: #f44336; }
    .kyc-banner-text { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 200px; }
    .kyc-banner-text strong { font-size: 0.9375rem; color: var(--color-text-primary); }
    .kyc-banner-text span { font-size: 0.8125rem; color: var(--color-text-muted); }
    .kyc-restricted-list {
      display: flex; flex-wrap: wrap; gap: 4px; align-items: center; width: 100%; margin-top: 4px;
    }
    .kyc-restricted-label { font-size: 0.75rem; color: var(--color-text-muted); margin-right: 4px; }
    .badge-sm { font-size: 0.6875rem; padding: 1px 6px; }
    /* KYC Card */
    .kyc-status-row { display: flex; align-items: center; gap: var(--space-md); margin-bottom: var(--space-md); }
    .kyc-status-badge {
      display: inline-flex; align-items: center; gap: 4px;
      padding: 4px 12px; border-radius: var(--radius-pill);
      font-size: 0.8125rem; font-weight: 600;
    }
    .kyc-status-badge.verified { background: rgba(76, 175, 80, 0.12); color: #4caf50; }
    .kyc-status-badge.rejected { background: rgba(244, 67, 54, 0.12); color: #f44336; }
    .kyc-status-badge.pending { background: rgba(255, 193, 7, 0.12); color: #ff9800; }
    .kyc-form { margin-top: var(--space-sm); }
    .kyc-hint { font-size: 0.8125rem; color: var(--color-text-muted); margin-bottom: var(--space-md); }
    /* KYC Mode Toggle */
    .kyc-mode-toggle {
      display: flex; gap: 2px; margin-bottom: var(--space-md);
      background: var(--color-surface-alt, rgba(255,255,255,0.04)); border-radius: var(--radius-md);
      padding: 3px; border: 1px solid var(--color-border);
    }
    .kyc-mode-btn {
      flex: 1; display: flex; align-items: center; justify-content: center; gap: 6px;
      padding: 8px 12px; border: none; border-radius: var(--radius-sm);
      font-size: 0.8125rem; font-weight: 600; cursor: pointer;
      background: transparent; color: var(--color-text-muted); transition: all 0.2s ease;
    }
    .kyc-mode-btn:hover { color: var(--color-text-primary); }
    .kyc-mode-btn.active {
      background: var(--gradient-accent, var(--color-accent));
      color: var(--color-text-inverse, #fff); box-shadow: 0 1px 4px rgba(0,0,0,0.15);
    }
    /* KYC Upload Zones */
    .kyc-upload-grid {
      display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); margin-bottom: var(--space-md);
    }
    @media (max-width: 480px) { .kyc-upload-grid { grid-template-columns: 1fr; } }
    .kyc-upload-zone {
      position: relative; display: flex; flex-direction: column; align-items: center; justify-content: center;
      gap: 6px; padding: var(--space-lg); border: 2px dashed var(--color-border);
      border-radius: var(--radius-md); cursor: pointer; transition: all 0.2s ease;
      min-height: 130px; background: var(--color-surface-alt, rgba(255,255,255,0.02));
    }
    .kyc-upload-zone:hover { border-color: var(--color-accent); background: rgba(var(--color-accent-rgb, 99,102,241), 0.04); }
    .kyc-upload-icon { font-size: 32px; color: var(--color-text-muted); opacity: 0.5; }
    .kyc-upload-label { font-size: 0.8125rem; font-weight: 600; color: var(--color-text-primary); }
    .kyc-upload-sublabel { font-size: 0.6875rem; color: var(--color-text-muted); }
    .kyc-upload-preview {
      width: 100%; max-height: 120px; object-fit: contain; border-radius: var(--radius-sm);
    }
    .kyc-upload-change {
      position: absolute; bottom: 6px; right: 8px; font-size: 0.6875rem;
      padding: 2px 8px; border-radius: var(--radius-pill);
      background: rgba(0,0,0,0.5); color: #fff;
    }
    /* Security */
    .security-section { display: flex; flex-direction: column; }
    .security-item {
      display: flex; align-items: center; justify-content: space-between;
      gap: var(--space-md); padding: var(--space-md) 0;
    }
    .security-info { display: flex; flex-direction: column; gap: 2px; }
    .security-label { font-size: 0.875rem; font-weight: 600; color: var(--color-text-primary); }
    .security-description { font-size: 0.75rem; color: var(--color-text-muted); }
    .security-divider { height: 1px; background: var(--color-border); }
    .spinner-sm {
      display: inline-block; width: 14px; height: 14px;
      border: 2px solid rgba(255,255,255,0.2); border-top-color: currentColor;
      border-radius: 50%; animation: spin 600ms linear infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }
    @media (max-width: 600px) {
      .security-item { flex-direction: column; align-items: flex-start; gap: var(--space-sm); }
      .detail-row { flex-direction: column; align-items: flex-start; gap: 4px; }
      .detail-value { text-align: left; }
    }
  `]
})
export class ProfileComponent implements OnInit {
  auth = inject(AuthService);
  private api = inject(ApiService);
  private toast = inject(ToastService);

  user = signal<User | null>(null);
  loading = signal(true);

  // Email edit
  editingEmail = signal(false);
  savingEmail = signal(false);
  emailValue = '';

  // Job edit
  editingJob = signal(false);
  savingJob = signal(false);
  jobRole = '';
  jobTitle = '';

  // KYC
  kycNationalId = '';
  kycSerialNumber = '';
  submittingKYC = signal(false);
  kycActiveMode = signal<'UPLOAD' | 'MANUAL'>('UPLOAD');
  idFrontFile = signal<File | null>(null);
  idBackFile = signal<File | null>(null);
  idFrontPreview = signal<string | null>(null);
  idBackPreview = signal<string | null>(null);

  // Password
  showPasswordModal = signal(false);
  changingPassword = signal(false);
  currentPassword = '';
  newPassword = '';
  confirmPassword = '';

  ngOnInit(): void {
    this.auth.fetchProfile().subscribe({
      next: (res) => {
        this.user.set(res.data);
        // Set default KYC mode from tenant config
        if (res.data.kyc_verification_mode === 'MANUAL') {
          this.kycActiveMode.set('MANUAL');
        } else {
          this.kycActiveMode.set('UPLOAD');
        }
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  userInitials(): string {
    const u = this.user();
    if (!u) return '?';
    if (u.crew_profile) {
      const f = u.crew_profile.first_name?.[0] ?? '';
      const l = u.crew_profile.last_name?.[0] ?? '';
      return (f + l).toUpperCase() || u.phone.slice(-2);
    }
    return u.phone.slice(-2);
  }

  formatRole(role: string): string {
    return role.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, l => l.toUpperCase());
  }

  formatAction(action: string): string {
    return action.replace(/_/g, ' ').toLowerCase().replace(/\b\w/g, l => l.toUpperCase());
  }

  kycColor(): string {
    const status = this.user()?.crew_profile?.kyc_status;
    if (status === 'VERIFIED') return '#4caf50';
    if (status === 'REJECTED') return '#f44336';
    return '#ff9800';
  }

  // --- Email ---
  startEmailEdit(): void {
    this.emailValue = this.user()?.email ?? '';
    this.editingEmail.set(true);
  }
  cancelEmailEdit(): void {
    this.editingEmail.set(false);
    this.emailValue = '';
  }
  saveEmail(): void {
    this.savingEmail.set(true);
    this.auth.fetchProfile().subscribe({
      next: (res) => {
        this.user.set(res.data);
        this.savingEmail.set(false);
        this.editingEmail.set(false);
        this.toast.success('Profile refreshed');
      },
      error: () => this.savingEmail.set(false),
    });
  }

  // --- Job ---
  startJobEdit(): void {
    const cp = this.user()?.crew_profile;
    this.jobRole = cp?.role ?? 'OTHER';
    this.jobTitle = cp?.job_title ?? '';
    this.editingJob.set(true);
  }
  saveJob(): void {
    this.savingJob.set(true);
    this.api.updateProfile({ role: this.jobRole, job_title: this.jobTitle }).subscribe({
      next: () => {
        this.savingJob.set(false);
        this.editingJob.set(false);
        this.toast.success('Job / specialization updated');
        this.refreshProfile();
      },
      error: () => this.savingJob.set(false),
    });
  }

  // --- KYC ---
  onFileSelected(event: Event, side: 'front' | 'back'): void {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files[0]) {
      this.setFile(input.files[0], side);
    }
  }

  onFileDrop(event: DragEvent, side: 'front' | 'back'): void {
    event.preventDefault();
    if (event.dataTransfer?.files && event.dataTransfer.files[0]) {
      this.setFile(event.dataTransfer.files[0], side);
    }
  }

  private setFile(file: File, side: 'front' | 'back'): void {
    const reader = new FileReader();
    reader.onload = (e) => {
      if (side === 'front') {
        this.idFrontFile.set(file);
        this.idFrontPreview.set(e.target?.result as string);
      } else {
        this.idBackFile.set(file);
        this.idBackPreview.set(e.target?.result as string);
      }
    };
    reader.readAsDataURL(file);
  }

  submitKYCUpload(): void {
    const front = this.idFrontFile();
    if (!this.kycNationalId || !front) return;
    this.submittingKYC.set(true);
    this.api.uploadKYC(this.kycNationalId, front, this.idBackFile() ?? undefined).subscribe({
      next: (res) => {
        this.submittingKYC.set(false);
        this.toast.success(res.data.message || 'ID documents uploaded successfully');
        this.kycNationalId = '';
        this.idFrontFile.set(null);
        this.idBackFile.set(null);
        this.idFrontPreview.set(null);
        this.idBackPreview.set(null);
        this.refreshProfile();
      },
      error: () => this.submittingKYC.set(false),
    });
  }

  submitKYC(): void {
    if (!this.kycNationalId || !this.kycSerialNumber) return;
    this.submittingKYC.set(true);
    this.api.initiateKYC({ national_id: this.kycNationalId, serial_number: this.kycSerialNumber }).subscribe({
      next: (res) => {
        this.submittingKYC.set(false);
        this.toast.success(res.data.message || 'KYC verification initiated');
        this.kycNationalId = '';
        this.kycSerialNumber = '';
        this.refreshProfile();
      },
      error: () => this.submittingKYC.set(false),
    });
  }

  // --- Password ---
  changePassword(): void {
    if (!this.currentPassword || !this.newPassword || !this.confirmPassword) {
      this.toast.warning('All password fields are required');
      return;
    }
    if (this.newPassword.length < 8) {
      this.toast.warning('New password must be at least 8 characters');
      return;
    }
    if (this.newPassword !== this.confirmPassword) {
      this.toast.error('Passwords do not match');
      return;
    }
    this.changingPassword.set(true);
    this.auth.changePassword(this.currentPassword, this.newPassword).subscribe({
      next: () => {
        this.toast.success('Password changed successfully');
        this.showPasswordModal.set(false);
        this.changingPassword.set(false);
        this.currentPassword = '';
        this.newPassword = '';
        this.confirmPassword = '';
      },
      error: () => this.changingPassword.set(false),
    });
  }

  private refreshProfile(): void {
    this.auth.fetchProfile().subscribe({
      next: (res) => this.user.set(res.data),
    });
  }
}
