-- Revert wallet_transactions category constraint back to original 5 values.
-- Note: any existing rows with category='LOAN' or 'ADJUSTMENT' must be
-- updated/removed before this down migration can succeed.

ALTER TABLE wallet_transactions
    DROP CONSTRAINT IF EXISTS wallet_transactions_category_check;

ALTER TABLE wallet_transactions
    ADD CONSTRAINT wallet_transactions_category_check
    CHECK (category::text = ANY (ARRAY[
        'EARNING'::text,
        'WITHDRAWAL'::text,
        'DEDUCTION'::text,
        'TOP_UP'::text,
        'REVERSAL'::text
    ]));
