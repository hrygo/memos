-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Memo embedding table
CREATE TABLE memo_embedding (
    id SERIAL PRIMARY KEY,
    memo_id INTEGER NOT NULL,
    embedding vector(1024) NOT NULL,
    model VARCHAR(100) NOT NULL DEFAULT 'BAAI/bge-m3',
    created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,

    -- Foreign key constraint
    CONSTRAINT fk_memo_embedding_memo
        FOREIGN KEY (memo_id)
        REFERENCES memo(id)
        ON DELETE CASCADE,

    -- One embedding per memo per model
    CONSTRAINT uq_memo_embedding_memo_model
        UNIQUE (memo_id, model)
);

-- HNSW vector index (cosine similarity)
-- Optimized for 2C2G: m=8 (connections), ef_construction=32 (build candidates)
-- Lower parameters reduce memory/CPU usage, trading some accuracy
CREATE INDEX idx_memo_embedding_hnsw
ON memo_embedding USING hnsw (embedding vector_cosine_ops)
WITH (m = 8, ef_construction = 32);

-- memo_id index (accelerate JOIN)
CREATE INDEX idx_memo_embedding_memo_id
ON memo_embedding (memo_id);

-- Updated timestamp trigger
CREATE OR REPLACE FUNCTION update_memo_embedding_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_memo_embedding_updated_ts
    BEFORE UPDATE ON memo_embedding
    FOR EACH ROW
    EXECUTE FUNCTION update_memo_embedding_updated_ts();
