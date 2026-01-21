-- ===== Down Migration for 0.26 =====
-- Rollback schedule tables

-- Drop update timestamp trigger
DROP TRIGGER IF EXISTS trigger_schedule_updated_ts ON schedule;
DROP FUNCTION IF EXISTS update_schedule_updated_ts();

-- Drop indexes
DROP INDEX IF EXISTS idx_schedule_uid;
DROP INDEX IF EXISTS idx_schedule_start_ts;
DROP INDEX IF EXISTS idx_schedule_creator_status;
DROP INDEX IF EXISTS idx_schedule_creator_start;
DROP INDEX IF EXISTS idx_schedule_updated_ts;

-- Drop tables (cascade will automatically delete related records)
DROP TABLE IF EXISTS schedule_reminder;
DROP TABLE IF EXISTS schedule;

-- Log completion
DO $$
BEGIN
	RAISE NOTICE 'Down migration 0.26 completed: schedule tables dropped';
END $$;
