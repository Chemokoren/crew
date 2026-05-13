-- Extend wallet_transactions_category_check to include 'LOAN' (loan disbursements)
-- and 'ADJUSTMENT' (manual balance corrections by admins).
-- The existing constraint must be dropped and re-created because PostgreSQL
-- does not support ALTER TABLE ... ADD VALUE to a CHECK constraint.

ALTER TABLE wallet_transactions
    DROP CONSTRAINT IF EXISTS wallet_transactions_category_check;

ALTER TABLE wallet_transactions
    ADD CONSTRAINT wallet_transactions_category_check
    CHECK (category::text = ANY (ARRAY[
        'EARNING'::text,
        'WITHDRAWAL'::text,
        'DEDUCTION'::text,
        'TOP_UP'::text,
        'REVERSAL'::text,
        'LOAN'::text,
        'ADJUSTMENT'::text
    ]));
