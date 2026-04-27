DROP INDEX IF EXISTS idx_loans_overdue;

ALTER TABLE loan_applications
    DROP COLUMN IF EXISTS repaid_at,
    DROP COLUMN IF EXISTS total_repaid_cents,
    DROP COLUMN IF EXISTS days_past_due;
