-- Make audit_logs.user_id nullable to allow org-level operations
-- (e.g., float top-ups) that don't have a direct user actor.
-- Also drop the FK constraint that requires user_id to exist in users table,
-- since the column may now be NULL.

ALTER TABLE audit_logs ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_user_id_fkey;
