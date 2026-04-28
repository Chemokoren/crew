-- Rollback: Remove loan category columns.

DROP INDEX IF EXISTS idx_loan_crew_status;
DROP INDEX IF EXISTS idx_loan_crew_cat;

ALTER TABLE loan_applications
  DROP COLUMN IF EXISTS purpose,
  DROP COLUMN IF EXISTS category;
