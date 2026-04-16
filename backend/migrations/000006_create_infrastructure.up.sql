-- 000006_create_infrastructure.up.sql
-- Webhooks, notifications, documents, audit logs

CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(20) NOT NULL CHECK (source IN ('JAMBOPAY', 'PERPAY', 'IPRS')),
    event_type VARCHAR(50) NOT NULL,
    external_ref VARCHAR(255),
    payload JSONB NOT NULL,
    is_processed BOOLEAN NOT NULL DEFAULT false,
    processed_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_events_source_processed ON webhook_events (source, is_processed);
CREATE INDEX idx_webhook_events_external_ref ON webhook_events (external_ref) WHERE external_ref IS NOT NULL;

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    channel VARCHAR(10) NOT NULL CHECK (channel IN ('SMS', 'PUSH', 'IN_APP')),
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    data JSONB,
    status VARCHAR(10) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'SENT', 'FAILED', 'READ')),
    sent_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_status ON notifications (user_id, status);
CREATE INDEX idx_notifications_created ON notifications (created_at);

CREATE TABLE notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_name VARCHAR(100) NOT NULL,
    channel VARCHAR(10) NOT NULL CHECK (channel IN ('SMS', 'PUSH', 'IN_APP')),
    title_template VARCHAR(500) NOT NULL,
    body_template TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX idx_notif_templates_event_channel ON notification_templates (event_name, channel);

CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    crew_member_id UUID REFERENCES crew_members(id),
    sacco_id UUID REFERENCES saccos(id),
    vehicle_id UUID REFERENCES vehicles(id),
    document_type VARCHAR(30) NOT NULL CHECK (document_type IN ('KYC_ID_FRONT', 'KYC_ID_BACK', 'KYC_SELFIE', 'SACCO_REGISTRATION', 'VEHICLE_LOGBOOK', 'OTHER')),
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT,
    mime_type VARCHAR(100),
    storage_path VARCHAR(500) NOT NULL,   -- MinIO object key
    uploaded_by_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_crew_member ON documents (crew_member_id) WHERE crew_member_id IS NOT NULL;
CREATE INDEX idx_documents_sacco ON documents (sacco_id) WHERE sacco_id IS NOT NULL;
CREATE INDEX idx_documents_vehicle ON documents (vehicle_id) WHERE vehicle_id IS NOT NULL;

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    action VARCHAR(20) NOT NULL,         -- CREATE, UPDATE, DELETE
    resource VARCHAR(50) NOT NULL,
    resource_id UUID,
    old_value JSONB,
    new_value JSONB,
    ip_address VARCHAR(50),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_created ON audit_logs (user_id, created_at);
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource, resource_id);
CREATE INDEX idx_audit_logs_created ON audit_logs (created_at);
