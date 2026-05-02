-- 000015_add_tenant_config.up.sql
-- Tenant & Industry Configuration Layer: Multi-industry workforce OS foundation.
-- Adds industry awareness to SACCOs (now "organizations"), configurable job types, and pay schedules.

-- 1. Extend saccos with tenant-level industry configuration
ALTER TABLE saccos
    ADD COLUMN IF NOT EXISTS industry_type VARCHAR(30) NOT NULL DEFAULT 'TRANSPORT'
        CHECK (industry_type IN ('TRANSPORT', 'CONSTRUCTION', 'HEALTH', 'LOGISTICS', 'AGRICULTURE', 'HOSPITALITY', 'GENERAL', 'CUSTOM')),
    ADD COLUMN IF NOT EXISTS tenant_config JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);

CREATE INDEX idx_saccos_industry_type ON saccos (industry_type);

-- 2. Configurable job types per tenant (replaces hardcoded DRIVER/CONDUCTOR/RIDER/OTHER)
CREATE TABLE tenant_job_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sacco_id UUID NOT NULL REFERENCES saccos(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,                 -- e.g. DRIVER, MASON, CHV, BOOKING_AGENT
    display_name VARCHAR(100) NOT NULL,        -- e.g. "Driver", "Mason", "Community Health Volunteer"
    category VARCHAR(30) NOT NULL DEFAULT 'PRIMARY'
        CHECK (category IN ('PRIMARY', 'FACILITATOR', 'SUPPORT', 'SUPERVISOR')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_job_types_sacco ON tenant_job_types (sacco_id);
CREATE UNIQUE INDEX idx_tenant_job_types_code ON tenant_job_types (sacco_id, code) WHERE is_active = true;
CREATE INDEX idx_tenant_job_types_category ON tenant_job_types (category);

-- 3. Configurable pay schedules per tenant
CREATE TABLE pay_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sacco_id UUID NOT NULL REFERENCES saccos(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,                -- e.g. "Daily Payout", "Weekly Friday", "Monthly End"
    frequency VARCHAR(20) NOT NULL
        CHECK (frequency IN ('DAILY', 'WEEKLY', 'BI_WEEKLY', 'MONTHLY')),
    pay_day INTEGER,                           -- Day of week (1=Mon..7=Sun) for WEEKLY; day of month for MONTHLY
    cutoff_hour INTEGER NOT NULL DEFAULT 17,   -- Hour (0-23) when period closes for earning aggregation
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pay_schedules_sacco ON pay_schedules (sacco_id);
CREATE UNIQUE INDEX idx_pay_schedules_default ON pay_schedules (sacco_id) WHERE is_default = true AND is_active = true;

-- 4. Link crew members to tenant-specific job types
ALTER TABLE crew_members
    ADD COLUMN IF NOT EXISTS job_type_id UUID REFERENCES tenant_job_types(id),
    ADD COLUMN IF NOT EXISTS job_title VARCHAR(100);

CREATE INDEX idx_crew_members_job_type ON crew_members (job_type_id) WHERE job_type_id IS NOT NULL;

-- 5. Backfill existing SACCOs: create default transport job types
-- This is done via a DO block so it's idempotent on re-run.
DO $$
DECLARE
    sacco_record RECORD;
BEGIN
    FOR sacco_record IN SELECT id FROM saccos WHERE deleted_at IS NULL LOOP
        -- Create default transport job types if none exist
        IF NOT EXISTS (SELECT 1 FROM tenant_job_types WHERE sacco_id = sacco_record.id) THEN
            INSERT INTO tenant_job_types (sacco_id, code, display_name, category, sort_order) VALUES
                (sacco_record.id, 'DRIVER',    'Driver',    'PRIMARY',     1),
                (sacco_record.id, 'CONDUCTOR', 'Conductor', 'PRIMARY',     2),
                (sacco_record.id, 'RIDER',     'Rider',     'PRIMARY',     3),
                (sacco_record.id, 'OTHER',     'Other',     'SUPPORT',     4);
        END IF;

        -- Create default daily pay schedule if none exists
        IF NOT EXISTS (SELECT 1 FROM pay_schedules WHERE sacco_id = sacco_record.id) THEN
            INSERT INTO pay_schedules (sacco_id, name, frequency, is_default) VALUES
                (sacco_record.id, 'Daily Payout', 'DAILY', true);
        END IF;
    END LOOP;
END $$;
