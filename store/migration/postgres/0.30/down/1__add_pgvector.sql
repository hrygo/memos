-- ===== Down Migration for 0.30 =====
-- Rollback pgvector extension and indexes

-- Drop vector similarity search function (if created)
DROP FUNCTION IF EXISTS memo_similarity_search CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS memo_embedding_idx;
DROP INDEX IF EXISTS memo_embedding_model_idx;

-- Note: We do NOT drop the vector extension here as it may be used by other tables
-- To completely remove pgvector, manually run: DROP EXTENSION IF EXISTS vector CASCADE;

-- Log completion
DO $$
BEGIN
	RAISE NOTICE 'Down migration 0.30 completed: vector indexes dropped';
	RAISE NOTICE 'To drop vector extension, manually run: DROP EXTENSION IF EXISTS vector CASCADE;';
END $$;
