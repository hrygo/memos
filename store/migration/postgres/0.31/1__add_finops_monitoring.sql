-- Query cost log table for FinOps monitoring
CREATE TABLE query_cost_log (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    query TEXT NOT NULL,
    strategy VARCHAR(50) NOT NULL,

    -- Cost breakdown (in USD)
    vector_cost DECIMAL(10, 6) NOT NULL DEFAULT 0,
    reranker_cost DECIMAL(10, 6) NOT NULL DEFAULT 0,
    llm_cost DECIMAL(10, 6) NOT NULL DEFAULT 0,
    total_cost DECIMAL(10, 6) NOT NULL,

    -- Performance metrics
    latency_ms INTEGER NOT NULL,

    -- Result metrics
    result_count INTEGER NOT NULL,

    -- Optional: User satisfaction feedback
    user_satisfied DECIMAL(3, 2) CHECK (user_satisfied IS NULL OR (user_satisfied >= 0 AND user_satisfied <= 1)) -- 0.00-1.00, NULL means not rated
);

-- P0 改进：添加 CHECK 约束确保数据完整性
ALTER TABLE query_cost_log
ADD CONSTRAINT chk_cost_log_costs CHECK (
    vector_cost >= 0 AND
    reranker_cost >= 0 AND
    llm_cost >= 0 AND
    total_cost >= 0 AND
    total_cost = (vector_cost + reranker_cost + llm_cost)
);

ALTER TABLE query_cost_log
ADD CONSTRAINT chk_cost_log_metrics CHECK (
    latency_ms >= 0 AND
    result_count >= 0
);

-- Index for user-time queries (performance)
CREATE INDEX idx_cost_log_user_time
ON query_cost_log (user_id, timestamp DESC);

-- Index for strategy analysis
CREATE INDEX idx_cost_log_strategy
ON query_cost_log (strategy, timestamp DESC);

-- Index for cost monitoring
CREATE INDEX idx_cost_log_cost
ON query_cost_log (total_cost DESC, timestamp DESC);

-- P0 改进：添加复合索引用于常见查询模式
CREATE INDEX idx_cost_log_strategy_time
ON query_cost_log (strategy, timestamp DESC)
WHERE timestamp > NOW() - INTERVAL '90 days'; -- 部分索引，只索引最近 90 天的数据

CREATE INDEX idx_cost_log_user_strategy_time
ON query_cost_log (user_id, strategy, timestamp DESC)
WHERE timestamp > NOW() - INTERVAL '90 days';

-- Comments
COMMENT ON TABLE query_cost_log IS 'FinOps monitoring: tracks AI query costs and performance metrics';
COMMENT ON COLUMN query_cost_log.strategy IS 'Routing strategy used: schedule_bm25_only, memo_semantic_only, hybrid_standard, full_pipeline_with_reranker, etc.';
COMMENT ON COLUMN query_cost_log.user_satisfied IS 'User satisfaction rating (0.0-1.0), collected optionally via feedback';

-- P0 改进：添加数据保留策略说明
-- 建议创建以下函数来定期清理旧数据：
-- CREATE OR REPLACE FUNCTION cleanup_old_cost_logs()
-- RETURNS void AS $$
-- BEGIN
--     DELETE FROM query_cost_log
--     WHERE timestamp < NOW() - INTERVAL '90 days';
-- END;
-- $$ LANGUAGE plpgsql;
--
-- 然后使用 pg_cron 或类似工具定期执行：
-- SELECT cron.schedule('cleanup-cost-logs', '0 2 * * *', 'SELECT cleanup_old_cost_logs()');

