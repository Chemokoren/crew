-- 000007_create_financial_services.up.sql
-- Credit scores, loans, insurance (Phase 3 — tables created now, logic later)

CREATE TABLE credit_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID NOT NULL REFERENCES crew_members(id),
    score INTEGER NOT NULL DEFAULT 0,    -- 0-1000
    factors JSONB,
    last_calculated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_credit_scores_crew ON credit_scores (crew_member_id);

CREATE TABLE loan_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID NOT NULL REFERENCES crew_members(id),
    amount_requested_cents BIGINT,
    amount_approved_cents BIGINT,
    interest_rate DECIMAL(5,4),
    tenure_days INTEGER,
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    status VARCHAR(20) NOT NULL DEFAULT 'APPLIED' CHECK (status IN ('APPLIED', 'APPROVED', 'DISBURSED', 'REPAYING', 'COMPLETED', 'DEFAULTED')),
    lender_id UUID REFERENCES users(id),
    disbursed_at TIMESTAMPTZ,
    due_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loans_crew ON loan_applications (crew_member_id);
CREATE INDEX idx_loans_status ON loan_applications (status);
CREATE INDEX idx_loans_lender ON loan_applications (lender_id) WHERE lender_id IS NOT NULL;

CREATE TABLE insurance_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID NOT NULL REFERENCES crew_members(id),
    provider VARCHAR(100) NOT NULL,
    policy_type VARCHAR(50) NOT NULL,
    premium_amount_cents BIGINT,
    premium_frequency VARCHAR(20) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'LAPSED', 'CLAIMED')),
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_insurance_crew ON insurance_policies (crew_member_id);
CREATE INDEX idx_insurance_status ON insurance_policies (status);
