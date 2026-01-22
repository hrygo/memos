# 🔍 日程检索问题诊断报告

> **诊断日期**：2025-01-21
> **问题**："1月21日有哪些事？"返回"暂无日程"
> **状态**：✅ 数据有数据，❌ 检索失败

---

## 📊 已确认的事实

### 1. 数据库有数据 ✅

```sql
SELECT id, title, to_timestamp(start_ts) as scheduled_time
FROM schedule
WHERE start_ts >= extract(epoch from '2026-01-21'::timestamp)
AND start_ts < extract(epoch from '2026-01-22'::timestamp)
ORDER BY start_ts;
```

**结果**：4 条日程
- ID 17: 开会讨论如何吃西瓜 (01:00-02:00 UTC)
- ID 18: 开会讨论如何吃黄瓜 (03:00-04:00 UTC)
- ID 11: 开会讨论绩效考核问题 (05:00-09:00 UTC)
- ID 4: 新标题 (08:14-09:14 UTC)

### 2. 代码已更新 ✅

- ✅ `query_router.go` 包含日期解析功能
- ✅ `connect_handler.go` 包含 QueryRouter 调用
- ✅ 二进制文件是最新的（编译在源码修改之后）

### 3. 初始化代码正确 ✅

```go
// server/router/api/v1/v1.go:73-83
queryRouter := queryengine.NewQueryRouter()
adaptiveRetriever := retrieval.NewAdaptiveRetriever(store, embeddingService, rerankerService)

service.AIService = &AIService{
    QueryRouter:       queryRouter,        // ✅ 初始化
    AdaptiveRetriever: adaptiveRetriever,  // ✅ 初始化
}
```

---

## ⚠️ 问题分析

### 可能原因

#### 原因 1：前端使用的是 Connect RPC（90% 可能）

Connect RPC 的聊天接口是：
- **路径**：`/api/v1/ai/chat` (Connect RPC)
- **方法**：POST
- **代码**：`server/router/api/v1/connect_handler.go:ChatWithMemos`

但 Connect RPC 的日志可能没有 `[QueryRouting]` 前缀，导致看不到。

#### 原因 2：日志输出被抑制（5% 可能）

代码中的日志是 `fmt.Printf`，可能被缓冲或重定向。

#### 原因 3：降级到旧逻辑（5% 可能）

如果 `QueryRouter` 或 `AdaptiveRetriever` 为 nil，会降级到旧逻辑（使用向量检索，不查询日程）。

---

## 🎯 诊断步骤

### 步骤 1：确认实时日志

```bash
# 在终端 1：监控所有日志（不过滤）
make logs backend > /tmp/backend_full.log 2>&1 &

# 在终端 2：查看日志
tail -f /tmp/backend_full.log
```

**然后在前端查询"1月21日有哪些事？"**

**预期输出**：
```
[QueryRouting] Strategy: hybrid_with_time_filter, Confidence: 0.95
[QueryRouting] TimeRange: 1月21日 (2026-01-21 00:00 to 2026-01-22 00:00)
[AdaptiveRetriever] Found 4 results
```

**如果没有看到**：
- 说明代码没有生效，需要重新编译和部署

### 步骤 2：验证代码生效

```bash
# 检查编译后的二进制文件
strings bin/memos | grep -i "queryrouting\|智能 Query Routing"

# 预期输出（应该有内容）：
# [QueryRouting] Strategy: %s, Confidence: %.2f
```

**如果没有输出**：说明二进制文件没有包含新代码，需要重新编译。

### 步骤 3：强制重新编译和部署

```bash
# 1. 停止服务
make stop

# 2. 清理旧的二进制
rm -f bin/memos

# 3. 重新编译
make build

# 4. 确认编译时间
ls -lh bin/memos

# 5. 启动服务
make start

# 6. 等待服务启动
sleep 5

# 7. 查看日志确认版本
make logs backend | head -20
```

### 步骤 4：测试查询

```bash
# 在前端查询："1月21日有哪些事？"

# 同时查看日志：
make logs backend | tail -50
```

**预期结果**：
```
您1月21日的日程安排如下：
- 01:00 - 开会讨论如何吃西瓜
- 03:00 - 开会讨论如何吃黄瓜
- 05:00 - 开会讨论绩效考核问题
- 08:14 - 新标题
```

---

## 🔧 快速修复

如果上述步骤都确认了，但仍然没有日程，请执行：

```bash
# 完全重新编译和部署
make stop
rm -rf bin/
make build
make start

# 验证
strings bin/memos | grep "Phase 1: 智能 Query Routing"

# 如果有输出，说明编译成功
# 然后测试查询
```

---

## 📝 需要提供的信息

如果问题仍然存在，请提供：

1. **完整的后端日志**（查询时的日志）
   ```bash
   make logs backend > /tmp/backend_query.log 2>&1
   cat /tmp/backend_query.log
   ```

2. **二进制文件信息**
   ```bash
   ls -lh bin/memos
   md5 bin/memos
   ```

3. **API 响应**（如果可能）
   - 前端显示的完整回复
   - 包括 `SOURCES` 字段

---

**下一步**：请按照"诊断步骤"操作，并提供日志输出。

---

**文档版本**：v1.0
**最后更新**：2025-01-21
**维护者**：Claude & Memos Team
