-- Revert: make audit_logs.user_id NOT NULL and restore FK constraint.

ALTER TABLE audit_logs ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);
