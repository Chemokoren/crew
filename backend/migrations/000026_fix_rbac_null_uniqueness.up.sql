-- 000026_fix_rbac_null_uniqueness.up.sql
-- Fix RBAC uniqueness for nullable tenant_id columns.
--
-- PostgreSQL UNIQUE constraints allow multiple NULL values. The original
-- constraints therefore allowed duplicate global roles and duplicate global
-- user-role assignments. Partial unique indexes close that gap while keeping
-- tenant-scoped uniqueness explicit.

ALTER TABLE roles DROP CONSTRAINT IF EXISTS uq_roles_slug_tenant;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS uq_user_role_tenant;

CREATE UNIQUE INDEX IF NOT EXISTS uq_roles_slug_tenant_active
    ON roles (slug, tenant_id)
    WHERE tenant_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_roles_slug_global_active
    ON roles (slug)
    WHERE tenant_id IS NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_user_role_tenant_active
    ON user_roles (user_id, role_id, tenant_id)
    WHERE tenant_id IS NOT NULL AND is_active = true;

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_user_role_global_active
    ON user_roles (user_id, role_id)
    WHERE tenant_id IS NULL AND is_active = true;
