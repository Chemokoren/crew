-- 000009_add_pin_hash_to_users.up.sql
-- Add transaction PIN support for USSD withdrawals

ALTER TABLE users ADD COLUMN IF NOT EXISTS pin_hash VARCHAR(255);
