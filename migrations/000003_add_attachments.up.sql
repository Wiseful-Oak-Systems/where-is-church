-- 000003_add_attachments.up.sql
-- Add file attachment support for suggestions, claims, and churches

CREATE TABLE IF NOT EXISTS attachments (
    id BIGSERIAL PRIMARY KEY,
    uploaded_by_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attachable_type VARCHAR(50) NOT NULL,
    attachable_id BIGINT NOT NULL,
    filename VARCHAR(255) NOT NULL,
    storage_key VARCHAR(500) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_attachments_uploaded_by ON attachments (uploaded_by_id);
CREATE INDEX idx_attachments_attachable ON attachments (attachable_type, attachable_id);
