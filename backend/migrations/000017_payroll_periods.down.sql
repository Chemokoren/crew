-- 000017_payroll_periods.down.sql

ALTER TABLE crew_sacco_memberships DROP COLUMN IF EXISTS pay_schedule_id;

DROP INDEX IF EXISTS idx_payroll_runs_schedule;
ALTER TABLE payroll_runs
    DROP COLUMN IF EXISTS pay_period_id,
    DROP COLUMN IF EXISTS pay_schedule_id;

DROP INDEX IF EXISTS idx_pay_periods_dates;
DROP INDEX IF EXISTS idx_pay_periods_status;
DROP INDEX IF EXISTS idx_pay_periods_sacco;
DROP INDEX IF EXISTS idx_pay_periods_schedule;
DROP TABLE IF EXISTS pay_periods;
