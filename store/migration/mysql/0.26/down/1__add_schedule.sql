-- ===== Down Migration for 0.26 =====
-- Rollback schedule tables for MySQL

-- Drop trigger
DROP TRIGGER IF EXISTS trigger_schedule_updated_ts ON schedule;
DROP FUNCTION IF EXISTS update_schedule_updated_ts;

-- Drop indexes
DROP INDEX IF EXISTS idx_schedule_uid ON schedule;
DROP INDEX IF EXISTS idx_schedule_start_ts ON schedule;
DROP INDEX IF EXISTS idx_schedule_creator_status ON schedule;
DROP INDEX IF EXISTS idx_schedule_creator_start ON schedule;

-- Drop tables
DROP TABLE IF EXISTS schedule_reminder;
DROP TABLE IF EXISTS schedule;
