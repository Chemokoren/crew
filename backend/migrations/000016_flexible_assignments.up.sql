-- 000016_flexible_assignments.up.sql
-- Generalize assignments to support non-transport work types.
-- vehicle_id becomes NULLABLE; new fields for work_type, work_site, hours, rates, check-in/out.

-- 1. Make vehicle_id nullable (was NOT NULL for transport-only)
ALTER TABLE assignments
    ALTER COLUMN vehicle_id DROP NOT NULL;

-- 2. Add generalized assignment fields
ALTER TABLE assignments
    ADD COLUMN IF NOT EXISTS work_type VARCHAR(20) NOT NULL DEFAULT 'SHIFT'
        CHECK (work_type IN ('SHIFT', 'DAILY', 'HOURLY', 'TASK', 'PROJECT', 'BOOKING')),
    ADD COLUMN IF NOT EXISTS work_site VARCHAR(255),
    ADD COLUMN IF NOT EXISTS project_ref VARCHAR(100),
    ADD COLUMN IF NOT EXISTS hours_worked DECIMAL(6,2),
    ADD COLUMN IF NOT EXISTS units_completed INTEGER,
    ADD COLUMN IF NOT EXISTS hourly_rate_cents BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS daily_rate_cents BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS per_unit_rate_cents BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS overtime_hours DECIMAL(6,2),
    ADD COLUMN IF NOT EXISTS overtime_rate_cents BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS check_in_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS check_out_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS pay_schedule_id UUID REFERENCES pay_schedules(id);

CREATE INDEX idx_assignments_work_type ON assignments (work_type);
CREATE INDEX idx_assignments_work_site ON assignments (work_site) WHERE work_site IS NOT NULL;
CREATE INDEX idx_assignments_project_ref ON assignments (project_ref) WHERE project_ref IS NOT NULL;
CREATE INDEX idx_assignments_check_in ON assignments (check_in_at) WHERE check_in_at IS NOT NULL;
CREATE INDEX idx_assignments_pay_schedule ON assignments (pay_schedule_id) WHERE pay_schedule_id IS NOT NULL;

-- 3. Extend earning_model enum: add new earning calculation types
-- (PostgreSQL CHECK constraints on assignment.earning_model — the model enum is in Go code)

-- 4. Add new earning types for non-transport work
-- (EarningType is a Go-level enum stored as varchar — no DB-level change needed)

-- 5. Backfill: tag existing assignments as SHIFT work type (already defaulted above)
