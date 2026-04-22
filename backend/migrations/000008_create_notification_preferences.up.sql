CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    sms_opt_in BOOLEAN DEFAULT TRUE,
    push_opt_in BOOLEAN DEFAULT TRUE,
    in_app_opt_in BOOLEAN DEFAULT TRUE,
    marketing_opt_in BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notification_preferences_user_id ON notification_preferences(user_id);
