-- Remove EXCLUDE constraint and time_range column

ALTER TABLE schedule
DROP CONSTRAINT IF EXISTS no_overlapping_schedules;

DROP INDEX IF EXISTS idx_schedule_time_range;

ALTER TABLE schedule
DROP COLUMN IF EXISTS time_range;
