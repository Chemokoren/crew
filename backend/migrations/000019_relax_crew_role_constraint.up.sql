-- 000019_relax_crew_role_constraint.up.sql
-- Remove the restrictive CHECK constraint on crew_members.role to support
-- dynamic tenant job type codes (e.g., MASON, BOOKING_AGENT, SUPERVISOR).
-- Also widen the column from VARCHAR(20) to VARCHAR(50) to accommodate longer codes.

ALTER TABLE crew_members DROP CONSTRAINT IF EXISTS crew_members_role_check;
ALTER TABLE crew_members ALTER COLUMN role TYPE VARCHAR(50);
