-- Remove sync tracking columns from sacco_float_transactions
ALTER TABLE sacco_float_transactions
    DROP COLUMN IF EXISTS sync_method,
    DROP COLUMN IF EXISTS synced_at;
