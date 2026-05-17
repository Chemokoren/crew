import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import {
  ApiResponse, ApiListResponse,
  CrewMember, CrewProfile, Assignment, Wallet, WalletTransaction,
  Organization, Vehicle, Route, PayrollRun, PayrollEntry,
  Earning, DailySummary, CreditScore, DetailedScoreResult, CreditScoreHistory,
  LoanApplication, LoanTier, InsurancePolicy, Notification, NotificationPreference,
  AuditLog, AdminUser, NotificationTemplate, Document,
  SystemStats, SACCOFloat, SACCOMembership, SACCOFloatTransaction,
  StatutoryRate, TenantJobType, PaySchedule, PayPeriod, FinancialProfile,
  BootstrapResult, WorkSite, SystemSetting, SystemAnnouncement
} from '../models';

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly API = environment.apiUrl;

  constructor(private http: HttpClient) {}

  // --- Crew ---
  getCrewMembers(params?: Record<string, string>): Observable<ApiListResponse<CrewMember>> {
    return this.http.get<ApiListResponse<CrewMember>>(`${this.API}/crew`, { params: this.buildParams(params) });
  }

  getCrewMember(id: string): Observable<ApiResponse<CrewMember>> {
    return this.http.get<ApiResponse<CrewMember>>(`${this.API}/crew/${id}`);
  }

  createCrewMember(data: { national_id: string; phone?: string; first_name: string; last_name: string; role: string }): Observable<ApiResponse<CrewMember>> {
    return this.http.post<ApiResponse<CrewMember>>(`${this.API}/crew`, data);
  }

  updateKYC(id: string, data: { kyc_status: string; serial_number?: string; reason?: string }): Observable<ApiResponse<CrewMember>> {
    return this.http.put<ApiResponse<CrewMember>>(`${this.API}/crew/${id}/kyc`, data);
  }

  verifyNationalID(id: string, serialNumber: string): Observable<ApiResponse<CrewMember>> {
    return this.http.post<ApiResponse<CrewMember>>(`${this.API}/crew/${id}/verify`, { serial_number: serialNumber });
  }

  deactivateCrewMember(id: string): Observable<unknown> {
    return this.http.delete(`${this.API}/crew/${id}`);
  }

  bulkImportCrew(members: Array<{ national_id: string; phone?: string; first_name: string; last_name: string; role: string }>): Observable<unknown> {
    return this.http.post(`${this.API}/crew/bulk-import`, { members });
  }

  searchByNationalID(nationalId: string): Observable<ApiResponse<CrewMember>> {
    return this.http.get<ApiResponse<CrewMember>>(`${this.API}/crew/search`, { params: { national_id: nationalId } });
  }

  /** Lookup employee by national ID before adding — checks if already registered */
  lookupByNationalID(nationalId: string): Observable<ApiResponse<{ found: boolean; linked: boolean; crew_member?: CrewMember }>> {
    return this.http.get<ApiResponse<{ found: boolean; linked: boolean; crew_member?: CrewMember }>>(`${this.API}/crew/lookup`, { params: { national_id: nationalId } });
  }

  resendCredentials(id: string): Observable<ApiResponse<{ message: string }>> {
    return this.http.post<ApiResponse<{ message: string }>>(`${this.API}/crew/${id}/resend-credentials`, {});
  }

  // --- Assignments ---
  getAssignments(params?: Record<string, string>): Observable<ApiListResponse<Assignment>> {
    return this.http.get<ApiListResponse<Assignment>>(`${this.API}/assignments`, { params: this.buildParams(params) });
  }

  getAssignment(id: string): Observable<ApiResponse<Assignment>> {
    return this.http.get<ApiResponse<Assignment>>(`${this.API}/assignments/${id}`);
  }

  createAssignment(data: Record<string, unknown>): Observable<ApiResponse<Assignment>> {
    return this.http.post<ApiResponse<Assignment>>(`${this.API}/assignments`, data);
  }

  updateAssignment(id: string, data: Record<string, unknown>): Observable<ApiResponse<Assignment>> {
    return this.http.put<ApiResponse<Assignment>>(`${this.API}/assignments/${id}`, data);
  }

  completeAssignment(id: string, totalRevenueCents: number): Observable<unknown> {
    return this.http.post(`${this.API}/assignments/${id}/complete`, { total_revenue_cents: totalRevenueCents });
  }

  cancelAssignment(id: string, reason: string): Observable<unknown> {
    return this.http.post(`${this.API}/assignments/${id}/cancel`, { reason });
  }

  reassignAssignment(id: string, newCrewMemberId: string): Observable<unknown> {
    return this.http.post(`${this.API}/assignments/${id}/reassign`, { new_crew_member_id: newCrewMemberId });
  }

  // --- Wallets ---
  getWalletBalance(crewMemberId: string): Observable<ApiResponse<Wallet>> {
    return this.http.get<ApiResponse<Wallet>>(`${this.API}/wallets/${crewMemberId}`);
  }

  getWalletTransactions(crewMemberId: string, params?: Record<string, string>): Observable<ApiListResponse<WalletTransaction>> {
    return this.http.get<ApiListResponse<WalletTransaction>>(`${this.API}/wallets/${crewMemberId}/transactions`, { params: this.buildParams(params) });
  }

  creditWallet(data: { crew_member_id: string; amount_cents: number; category: string; reference?: string; description?: string }, idempotencyKey: string): Observable<unknown> {
    return this.http.post(`${this.API}/wallets/credit`, data, { headers: { 'Idempotency-Key': idempotencyKey } });
  }

  debitWallet(data: { crew_member_id: string; amount_cents: number; category: string; reference?: string; description?: string }, idempotencyKey: string): Observable<unknown> {
    return this.http.post(`${this.API}/wallets/debit`, data, { headers: { 'Idempotency-Key': idempotencyKey } });
  }

  initiatePayout(crewMemberId: string, data: Record<string, unknown>, idempotencyKey: string): Observable<unknown> {
    return this.http.post(`${this.API}/wallets/${crewMemberId}/payout`, data, { headers: { 'Idempotency-Key': idempotencyKey } });
  }

  exportWalletCSV(crewMemberId: string): Observable<Blob> {
    return this.http.get(`${this.API}/wallets/${crewMemberId}/export`, { responseType: 'blob' });
  }

  // --- Atomic Transactions (single endpoint, all-or-nothing) ---

  /** Atomically debit org float (gross) and credit employee wallet (net) */
  employeePayout(data: {
    crew_member_id: string;
    gross_cents: number;
    net_cents: number;
    idempotency_key: string;
    description?: string;
  }): Observable<unknown> {
    return this.http.post(`${this.API}/transactions/employee-payout`, data);
  }

  /** Process multiple employee payouts sequentially, returning per-item success/failure */
  bulkEmployeePayout(data: {
    payouts: Array<{ crew_member_id: string; gross_cents: number; net_cents: number; description?: string; }>;
    idempotency_prefix: string;
  }): Observable<unknown> {
    return this.http.post(`${this.API}/transactions/bulk-employee-payout`, data);
  }

  /** Atomically debit sender and credit recipient wallet */
  walletTransfer(data: {
    to_crew_member_id: string;
    amount_cents: number;
    idempotency_key: string;
    description?: string;
  }): Observable<unknown> {
    return this.http.post(`${this.API}/transactions/transfer`, data);
  }

  // --- Organizations ---
  getOrganizations(params?: Record<string, string>): Observable<ApiListResponse<Organization>> {
    return this.http.get<ApiListResponse<Organization>>(`${this.API}/organizations`, { params: this.buildParams(params) });
  }

  getOrganization(id: string): Observable<ApiResponse<Organization>> {
    return this.http.get<ApiResponse<Organization>>(`${this.API}/organizations/${id}`);
  }

  createOrganization(data: Record<string, unknown>): Observable<ApiResponse<Organization>> {
    return this.http.post<ApiResponse<Organization>>(`${this.API}/organizations`, data);
  }

  updateOrganization(id: string, data: Record<string, unknown>): Observable<ApiResponse<Organization>> {
    return this.http.put<ApiResponse<Organization>>(`${this.API}/organizations/${id}`, data);
  }

  deleteOrganization(id: string): Observable<unknown> {
    return this.http.delete(`${this.API}/organizations/${id}`);
  }

  getSACCOMembers(saccoId: string, params?: Record<string, string>): Observable<ApiListResponse<SACCOMembership>> {
    return this.http.get<ApiListResponse<SACCOMembership>>(`${this.API}/organizations/${saccoId}/members`, { params: this.buildParams(params) });
  }

  addSACCOMember(saccoId: string, data: { crew_member_id: string; role: string; joined_at?: string }): Observable<unknown> {
    return this.http.post(`${this.API}/organizations/${saccoId}/members`, data);
  }

  updateOrganizationMember(saccoId: string, membershipId: string, data: { role_in_sacco: string; joined_at?: string }): Observable<unknown> {
    return this.http.put(`${this.API}/organizations/${saccoId}/members/${membershipId}`, data);
  }

  removeSACCOMember(saccoId: string, membershipId: string): Observable<unknown> {
    return this.http.delete(`${this.API}/organizations/${saccoId}/members/${membershipId}`);
  }

  getSACCOFloat(saccoId: string): Observable<ApiResponse<SACCOFloat>> {
    return this.http.get<ApiResponse<SACCOFloat>>(`${this.API}/organizations/${saccoId}/float`);
  }

  creditSACCOFloat(saccoId: string, data: Record<string, unknown>): Observable<unknown> {
    return this.http.post(`${this.API}/organizations/${saccoId}/float/credit`, data);
  }

  topupSACCOFloat(saccoId: string, data: Record<string, unknown>): Observable<unknown> {
    return this.http.post(`${this.API}/organizations/${saccoId}/float/topup`, data);
  }

  confirmTopUp(saccoId: string, txId: string): Observable<unknown> {
    return this.http.post(`${this.API}/organizations/${saccoId}/float/topup/${txId}/confirm`, {});
  }

  rejectTopUp(saccoId: string, txId: string, reason?: string): Observable<unknown> {
    return this.http.post(`${this.API}/organizations/${saccoId}/float/topup/${txId}/reject`, { reason: reason || '' });
  }

  debitSACCOFloat(saccoId: string, data: Record<string, unknown>): Observable<unknown> {
    return this.http.post(`${this.API}/organizations/${saccoId}/float/debit`, data);
  }

  getFloatTransactions(saccoId: string, params?: Record<string, string>): Observable<ApiListResponse<SACCOFloatTransaction>> {
    return this.http.get<ApiListResponse<SACCOFloatTransaction>>(`${this.API}/organizations/${saccoId}/float/transactions`, { params: this.buildParams(params) });
  }

  /** Poll JamboPay for status of pending STK push transactions */
  pollSTK(saccoId: string): Observable<ApiResponse<{
    message: string;
    checked: number;
    confirmed: number;
    failed: number;
    skipped: number;
    results: Array<{ tx_id: string; order_id: string; jp_status?: string; action?: string; error?: string }>;
  }>> {
    return this.http.post<any>(`${this.API}/organizations/${saccoId}/float/poll-stk`, {});
  }

  /** Poll JamboPay for status of a single pending STK push transaction */
  pollSingleSTK(saccoId: string, txId: string): Observable<ApiResponse<{
    tx_id: string; order_id: string; jp_status?: string; action?: string; error?: string;
  }>> {
    return this.http.post<any>(`${this.API}/organizations/${saccoId}/float/poll-stk/${txId}`, {});
  }

  // --- Vehicles ---
  getVehicles(params?: Record<string, string>): Observable<ApiListResponse<Vehicle>> {
    return this.http.get<ApiListResponse<Vehicle>>(`${this.API}/vehicles`, { params: this.buildParams(params) });
  }

  getVehicle(id: string): Observable<ApiResponse<Vehicle>> {
    return this.http.get<ApiResponse<Vehicle>>(`${this.API}/vehicles/${id}`);
  }

  createVehicle(data: Record<string, unknown>): Observable<ApiResponse<Vehicle>> {
    return this.http.post<ApiResponse<Vehicle>>(`${this.API}/vehicles`, data);
  }

  updateVehicle(id: string, data: Record<string, unknown>): Observable<ApiResponse<Vehicle>> {
    return this.http.put<ApiResponse<Vehicle>>(`${this.API}/vehicles/${id}`, data);
  }

  deleteVehicle(id: string): Observable<unknown> {
    return this.http.delete(`${this.API}/vehicles/${id}`);
  }

  // --- Routes ---
  getRoutes(params?: Record<string, string>): Observable<ApiListResponse<Route>> {
    return this.http.get<ApiListResponse<Route>>(`${this.API}/routes`, { params: this.buildParams(params) });
  }

  getRoute(id: string): Observable<ApiResponse<Route>> {
    return this.http.get<ApiResponse<Route>>(`${this.API}/routes/${id}`);
  }

  createRoute(data: Record<string, unknown>): Observable<ApiResponse<Route>> {
    return this.http.post<ApiResponse<Route>>(`${this.API}/routes`, data);
  }

  updateRoute(id: string, data: Record<string, unknown>): Observable<ApiResponse<Route>> {
    return this.http.put<ApiResponse<Route>>(`${this.API}/routes/${id}`, data);
  }

  deleteRoute(id: string): Observable<unknown> {
    return this.http.delete(`${this.API}/routes/${id}`);
  }

  // --- Payroll ---
  getPayrollRuns(params?: Record<string, string>): Observable<ApiListResponse<PayrollRun>> {
    return this.http.get<ApiListResponse<PayrollRun>>(`${this.API}/payroll`, { params: this.buildParams(params) });
  }

  getPayrollRun(id: string): Observable<ApiResponse<PayrollRun>> {
    return this.http.get<ApiResponse<PayrollRun>>(`${this.API}/payroll/${id}`);
  }

  createPayrollRun(data: Record<string, unknown>): Observable<ApiResponse<PayrollRun>> {
    return this.http.post<ApiResponse<PayrollRun>>(`${this.API}/payroll`, data);
  }

  getPayrollEntries(id: string): Observable<ApiResponse<PayrollEntry[]>> {
    return this.http.get<ApiResponse<PayrollEntry[]>>(`${this.API}/payroll/${id}/entries`);
  }

  processPayrollRun(id: string): Observable<ApiResponse<PayrollRun>> {
    return this.http.post<ApiResponse<PayrollRun>>(`${this.API}/payroll/${id}/process`, {});
  }

  approvePayrollRun(id: string): Observable<ApiResponse<PayrollRun>> {
    return this.http.post<ApiResponse<PayrollRun>>(`${this.API}/payroll/${id}/approve`, {});
  }

  submitPayrollRun(id: string): Observable<ApiResponse<PayrollRun>> {
    return this.http.post<ApiResponse<PayrollRun>>(`${this.API}/payroll/${id}/submit`, {});
  }

  // --- Earnings ---
  getEarnings(params?: Record<string, string>): Observable<ApiListResponse<Earning>> {
    return this.http.get<ApiListResponse<Earning>>(`${this.API}/earnings`, { params: this.buildParams(params) });
  }

  getEarningSummary(crewMemberId: string, date?: string): Observable<ApiResponse<DailySummary>> {
    const params = this.buildParams(date ? { date } : undefined);
    return this.http.get<ApiResponse<DailySummary>>(`${this.API}/earnings/summary/${crewMemberId}`, { params });
  }

  // --- Credit ---
  getCreditScore(crewMemberId: string): Observable<ApiResponse<CreditScore>> {
    return this.http.get<ApiResponse<CreditScore>>(`${this.API}/credit/${crewMemberId}`);
  }

  getDetailedScore(crewMemberId: string): Observable<ApiResponse<DetailedScoreResult>> {
    return this.http.get<ApiResponse<DetailedScoreResult>>(`${this.API}/credit/${crewMemberId}/detailed`);
  }

  getScoreHistory(crewMemberId: string, limit?: number): Observable<ApiResponse<CreditScoreHistory[]>> {
    const params = this.buildParams(limit ? { limit: limit.toString() } : undefined);
    return this.http.get<ApiResponse<CreditScoreHistory[]>>(`${this.API}/credit/${crewMemberId}/history`, { params });
  }

  calculateScore(crewMemberId: string): Observable<ApiResponse<CreditScore>> {
    return this.http.post<ApiResponse<CreditScore>>(`${this.API}/credit/${crewMemberId}/calculate`, {});
  }

  // --- Loans ---
  getLoans(params?: Record<string, string>): Observable<ApiListResponse<LoanApplication>> {
    return this.http.get<ApiListResponse<LoanApplication>>(`${this.API}/loans`, { params: this.buildParams(params) });
  }

  getLoan(id: string): Observable<ApiResponse<LoanApplication>> {
    return this.http.get<ApiResponse<LoanApplication>>(`${this.API}/loans/${id}`);
  }

  getLoanTier(crewMemberId: string): Observable<ApiResponse<LoanTier>> {
    return this.http.get<ApiResponse<LoanTier>>(`${this.API}/loans/tier/${crewMemberId}`, {
      headers: { 'X-Skip-Error-Toast': 'true' }
    });
  }

  applyForLoan(data: { crew_member_id: string; amount_cents: number; tenure_days: number; category?: string; purpose?: string }): Observable<ApiResponse<LoanApplication>> {
    return this.http.post<ApiResponse<LoanApplication>>(`${this.API}/loans`, data);
  }

  approveLoan(id: string, data: { approved_amount_cents: number; interest_rate: number }): Observable<ApiResponse<LoanApplication>> {
    return this.http.post<ApiResponse<LoanApplication>>(`${this.API}/loans/${id}/approve`, data);
  }

  rejectLoan(id: string): Observable<ApiResponse<LoanApplication>> {
    return this.http.post<ApiResponse<LoanApplication>>(`${this.API}/loans/${id}/reject`, {});
  }

  disburseLoan(id: string): Observable<ApiResponse<LoanApplication>> {
    return this.http.post<ApiResponse<LoanApplication>>(`${this.API}/loans/${id}/disburse`, {});
  }

  repayLoan(id: string, amountCents: number): Observable<ApiResponse<LoanApplication>> {
    return this.http.post<ApiResponse<LoanApplication>>(`${this.API}/loans/${id}/repay`, { amount_cents: amountCents });
  }

  // --- Insurance ---
  getInsurancePolicies(params?: Record<string, string>): Observable<ApiListResponse<InsurancePolicy>> {
    return this.http.get<ApiListResponse<InsurancePolicy>>(`${this.API}/insurance`, { params: this.buildParams(params) });
  }

  createInsurancePolicy(data: Record<string, unknown>): Observable<ApiResponse<InsurancePolicy>> {
    return this.http.post<ApiResponse<InsurancePolicy>>(`${this.API}/insurance`, data);
  }

  lapseInsurancePolicy(id: string): Observable<unknown> {
    return this.http.post(`${this.API}/insurance/${id}/lapse`, {});
  }

  // --- Notifications ---
  getNotifications(params?: Record<string, string>): Observable<ApiListResponse<Notification>> {
    return this.http.get<ApiListResponse<Notification>>(`${this.API}/notifications`, { params: this.buildParams(params) });
  }

  markNotificationRead(id: string): Observable<unknown> {
    return this.http.put(`${this.API}/notifications/${id}/read`, {});
  }

  getNotificationPreferences(): Observable<ApiResponse<NotificationPreference>> {
    return this.http.get<ApiResponse<NotificationPreference>>(`${this.API}/notifications/preferences`);
  }

  updateNotificationPreferences(data: Partial<NotificationPreference>): Observable<ApiResponse<NotificationPreference>> {
    return this.http.put<ApiResponse<NotificationPreference>>(`${this.API}/notifications/preferences`, data);
  }

  // --- Admin ---
  getSystemStats(): Observable<ApiResponse<SystemStats>> {
    return this.http.get<ApiResponse<SystemStats>>(`${this.API}/admin/stats`);
  }

  getUsers(params?: Record<string, string>): Observable<ApiListResponse<AdminUser>> {
    return this.http.get<ApiListResponse<AdminUser>>(`${this.API}/admin/users`, { params: this.buildParams(params) });
  }

  disableAccount(userId: string): Observable<unknown> {
    return this.http.post(`${this.API}/admin/users/${userId}/disable`, {});
  }

  enableAccount(userId: string): Observable<unknown> {
    return this.http.post(`${this.API}/admin/users/${userId}/enable`, {});
  }

  resetPassword(userId: string, newPassword: string): Observable<unknown> {
    return this.http.post(`${this.API}/admin/users/${userId}/reset-password`, { new_password: newPassword });
  }

  getAuditLogs(params?: Record<string, string>): Observable<ApiListResponse<AuditLog>> {
    return this.http.get<ApiListResponse<AuditLog>>(`${this.API}/admin/audit-logs`, { params: this.buildParams(params) });
  }

  getStatutoryRates(): Observable<ApiResponse<StatutoryRate[]>> {
    return this.http.get<ApiResponse<StatutoryRate[]>>(`${this.API}/admin/statutory-rates`);
  }

  getNotificationTemplates(): Observable<ApiResponse<NotificationTemplate[]>> {
    return this.http.get<ApiResponse<NotificationTemplate[]>>(`${this.API}/admin/notifications/templates`);
  }

  createNotificationTemplate(data: Partial<NotificationTemplate>): Observable<ApiResponse<NotificationTemplate>> {
    const payload = { ...data };
    if (!payload.id) delete payload.id;
    return this.http.post<ApiResponse<NotificationTemplate>>(`${this.API}/admin/notifications/templates`, payload);
  }

  updateNotificationTemplate(data: Partial<NotificationTemplate>): Observable<ApiResponse<NotificationTemplate>> {
    return this.http.put<ApiResponse<NotificationTemplate>>(`${this.API}/admin/notifications/templates`, data);
  }

  // --- Documents ---
  getDocuments(params?: Record<string, string>): Observable<ApiListResponse<Document>> {
    return this.http.get<ApiListResponse<Document>>(`${this.API}/documents`, { params: this.buildParams(params) });
  }

  uploadDocument(formData: FormData): Observable<ApiResponse<Document>> {
    return this.http.post<ApiResponse<Document>>(`${this.API}/documents/upload`, formData);
  }

  downloadDocument(id: string): Observable<ApiResponse<{ download_url: string }>> {
    return this.http.get<ApiResponse<{ download_url: string }>>(`${this.API}/documents/${id}/download`);
  }

  deleteDocument(id: string): Observable<unknown> {
    return this.http.delete(`${this.API}/documents/${id}`);
  }

  // --- Tenant Config (Phase F3) ---
  getJobTypes(saccoId: string): Observable<ApiListResponse<TenantJobType>> {
    return this.http.get<ApiListResponse<TenantJobType>>(`${this.API}/organizations/${saccoId}/job-types`);
  }

  createJobType(saccoId: string, data: Partial<TenantJobType>): Observable<ApiResponse<TenantJobType>> {
    return this.http.post<ApiResponse<TenantJobType>>(`${this.API}/organizations/${saccoId}/job-types`, data);
  }

  updateJobType(saccoId: string, id: string, data: Partial<TenantJobType>): Observable<ApiResponse<TenantJobType>> {
    return this.http.put<ApiResponse<TenantJobType>>(`${this.API}/organizations/${saccoId}/job-types/${id}`, data);
  }

  deleteJobType(saccoId: string, id: string): Observable<unknown> {
    return this.http.delete(`${this.API}/organizations/${saccoId}/job-types/${id}`);
  }

  // --- Pay Schedules (Phase F3) ---
  getPaySchedules(saccoId: string): Observable<ApiListResponse<PaySchedule>> {
    return this.http.get<ApiListResponse<PaySchedule>>(`${this.API}/organizations/${saccoId}/pay-schedules`);
  }

  createPaySchedule(saccoId: string, data: Partial<PaySchedule>): Observable<ApiResponse<PaySchedule>> {
    return this.http.post<ApiResponse<PaySchedule>>(`${this.API}/organizations/${saccoId}/pay-schedules`, data);
  }

  updatePaySchedule(saccoId: string, id: string, data: Partial<PaySchedule>): Observable<ApiResponse<PaySchedule>> {
    return this.http.put<ApiResponse<PaySchedule>>(`${this.API}/organizations/${saccoId}/pay-schedules/${id}`, data);
  }

  deletePaySchedule(saccoId: string, id: string): Observable<unknown> {
    return this.http.delete(`${this.API}/organizations/${saccoId}/pay-schedules/${id}`);
  }

  // --- Industry Bootstrap (AD-13) ---
  bootstrapIndustry(orgId: string, industry: string): Observable<ApiResponse<BootstrapResult>> {
    return this.http.post<ApiResponse<BootstrapResult>>(`${this.API}/organizations/${orgId}/bootstrap`, { industry_type: industry });
  }

  // --- Pay Periods (Phase F3) ---
  getPayPeriods(params?: Record<string, string>): Observable<ApiListResponse<PayPeriod>> {
    return this.http.get<ApiListResponse<PayPeriod>>(`${this.API}/payroll/periods`, { params: this.buildParams(params) });
  }

  generatePayPeriod(scheduleId: string, data: { reference_date: string }): Observable<ApiResponse<PayPeriod>> {
    return this.http.post<ApiResponse<PayPeriod>>(`${this.API}/payroll/schedule/${scheduleId}/generate-period`, data);
  }

  closePayPeriod(periodId: string): Observable<ApiResponse<PayPeriod>> {
    return this.http.post<ApiResponse<PayPeriod>>(`${this.API}/payroll/periods/${periodId}/close`, {});
  }

  runScheduledPayroll(scheduleId: string, data: { reference_date: string; organization_id: string }): Observable<ApiResponse<PayrollRun>> {
    return this.http.post<ApiResponse<PayrollRun>>(`${this.API}/payroll/schedule/${scheduleId}/run`, data);
  }

  // --- Financial Profile (Phase F3) ---
  getFinancialProfile(crewMemberId: string): Observable<ApiResponse<FinancialProfile>> {
    return this.http.get<ApiResponse<FinancialProfile>>(`${this.API}/financial-profile/${crewMemberId}`);
  }

  // --- Check-in/Check-out (Phase F3) ---
  checkInAssignment(assignmentId: string): Observable<ApiResponse<Assignment>> {
    return this.http.post<ApiResponse<Assignment>>(`${this.API}/assignments/${assignmentId}/check-in`, {});
  }

  checkOutAssignment(assignmentId: string, data?: { hours_worked?: number }): Observable<ApiResponse<Assignment>> {
    return this.http.post<ApiResponse<Assignment>>(`${this.API}/assignments/${assignmentId}/check-out`, data || {});
  }

  // --- Bulk Assignments (Phase F3) ---
  bulkCreateAssignments(data: { assignments: Array<Record<string, unknown>> }): Observable<unknown> {
    return this.http.post(`${this.API}/assignments/bulk`, data);
  }

  // --- Tenant Config Update ---
  updateTenantConfig(saccoId: string, config: Record<string, unknown>): Observable<ApiResponse<Organization>> {
    return this.http.put<ApiResponse<Organization>>(`${this.API}/organizations/${saccoId}/config`, config);
  }

  // --- USSD Gateway Admin ---
  // Routes through /ussd-admin proxy → USSD gateway at :8090/admin/...
  refreshUSSDRoleCache(): Observable<{ status: string; message: string }> {
    return this.http.post<{ status: string; message: string }>('/ussd-admin/cache/refresh', {});
  }

  // --- Helpers ---
  private buildParams(params?: Record<string, string>): HttpParams {
    let httpParams = new HttpParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          httpParams = httpParams.set(key, value);
        }
      });
    }
    return httpParams;
  }

  // --- Work Sites (Phase H) ---
  getWorkSites(params?: Record<string, string>): Observable<ApiListResponse<WorkSite>> {
    return this.http.get<ApiListResponse<WorkSite>>(`${this.API}/work-sites`, { params: this.buildParams(params) });
  }

  getWorkSite(id: string): Observable<ApiResponse<WorkSite>> {
    return this.http.get<ApiResponse<WorkSite>>(`${this.API}/work-sites/${id}`);
  }

  createWorkSite(data: Partial<WorkSite>): Observable<ApiResponse<WorkSite>> {
    return this.http.post<ApiResponse<WorkSite>>(`${this.API}/work-sites`, data);
  }

  updateWorkSite(id: string, data: Partial<WorkSite>): Observable<ApiResponse<WorkSite>> {
    return this.http.put<ApiResponse<WorkSite>>(`${this.API}/work-sites/${id}`, data);
  }

  deleteWorkSite(id: string): Observable<void> {
    return this.http.delete<void>(`${this.API}/work-sites/${id}`);
  }

  // --- Profile & KYC ---
  updateProfile(data: { role?: string; job_title?: string; first_name?: string; last_name?: string }): Observable<ApiResponse<CrewProfile>> {
    return this.http.put<ApiResponse<CrewProfile>>(`${this.API}/auth/profile`, data);
  }

  initiateKYC(data: { national_id: string; serial_number: string }): Observable<ApiResponse<{ kyc_status: string; crew_id: string; message: string }>> {
    return this.http.post<ApiResponse<{ kyc_status: string; crew_id: string; message: string }>>(`${this.API}/auth/kyc/initiate`, data);
  }

  uploadKYC(nationalId: string, idFront: File, idBack?: File): Observable<ApiResponse<{ kyc_status: string; crew_id: string; message: string }>> {
    const fd = new FormData();
    fd.append('national_id', nationalId);
    fd.append('id_front', idFront);
    if (idBack) {
      fd.append('id_back', idBack);
    }
    return this.http.post<ApiResponse<{ kyc_status: string; crew_id: string; message: string }>>(`${this.API}/auth/kyc/upload`, fd);
  }

  // --- Platform System Settings ---

  createStatutoryRate(data: Partial<StatutoryRate>): Observable<ApiResponse<StatutoryRate>> {
    return this.http.post<ApiResponse<StatutoryRate>>(`${this.API}/admin/statutory-rates`, data);
  }

  updateStatutoryRate(id: string, data: Partial<StatutoryRate>): Observable<ApiResponse<StatutoryRate>> {
    return this.http.put<ApiResponse<StatutoryRate>>(`${this.API}/admin/statutory-rates/${id}`, data);
  }

  // --- System Settings (key-value) ---

  getSystemSettings(prefix?: string): Observable<ApiResponse<SystemSetting[]>> {
    const params = this.buildParams(prefix ? { prefix } : undefined);
    return this.http.get<ApiResponse<SystemSetting[]>>(`${this.API}/admin/system-settings`, { params });
  }

  upsertSystemSetting(setting: Partial<SystemSetting>): Observable<ApiResponse<SystemSetting>> {
    return this.http.put<ApiResponse<SystemSetting>>(`${this.API}/admin/system-settings`, setting);
  }

  bulkUpsertSystemSettings(settings: Partial<SystemSetting>[]): Observable<ApiResponse<{ message: string; count: number }>> {
    return this.http.put<ApiResponse<{ message: string; count: number }>>(`${this.API}/admin/system-settings/bulk`, { settings });
  }

  deleteSystemSetting(key: string): Observable<ApiResponse<{ message: string }>> {
    return this.http.delete<ApiResponse<{ message: string }>>(`${this.API}/admin/system-settings/${key}`);
  }

  // --- System Announcements ---

  getAnnouncements(params?: Record<string, string>): Observable<ApiListResponse<SystemAnnouncement>> {
    return this.http.get<ApiListResponse<SystemAnnouncement>>(`${this.API}/admin/announcements`, { params: this.buildParams(params) });
  }

  createAnnouncement(data: Partial<SystemAnnouncement>): Observable<ApiResponse<SystemAnnouncement>> {
    return this.http.post<ApiResponse<SystemAnnouncement>>(`${this.API}/admin/announcements`, data);
  }

  updateAnnouncement(id: string, data: Partial<SystemAnnouncement>): Observable<ApiResponse<SystemAnnouncement>> {
    return this.http.put<ApiResponse<SystemAnnouncement>>(`${this.API}/admin/announcements/${id}`, data);
  }

  deleteAnnouncement(id: string): Observable<ApiResponse<{ message: string }>> {
    return this.http.delete<ApiResponse<{ message: string }>>(`${this.API}/admin/announcements/${id}`);
  }

  // Active announcements (all authenticated users)
  getActiveAnnouncements(): Observable<ApiResponse<SystemAnnouncement[]>> {
    return this.http.get<ApiResponse<SystemAnnouncement[]>>(`${this.API}/announcements/active`);
  }
}
