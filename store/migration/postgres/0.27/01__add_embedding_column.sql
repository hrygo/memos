-- Add embedding column to memos table for semantic search
-- vector(1024) is the dimension of BAAI/bge-m3 embedding model

ALTER TABLE memo ADD COLUMN IF NOT EXISTS embedding vector(1024);

-- Create HNSW index for fast vector similarity search
-- Using cosine distance (vector_cosine_ops) which is suitable for semantic search
-- m=16: number of bi-directional links for each node
-- ef_construction=64: size of dynamic candidate list for construction

CREATE INDEX IF NOT EXISTS memo_embedding_idx
ON memo USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Add comment for documentation
COMMENT ON COLUMN memo.embedding IS 'Vector embedding for semantic search (1024 dimensions, BAAI/bge-m3 model)';
