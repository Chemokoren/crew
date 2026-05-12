-- Add sync tracking columns to sacco_float_transactions
ALTER TABLE sacco_float_transactions
    ADD COLUMN IF NOT EXISTS sync_method VARCHAR(20) DEFAULT '' NOT NULL,
    ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ;
