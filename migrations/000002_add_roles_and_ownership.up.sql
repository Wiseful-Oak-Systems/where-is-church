-- 000002_add_roles_and_ownership.up.sql
-- Add community_manager and church_owner roles, church ownership claims

-- Update role check constraint (if exists) or just allow new values
-- PostgreSQL enum approach: roles are stored as VARCHAR, no constraint to change

-- Add church ownership claims table
CREATE TABLE IF NOT EXISTS church_ownerships (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    church_id BIGINT NOT NULL REFERENCES churches(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    evidence TEXT,
    reviewed_by_id BIGINT REFERENCES users(id),
    review_note TEXT,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX idx_church_ownerships_user_id ON church_ownerships (user_id);
CREATE INDEX idx_church_ownerships_church_id ON church_ownerships (church_id);
CREATE INDEX idx_church_ownerships_status ON church_ownerships (status);

-- Add unique constraint to prevent duplicate active claims
CREATE UNIQUE INDEX idx_church_ownerships_active_claim
    ON church_ownerships (user_id, church_id)
    WHERE status IN ('pending', 'approved');
