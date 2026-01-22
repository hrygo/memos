# RAG 优化测试指南

> **版本**：v1.0
> **日期**：2025-01-21
> **范围**：Phase 1 优化功能测试

---

## 📋 测试清单

### 1. 环境准备

#### 1.1 数据库迁移
```bash
# 确保 PostgreSQL 正在运行
make docker-up

# 应用迁移（版本 0.31）
psql -h localhost -p 25432 -U memos -d memos \
  -f store/migration/postgres/0.31/1__add_finops_monitoring.sql

# 验证表创建
psql -h localhost -p 25432 -U memos -d memos \
  -c "\d query_cost_log"
```

预期输出：
```
Column          | Type                    | Nullable
----------------+-------------------------+----------
id              | bigint                  | not null
timestamp       | timestamp               | not null
user_id         | integer                 | not null
query           | text                    | not null
strategy        | character varying(50)   | not null
vector_cost     | numeric(10,6)           | not null
reranker_cost   | numeric(10,6)           | not null
llm_cost        | numeric(10,6)           | not null
total_cost      | numeric(10,6)           | not null
latency_ms      | integer                 | not null
result_count    | integer                 | not null
user_satisfied  | numeric(3,2)            |
```

#### 1.2 环境变量配置
```bash
# .env 文件
MEMOS_DRIVER=postgres
MEMOS_DSN=postgres://memos:memos@localhost:25432/memos?sslmode=disable

# AI 功能
MEMOS_AI_ENABLED=true
MEMOS_AI_EMBEDDING_PROVIDER=siliconflow
MEMOS_AI_EMBEDDING_MODEL=BAAI/bge-m3
MEMOS_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3
MEMOS_AI_LLM_PROVIDER=deepseek
MEMOS_AI_LLM_MODEL=deepseek-chat
MEMOS_AI_DEEPSEEK_API_KEY=your_api_key
```

---

### 2. 单元测试

#### 2.1 Query Routing 测试

```bash
# 运行 Query Router 测试
go test ./server/queryengine/... -v

# 运行特定测试
go test ./server/queryengine/... -v -run TestQueryRouter_Route

# 性能基准测试
go test ./server/queryengine/... -bench=. -benchmem
```

**预期结果**：
- ✅ 所有测试通过
- ✅ 平均路由时间 < 10ms
- ✅ 时间范围检测准确率 > 95%

**关键测试场景**：
1. **纯日程查询**："今天有什么安排" → `schedule_bm25_only`
2. **混合查询**："今天下午关于AI的会议" → `hybrid_with_time_filter`
3. **笔记查询**："搜索关于Python的笔记" → `memo_semantic_only`
4. **通用问答**："总结我的工作计划" → `full_pipeline_with_reranker`

#### 2.2 Adaptive Retrieval 测试

```bash
# 运行 Adaptive Retrieval 测试
go test ./server/retrieval/... -v

# 运行特定测试
go test ./server/retrieval/... -v -run TestAdaptiveRetriever_EvaluateQuality
```

**预期结果**：
- ✅ 质量评估逻辑正确
- ✅ Selective Reranker 规则生效
- ✅ 结果合并、去重、排序正确

**关键测试场景**：
1. **高质量结果**：前2名分数差距 >0.20 → `HighQuality`
2. **中等质量结果**：前2名分数差距 0.15-0.20 → `MediumQuality`
3. **低质量结果**：第1名分数 <0.70 → `LowQuality`

#### 2.3 Cost Monitor 测试

```bash
# 运行 Cost Monitor 测试
go test ./server/finops/... -v

# 运行特定测试
go test ./server/finops/... -v -run TestCostMonitor_CalculateTotalCost
```

**预期结果**：
- ✅ 成本计算正确
- ✅ 成本估算合理
- ✅ 周期时间计算准确

**关键测试场景**：
1. **成本计算**：`0.001 + 0.005 + 0.01 = 0.016`
2. **Embedding 成本**：1000 字符 ≈ $0.00003
3. **Reranker 成本**：10 个文档 × 100 字符 ≈ $0.0003
4. **LLM 成本**：2000 输入 + 1000 输出 ≈ $0.0005

---

### 3. 集成测试

#### 3.1 启动服务

```bash
# 启动所有服务
make start

# 查看日志
make logs

# 检查服务状态
curl http://localhost:25173/healthz
```

预期输出：
```json
{"status":"Service ready."}
```

#### 3.2 AI Chat 功能测试

使用前端界面或 API 测试：

```bash
# API 端点
# POST /api/v1/ai/chat
```

**测试场景**：

##### 场景 1：纯日程查询
```json
{
  "message": "今天有什么安排"
}
```

**预期**：
- ✅ 路由策略：`schedule_bm25_only`
- ✅ 响应延迟：< 100ms
- ✅ 返回日程列表（如果有的话）
- ✅ 成本记录：$0.006

**日志验证**：
```
[QueryRouting] Strategy: schedule_bm25_only, Confidence: 0.95
[Retrieval] Completed in 50ms, found 3 results
[ChatWithMemos] Completed - Retrieval: 50ms, LLM: 150ms, Total: 200ms
```

##### 场景 2：笔记查询
```json
{
  "message": "搜索关于React的笔记"
}
```

**预期**：
- ✅ 路由策略：`memo_semantic_only`
- ✅ 响应延迟：< 200ms
- ✅ 返回相关笔记
- ✅ 成本记录：$0.005

**日志验证**：
```
[QueryRouting] Strategy: memo_semantic_only, Confidence: 0.90
[Retrieval] Completed in 150ms, found 5 results
```

##### 场景 3：混合查询
```json
{
  "message": "今天下午关于AI项目的会议"
}
```

**预期**：
- ✅ 路由策略：`hybrid_with_time_filter`
- ✅ 返回日程和笔记
- ✅ 成本记录：$0.010

##### 场景 4：通用问答
```json
{
  "message": "总结一下我的工作计划"
}
```

**预期**：
- ✅ 路由策略：`full_pipeline_with_reranker`
- ✅ 使用 Reranker
- ✅ 返回总结性回答
- ✅ 成本记录：$0.060

---

### 4. 性能测试

#### 4.1 延迟测试

```bash
# 使用 ab 或 wrk 进行压测
# ab -n 100 -c 10 http://localhost:25173/healthz

# 或使用 curl 测试单次请求
time curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"今天有什么安排"}'
```

**预期指标**：

| 场景 | P50 (平均) | P95 | P99 |
|------|-----------|-----|-----|
| schedule_bm25_only | < 50ms | < 80ms | < 100ms |
| memo_semantic_only | < 150ms | < 200ms | < 250ms |
| hybrid_standard | < 200ms | < 300ms | < 400ms |
| full_pipeline | < 500ms | < 700ms | < 900ms |

#### 4.2 成本验证

```sql
-- 查询成本记录
SELECT
    strategy,
    COUNT(*) as query_count,
    AVG(total_cost) as avg_cost,
    AVG(latency_ms) as avg_latency,
    AVG(result_count) as avg_results
FROM query_cost_log
WHERE timestamp > NOW() - INTERVAL '1 hour'
GROUP BY strategy
ORDER BY strategy;
```

**预期结果**：

| Strategy | Avg Cost | Avg Latency |
|----------|----------|-------------|
| schedule_bm25_only | $0.006 | < 100ms |
| memo_semantic_only | $0.005 | < 200ms |
| hybrid_standard | $0.010 | < 250ms |
| full_pipeline | $0.060 | < 600ms |

---

### 5. FinOps 监控验证

#### 5.1 查看成本报告

```sql
-- 每日成本报告
SELECT
    DATE(timestamp) as date,
    COUNT(*) as total_queries,
    SUM(total_cost) as total_cost,
    AVG(latency_ms) as avg_latency
FROM query_cost_log
WHERE timestamp > NOW() - INTERVAL '7 days'
GROUP BY DATE(timestamp)
ORDER BY date DESC;
```

#### 5.2 策略分布分析

```sql
-- 策略使用分布
SELECT
    strategy,
    COUNT(*) as usage_count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage,
    AVG(total_cost) as avg_cost,
    AVG(latency_ms) as avg_latency
FROM query_cost_log
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY strategy
ORDER BY usage_count DESC;
```

**预期分布**：
- `schedule_bm25_only`: ~35%
- `memo_semantic_only`: ~30%
- `hybrid_standard`: ~30%
- `full_pipeline`: ~5%

---

### 6. 回归测试

#### 6.1 兼容性测试

确保新功能不影响旧功能：

```bash
# 测试原有的 SemanticSearch API
curl -X POST http://localhost:28081/api/v1/ai/search \
  -H "Content-Type: application/json" \
  -d '{"query":"测试","limit":10}'

# 测试 GetRelatedMemos API
curl http://localhost:28081/api/v1/memos/xxx/related
```

#### 6.2 错误处理测试

```bash
# 测试 AI 功能禁用时的行为
# 设置 MEMOS_AI_ENABLED=false
make restart

# 预期：返回 "AI features are disabled"
```

---

### 7. 压力测试

#### 7.1 并发测试

```bash
# 使用 wrk 进行并发测试
wrk -t4 -c100 -d30s --latency \
  -H "Content-Type: application/json" \
  -s/post_chat.lua \
  http://localhost:28081/api/v1/ai/chat
```

**post_chat.lua 内容**：
```lua
wrk.method = "POST"
wrk.body   = '{"message":"今天有什么安排"}'
wrk.headers["Content-Type"] = "application/json"
```

**预期指标**：
- QPS > 100
- P95 延迟 < 500ms
- 错误率 < 1%

#### 7.2 成本压力测试

```sql
-- 查询高成本查询
SELECT
    query,
    strategy,
    total_cost,
    latency_ms,
    timestamp
FROM query_cost_log
WHERE total_cost > 0.05
ORDER BY total_cost DESC
LIMIT 10;
```

---

## 📊 测试报告模板

### 测试执行摘要

- **测试日期**：YYYY-MM-DD
- **测试人员**：[姓名]
- **环境**：开发/测试/生产
- **版本**：v1.0

### 测试结果

| 测试项 | 通过 | 失败 | 阻塞 | 通过率 |
|--------|------|------|------|--------|
| 单元测试 | 45 | 0 | 0 | 100% |
| 集成测试 | 12 | 0 | 0 | 100% |
| 性能测试 | 8 | 1 | 0 | 87.5% |
| **总计** | **65** | **1** | **0** | **98.5%** |

### 性能指标

| 指标 | 目标值 | 实际值 | 状态 |
|------|--------|--------|------|
| 平均延迟 | < 200ms | 180ms | ✅ |
| P95 延迟 | < 500ms | 420ms | ✅ |
| 每查询成本 | < $0.10 | $0.08 | ✅ |
| QPS | > 100 | 120 | ✅ |

### 问题清单

| ID | 问题描述 | 严重程度 | 状态 | 负责人 |
|----|---------|---------|------|--------|
| 1 | [问题描述] | 高/中/低 | 待修复/已修复 | [姓名] |

---

## ✅ 验收标准

### 功能验收

- [ ] Query Routing 覆盖率 > 95%
- [ ] FinOps 监控正常记录
- [ ] Selective Reranker 正常工作
- [ ] 无回归问题

### 性能验收

- [ ] 平均延迟 < 350ms
- [ ] P95 延迟 < 700ms
- [ ] 成本降低 > 30%

### 准确度验收

- [ ] 用户满意度 > 4.0/5
- [ ] NDCG@10 持平或略有提升

---

## 🐛 已知问题

### 问题 1：时间范围检测在组合时间词时可能不精确

**描述**："今天下午"只匹配到"下午"而不是"今天下午"

**影响**：低

**解决方案**：优化时间关键词匹配优先级

### 问题 2：FinOps 成本估算不够精确

**描述**：使用固定的 Token 估算，可能与实际有偏差

**影响**：中

**解决方案**：从 AI 服务提供商获取实际 Token 使用量

---

## 📝 测试注意事项

1. **测试数据**：使用真实的用户数据场景，包含笔记和日程
2. **环境隔离**：测试环境与生产环境分离
3. **数据清理**：测试后清理 `query_cost_log` 表
4. **性能监控**：测试期间监控系统资源使用
5. **日志收集**：保存完整的测试日志用于分析

---

## 🚀 自动化测试

### CI/CD 集成

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      - name: Run tests
        run: |
          go test ./server/queryengine/... -v
          go test ./server/retrieval/... -v
          go test ./server/finops/... -v
```

---

**最后更新**：2025-01-21
**文档版本**：v1.0
