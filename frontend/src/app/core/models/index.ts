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
  organization_id?: string;
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

// --- Multi-Industry Types (Phase F2) ---
export type IndustryType = 'TRANSPORT' | 'CONSTRUCTION' | 'HEALTH' | 'LOGISTICS' | 'AGRICULTURE' | 'HOSPITALITY' | 'GENERAL' | 'CUSTOM';
export type WorkType = 'SHIFT' | 'DAILY' | 'HOURLY' | 'PER_TRIP' | 'PROJECT' | 'VISIT' | 'CONTRACT' | 'TASK';
export type PayFrequency = 'DAILY' | 'WEEKLY' | 'BI_WEEKLY' | 'MONTHLY';
export type PeriodStatus = 'OPEN' | 'CLOSED' | 'PROCESSING';
export type JobTypeCategory = 'PRIMARY' | 'FACILITATOR' | 'SUPPORT' | 'SUPERVISOR';

export interface TenantJobType {
  id: string;
  organization_id: string;
  code: string;
  display_name: string;
  category: JobTypeCategory;
  is_active: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface PaySchedule {
  id: string;
  organization_id: string;
  name: string;
  frequency: PayFrequency;
  pay_day?: number;
  cutoff_hour: number;
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface PayPeriod {
  id: string;
  pay_schedule_id: string;
  period_start: string;
  period_end: string;
  status: PeriodStatus;
  closed_at?: string;
  created_at: string;
  worker_count?: number;
  total_amount_cents?: number;
}

export interface Assignment {
  id: string;
  crew_member_id: string;
  crew_member_name: string;
  vehicle_id?: string;
  vehicle_registration_no?: string;
  organization_id: string;
  organization_name: string;
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
  // Generalized fields (Phase C)
  work_type: WorkType;
  work_site?: string;
  project_ref?: string;
  hours_worked?: number;
  hourly_rate_cents?: number;
  supervisor_id?: string;
  check_in_at?: string;
  check_out_at?: string;
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

export type OrganizationType = 'SACCO' | 'CONSTRUCTION_FIRM' | 'LOGISTICS_COMPANY' | 'HEALTH_NGO' | 'AGRICULTURE_COOP' | 'HOSPITALITY_GROUP' | 'GENERAL';

export interface Organization {
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
  // D1: Organization type + industry
  organization_type?: OrganizationType;
  industry_type?: IndustryType;
  default_language?: string;
  tenant_config?: TenantConfig;
}

export interface TenantConfig {
  credit_scoring_weights?: Record<string, number>;
  statutory_exemptions?: string[];
  ui_labels?: Record<string, string>;
  features?: Record<string, boolean>;
}

// AD-13: Bootstrap result from industry template seeding
export interface BootstrapResult {
  industry_set: boolean;
  job_types_seeded: string[];
  job_types_skipped: boolean;
  schedules_seeded: string[];
  schedules_skipped: boolean;
  config_seeded: boolean;
}

// AD-9: Industry template configuration
export interface IndustryTemplate {
  industry_type: IndustryType;
  organization_type: OrganizationType;
  display_label: string;
  assignment_types: string[];
  earning_models: string[];
  payment_frequencies: string[];
  statutory_bodies: string[];
  default_job_types: { code: string; display_name: string; category: JobTypeCategory }[];
  ui_labels: Record<string, string>;
}

// AD-12: Role-based permission matrix
export interface IndustryPermission {
  role_category: JobTypeCategory;
  can_create_assignments: boolean;
  can_approve_earnings: boolean;
  can_manage_payroll: boolean;
  can_view_financial_profiles: boolean;
  can_manage_crew: boolean;
  can_manage_settings: boolean;
}

export interface Vehicle {
  id: string;
  organization_id: string;
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
  start_point: string;
  end_point: string;
  estimated_distance_km: number | null;
  is_active: boolean;
  created_at: string;
}

export interface PayrollRun {
  id: string;
  organization_id: string;
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
  // Phase D additions
  pay_schedule_id?: string;
  pay_period_id?: string;
}

export type PayrollStatus = 'DRAFT' | 'PROCESSED' | 'APPROVED' | 'SUBMITTED' | 'COMPLETED' | 'FAILED';

export interface PayrollEntry {
  id: string;
  payroll_run_id: string;
  crew_member_id: string;
  gross_earnings_cents: number;
  sha_deduction_cents: number;
  nssf_deduction_cents: number;
  housing_levy_deduction_cents: number;
  other_deductions_cents: number;
  net_pay_cents: number;
  created_at: string;
}

export type RateType = 'PERCENTAGE' | 'FIXED' | 'TIERED';

export interface StatutoryRate {
  id: string;
  name: string;
  rate: number;
  rate_type: RateType;
  effective_from: string;
  is_active: boolean;
  created_at: string;
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
  // F12: Multi-industry grouping
  work_type?: WorkType;
  work_site?: string;
  project_ref?: string;
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

export interface ScoreFactor {
  category: string;
  name: string;
  points: number;
  max_points: number;
  percentage: number;
  description: string;
  impact: 'POSITIVE' | 'NEUTRAL' | 'NEGATIVE';
}

export interface DetailedScoreResult {
  score: number;
  grade: string;
  factors: ScoreFactor[];
  suggestions: string[];
  model_version: string;
  computed_at: string;
  features?: Record<string, unknown>;
}

export interface CreditScoreHistory {
  id: string;
  crew_member_id: string;
  score: number;
  grade: string;
  model_version: string;
  factors?: unknown;
  suggestions?: unknown;
  computed_at: string;
}

export interface LoanApplication {
  id: string;
  crew_member_id: string;
  amount_cents: number;
  amount_requested_cents: number;
  approved_amount_cents?: number;
  interest_rate?: number;
  tenure_days: number;
  category: string;
  purpose?: string;
  status: LoanStatus;
  disbursed_at?: string;
  due_date?: string;
  due_at?: string;
  total_repaid_cents: number;
  approved_by?: string;
  lender_id?: string;
  created_at: string;
}

export type LoanStatus = 'APPLIED' | 'PENDING' | 'APPROVED' | 'REJECTED' | 'DISBURSED' | 'REPAYING' | 'COMPLETED' | 'REPAID' | 'DEFAULTED';

export type LoanCategory = 'PERSONAL' | 'EMERGENCY' | 'EDUCATION' | 'BUSINESS' | 'ASSET';

export interface LoanTier {
  score: number;
  grade: string;
  max_loan_kes: number;
  interest_rate: number;
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

export interface NotificationPreference {
  id?: string;
  user_id?: string;
  sms_opt_in: boolean;
  push_opt_in: boolean;
  in_app_opt_in: boolean;
  marketing_opt_in: boolean;
  updated_at?: string;
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

export interface AdminUser {
  id: string;
  phone: string;
  email?: string;
  system_role: string;
  crew_member_id?: string;
  organization_id?: string;
  is_active: boolean;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface NotificationTemplate {
  id: string;
  event_name: string;
  channel: string;
  title_template: string;
  body_template: string;
  is_active: boolean;
}

export type DocumentType = 'KYC_ID_FRONT' | 'KYC_ID_BACK' | 'KYC_SELFIE' | 'SACCO_REGISTRATION' | 'VEHICLE_LOGBOOK' | 'OTHER';

export interface Document {
  id: string;
  crew_member_id?: string;
  organization_id?: string;
  vehicle_id?: string;
  document_type: DocumentType;
  file_name: string;
  file_size: number;
  mime_type: string;
  uploaded_by_id: string;
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
  organization_id: string;
  balance_cents: number;
  currency: string;
}

export interface SACCOMembership {
  id: string;
  organization_id: string;
  crew_member_id: string;
  role_in_sacco: string;
  joined_at: string;
  left_at?: string;
  is_active: boolean;
  pay_schedule_id?: string;
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

// --- Financial Profile (Phase E) ---

export interface OrgProfile {
  org_id: string;
  org_name: string;
  industry: IndustryType;
  role: string;
  joined_at: string;
  tenure_months: number;
  is_active: boolean;
  earnings_30d_cents: number;
  assignment_count_30d: number;
}

export interface LoanProduct {
  category: string;
  label: string;
  description?: string;
  max_amount_kes?: number;
}

export interface InsuranceProduct {
  type: string;
  label: string;
  description: string;
  provider?: string;
}

export interface FinancialProfile {
  crew_member_id: string;
  full_name: string;
  national_id: string;
  kyc_status: string;
  primary_work_type?: string;
  computed_at: string;
  composite_score: number;
  score_grade: string;
  org_count: number;
  cross_org_tenure_months: number;
  org_profiles: OrgProfile[];
  total_earnings_30d_cents: number;
  total_earnings_90d_cents: number;
  avg_daily_earnings_cents: number;
  earning_trend: string;
  wallet_balance_cents: number;
  savings_rate: number;
  total_loans_completed: number;
  total_loans_defaulted: number;
  on_time_repayment_rate: number;
  active_insurance_policies: number;
  available_loan_products?: LoanProduct[];
  available_insurance?: InsuranceProduct[];
  factors?: ScoreFactor[];
  suggestions?: string[];
}

// --- USSD Admin Models ---

export type MNOProvider = 'SAFARICOM' | 'AIRTEL' | 'TELKOM';
export type ShortcodeStatus = 'PENDING' | 'PROVISIONED' | 'ACTIVE' | 'SUSPENDED' | 'REJECTED';
export type ABTestStatus = 'DRAFT' | 'RUNNING' | 'PAUSED' | 'COMPLETED';

export interface ServiceCodeRoute {
  id: string;
  service_code: string;
  industry_type: IndustryType;
  organization_id?: string;
  organization_name?: string;
  is_active: boolean;
  roles: string[];
  created_at: string;
  updated_at: string;
}

export interface ShortcodeRequest {
  id: string;
  service_code: string;
  mno: MNOProvider;
  status: ShortcodeStatus;
  submitted_at: string;
  provisioned_at?: string;
  rejected_reason?: string;
  callback_url?: string;
}

export interface ABTest {
  id: string;
  name: string;
  service_code: string;
  variant_a_label: string;
  variant_b_label: string;
  variant_a_roles: string[];
  variant_b_roles: string[];
  traffic_split_pct: number;
  status: ABTestStatus;
  impressions_a: number;
  impressions_b: number;
  conversions_a: number;
  conversions_b: number;
  started_at?: string;
  ended_at?: string;
  created_at: string;
}
