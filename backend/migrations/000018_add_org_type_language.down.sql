-- 000018_add_org_type_language.down.sql
ALTER TABLE saccos
    DROP COLUMN IF EXISTS organization_type,
    DROP COLUMN IF EXISTS default_language;

ALTER TABLE users
    DROP COLUMN IF EXISTS preferred_language;
