import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import {
  ApiResponse, ApiListResponse,
  CrewMember, Assignment, Wallet, WalletTransaction,
  SACCO, Vehicle, Route, PayrollRun, PayrollEntry,
  Earning, DailySummary, CreditScore, LoanApplication,
  LoanTier, InsurancePolicy, Notification, AuditLog,
  SystemStats, SACCOFloat, SACCOMembership, SACCOFloatTransaction
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

  createCrewMember(data: { national_id: string; first_name: string; last_name: string; role: string }): Observable<ApiResponse<CrewMember>> {
    return this.http.post<ApiResponse<CrewMember>>(`${this.API}/crew`, data);
  }

  updateKYC(id: string, data: { kyc_status: string; serial_number?: string }): Observable<ApiResponse<CrewMember>> {
    return this.http.put<ApiResponse<CrewMember>>(`${this.API}/crew/${id}/kyc`, data);
  }

  verifyNationalID(id: string, serialNumber: string): Observable<ApiResponse<CrewMember>> {
    return this.http.post<ApiResponse<CrewMember>>(`${this.API}/crew/${id}/verify`, { serial_number: serialNumber });
  }

  deactivateCrewMember(id: string): Observable<unknown> {
    return this.http.delete(`${this.API}/crew/${id}`);
  }

  bulkImportCrew(members: Array<{ national_id: string; first_name: string; last_name: string; role: string }>): Observable<unknown> {
    return this.http.post(`${this.API}/crew/bulk-import`, { members });
  }

  searchByNationalID(nationalId: string): Observable<ApiResponse<CrewMember>> {
    return this.http.get<ApiResponse<CrewMember>>(`${this.API}/crew/search`, { params: { national_id: nationalId } });
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

  // --- SACCOs ---
  getSACCOs(params?: Record<string, string>): Observable<ApiListResponse<SACCO>> {
    return this.http.get<ApiListResponse<SACCO>>(`${this.API}/saccos`, { params: this.buildParams(params) });
  }

  getSACCO(id: string): Observable<ApiResponse<SACCO>> {
    return this.http.get<ApiResponse<SACCO>>(`${this.API}/saccos/${id}`);
  }

  createSACCO(data: Record<string, unknown>): Observable<ApiResponse<SACCO>> {
    return this.http.post<ApiResponse<SACCO>>(`${this.API}/saccos`, data);
  }

  updateSACCO(id: string, data: Record<string, unknown>): Observable<ApiResponse<SACCO>> {
    return this.http.put<ApiResponse<SACCO>>(`${this.API}/saccos/${id}`, data);
  }

  deleteSACCO(id: string): Observable<unknown> {
    return this.http.delete(`${this.API}/saccos/${id}`);
  }

  getSACCOMembers(saccoId: string, params?: Record<string, string>): Observable<ApiListResponse<SACCOMembership>> {
    return this.http.get<ApiListResponse<SACCOMembership>>(`${this.API}/saccos/${saccoId}/members`, { params: this.buildParams(params) });
  }

  addSACCOMember(saccoId: string, data: { crew_member_id: string; role: string }): Observable<unknown> {
    return this.http.post(`${this.API}/saccos/${saccoId}/members`, data);
  }

  removeSACCOMember(saccoId: string, membershipId: string): Observable<unknown> {
    return this.http.delete(`${this.API}/saccos/${saccoId}/members/${membershipId}`);
  }

  getSACCOFloat(saccoId: string): Observable<ApiResponse<SACCOFloat>> {
    return this.http.get<ApiResponse<SACCOFloat>>(`${this.API}/saccos/${saccoId}/float`);
  }

  creditSACCOFloat(saccoId: string, data: Record<string, unknown>): Observable<unknown> {
    return this.http.post(`${this.API}/saccos/${saccoId}/float/credit`, data);
  }

  debitSACCOFloat(saccoId: string, data: Record<string, unknown>): Observable<unknown> {
    return this.http.post(`${this.API}/saccos/${saccoId}/float/debit`, data);
  }

  getFloatTransactions(saccoId: string, params?: Record<string, string>): Observable<ApiListResponse<SACCOFloatTransaction>> {
    return this.http.get<ApiListResponse<SACCOFloatTransaction>>(`${this.API}/saccos/${saccoId}/float/transactions`, { params: this.buildParams(params) });
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

  getDetailedScore(crewMemberId: string): Observable<ApiResponse<unknown>> {
    return this.http.get<ApiResponse<unknown>>(`${this.API}/credit/${crewMemberId}/detailed`);
  }

  getScoreHistory(crewMemberId: string, limit?: number): Observable<ApiResponse<unknown>> {
    const params = this.buildParams(limit ? { limit: limit.toString() } : undefined);
    return this.http.get<ApiResponse<unknown>>(`${this.API}/credit/${crewMemberId}/history`, { params });
  }

  // --- Loans ---
  getLoans(params?: Record<string, string>): Observable<ApiListResponse<LoanApplication>> {
    return this.http.get<ApiListResponse<LoanApplication>>(`${this.API}/loans`, { params: this.buildParams(params) });
  }

  getLoanTier(crewMemberId: string): Observable<ApiResponse<LoanTier>> {
    return this.http.get<ApiResponse<LoanTier>>(`${this.API}/loans/tier/${crewMemberId}`);
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

  getNotificationPreferences(): Observable<ApiResponse<unknown>> {
    return this.http.get<ApiResponse<unknown>>(`${this.API}/notifications/preferences`);
  }

  updateNotificationPreferences(data: Record<string, unknown>): Observable<unknown> {
    return this.http.put(`${this.API}/notifications/preferences`, data);
  }

  // --- Admin ---
  getSystemStats(): Observable<ApiResponse<SystemStats>> {
    return this.http.get<ApiResponse<SystemStats>>(`${this.API}/admin/stats`);
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

  getStatutoryRates(): Observable<ApiResponse<unknown>> {
    return this.http.get<ApiResponse<unknown>>(`${this.API}/admin/statutory-rates`);
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
}
