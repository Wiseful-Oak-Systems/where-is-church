-- 000005_add_confirmations_and_quality.down.sql
DROP TABLE IF EXISTS church_confirmations;
ALTER TABLE churches DROP COLUMN IF EXISTS data_quality;
ALTER TABLE churches DROP COLUMN IF EXISTS confirmations;
