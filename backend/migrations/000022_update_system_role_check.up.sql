-- 000022_update_system_role_check.up.sql
-- Updates the system_role check constraint to use new role names:
--   SACCO_ADMIN → EMPLOYER
--   CREW        → EMPLOYEE

-- Step 1: Drop the old check constraint first (needed because some rows
-- may already contain the new values from application-level writes)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_system_role_check;

-- Step 2: Migrate any remaining old-style role values to the new names
UPDATE users SET system_role = 'EMPLOYER' WHERE system_role = 'SACCO_ADMIN';
UPDATE users SET system_role = 'EMPLOYEE' WHERE system_role = 'CREW';

-- Step 3: Add the new check constraint with updated role values
ALTER TABLE users ADD CONSTRAINT users_system_role_check
    CHECK (system_role IN ('SYSTEM_ADMIN', 'EMPLOYER', 'EMPLOYEE', 'LENDER', 'INSURER'));
