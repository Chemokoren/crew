-- 000020_create_work_sites.up.sql
CREATE TABLE IF NOT EXISTS work_sites (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID        NOT NULL REFERENCES saccos(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    project_ref     VARCHAR(100),
    address         VARCHAR(500),
    description     TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id   UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_work_sites_org ON work_sites(organization_id);
CREATE INDEX IF NOT EXISTS idx_work_sites_deleted_at ON work_sites(deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_work_sites_org_name
    ON work_sites(organization_id, name)
    WHERE deleted_at IS NULL;
