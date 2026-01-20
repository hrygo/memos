-- ===== Down Migration for 0.26 =====
-- Rollback schedule tables for SQLite
-- SQLite doesn't support DROP TRIGGER IF EXISTS, so we just drop tables

-- Drop tables (SQLite will automatically drop triggers)
DROP TABLE IF EXISTS schedule_reminder;
DROP TABLE IF EXISTS schedule;
