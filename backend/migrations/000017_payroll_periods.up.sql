-- 000017_payroll_periods.up.sql
-- Phase D: Multi-schedule pay periods, per-worker schedule overrides, statutory exemptions.

-- 1. Pay periods track individual pay windows within a schedule
CREATE TABLE IF NOT EXISTS pay_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pay_schedule_id UUID NOT NULL REFERENCES pay_schedules(id),
    sacco_id UUID NOT NULL REFERENCES saccos(id),
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN'
        CHECK (status IN ('OPEN', 'CLOSED', 'PROCESSING', 'COMPLETED')),
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (pay_schedule_id, period_start)
);

CREATE INDEX idx_pay_periods_schedule ON pay_periods (pay_schedule_id);
CREATE INDEX idx_pay_periods_sacco ON pay_periods (sacco_id);
CREATE INDEX idx_pay_periods_status ON pay_periods (status);
CREATE INDEX idx_pay_periods_dates ON pay_periods (period_start, period_end);

-- 2. Link payroll runs to a pay schedule and pay period
ALTER TABLE payroll_runs
    ADD COLUMN IF NOT EXISTS pay_schedule_id UUID REFERENCES pay_schedules(id),
    ADD COLUMN IF NOT EXISTS pay_period_id UUID REFERENCES pay_periods(id);

CREATE INDEX idx_payroll_runs_schedule ON payroll_runs (pay_schedule_id) WHERE pay_schedule_id IS NOT NULL;

-- 3. Per-worker pay schedule override on membership
ALTER TABLE crew_sacco_memberships
    ADD COLUMN IF NOT EXISTS pay_schedule_id UUID REFERENCES pay_schedules(id);

CREATE INDEX idx_memberships_pay_schedule ON crew_sacco_memberships (pay_schedule_id) WHERE pay_schedule_id IS NOT NULL;

-- 4. Earning filter by sacco_id for schedule-aware payroll aggregation
-- (earning.crew_member → crew_member.sacco via membership — no schema change needed, just service logic)

-- 5. Statutory exemptions per job type (stored in tenant_config JSONB, no schema change)
