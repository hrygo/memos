-- Drop FinOps monitoring table and indexes

-- Drop indexes
DROP INDEX IF EXISTS idx_cost_log_cost;
DROP INDEX IF EXISTS idx_cost_log_strategy;
DROP INDEX IF EXISTS idx_cost_log_user_time;
DROP INDEX IF EXISTS idx_cost_log_strategy_time;
DROP INDEX IF EXISTS idx_cost_log_user_strategy_time;

-- Drop constraints
ALTER TABLE query_cost_log DROP CONSTRAINT IF EXISTS chk_cost_log_costs;
ALTER TABLE query_cost_log DROP CONSTRAINT IF EXISTS chk_cost_log_metrics;

-- Drop table
DROP TABLE IF EXISTS query_cost_log;
