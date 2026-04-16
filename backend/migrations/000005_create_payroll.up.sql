-- 000005_create_payroll.up.sql
-- Payroll runs, entries, statutory rates

CREATE TABLE payroll_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sacco_id UUID NOT NULL REFERENCES saccos(id),
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PROCESSING', 'APPROVED', 'SUBMITTED', 'COMPLETED')),
    total_gross_cents BIGINT,
    total_deductions_cents BIGINT,
    total_net_cents BIGINT,
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    processed_by_id UUID REFERENCES users(id),
    approved_by_id UUID REFERENCES users(id),
    submitted_at TIMESTAMPTZ,
    perpay_reference VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payroll_runs_sacco ON payroll_runs (sacco_id);
CREATE INDEX idx_payroll_runs_status ON payroll_runs (status);
CREATE INDEX idx_payroll_runs_period ON payroll_runs (period_start, period_end);

CREATE TABLE payroll_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_run_id UUID NOT NULL REFERENCES payroll_runs(id) ON DELETE CASCADE,
    crew_member_id UUID NOT NULL REFERENCES crew_members(id),
    gross_earnings_cents BIGINT,
    sha_deduction_cents BIGINT,
    nssf_deduction_cents BIGINT,
    housing_levy_deduction_cents BIGINT,
    other_deductions_cents BIGINT,
    net_pay_cents BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payroll_entries_run ON payroll_entries (payroll_run_id);
CREATE INDEX idx_payroll_entries_crew ON payroll_entries (crew_member_id);

CREATE TABLE statutory_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL,          -- SHA | NSSF | HOUSING_LEVY
    rate DECIMAL(8,4),
    rate_type VARCHAR(20) NOT NULL CHECK (rate_type IN ('PERCENTAGE', 'FIXED', 'TIERED')),
    effective_from DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_statutory_rates_name ON statutory_rates (name, is_active);
