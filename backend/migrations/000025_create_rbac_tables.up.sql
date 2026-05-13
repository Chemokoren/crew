-- 000025_create_rbac_tables.up.sql
-- Enterprise-Grade RBAC schema for AMY Workforce Financial Operating System.
-- Tables: roles, permissions, role_permissions, user_roles, policies, role_templates

-- ============================================================================
-- 1. PERMISSIONS — Central permission registry (seeded from Go registry)
-- ============================================================================
CREATE TABLE IF NOT EXISTS permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key         VARCHAR(120) NOT NULL,
    module      VARCHAR(60)  NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    risk_level  VARCHAR(20)  NOT NULL DEFAULT 'low',   -- low, medium, high, critical
    category    VARCHAR(60)  NOT NULL DEFAULT '',       -- CRUD, workflow, financial, compliance
    is_system   BOOLEAN      NOT NULL DEFAULT true,     -- system-defined (immutable key)
    depends_on  TEXT[]       DEFAULT '{}',              -- permission keys this depends on
    metadata    JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_permissions_key UNIQUE (key)
);

CREATE INDEX idx_permissions_module   ON permissions (module);
CREATE INDEX idx_permissions_category ON permissions (category);

-- ============================================================================
-- 2. ROLES — Tenant-scoped or global roles
-- ============================================================================
CREATE TABLE IF NOT EXISTS roles (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(120) NOT NULL,
    slug           VARCHAR(120) NOT NULL,
    description    TEXT         NOT NULL DEFAULT '',
    tenant_id      UUID         REFERENCES saccos(id) ON DELETE CASCADE,  -- NULL = global
    industry_type  VARCHAR(30)  NOT NULL DEFAULT '',
    is_system      BOOLEAN      NOT NULL DEFAULT false,  -- system-defined (cannot delete)
    is_template    BOOLEAN      NOT NULL DEFAULT false,   -- template role (cloneable)
    is_active      BOOLEAN      NOT NULL DEFAULT true,
    parent_role_id UUID         REFERENCES roles(id) ON DELETE SET NULL,
    metadata       JSONB        NOT NULL DEFAULT '{}',
    created_by     UUID         REFERENCES users(id) ON DELETE SET NULL,
    updated_by     UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,

    -- Slug must be unique within a tenant (or globally if tenant_id IS NULL)
    CONSTRAINT uq_roles_slug_tenant UNIQUE (slug, tenant_id)
);

CREATE INDEX idx_roles_tenant       ON roles (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_roles_industry     ON roles (industry_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_roles_system       ON roles (is_system) WHERE deleted_at IS NULL;
CREATE INDEX idx_roles_template     ON roles (is_template) WHERE deleted_at IS NULL;
CREATE INDEX idx_roles_active       ON roles (is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_roles_deleted      ON roles (deleted_at) WHERE deleted_at IS NOT NULL;

-- ============================================================================
-- 3. ROLE_PERMISSIONS — Many-to-many join: which permissions each role has
-- ============================================================================
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    granted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role ON role_permissions (role_id);
CREATE INDEX idx_role_permissions_perm ON role_permissions (permission_id);

-- ============================================================================
-- 4. USER_ROLES — Maps users to roles within a tenant context
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    tenant_id   UUID        REFERENCES saccos(id) ON DELETE CASCADE,  -- NULL = global assignment
    assigned_by UUID        REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ,                                          -- NULL = permanent
    is_active   BOOLEAN     NOT NULL DEFAULT true,

    -- A user can only hold a given role once per tenant
    CONSTRAINT uq_user_role_tenant UNIQUE (user_id, role_id, tenant_id)
);

CREATE INDEX idx_user_roles_user    ON user_roles (user_id) WHERE is_active = true;
CREATE INDEX idx_user_roles_role    ON user_roles (role_id) WHERE is_active = true;
CREATE INDEX idx_user_roles_tenant  ON user_roles (tenant_id) WHERE is_active = true;
CREATE INDEX idx_user_roles_expires ON user_roles (expires_at) WHERE expires_at IS NOT NULL AND is_active = true;

-- ============================================================================
-- 5. POLICIES — Dynamic policy-based access control (optional layer)
-- ============================================================================
CREATE TABLE IF NOT EXISTS policies (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(120) NOT NULL,
    description    TEXT         NOT NULL DEFAULT '',
    permission_key VARCHAR(120) NOT NULL,                   -- which permission this constrains
    conditions     JSONB        NOT NULL DEFAULT '{}',      -- JSON condition tree
    effect         VARCHAR(10)  NOT NULL DEFAULT 'DENY',    -- ALLOW or DENY
    is_active      BOOLEAN      NOT NULL DEFAULT true,
    priority       INT          NOT NULL DEFAULT 0,         -- higher = evaluated first
    tenant_id      UUID         REFERENCES saccos(id) ON DELETE CASCADE,
    created_by     UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_policy_effect CHECK (effect IN ('ALLOW', 'DENY'))
);

CREATE INDEX idx_policies_perm_key ON policies (permission_key) WHERE is_active = true;
CREATE INDEX idx_policies_tenant   ON policies (tenant_id) WHERE is_active = true;

-- ============================================================================
-- 6. ROLE_TEMPLATES — Pre-built industry role templates (seeded)
-- ============================================================================
CREATE TABLE IF NOT EXISTS role_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    industry_type VARCHAR(30)  NOT NULL,
    role_name     VARCHAR(120) NOT NULL,
    role_slug     VARCHAR(120) NOT NULL,
    description   TEXT         NOT NULL DEFAULT '',
    permissions   JSONB        NOT NULL DEFAULT '[]',   -- array of permission key strings
    is_default    BOOLEAN      NOT NULL DEFAULT false,
    sort_order    INT          NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_role_templates_slug_industry UNIQUE (role_slug, industry_type)
);

CREATE INDEX idx_role_templates_industry ON role_templates (industry_type);
