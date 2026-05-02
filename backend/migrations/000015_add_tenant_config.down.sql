-- 000015_add_tenant_config.down.sql
-- Rollback: Remove tenant configuration layer

ALTER TABLE crew_members DROP COLUMN IF EXISTS job_title;
ALTER TABLE crew_members DROP COLUMN IF EXISTS job_type_id;

DROP TABLE IF EXISTS pay_schedules;
DROP TABLE IF EXISTS tenant_job_types;

DROP INDEX IF EXISTS idx_saccos_industry_type;
ALTER TABLE saccos DROP COLUMN IF EXISTS display_name;
ALTER TABLE saccos DROP COLUMN IF EXISTS tenant_config;
ALTER TABLE saccos DROP COLUMN IF EXISTS industry_type;
