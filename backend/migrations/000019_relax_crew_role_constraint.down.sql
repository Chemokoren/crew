-- 000019_relax_crew_role_constraint.down.sql
-- Restore the original CHECK constraint and column type.

ALTER TABLE crew_members ALTER COLUMN role TYPE VARCHAR(20);
ALTER TABLE crew_members ADD CONSTRAINT crew_members_role_check CHECK (role IN ('DRIVER', 'CONDUCTOR', 'RIDER', 'OTHER'));
