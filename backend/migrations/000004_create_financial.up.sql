-- 000004_create_financial.up.sql
-- Wallets, transactions, SACCO float

CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID NOT NULL REFERENCES crew_members(id),
    balance_cents BIGINT NOT NULL DEFAULT 0,
    total_credited_cents BIGINT NOT NULL DEFAULT 0,
    total_debited_cents BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    version INTEGER NOT NULL DEFAULT 0,    -- Optimistic lock
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_payout_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_wallets_crew_member ON wallets (crew_member_id);

CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    idempotency_key VARCHAR(255),
    transaction_type VARCHAR(10) NOT NULL CHECK (transaction_type IN ('CREDIT', 'DEBIT')),
    category VARCHAR(20) NOT NULL CHECK (category IN ('EARNING', 'WITHDRAWAL', 'DEDUCTION', 'TOP_UP', 'REVERSAL')),
    amount_cents BIGINT NOT NULL,
    balance_after_cents BIGINT,
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    reference VARCHAR(255),
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'COMPLETED', 'FAILED', 'REVERSED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_wallet_tx_idempotency ON wallet_transactions (idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_wallet_tx_wallet_created ON wallet_transactions (wallet_id, created_at);
CREATE INDEX idx_wallet_tx_status ON wallet_transactions (status);

CREATE TABLE sacco_floats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sacco_id UUID NOT NULL REFERENCES saccos(id),
    balance_cents BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    version INTEGER NOT NULL DEFAULT 0,    -- Optimistic lock
    last_funded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_sacco_floats_sacco ON sacco_floats (sacco_id);

CREATE TABLE sacco_float_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sacco_float_id UUID NOT NULL REFERENCES sacco_floats(id),
    idempotency_key VARCHAR(255),
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('FUND', 'PAYOUT', 'ADJUSTMENT')),
    amount_cents BIGINT NOT NULL,
    balance_after_cents BIGINT,
    currency VARCHAR(3) NOT NULL DEFAULT 'KES',
    reference VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'COMPLETED', 'FAILED', 'REVERSED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_sacco_float_tx_idempotency ON sacco_float_transactions (idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_sacco_float_tx_float_created ON sacco_float_transactions (sacco_float_id, created_at);
