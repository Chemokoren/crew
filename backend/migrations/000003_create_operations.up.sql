-- 000003_create_operations.up.sql
-- Assignments, earnings, daily summaries

CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID NOT NULL REFERENCES crew_members(id),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id),
    sacco_id UUID NOT NULL REFERENCES saccos(id),
    route_id UUID REFERENCES routes(id),
    shift_date DATE NOT NULL,
    shift_start TIMESTAMPTZ NOT NULL,
    shift_end TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'ACTIVE', 'COMPLETED', 'CANCELLED')),

    -- Earning configuration
    earning_model VARCHAR(20) NOT NULL CHECK (earning_model IN ('FIXED', 'COMMISSION', 'HYBRID')),
    fixed_amount_cents BIGINT NOT NULL DEFAULT 0,
    commission_rate DECIMAL(5,4),           -- 0.1500 = 15%
    hybrid_base_cents BIGINT NOT NULL DEFAULT 0,
    commission_basis VARCHAR(20) CHECK (commission_basis IN ('FARE_TOTAL', 'TRIP_COUNT', 'REVENUE')),

    notes TEXT,
    created_by_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assignments_crew_date ON assignments (crew_member_id, shift_date);
CREATE INDEX idx_assignments_vehicle_date ON assignments (vehicle_id, shift_date);
CREATE INDEX idx_assignments_sacco_status ON assignments (sacco_id, status);
CREATE INDEX idx_assignments_status ON assignments (status);
CREATE INDEX idx_assignments_shift_date ON assignments (shift_date);

CREATE TABLE earnings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES assignments(id),
    crew_member_id UUID NOT NULL REFERENCES crew_members(id),
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    earning_type VARCHAR(20) NOT NULL CHECK (earning_type IN ('SHIFT_PAY', 'COMMISSION', 'BONUS', 'ADJUSTMENT')),
    description TEXT,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    earned_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_earnings_crew_date ON earnings (crew_member_id, earned_at);
CREATE INDEX idx_earnings_assignment ON earnings (assignment_id);
CREATE INDEX idx_earnings_earned_at ON earnings (earned_at);

CREATE TABLE daily_earnings_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID NOT NULL REFERENCES crew_members(id),
    date DATE NOT NULL,
    total_earned_cents BIGINT NOT NULL DEFAULT 0,
    total_deductions_cents BIGINT NOT NULL DEFAULT 0,
    net_amount_cents BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    assignment_count INTEGER NOT NULL DEFAULT 0,
    is_processed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_daily_summary_unique ON daily_earnings_summaries (crew_member_id, date);
CREATE INDEX idx_daily_summary_date ON daily_earnings_summaries (date);
CREATE INDEX idx_daily_summary_processed ON daily_earnings_summaries (is_processed) WHERE is_processed = false;
