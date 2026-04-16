-- 000002_create_identity_registry.up.sql
-- SACCOs, crew members, routes, vehicles, memberships

-- Crew ID sequence (CRW-00001, CRW-00002, ...)
CREATE SEQUENCE crew_id_seq START 1;

CREATE TABLE saccos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    registration_number VARCHAR(100) NOT NULL,
    county VARCHAR(100) NOT NULL,
    sub_county VARCHAR(100),
    contact_phone VARCHAR(20) NOT NULL,
    contact_email VARCHAR(255),
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ              -- Soft delete
);

CREATE UNIQUE INDEX idx_saccos_registration_number ON saccos (registration_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_saccos_county ON saccos (county);
CREATE INDEX idx_saccos_deleted_at ON saccos (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE crew_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_id VARCHAR(20) NOT NULL,       -- Human-readable: CRW-00001
    national_id VARCHAR(255) NOT NULL,  -- Should be encrypted at application level
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    kyc_status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (kyc_status IN ('PENDING', 'VERIFIED', 'REJECTED')),
    kyc_verified_at TIMESTAMPTZ,
    photo_url TEXT,
    role VARCHAR(20) NOT NULL CHECK (role IN ('DRIVER', 'CONDUCTOR', 'RIDER', 'OTHER')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ              -- Soft delete
);

CREATE UNIQUE INDEX idx_crew_members_crew_id ON crew_members (crew_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_crew_members_kyc_status ON crew_members (kyc_status);
CREATE INDEX idx_crew_members_role ON crew_members (role);
CREATE INDEX idx_crew_members_deleted_at ON crew_members (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    start_point VARCHAR(255) NOT NULL,
    end_point VARCHAR(255) NOT NULL,
    estimated_distance_km DOUBLE PRECISION,
    base_fare_cents BIGINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_routes_deleted_at ON routes (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sacco_id UUID NOT NULL REFERENCES saccos(id),
    registration_no VARCHAR(20) NOT NULL,
    vehicle_type VARCHAR(20) NOT NULL CHECK (vehicle_type IN ('MATATU', 'BODA', 'TUK_TUK')),
    route_id UUID REFERENCES routes(id),
    capacity INTEGER,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_vehicles_registration_no ON vehicles (registration_no) WHERE deleted_at IS NULL;
CREATE INDEX idx_vehicles_sacco_id ON vehicles (sacco_id);
CREATE INDEX idx_vehicles_route_id ON vehicles (route_id) WHERE route_id IS NOT NULL;
CREATE INDEX idx_vehicles_deleted_at ON vehicles (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE crew_sacco_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID NOT NULL REFERENCES crew_members(id),
    sacco_id UUID NOT NULL REFERENCES saccos(id),
    role_in_sacco VARCHAR(20) NOT NULL DEFAULT 'MEMBER' CHECK (role_in_sacco IN ('MEMBER', 'ADMIN', 'CHAIRMAN')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_memberships_crew_member ON crew_sacco_memberships (crew_member_id);
CREATE INDEX idx_memberships_sacco ON crew_sacco_memberships (sacco_id);
CREATE UNIQUE INDEX idx_memberships_active ON crew_sacco_memberships (crew_member_id, sacco_id) WHERE is_active = true;

-- Now add FKs from users to crew_members and saccos
ALTER TABLE users ADD CONSTRAINT fk_users_crew_member FOREIGN KEY (crew_member_id) REFERENCES crew_members(id);
ALTER TABLE users ADD CONSTRAINT fk_users_sacco FOREIGN KEY (sacco_id) REFERENCES saccos(id);
