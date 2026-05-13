-- 000025_create_rbac_tables.down.sql
-- Drop RBAC tables in reverse dependency order.

DROP TABLE IF EXISTS role_templates CASCADE;
DROP TABLE IF EXISTS policies CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
