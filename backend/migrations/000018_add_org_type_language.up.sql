-- 000018_add_org_type_language.up.sql
-- Adds organization_type and default_language columns to saccos table.
-- Required by Organization model (Phase F) for industry-agnostic support.

ALTER TABLE saccos
    ADD COLUMN IF NOT EXISTS organization_type VARCHAR(50) NOT NULL DEFAULT 'SACCO'
        CHECK (organization_type IN ('SACCO', 'CONSTRUCTION_FIRM', 'LOGISTICS_COMPANY', 'HEALTH_NGO', 'AGRICULTURE_COOP', 'HOSPITALITY_GROUP', 'GENERAL')),
    ADD COLUMN IF NOT EXISTS default_language VARCHAR(10) NOT NULL DEFAULT 'sw';

-- Add preferred_language to users if missing (D5: user > org > system)
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS preferred_language VARCHAR(10) NOT NULL DEFAULT 'sw';
