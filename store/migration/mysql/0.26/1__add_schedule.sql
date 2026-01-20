-- Schedule table for AI-powered schedule assistant (MySQL)
--
-- ⚠️ DEPRECATION NOTICE
--
-- This migration file is provided AS-IS and may contain syntax errors.
-- The schedule feature is NOT officially supported on MySQL.
--
-- Please use PostgreSQL or SQLite instead.
-- See: /docs/schedule-assistant-implementation-plan.md
--
-- Known issues:
--   - Line 33: Missing column name in payload field definition
--   - JSON constraints not validated
--   - Limited trigger capabilities
--
CREATE TABLE schedule (
  id INT AUTO_INCREMENT PRIMARY KEY,
  uid VARCHAR(255) NOT NULL UNIQUE,
  creator_id INT NOT NULL,

  -- Standard fields
  created_ts BIGINT NOT NULL DEFAULT (UNIX_TIMESTAMP()),
  updated_ts BIGINT NOT NULL DEFAULT (UNIX_TIMESTAMP()),
  row_status VARCHAR(255) NOT NULL DEFAULT 'NORMAL',

  -- Schedule core fields
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  location TEXT DEFAULT '',

  -- Time fields (UTC timestamps)
  start_ts BIGINT NOT NULL,
  end_ts BIGINT,
  all_day BOOLEAN NOT NULL DEFAULT FALSE,

  -- Timezone
  timezone VARCHAR(255) NOT NULL DEFAULT 'Asia/Shanghai',

  -- Recurrence rule (JSON format as TEXT)
  recurrence_rule TEXT,
  recurrence_end_ts BIGINT,

  -- Reminders (JSON array format as TEXT)
  reminders TEXT NOT NULL DEFAULT '[]',

  -- Extension (JSON format)
  JSON,

  -- Foreign key constraint
  CONSTRAINT fk_schedule_creator
    FOREIGN KEY (creator_id)
    REFERENCES user(id)
    ON DELETE CASCADE,

  -- Check constraints (MySQL 8.0.16+)
  CONSTRAINT chk_schedule_time_range
    CHECK (end_ts IS NULL OR end_ts >= start_ts)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Performance indexes
CREATE INDEX idx_schedule_creator_start ON schedule(creator_id, start_ts);
CREATE INDEX idx_schedule_creator_status ON schedule(creator_id, row_status);
CREATE INDEX idx_schedule_start_ts ON schedule(start_ts);
CREATE INDEX idx_schedule_uid ON schedule(uid);

-- Updated timestamp trigger
DELIMITER $$
CREATE TRIGGER trigger_schedule_updated_ts
  BEFORE UPDATE ON schedule
  FOR EACH ROW
BEGIN
  SET NEW.updated_ts = UNIX_TIMESTAMP();
END$$
DELIMITER ;
