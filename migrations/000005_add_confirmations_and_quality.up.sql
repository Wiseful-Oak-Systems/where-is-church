-- 000005_add_confirmations_and_quality.up.sql
-- Add church confirmation system and data quality states

ALTER TABLE churches ADD COLUMN IF NOT EXISTS data_quality VARCHAR(50) NOT NULL DEFAULT 'unverified';
ALTER TABLE churches ADD COLUMN IF NOT EXISTS confirmations INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_churches_data_quality ON churches (data_quality);

CREATE TABLE IF NOT EXISTS church_confirmations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    church_id BIGINT NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, church_id)
);

CREATE INDEX idx_church_confirmations_church ON church_confirmations (church_id);
