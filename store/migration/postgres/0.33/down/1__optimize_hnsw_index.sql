-- Rollback HNSW index optimization

-- Drop optimized index
DROP INDEX IF EXISTS idx_memo_embedding_hnsw;

-- Recreate with old parameters (2C2G optimized)
CREATE INDEX idx_memo_embedding_hnsw
ON memo_embedding USING hnsw (embedding vector_cosine_ops)
WITH (m = 8, ef_construction = 32);
