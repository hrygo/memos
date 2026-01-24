-- AI conversation and message tables for PostgreSQL

-- ai_conversation
CREATE TABLE ai_conversation (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  parrot_id TEXT NOT NULL DEFAULT '',
  pinned BOOLEAN NOT NULL DEFAULT FALSE,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',
  CONSTRAINT fk_ai_conversation_creator
    FOREIGN KEY (creator_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_ai_conversation_row_status
    CHECK (row_status IN ('NORMAL', 'ARCHIVED'))
);

CREATE INDEX idx_ai_conversation_creator ON ai_conversation(creator_id);
CREATE INDEX idx_ai_conversation_updated ON ai_conversation(updated_ts DESC);

-- ai_message
CREATE TABLE ai_message (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  conversation_id INTEGER NOT NULL,
  type TEXT NOT NULL DEFAULT 'MESSAGE',
  role TEXT NOT NULL DEFAULT 'USER',
  content TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_ai_message_conversation
    FOREIGN KEY (conversation_id)
    REFERENCES ai_conversation(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_ai_message_type
    CHECK (type IN ('MESSAGE', 'SEPARATOR')),
  CONSTRAINT chk_ai_message_role
    CHECK (role IN ('USER', 'ASSISTANT', 'SYSTEM'))
);

CREATE INDEX idx_ai_message_conversation ON ai_message(conversation_id);
CREATE INDEX idx_ai_message_created ON ai_message(created_ts ASC);

-- Trigger to update updated_ts on ai_conversation
CREATE OR REPLACE FUNCTION update_ai_conversation_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_ai_conversation_updated_ts
  BEFORE UPDATE ON ai_conversation
  FOR EACH ROW
  EXECUTE FUNCTION update_ai_conversation_updated_ts();
