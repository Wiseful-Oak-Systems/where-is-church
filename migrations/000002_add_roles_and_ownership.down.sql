-- 000002_add_roles_and_ownership.down.sql
-- Rollback: remove church ownership claims

DROP INDEX IF EXISTS idx_church_ownerships_active_claim;
DROP TABLE IF EXISTS church_ownerships;

-- Revert any users with new roles back to 'user'
UPDATE users SET role = 'user' WHERE role IN ('church_owner', 'community_manager');
