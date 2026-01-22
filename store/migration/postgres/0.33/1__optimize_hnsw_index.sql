-- Optimize HNSW index parameters for better performance
-- Phase 2.4: HNSW Index Optimization

-- Drop old index with suboptimal parameters
DROP INDEX IF EXISTS idx_memo_embedding_hnsw;

-- Recreate with optimized parameters for 2C4G configuration
-- m = 16: number of bi-directional links (higher = better recall, more memory)
-- ef_construction = 64: size of dynamic candidate list during construction
-- ef_search is set at runtime via: SET hnsw.ef_search = 100;
CREATE INDEX idx_memo_embedding_hnsw
ON memo_embedding USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Add comment for documentation
COMMENT ON INDEX idx_memo_embedding_hnsw IS
'HNSW index for vector similarity search with m=16, ef_construction=64. Use SET hnsw.ef_search = 100 for queries.';
