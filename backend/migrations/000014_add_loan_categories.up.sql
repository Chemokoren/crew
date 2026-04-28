-- Add loan category and purpose columns to loan_applications.
-- Part of the concurrent loan policy system.
-- Categories allow per-category concurrency (e.g., Personal + Education simultaneously).

ALTER TABLE loan_applications
  ADD COLUMN IF NOT EXISTS category VARCHAR(30) NOT NULL DEFAULT 'PERSONAL',
  ADD COLUMN IF NOT EXISTS purpose VARCHAR(255);

-- Composite index for per-category active loan lookups
CREATE INDEX IF NOT EXISTS idx_loan_crew_cat ON loan_applications(crew_member_id, category);

-- Index for active loan status queries
CREATE INDEX IF NOT EXISTS idx_loan_crew_status ON loan_applications(crew_member_id, status);
