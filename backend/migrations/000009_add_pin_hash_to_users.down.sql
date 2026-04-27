-- 000009_add_pin_hash_to_users.down.sql
ALTER TABLE users DROP COLUMN IF EXISTS pin_hash;
