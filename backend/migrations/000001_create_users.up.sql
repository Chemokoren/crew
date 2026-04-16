-- 000001_create_users.up.sql
-- Core identity: users and authentication

CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- For gen_random_uuid()

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(20) NOT NULL,
    email VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    system_role VARCHAR(20) NOT NULL CHECK (system_role IN ('SYSTEM_ADMIN', 'SACCO_ADMIN', 'CREW', 'LENDER', 'INSURER')),
    crew_member_id UUID,          -- FK added after crew_members table exists
    sacco_id UUID,                -- FK added after saccos table exists
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_phone ON users (phone);
CREATE INDEX idx_users_system_role ON users (system_role);
CREATE INDEX idx_users_crew_member_id ON users (crew_member_id) WHERE crew_member_id IS NOT NULL;
CREATE INDEX idx_users_sacco_id ON users (sacco_id) WHERE sacco_id IS NOT NULL;
