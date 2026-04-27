-- Add repayment tracking columns to loan_applications for accurate
-- credit scoring (on_time_repayment_rate computation).

ALTER TABLE loan_applications
    ADD COLUMN repaid_at         TIMESTAMPTZ,
    ADD COLUMN total_repaid_cents BIGINT DEFAULT 0,
    ADD COLUMN days_past_due     INT DEFAULT 0;

-- Index for the default-detection worker: find overdue disbursed loans
CREATE INDEX idx_loans_overdue ON loan_applications (status, due_at)
    WHERE status IN ('DISBURSED', 'REPAYING');
