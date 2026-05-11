-- 000022_update_system_role_check.down.sql
-- Reverts role names back to legacy values.

UPDATE users SET system_role = 'SACCO_ADMIN' WHERE system_role = 'EMPLOYER';
UPDATE users SET system_role = 'CREW' WHERE system_role = 'EMPLOYEE';

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_system_role_check;
ALTER TABLE users ADD CONSTRAINT users_system_role_check
    CHECK (system_role IN ('SYSTEM_ADMIN', 'SACCO_ADMIN', 'CREW', 'LENDER', 'INSURER'));
