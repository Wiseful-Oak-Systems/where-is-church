-- 000001_init.down.sql
-- Rollback initial schema

DROP TABLE IF EXISTS favorites;
DROP TABLE IF EXISTS suggestions;
DROP TABLE IF EXISTS check_ins;
DROP TABLE IF EXISTS mass_schedules;
DROP TABLE IF EXISTS churches;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS postgis;
