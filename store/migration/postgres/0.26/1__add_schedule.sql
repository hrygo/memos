-- Schedule table for AI-powered schedule assistant
CREATE TABLE schedule (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,

  -- Standard fields
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',

  -- Schedule core fields
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  location TEXT DEFAULT '',

  -- Time fields (UTC timestamps)
  start_ts BIGINT NOT NULL,
  end_ts BIGINT,
  all_day BOOLEAN NOT NULL DEFAULT FALSE,

  -- Timezone
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',

  -- Recurrence rule (JSON format)
  recurrence_rule TEXT,
  recurrence_end_ts BIGINT,

  -- Reminders (JSON array format)
  reminders TEXT NOT NULL DEFAULT '[]',

  -- Extension
  payload JSONB NOT NULL DEFAULT '{}',

  -- Foreign key constraint
  CONSTRAINT fk_schedule_creator
    FOREIGN KEY (creator_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,

  -- Check constraints
  CONSTRAINT chk_schedule_time_range
    CHECK (end_ts IS NULL OR end_ts >= start_ts),
  CONSTRAINT chk_schedule_reminders_json
    CHECK (reminders::jsonb IS NOT NULL)
);

-- Performance indexes
CREATE INDEX idx_schedule_creator_start ON schedule(creator_id, start_ts);
CREATE INDEX idx_schedule_creator_status ON schedule(creator_id, row_status);
CREATE INDEX idx_schedule_start_ts ON schedule(start_ts);
CREATE INDEX idx_schedule_uid ON schedule(uid);

-- Updated timestamp trigger
CREATE OR REPLACE FUNCTION update_schedule_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_schedule_updated_ts
  BEFORE UPDATE ON schedule
  FOR EACH ROW
  EXECUTE FUNCTION update_schedule_updated_ts();
