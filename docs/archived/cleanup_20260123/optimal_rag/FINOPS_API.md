# FinOps API 文档

> **版本**：v1.0
> **日期**：2025-01-21
> **功能**：AI 查询成本监控与优化

---

## 📋 API 概述

FinOps（Financial Operations）API 提供了 AI 查询成本监控、分析和优化的完整功能。

### 主要功能

1. **成本追踪**：记录每次 AI 查询的详细成本
2. **策略分析**：分析不同检索策略的成本效益
3. **性能监控**：追踪查询延迟和吞吐量
4. **优化建议**：基于数据提供策略优化建议

---

## 🔧 API 端点

### 1. 获取成本报告

获取指定时间段的成本报告。

**请求**：
```http
GET /api/v1/ai/cost-report?period=daily
```

**参数**：
| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `period` | string | 否 | 时间周期：`daily`, `weekly`, `monthly`，默认 `daily` |

**响应**：
```json
{
  "period": "daily",
  "total_cost_usd": 12.50,
  "query_count": 150,
  "avg_latency_ms": 180,
  "by_strategy": {
    "schedule_bm25_only": {
      "strategy": "schedule_bm25_only",
      "query_count": 50,
      "cost_usd": 3.00,
      "avg_latency_ms": 60,
      "avg_result_count": 3
    },
    "memo_semantic_only": {
      "strategy": "memo_semantic_only",
      "query_count": 45,
      "cost_usd": 2.25,
      "avg_latency_ms": 150,
      "avg_result_count": 5
    },
    "hybrid_standard": {
      "strategy": "hybrid_standard",
      "query_count": 45,
      "cost_usd": 4.50,
      "avg_latency_ms": 200,
      "avg_result_count": 8
    },
    "full_pipeline_with_reranker": {
      "strategy": "full_pipeline_with_reranker",
      "query_count": 10,
      "cost_usd": 2.75,
      "avg_latency_ms": 500,
      "avg_result_count": 10
    }
  },
  "top_expenses": [
    {
      "query": "总结我的工作计划",
      "strategy": "full_pipeline_with_reranker",
      "cost_usd": 0.060,
      "timestamp": "2025-01-21T10:30:00Z"
    }
  ]
}
```

---

### 2. 获取策略统计

获取各个路由策略的使用统计和性能指标。

**请求**：
```http
GET /api/v1/ai/strategy-stats?period=weekly
```

**参数**：
| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `period` | string | 否 | 时间周期：`daily`, `weekly`, `monthly`，默认 `weekly` |

**响应**：
```json
{
  "period": "weekly",
  "total_queries": 1050,
  "strategy_distribution": {
    "schedule_bm25_only": 35.0,
    "memo_semantic_only": 30.0,
    "hybrid_bm25_weighted": 15.0,
    "hybrid_with_time_filter": 15.0,
    "hybrid_standard": 5.0,
    "full_pipeline_with_reranker": 0.0
  },
  "performance_metrics": {
    "p50_latency_ms": 180,
    "p95_latency_ms": 350,
    "p99_latency_ms": 500,
    "throughput_qps": 120
  },
  "cost_optimization": {
    "current_monthly_cost": 28000,
    "projected_monthly_cost": 28500,
    "potential_savings": 7000,
    "optimization_suggestions": [
      "策略 'full_pipeline_with_reranker' 使用率过高，考虑降级到 'hybrid_standard'",
      "高成本查询占 5%，建议添加缓存"
    ]
  }
}
```

---

### 3. 查询成本日志

查询原始的成本日志记录（用于高级分析）。

**请求**：
```http
GET /api/v1/ai/cost-logs?start_date=2025-01-01&end_date=2025-01-21&limit=100
```

**参数**：
| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `start_date` | string | 否 | 开始日期（ISO 8601） |
| `end_date` | string | 否 | 结束日期（ISO 8601） |
| `limit` | int | 否 | 返回数量，默认 100 |
| `offset` | int | 否 | 偏移量，默认 0 |
| `strategy` | string | 否 | 过滤策略 |
| `user_id` | int | 否 | 过滤用户 ID |

**响应**：
```json
{
  "logs": [
    {
      "id": 12345,
      "timestamp": "2025-01-21T10:30:00Z",
      "user_id": 1,
      "query": "今天有什么安排",
      "strategy": "schedule_bm25_only",
      "vector_cost_usd": 0.001,
      "reranker_cost_usd": 0.0,
      "llm_cost_usd": 0.002,
      "total_cost_usd": 0.003,
      "latency_ms": 150,
      "result_count": 3
    }
  ],
  "total_count": 1500,
  "limit": 100,
  "offset": 0
}
```

---

## 📊 数据模型

### QueryCostLog

成本日志记录模型。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | bigint | 主键 |
| `timestamp` | timestamp | 记录时间 |
| `user_id` | integer | 用户 ID |
| `query` | text | 查询内容 |
| `strategy` | varchar(50) | 路由策略 |
| `vector_cost` | numeric(10,6) | 向量检索成本（美元） |
| `reranker_cost` | numeric(10,6) | Reranker 成本（美元） |
| `llm_cost` | numeric(10,6) | LLM 成本（美元） |
| `total_cost` | numeric(10,6) | 总成本（美元） |
| `latency_ms` | integer | 延迟（毫秒） |
| `result_count` | integer | 结果数量 |
| `user_satisfied` | numeric(3,2) | 用户满意度（0-1） |

### RouteDecision

路由决策模型。

| 字段 | 类型 | 说明 |
|------|------|------|
| `strategy` | string | 策略名称 |
| `confidence` | float32 | 置信度（0-1） |
| `time_range` | TimeRange | 时间范围 |
| `semantic_query` | string | 清理后的查询 |
| `needs_reranker` | bool | 是否需要 Reranker |

### 策略类型

| 策略 | 说明 | 平均成本 | 平均延迟 | 使用率 |
|------|------|---------|---------|--------|
| `schedule_bm25_only` | 纯日程查询 | $0.006 | 50ms | 35% |
| `memo_semantic_only` | 纯笔记查询 | $0.005 | 150ms | 30% |
| `hybrid_bm25_weighted` | 混合检索（BM25 加权） | $0.010 | 200ms | 15% |
| `hybrid_with_time_filter` | 混合检索（时间过滤） | $0.010 | 200ms | 15% |
| `hybrid_standard` | 标准混合检索 | $0.010 | 200ms | 5% |
| `full_pipeline_with_reranker` | 完整流程 | $0.060 | 500ms | <1% |

---

## 💡 使用示例

### 示例 1：监控每日成本

```bash
curl -X GET "http://localhost:28081/api/v1/ai/cost-report?period=daily" \
  -H "Authorization: Bearer <token>"
```

### 示例 2：分析策略分布

```bash
curl -X GET "http://localhost:28081/api/v1/ai/strategy-stats?period=weekly" \
  -H "Authorization: Bearer <token>"
```

### 示例 3：查询高成本查询

```sql
-- 直接 SQL 查询
SELECT
    query,
    strategy,
    total_cost_usd,
    latency_ms,
    timestamp
FROM query_cost_log
WHERE total_cost_usd > 0.05
ORDER BY total_cost_usd DESC
LIMIT 10;
```

---

## 🔐 权限要求

所有 FinOps API 端点都需要认证：

- **用户权限**：可以查看自己的成本数据
- **管理员权限**：可以查看所有用户的成本数据

---

## 📈 成本计算

### 向量检索成本

```go
cost = (textLength / 3.0) * (0.0001 / 1000000.0)
```

- 基于 SiliconFlow BGE-M3 模型
- 价格：$0.0001 / 1M tokens

### Reranker 成本

```go
cost = ((queryLength + docCount * avgDocLength) / 3.0 / 1000.0) * 0.0001
```

- 基于 SiliconFlow BGE Reranker
- 价格：$0.0001 / 1K tokens

### LLM 成本

```go
cost = (inputTokens * 0.14 / 1000000.0) + (outputTokens * 0.28 / 1000000.0)
```

- 基于 DeepSeek Chat 模型
- 输入价格：$0.14 / 1M tokens
- 输出价格：$0.28 / 1M tokens

---

## 🚀 客户端集成

### JavaScript/TypeScript

```typescript
interface CostReport {
  period: string;
  total_cost_usd: number;
  query_count: number;
  avg_latency_ms: number;
  by_strategy: {
    [strategy: string]: {
      query_count: number;
      cost_usd: number;
      avg_latency_ms: number;
      avg_result_count: number;
    };
  };
}

async function getCostReport(period: 'daily' | 'weekly' | 'monthly'): Promise<CostReport> {
  const response = await fetch(`/api/v1/ai/cost-report?period=${period}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  return response.json();
}

// 使用示例
const report = await getCostReport('daily');
console.log(`今日成本: $${report.total_cost_usd}`);
console.log(`平均延迟: ${report.avg_latency_ms}ms`);
```

### Python

```python
import requests

def get_cost_report(period='daily', token=None):
    """获取成本报告"""
    headers = {}
    if token:
        headers['Authorization'] = f'Bearer {token}'

    response = requests.get(
        f'http://localhost:28081/api/v1/ai/cost-report?period={period}',
        headers=headers
    )

    return response.json()

# 使用示例
report = get_cost_report('daily')
print(f"今日成本: ${report['total_cost_usd']}")
print(f"平均延迟: {report['avg_latency_ms']}ms")
```

---

## 📊 Grafana 集成

### 数据源配置

```json
{
  "name": "Memos PostgreSQL",
  "type": "postgres",
  "url": "postgres://memos:memos@localhost:25432/memos",
  "database": "memos"
}
```

### 推荐面板

1. **成本概览**
   - 总成本趋势
   - 策略分布饼图
   - 每日成本柱状图

2. **性能监控**
   - P50/P95/P99 延迟
   - QPS 趋势
   - 错误率

3. **策略分析**
   - 各策略使用率
   - 各策略平均成本
   - 各策略平均延迟

---

**最后更新**：2025-01-21
**文档版本**：v1.0
