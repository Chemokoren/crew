import { IndustryType, IndustryTemplate, IndustryPermission, JobTypeCategory } from '../models';

/**
 * Client-side mirror of backend IndustryTemplate configurations.
 * Used by the dashboard, sidebar, forms, and settings UI to adapt
 * labels, icons, fields, and available options per industry.
 *
 * Implements AD-9 (construction-first pilot) and AD-13 (template bootstrap).
 */
export const INDUSTRY_TEMPLATES: Record<string, IndustryTemplate> = {
  TRANSPORT: {
    industry_type: 'TRANSPORT',
    organization_type: 'SACCO',
    display_label: 'Transport SACCO',
    assignment_types: ['SHIFT'],
    earning_models: ['FIXED', 'COMMISSION', 'HYBRID'],
    payment_frequencies: ['DAILY', 'WEEKLY'],
    statutory_bodies: ['SHA', 'NSSF', 'HousingLevy'],
    default_job_types: [
      { code: 'DRIVER', display_name: 'Driver', category: 'PRIMARY' },
      { code: 'CONDUCTOR', display_name: 'Conductor', category: 'PRIMARY' },
      { code: 'RIDER', display_name: 'Boda Rider', category: 'PRIMARY' },
      { code: 'BOOKING_AGENT', display_name: 'Booking Agent', category: 'FACILITATOR' },
      { code: 'DISPATCHER', display_name: 'Dispatcher', category: 'FACILITATOR' },
      { code: 'SUPERVISOR', display_name: 'Fleet Supervisor', category: 'SUPERVISOR' },
      { code: 'OFFICE_ADMIN', display_name: 'Office Administrator', category: 'SUPPORT' },
    ],
    ui_labels: {
      assignment: 'Shift', work_site: 'Route', worker: 'Crew Member',
      organization: 'SACCO', vehicle: 'Vehicle', dashboard_title: 'Fleet Overview',
    },
  },
  CONSTRUCTION: {
    industry_type: 'CONSTRUCTION',
    organization_type: 'CONSTRUCTION_FIRM',
    display_label: 'Construction Firm',
    assignment_types: ['DAILY', 'HOURLY', 'PROJECT'],
    earning_models: ['DAILY_RATE', 'HOURLY', 'PER_TASK', 'FIXED'],
    payment_frequencies: ['DAILY', 'WEEKLY', 'BI_WEEKLY'],
    statutory_bodies: [],
    default_job_types: [
      { code: 'MASON', display_name: 'Mason', category: 'PRIMARY' },
      { code: 'CARPENTER', display_name: 'Carpenter', category: 'PRIMARY' },
      { code: 'PLUMBER', display_name: 'Plumber', category: 'PRIMARY' },
      { code: 'ELECTRICIAN', display_name: 'Electrician', category: 'PRIMARY' },
      { code: 'LABORER', display_name: 'General Laborer', category: 'PRIMARY' },
      { code: 'FOREMAN', display_name: 'Foreman', category: 'SUPERVISOR' },
      { code: 'SITE_MANAGER', display_name: 'Site Manager', category: 'SUPERVISOR' },
      { code: 'SAFETY_OFFICER', display_name: 'Safety Officer', category: 'SUPPORT' },
    ],
    ui_labels: {
      assignment: 'Job', work_site: 'Site', worker: 'Worker',
      organization: 'Contractor', vehicle: 'Equipment', dashboard_title: 'Site Overview',
    },
  },
  HEALTH: {
    industry_type: 'HEALTH',
    organization_type: 'HEALTH_NGO',
    display_label: 'Health Organization',
    assignment_types: ['DAILY', 'HOURLY', 'TASK'],
    earning_models: ['DAILY_RATE', 'PER_TASK', 'SALARY'],
    payment_frequencies: ['MONTHLY', 'BI_WEEKLY'],
    statutory_bodies: ['SHA'],
    default_job_types: [
      { code: 'CHV', display_name: 'Community Health Volunteer', category: 'PRIMARY' },
      { code: 'CHP', display_name: 'Community Health Promoter', category: 'PRIMARY' },
      { code: 'NURSE', display_name: 'Nurse', category: 'PRIMARY' },
      { code: 'COORDINATOR', display_name: 'Coordinator', category: 'SUPERVISOR' },
      { code: 'DATA_CLERK', display_name: 'Data Clerk', category: 'SUPPORT' },
    ],
    ui_labels: {
      assignment: 'Visit', work_site: 'Coverage Area', worker: 'Health Worker',
      organization: 'Health Facility', vehicle: 'Transport', dashboard_title: 'Health Dashboard',
    },
  },
  LOGISTICS: {
    industry_type: 'LOGISTICS',
    organization_type: 'LOGISTICS_COMPANY',
    display_label: 'Logistics Company',
    assignment_types: ['TASK', 'SHIFT', 'DAILY'],
    earning_models: ['PER_TASK', 'FIXED', 'COMMISSION'],
    payment_frequencies: ['DAILY', 'WEEKLY'],
    statutory_bodies: [],
    default_job_types: [
      { code: 'RIDER', display_name: 'Delivery Rider', category: 'PRIMARY' },
      { code: 'DRIVER', display_name: 'Driver', category: 'PRIMARY' },
      { code: 'LOADER', display_name: 'Loader', category: 'PRIMARY' },
      { code: 'DISPATCHER', display_name: 'Dispatcher', category: 'FACILITATOR' },
      { code: 'WAREHOUSE_MGR', display_name: 'Warehouse Manager', category: 'SUPERVISOR' },
    ],
    ui_labels: {
      assignment: 'Delivery', work_site: 'Warehouse', worker: 'Worker',
      organization: 'Company', vehicle: 'Vehicle', dashboard_title: 'Logistics Dashboard',
    },
  },
  AGRICULTURE: {
    industry_type: 'AGRICULTURE',
    organization_type: 'AGRICULTURE_COOP',
    display_label: 'Agricultural Cooperative',
    assignment_types: ['DAILY', 'HOURLY', 'PROJECT'],
    earning_models: ['DAILY_RATE', 'HOURLY', 'PER_PIECE'],
    payment_frequencies: ['DAILY', 'WEEKLY'],
    statutory_bodies: [],
    default_job_types: [
      { code: 'PICKER', display_name: 'Picker / Harvester', category: 'PRIMARY' },
      { code: 'FIELD_WORKER', display_name: 'Field Worker', category: 'PRIMARY' },
      { code: 'SORTER', display_name: 'Sorter / Grader', category: 'PRIMARY' },
      { code: 'TEAM_LEAD', display_name: 'Team Leader', category: 'SUPERVISOR' },
      { code: 'WEIGHBRIDGE', display_name: 'Weighbridge Operator', category: 'SUPPORT' },
    ],
    ui_labels: {
      assignment: 'Task', work_site: 'Farm Block', worker: 'Worker',
      organization: 'Cooperative', vehicle: 'Tractor', dashboard_title: 'Farm Dashboard',
    },
  },
  HOSPITALITY: {
    industry_type: 'HOSPITALITY',
    organization_type: 'HOSPITALITY_GROUP',
    display_label: 'Hospitality Group',
    assignment_types: ['SHIFT', 'DAILY'],
    earning_models: ['DAILY_RATE', 'HOURLY', 'SALARY'],
    payment_frequencies: ['WEEKLY', 'BI_WEEKLY', 'MONTHLY'],
    statutory_bodies: ['SHA', 'NSSF'],
    default_job_types: [
      { code: 'WAITER', display_name: 'Waiter / Waitress', category: 'PRIMARY' },
      { code: 'COOK', display_name: 'Cook', category: 'PRIMARY' },
      { code: 'HOUSEKEEPER', display_name: 'Housekeeper', category: 'PRIMARY' },
      { code: 'RECEPTIONIST', display_name: 'Receptionist', category: 'SUPPORT' },
      { code: 'SHIFT_LEAD', display_name: 'Shift Lead', category: 'SUPERVISOR' },
    ],
    ui_labels: {
      assignment: 'Shift', work_site: 'Location', worker: 'Staff',
      organization: 'Establishment', vehicle: 'Transport', dashboard_title: 'Operations Dashboard',
    },
  },
  GENERAL: {
    industry_type: 'GENERAL',
    organization_type: 'GENERAL',
    display_label: 'Organization',
    assignment_types: ['DAILY', 'SHIFT', 'HOURLY', 'TASK', 'PROJECT'],
    earning_models: ['FIXED', 'DAILY_RATE', 'HOURLY', 'PER_TASK', 'COMMISSION', 'SALARY'],
    payment_frequencies: ['DAILY', 'WEEKLY', 'BI_WEEKLY', 'MONTHLY'],
    statutory_bodies: ['SHA', 'NSSF', 'HousingLevy'],
    default_job_types: [
      { code: 'WORKER', display_name: 'Worker', category: 'PRIMARY' },
      { code: 'TEAM_LEAD', display_name: 'Team Lead', category: 'SUPERVISOR' },
      { code: 'ADMIN', display_name: 'Administrator', category: 'SUPPORT' },
    ],
    ui_labels: {
      assignment: 'Assignment', work_site: 'Location', worker: 'Worker',
      organization: 'Organization', vehicle: 'Vehicle', dashboard_title: 'Dashboard',
    },
  },
  CUSTOM: {
    industry_type: 'CUSTOM' as IndustryType,
    organization_type: 'GENERAL',
    display_label: 'Custom Industry',
    assignment_types: ['DAILY', 'SHIFT', 'HOURLY', 'TASK', 'PROJECT', 'BOOKING'],
    earning_models: ['FIXED', 'DAILY_RATE', 'HOURLY', 'PER_TASK', 'PER_PIECE', 'COMMISSION', 'SALARY', 'HYBRID'],
    payment_frequencies: ['DAILY', 'WEEKLY', 'BI_WEEKLY', 'MONTHLY'],
    statutory_bodies: ['SHA', 'NSSF', 'HousingLevy'],
    default_job_types: [
      { code: 'WORKER', display_name: 'Worker', category: 'PRIMARY' },
      { code: 'SUPERVISOR', display_name: 'Supervisor', category: 'SUPERVISOR' },
    ],
    ui_labels: {
      assignment: 'Assignment', work_site: 'Location', worker: 'Worker',
      organization: 'Organization', vehicle: 'Asset', dashboard_title: 'Dashboard',
    },
  },
};

/**
 * Returns the IndustryTemplate for a given industry type.
 * Falls back to GENERAL if the industry is not in the registry.
 */
export function getIndustryTemplate(industry?: IndustryType | string): IndustryTemplate {
  if (industry && INDUSTRY_TEMPLATES[industry]) {
    return INDUSTRY_TEMPLATES[industry];
  }
  return INDUSTRY_TEMPLATES['GENERAL'];
}

/**
 * Returns a UI label for a given key, adapted to the current industry.
 * Example: getLabel('CONSTRUCTION', 'assignment') → 'Job'
 */
export function getIndustryLabel(industry: IndustryType | string | undefined, key: string): string {
  const tmpl = getIndustryTemplate(industry);
  return tmpl.ui_labels[key] || key;
}

/**
 * Role-based permission matrix — defines default permissions by JobTypeCategory.
 * Organizations can override these in their config (AD-7 RoleConfig).
 */
export const ROLE_PERMISSIONS: Record<string, IndustryPermission> = {
  PRIMARY: {
    role_category: 'PRIMARY',
    can_create_assignments: false,
    can_approve_earnings: false,
    can_manage_payroll: false,
    can_view_financial_profiles: false,
    can_manage_crew: false,
    can_manage_settings: false,
  },
  FACILITATOR: {
    role_category: 'FACILITATOR',
    can_create_assignments: true,
    can_approve_earnings: false,
    can_manage_payroll: false,
    can_view_financial_profiles: false,
    can_manage_crew: false,
    can_manage_settings: false,
  },
  SUPERVISOR: {
    role_category: 'SUPERVISOR',
    can_create_assignments: true,
    can_approve_earnings: true,
    can_manage_payroll: false,
    can_view_financial_profiles: true,
    can_manage_crew: true,
    can_manage_settings: false,
  },
  SUPPORT: {
    role_category: 'SUPPORT',
    can_create_assignments: false,
    can_approve_earnings: false,
    can_manage_payroll: true,
    can_view_financial_profiles: true,
    can_manage_crew: true,
    can_manage_settings: true,
  },
};

/**
 * Returns the default permission set for a job type category.
 */
export function getRolePermissions(category: JobTypeCategory | string): IndustryPermission {
  return ROLE_PERMISSIONS[category] || ROLE_PERMISSIONS['PRIMARY'];
}

/**
 * Dashboard icon mapping per industry
 */
export const INDUSTRY_ICONS: Record<string, string> = {
  TRANSPORT: 'directions_bus',
  CONSTRUCTION: 'construction',
  HEALTH: 'health_and_safety',
  LOGISTICS: 'local_shipping',
  AGRICULTURE: 'agriculture',
  HOSPITALITY: 'hotel',
  GENERAL: 'business',
  CUSTOM: 'tune',
};
