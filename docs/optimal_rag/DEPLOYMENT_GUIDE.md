# RAG 优化部署指南

> **版本**：v1.0
> **日期**：2025-01-21
> **环境**：生产环境
> **适用范围**：Phase 1 优化功能

---

## 📋 部署前检查清单

### 1. 环境要求

- ✅ Go 1.25+
- ✅ PostgreSQL 14+ (支持 pgvector)
- ✅ Node.js 18+
- ✅ 至少 2C4G 资源

### 2. 依赖服务

- ✅ PostgreSQL (已安装 pgvector 扩展)
- ✅ AI 服务 API Key (SiliconFlow/DeepSeek/OpenAI 等)

### 3. 配置文件

- ✅ `.env` 文件已配置
- ✅ 数据库连接字符串正确
- ✅ AI 功能已启用

---

## 🚀 部署步骤

### 步骤 1：备份数据库

```bash
# 备份现有数据库
pg_dump -h localhost -U memos -d memos > backup_$(date +%Y%m%d).sql

# 或使用 Docker
docker exec memos-db pg_dump -U memos memos > backup.sql
```

### 步骤 2：应用数据库迁移

```bash
# 方式 1：通过 psql 应用
psql -h localhost -p 25432 -U memos -d memos \
  -f store/migration/postgres/0.31/1__add_finops_monitoring.sql

# 方式 2：通过应用自动迁移（推荐）
# 重启服务时自动应用
make restart
```

**验证迁移**：
```sql
-- 检查表是否创建成功
\d query_cost_log

-- 验证索引
SELECT indexname FROM pg_indexes
WHERE tablename = 'query_cost_log';
```

预期输出：
```
indexname
------------------------------------
idx_cost_log_user_time
idx_cost_log_strategy
idx_cost_log_cost
```

### 步骤 3：更新环境变量

```bash
# .env 文件添加以下配置
MEMOS_AI_ENABLED=true
MEMOS_AI_EMBEDDING_PROVIDER=siliconflow
MEMOS_AI_EMBEDDING_MODEL=BAAI/bge-m3
MEMOS_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3
MEMOS_AI_LLM_PROVIDER=deepseek
MEMOS_AI_LLM_MODEL=deepseek-chat
MEMOS_AI_DEEPSEEK_API_KEY=your_api_key_here
```

### 步骤 4：构建新版本

```bash
# 方式 1：使用 Make
make build-all

# 方式 2：手动构建
go build -o bin/memos ./cmd/memos
cd web && npm run build && cd ..
```

### 步骤 5：停止现有服务

```bash
# 停止所有服务
make stop

# 或分别停止
make docker-down  # 停止数据库
```

### 步骤 6：启动新服务

```bash
# 启动所有服务
make start

# 查看日志
make logs

# 检查健康状态
curl http://localhost:25173/healthz
```

**预期输出**：
```json
{"status":"Service ready."}
```

### 步骤 7：验证优化功能

```bash
# 1. 测试 AI Chat 功能
curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"message":"今天有什么安排","history":[]}'

# 2. 检查日志中的新标记
make logs backend | grep -E "\[QueryRouting\]|\[Retrieval\]|\[FinOps\]"
```

**预期日志**：
```
[QueryRouting] Strategy: schedule_bm25_only, Confidence: 0.95
[Retrieval] Completed in 50ms, found 3 results
[ChatWithMemos] Completed - Retrieval: 50ms, LLM: 150ms, Total: 200ms, Strategy: schedule_bm25_only
[FinOps] Successfully recorded cost: $0.008
```

---

## 📊 部署后验证

### 1. 功能验证

#### 测试场景 1：纯日程查询
```bash
curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"message":"今天有什么安排","history":[]}'
```

**预期**：
- ✅ 响应时间 < 100ms
- ✅ 路由策略：`schedule_bm25_only`
- ✅ 返回日程列表

#### 测试场景 2：笔记查询
```bash
curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"message":"搜索关于React的笔记","history":[]}'
```

**预期**：
- ✅ 响应时间 < 200ms
- ✅ 路由策略：`memo_semantic_only` 或 `hybrid_bm25_weighted`
- ✅ 返回相关笔记

#### 测试场景 3：通用问答
```bash
curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"message":"总结一下我的工作计划","history":[]}'
```

**预期**：
- ✅ 响应时间 < 500ms
- ✅ 路由策略：`full_pipeline_with_reranker`
- ✅ 返回总结性回答

### 2. 性能验证

#### 延迟测试
```bash
# 使用 wrk 进行压力测试
wrk -t4 -c100 -d30s --latency \
  -H "Content-Type: application/json" \
  -s/post_chat.lua \
  http://localhost:28081/api/v1/ai/chat
```

**目标指标**：
- P50 延迟 < 200ms
- P95 延迟 < 500ms
- QPS > 100

#### 成本验证
```sql
-- 查询最近 1 小时的成本
SELECT
    strategy,
    COUNT(*) as query_count,
    AVG(total_cost) as avg_cost,
    AVG(latency_ms) as avg_latency
FROM query_cost_log
WHERE timestamp > NOW() - INTERVAL '1 hour'
GROUP BY strategy
ORDER BY avg_cost DESC;
```

### 3. 数据验证

#### 检查成本记录
```sql
-- 确认成本记录正常写入
SELECT COUNT(*) FROM query_cost_log;
SELECT MAX(timestamp) as latest_record FROM query_cost_log;
```

#### 检查策略分布
```sql
-- 查看策略使用分布
SELECT
    strategy,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
FROM query_cost_log
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY strategy
ORDER BY count DESC;
```

**预期分布**：
- `schedule_bm25_only`: ~35%
- `memo_semantic_only`: ~30%
- `hybrid_bm25_weighted`: ~15%
- `hybrid_with_time_filter`: ~15%
- `hybrid_standard`: ~5%
- `full_pipeline`: ~<1%

---

## 🔧 故障排查

### 问题 1：迁移失败

**症状**：
```
ERROR: extension "vector" does not exist
```

**解决方案**：
```bash
# 连接到数据库
psql -h localhost -p 25432 -U memos -d memos

# 启用 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;

# 退出
\q
```

### 问题 2：成本记录未写入

**症状**：
```sql
SELECT COUNT(*) FROM query_cost_log;
-- 返回 0
```

**解决方案**：
```bash
# 检查 AI 功能是否启用
grep "MEMOS_AI_ENABLED" .env

# 检查日志
make logs backend | grep FinOps

# 检查 CostMonitor 是否初始化
make logs backend | grep "CostMonitor"
```

### 问题 3：路由策略不符合预期

**症状**：所有查询都使用 `hybrid_standard`

**解决方案**：
```bash
# 检查 QueryRouter 是否初始化
make logs backend | grep "QueryRouter"

# 检查路由逻辑
# 可以添加调试日志
```

### 问题 4：性能未改善

**症状**：延迟仍然很高

**解决方案**：
```sql
-- 1. 检查策略分布
SELECT strategy, AVG(latency_ms) FROM query_cost_log GROUP BY strategy;

-- 2. 检查是否有大量使用 full_pipeline
-- 3. 检查数据库索引
-- 4. 检查 AI 服务 API 限速
```

---

## 🔄 回滚方案

如果部署后出现严重问题，可以快速回滚：

### 方案 1：数据库回滚

```bash
# 回滚数据库迁移
psql -h localhost -p 25432 -U memos -d memos \
  -f store/migration/postgres/0.31/down/1__add_finops_monitoring.sql
```

### 方案 2：代码回滚

```bash
# 切换到旧版本
git checkout <previous-commit>

# 重新构建
make build-all

# 重启服务
make restart
```

### 方案 3：配置回滚

```bash
# 禁用新功能
# 在 .env 中添加
MEMOS_AI_ENABLED=false

# 重启服务
make restart
```

---

## 📈 监控设置

### 1. Prometheus 指标

添加以下指标到 Prometheus 配置：

```yaml
scrape_configs:
  - job_name: 'memos'
    static_configs:
      - targets: ['localhost:28081']
    metrics_path: /metrics
```

### 2. Grafana 仪表板

导入仪表板 JSON（见 `docs/grafana/rag-optimization-dashboard.json`）：

**面板包含**：
- 成本趋势图
- 策略分布饼图
- 延迟热力图
- QPS 时间序列

### 3. 告警配置

推荐告警规则：

```yaml
alerts:
  - alert: HighCostPerQuery
    expr: memos_cost_per_query_avg > 0.10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "平均查询成本过高"

  - alert: HighLatency
    expr: memos_query_latency_p95 > 500
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "P95 延迟过高"

  - alert: ImbalancedStrategyUsage
    expr: memos_full_pipeline_usage_rate > 0.20
    for: 10m
    labels:
      severity: info
    annotations:
      summary: "完整流程使用率过高，建议优化"
```

---

## 🎯 优化验证

### 对比测试

部署前后对比测试：

| 指标 | 部署前 | 部署后 | 目标 |
|------|--------|--------|------|
| 平均延迟 | 800ms | - | < 350ms |
| P95 延迟 | 1500ms | - | < 700ms |
| 每查询成本 | $0.175 | - | < $0.10 |
| 月成本 | $52.5K | - | < $32K |

### 用户反馈

收集用户反馈：

```bash
# 1. 添加满意度反馈功能
# 在 ChatWithMemosResponse 中添加满意度评分

# 2. 发送反馈请求
curl -X POST http://localhost:28081/api/v1/ai/feedback \
  -H "Content-Type: application/json" \
  -d '{"query_id":"xxx","satisfaction":0.9}'
```

---

## 📝 部署检查表

### 部署前

- [ ] 数据库备份完成
- [ ] 迁移脚本测试通过
- [ ] 环境变量配置正确
- [ ] AI 服务 API Key 有效
- [ ] 新版本编译成功
- [ ] 回滚方案准备就绪

### 部署中

- [ ] 数据库迁移成功
- [ ] 服务启动成功
- [ ] 健康检查通过
- [ ] 日志正常输出

### 部署后

- [ ] 功能测试通过
- [ ] 性能指标达标
- [ ] 成本记录正常
- [ ] 监控告警配置
- [ ] 用户反馈收集

---

## 🚀 快速部署脚本

创建 `deploy-optimization.sh` 脚本：

```bash
#!/bin/bash
set -e

echo "========================================="
echo "RAG 优化部署脚本"
echo "========================================="

# 1. 备份数据库
echo "步骤 1: 备份数据库..."
pg_dump -h localhost -p 25432 -U memos -d memos > backup_$(date +%Y%m%d_%H%M%S).sql
echo "✅ 数据库备份完成"

# 2. 应用迁移
echo "步骤 2: 应用数据库迁移..."
psql -h localhost -p 25432 -U memos -d memos \
  -f store/migration/postgres/0.31/1__add_finops_monitoring.sql
echo "✅ 数据库迁移完成"

# 3. 构建新版本
echo "步骤 3: 构建新版本..."
make build-all
echo "✅ 构建完成"

# 4. 重启服务
echo "步骤 4: 重启服务..."
make restart
sleep 5
echo "✅ 服务重启完成"

# 5. 验证部署
echo "步骤 5: 验证部署..."
curl -s http://localhost:25173/healthz > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ 部署成功！"
else
    echo "❌ 部署失败，请检查日志"
    make logs
    exit 1
fi

echo "========================================="
echo "部署完成！"
echo "查看日志: make logs"
echo "========================================="
```

使用方式：
```bash
chmod +x deploy-optimization.sh
./deploy-optimization.sh
```

---

## 📞 支持与联系

### 问题报告

如遇到部署问题，请提供以下信息：

1. 环境信息
```bash
go version
psql --version
uname -a
```

2. 错误日志
```bash
make logs backend > error.log 2>&1
```

3. 配置信息
```bash
# 移除敏感信息后提供
env | grep MEMOS
```

### 参考文档

- **优化总结**：`docs/OPTIMIZATION_SUMMARY.md`
- **测试指南**：`docs/TESTING_GUIDE.md`
- **API 文档**：`docs/FINOPS_API.md`
- **完成报告**：`docs/PHASE1_COMPLETION_REPORT.md`

---

**最后更新**：2025-01-21
**文档版本**：v1.0
**维护者**：Memos 团队
