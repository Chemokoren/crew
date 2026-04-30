/* Core interfaces mapping to backend DTOs */

export interface ApiResponse<T> {
  data: T;
}

export interface ApiListResponse<T> {
  data: T[];
  meta: PaginationMeta;
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface User {
  id: string;
  phone: string;
  email?: string;
  system_role: SystemRole;
  crew_member_id?: string;
  sacco_id?: string;
  is_active: boolean;
  last_login_at?: string;
  created_at: string;
}

export type SystemRole = 'SYSTEM_ADMIN' | 'SACCO_ADMIN' | 'CREW' | 'LENDER' | 'INSURER';

export interface AuthResponse {
  user: User;
  tokens: TokenPair;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface CrewMember {
  id: string;
  crew_id: string;
  first_name: string;
  last_name: string;
  full_name: string;
  role: CrewRole;
  kyc_status: KYCStatus;
  kyc_verified_at?: string;
  photo_url?: string;
  is_active: boolean;
  created_at: string;
}

export type CrewRole = 'DRIVER' | 'CONDUCTOR' | 'RIDER' | 'OTHER';
export type KYCStatus = 'PENDING' | 'VERIFIED' | 'REJECTED';

export interface Assignment {
  id: string;
  crew_member_id: string;
  crew_member_name: string;
  vehicle_id: string;
  vehicle_registration_no: string;
  sacco_id: string;
  sacco_name: string;
  route_id?: string;
  route_name?: string;
  shift_date: string;
  shift_start: string;
  shift_end?: string;
  status: AssignmentStatus;
  earning_model: EarningModel;
  fixed_amount_cents?: number;
  commission_rate?: number;
  hybrid_base_cents?: number;
  commission_basis?: CommissionBasis;
  notes?: string;
  created_by_id: string;
  created_at: string;
}

export type AssignmentStatus = 'SCHEDULED' | 'ACTIVE' | 'COMPLETED' | 'CANCELLED';
export type EarningModel = 'FIXED' | 'COMMISSION' | 'HYBRID';
export type CommissionBasis = 'FARE_TOTAL' | 'TRIP_COUNT' | 'REVENUE';

export interface Wallet {
  id: string;
  crew_member_id: string;
  balance_cents: number;
  balance_formatted: string;
  total_credited_cents: number;
  total_debited_cents: number;
  currency: string;
  is_active: boolean;
  last_payout_at?: string;
}

export interface WalletTransaction {
  id: string;
  transaction_type: 'CREDIT' | 'DEBIT';
  category: string;
  amount_cents: number;
  balance_after_cents: number;
  currency: string;
  reference?: string;
  description?: string;
  status: string;
  created_at: string;
}

export interface SACCO {
  id: string;
  name: string;
  registration_number: string;
  county: string;
  sub_county?: string;
  contact_phone: string;
  contact_email?: string;
  currency: string;
  is_active: boolean;
  created_at: string;
}

export interface Vehicle {
  id: string;
  sacco_id: string;
  registration_no: string;
  vehicle_type: VehicleType;
  route_id?: string;
  capacity: number;
  is_active: boolean;
  created_at: string;
}

export type VehicleType = 'MATATU' | 'BODA' | 'TUK_TUK';

export interface Route {
  id: string;
  name: string;
  code: string;
  origin: string;
  destination: string;
  distance_km?: number;
  is_active: boolean;
  created_at: string;
}

export interface PayrollRun {
  id: string;
  sacco_id: string;
  period_start: string;
  period_end: string;
  status: PayrollStatus;
  total_gross_cents: number;
  total_deductions_cents: number;
  total_net_cents: number;
  entry_count: number;
  approved_by?: string;
  approved_at?: string;
  created_at: string;
}

export type PayrollStatus = 'DRAFT' | 'PROCESSED' | 'APPROVED' | 'SUBMITTED' | 'COMPLETED' | 'FAILED';

export interface PayrollEntry {
  id: string;
  payroll_run_id: string;
  crew_member_id: string;
  gross_cents: number;
  sha_cents: number;
  nssf_cents: number;
  housing_levy_cents: number;
  net_cents: number;
}

export interface Earning {
  id: string;
  crew_member_id: string;
  assignment_id: string;
  amount_cents: number;
  currency: string;
  earning_type: string;
  description?: string;
  is_verified: boolean;
  earned_at: string;
  created_at: string;
}

export interface DailySummary {
  id: string;
  crew_member_id: string;
  date: string;
  total_earned_cents: number;
  total_deductions_cents: number;
  net_amount_cents: number;
  currency: string;
  assignment_count: number;
  is_processed: boolean;
  created_at: string;
}

export interface CreditScore {
  id: string;
  crew_member_id: string;
  score: number;
  grade: string;
  computed_at: string;
}

export interface LoanApplication {
  id: string;
  crew_member_id: string;
  amount_cents: number;
  approved_amount_cents?: number;
  interest_rate?: number;
  tenure_days: number;
  category: string;
  purpose?: string;
  status: LoanStatus;
  disbursed_at?: string;
  due_date?: string;
  total_repaid_cents: number;
  approved_by?: string;
  created_at: string;
}

export type LoanStatus = 'PENDING' | 'APPROVED' | 'REJECTED' | 'DISBURSED' | 'REPAID' | 'DEFAULTED';

export interface LoanTier {
  score: number;
  grade: string;
  max_loan_kes: string;
  interest_rate: string;
  max_tenure_days: number;
  cooldown_days: number;
  description: string;
}

export interface InsurancePolicy {
  id: string;
  crew_member_id: string;
  provider: string;
  policy_type: string;
  frequency: string;
  premium_cents: number;
  start_date: string;
  end_date: string;
  status: string;
  created_at: string;
}

export interface Notification {
  id: string;
  user_id: string;
  title: string;
  body: string;
  channel: string;
  status: string;
  read_at?: string;
  created_at: string;
}

export interface AuditLog {
  id: string;
  actor_id: string;
  resource: string;
  resource_id: string;
  action: string;
  details: string;
  created_at: string;
}

export interface SystemStats {
  total_users: number;
  active_users: number;
  total_crew: number;
  total_saccos: number;
  total_vehicles: number;
  total_assignments: number;
  total_wallet_balance_cents: number;
}

export interface SACCOFloat {
  id: string;
  sacco_id: string;
  balance_cents: number;
  currency: string;
}

export interface SACCOMembership {
  id: string;
  sacco_id: string;
  crew_member_id: string;
  role: string;
  joined_at: string;
}

export interface SACCOFloatTransaction {
  id: string;
  sacco_float_id: string;
  transaction_type: 'CREDIT' | 'DEBIT';
  amount_cents: number;
  balance_after_cents: number;
  reference?: string;
  idempotency_key: string;
  created_at: string;
}
