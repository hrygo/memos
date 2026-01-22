# Phase 1-3 重构代码审查报告

**审查日期**: 2025-01-22  
**审查范围**: Phase 1-3 智能助理系统重构

---

## 实施概览

### Phase 1: 附件管理 (OCR + 全文提取)
- 新增 `plugin/ocr/` - Tesseract OCR 集成
- 新增 `plugin/textextract/` - Apache Tika 文档解析
- 新增 `server/runner/ocr/` - 后台处理 runner
- 数据库迁移: 附件表新增 `extracted_text`, `ocr_text`, `thumbnail_path` 字段

### Phase 2: 缓存优化
- 新增 `store/cache/` - 三层缓存架构
- 默认 L1 内存缓存 (30min TTL)
- L2 Redis 可选 (需配置环境变量)

### Phase 3: 智能查询路由
- 实现 BM25 全文检索 (PostgreSQL tsvector)
- 实现 RRF (Reciprocal Rank Fusion) 融合算法
- 新增 `server/scheduler/rrule/` - RFC 5545 重复规则
- 新增 `server/scheduler/suggestion/` - 智能时间推荐
- 新增 `server/stats/` - 轻量级使用统计

---

## 本次重构相关问题

### 需立即修复 (P0)

#### 1. Tika 文件写入失败
**文件**: `plugin/textextract/tika.go:211-216`

**问题**: 
```go
outputFile.Close()  // 过早关闭
if _, err := outputFile.Write(data); err != nil {  // 失败
```

**影响**: 所有 PDF/Office 文档全文提取功能无法工作

---

#### 2. RRULE BYDAY 只处理第一天
**文件**: `server/scheduler/rrule/rrule.go:309-322`

**问题**: 循环中提前 return，导致 `BYDAY=MO,WE,FR` 只计算周一

**影响**: 重复日程生成不正确

---

#### 3. 统计字段未实现
**文件**: `server/stats/stats.go:33-36`

**问题**: `ActiveDays`, `LastActivityTime`, `StreakDays` 在 `GetSummary()` 中使用但从未赋值

**影响**: 统计摘要显示不准确

---

### 建议修复 (P1)

#### 4. 并发安全性
- `server/retrieval/adaptive_retrieval.go`: 添加 ctx.Done() 检查防止 goroutine 泄漏
- `server/runner/ocr/runner.go`: 添加 goroutine 限流机制

#### 5. 输入验证
- `store/memo_embedding.go`: 为 `BM25SearchOptions` 添加长度和限制验证

---

### 架构建议 (P2)

#### 6. 目录结构
- `server/scheduler/rrule/` → 考虑移至 `plugin/rrule/` 或 `internal/rrule/`
- `server/scheduler/suggestion/` → 考虑移至 `plugin/schedule/`
- `server/stats/` → 考虑移至 `internal/stats/`

#### 7. 测试覆盖
- 为 `plugin/ocr/` 和 `plugin/textextract/` 添加测试
- 为 RRF 融合算法添加单元测试

---

## 配置变更

### 新增环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MEMOS_OCR_ENABLED` | `false` | OCR 功能开关 |
| `MEMOS_TEXTEXTRACT_ENABLED` | `false` | 全文提取开关 |
| `MEMOS_CACHE_REDIS_ADDR` | - | Redis L2 缓存地址（可选） |

### 缓存配置

- L1 内存缓存: 1000 项, 30 分钟 TTL
- L2 Redis: 可选, 30 分钟 TTL

---

## 总结

本次重构实现了 Phase 1-3 的核心功能，但存在 3 个功能 Bug 需要立即修复。架构和测试问题可以在后续迭代中逐步改进。

**下一步**:
1. 修复 P0 问题
2. 添加测试覆盖
3. 提交 PR 到上游
