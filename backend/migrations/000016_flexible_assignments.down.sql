-- 000016_flexible_assignments.down.sql
-- Rollback: Remove flexible assignment fields

DROP INDEX IF EXISTS idx_assignments_pay_schedule;
DROP INDEX IF EXISTS idx_assignments_check_in;
DROP INDEX IF EXISTS idx_assignments_project_ref;
DROP INDEX IF EXISTS idx_assignments_work_site;
DROP INDEX IF EXISTS idx_assignments_work_type;

ALTER TABLE assignments
    DROP COLUMN IF EXISTS pay_schedule_id,
    DROP COLUMN IF EXISTS check_out_at,
    DROP COLUMN IF EXISTS check_in_at,
    DROP COLUMN IF EXISTS overtime_rate_cents,
    DROP COLUMN IF EXISTS overtime_hours,
    DROP COLUMN IF EXISTS per_unit_rate_cents,
    DROP COLUMN IF EXISTS daily_rate_cents,
    DROP COLUMN IF EXISTS hourly_rate_cents,
    DROP COLUMN IF EXISTS units_completed,
    DROP COLUMN IF EXISTS hours_worked,
    DROP COLUMN IF EXISTS project_ref,
    DROP COLUMN IF EXISTS work_site,
    DROP COLUMN IF EXISTS work_type;

-- Restore vehicle_id NOT NULL (requires all rows to have a value)
-- ALTER TABLE assignments ALTER COLUMN vehicle_id SET NOT NULL;
