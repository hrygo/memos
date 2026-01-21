-- Schedule table for AI-powered schedule assistant (SQLite)
CREATE TABLE schedule (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,

  -- Standard fields
  created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',

  -- Schedule core fields
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  location TEXT DEFAULT '',

  -- Time fields (UTC timestamps)
  start_ts INTEGER NOT NULL,
  end_ts INTEGER,
  all_day INTEGER NOT NULL DEFAULT 0,

  -- Timezone
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',

  -- Recurrence rule (JSON format as TEXT)
  recurrence_rule TEXT,
  recurrence_end_ts INTEGER,

  -- Reminders (JSON array format as TEXT)
  reminders TEXT NOT NULL DEFAULT '[]',

  -- Extension (JSON format as TEXT)
  payload TEXT NOT NULL DEFAULT '{}',

  -- Foreign key constraint
  FOREIGN KEY (creator_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,

  -- Check constraints
  CHECK (end_ts IS NULL OR end_ts >= start_ts)
);

-- Performance indexes
CREATE INDEX idx_schedule_creator_start ON schedule(creator_id, start_ts);
CREATE INDEX idx_schedule_creator_status ON schedule(creator_id, row_status);
CREATE INDEX idx_schedule_start_ts ON schedule(start_ts);
CREATE INDEX idx_schedule_uid ON schedule(uid);

-- Updated timestamp trigger
CREATE TRIGGER trigger_schedule_updated_ts
  AFTER UPDATE ON schedule
  FOR EACH ROW
  WHEN NEW.updated_ts <= OLD.updated_ts
BEGIN
  UPDATE schedule SET updated_ts = strftime('%s', 'now') WHERE id = NEW.id;
END;
