-- Negative event tracking for credit scoring.
-- Records fraud flags, disputes, account locks, and other risk signals.

CREATE TABLE credit_negative_events (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID NOT NULL,
    event_type     VARCHAR(50) NOT NULL,  -- 'FRAUD_FLAG', 'DISPUTE', 'ACCOUNT_LOCK', 'KYC_FAILURE', 'REVERSED_TX'
    severity       VARCHAR(20) NOT NULL DEFAULT 'MEDIUM',  -- 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
    description    TEXT,
    resolved       BOOLEAN DEFAULT false,
    resolved_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cne_crew ON credit_negative_events (crew_member_id, created_at DESC);
CREATE INDEX idx_cne_type ON credit_negative_events (event_type);
