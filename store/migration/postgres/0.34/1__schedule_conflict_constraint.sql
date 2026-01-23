-- Add EXCLUDE constraint to prevent concurrent schedule conflicts
-- This implements atomic conflict detection at the database level

-- Enable btree_gist extension if not already enabled (required for EXCLUDE with =)
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Add generated tsrange column for time range overlap detection
ALTER TABLE schedule
ADD COLUMN IF NOT EXISTS time_range tsrange
GENERATED ALWAYS AS (
    tsrange(
        start_ts,
        COALESCE(end_ts, start_ts + 3600)
    )
) STORED;

-- Create gist index for overlap detection
CREATE INDEX IF NOT EXISTS idx_schedule_time_range
ON schedule USING gist (creator_id WITH =, time_range WITH &&);

-- Add EXCLUDE constraint to prevent overlapping schedules for the same user
-- This ensures atomic conflict detection - no two schedules can overlap
-- even under high concurrency
ALTER TABLE schedule
DROP CONSTRAINT IF EXISTS no_overlapping_schedules;

ALTER TABLE schedule
ADD CONSTRAINT no_overlapping_schedules
EXCLUDE USING gist (
    creator_id WITH =,
    time_range WITH &&
);
