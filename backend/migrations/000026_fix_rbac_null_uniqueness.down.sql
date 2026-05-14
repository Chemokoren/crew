-- 000026_fix_rbac_null_uniqueness.down.sql

DROP INDEX IF EXISTS uq_user_roles_user_role_global_active;
DROP INDEX IF EXISTS uq_user_roles_user_role_tenant_active;
DROP INDEX IF EXISTS uq_roles_slug_global_active;
DROP INDEX IF EXISTS uq_roles_slug_tenant_active;

ALTER TABLE roles
    ADD CONSTRAINT uq_roles_slug_tenant UNIQUE (slug, tenant_id);

ALTER TABLE user_roles
    ADD CONSTRAINT uq_user_role_tenant UNIQUE (user_id, role_id, tenant_id);
