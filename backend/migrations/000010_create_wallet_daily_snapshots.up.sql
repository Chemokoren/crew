-- Daily wallet balance snapshots for accurate credit scoring.
-- One row per wallet per day, recording the closing balance.
-- Used by the credit scoring engine to compute true 30-day average balances.

CREATE TABLE wallet_daily_snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id       UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    crew_member_id  UUID NOT NULL,
    balance_cents   BIGINT NOT NULL,
    currency        VARCHAR(3) NOT NULL DEFAULT 'KES',
    snapshot_date   DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- One snapshot per wallet per day
    CONSTRAINT uq_wallet_snapshot_date UNIQUE (wallet_id, snapshot_date)
);

-- Fast lookups: "get last 30 snapshots for crew member X"
CREATE INDEX idx_wds_crew_date ON wallet_daily_snapshots (crew_member_id, snapshot_date DESC);

-- Fast lookups for the worker: "find all wallets that need snapshots today"
CREATE INDEX idx_wds_wallet_date ON wallet_daily_snapshots (wallet_id, snapshot_date DESC);
