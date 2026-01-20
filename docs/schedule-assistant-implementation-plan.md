# 日程助手功能实现计划

> **⚠️ 重要声明：MySQL 数据库不再支持**
>
> 从本功能开始，所有新功能仅支持 **PostgreSQL** 和 **SQLite** 数据库。
>
> **原因**：
> - MySQL 缺乏 JSON 字段约束和高级触发器支持
> - 向量搜索（pgvector）仅在 PostgreSQL 上可用
> - 维护三数据库兼容性的成本过高
> - MySQL 驱动存在已知 bug 且难以修复
>
> **建议**：现有 MySQL 用户请迁移到 PostgreSQL 或 SQLite。

## 功能概述

日程助手是一个基于 AI 的智能日程管理系统，通过自然语言交互，集成 AI 聊天、应用内通知，提供完整的日程管理服务。

## 第一期功能范围（MVP）

### 核心功能
- ✅ **独立数据模型**：创建独立的 Schedule 表
- ✅ **完整 CRUD**：创建、查询、编辑、删除日程
- ✅ **自然语言解析**：支持中文时间表达，自动提取日程信息
- ✅ **智能提醒**：支持多种提醒方式（15分钟前、1小时前等）
- ✅ **冲突检测**：创建时检测时间冲突
- ⏳ **AI 聊天集成**：待前端实现
- ⏳ **日历视图**：聊天页面内嵌小型日历组件- 待实现
- ⏳ **重复日程**：支持每日、每周、每月、工作日等重复规则- 待实现
- ⏳ **实时通知**：应用内通知（WebSocket/SSE）- 待实现

### 第二期功能
- 钉钉机器人集成（双向同步）
- 邮件提醒

## 架构设计

### 后端架构

```
plugin/ai/schedule/                    # 日程助手核心模块
├── parser.go                         # 自然语言时间/事件解析器 (待实现)
├── scheduler.go                      # 日程管理逻辑 (待实现)
├── recurrence.go                     # 重复日程规则处理 (待实现)
└── conflict.go                       # 冲突检测 (待实现)

server/router/api/v1/
├── schedule_service.go               # 日程 API 服务实现 ✅
└── schedule_service.pb.go            # Protobuf 生成代码 ✅

store/
├── schedule.go                       # 日程数据存储接口 ✅
└── db/
    ├── postgres/schedule.go          # PostgreSQL 实现 ✅
    ├── sqlite/schedule.go            # SQLite 实现 ✅
    └── mysql/schedule.go             # MySQL 实现 ✅

server/runner/schedule/
└── reminder.go                       # 提醒后台任务 (待实现)
```

### 前端架构

```
web/src/
├── pages/AIChat.tsx                  # 扩展：添加日程意图识别和日历 (待修改)
├── components/AIChat/
│   ├── ScheduleCalendar.tsx         # 内嵌日历组件 (待创建)
│   ├── ScheduleInput.tsx            # 日程快速输入/确认 (待创建)
│   └── ScheduleConflictAlert.tsx    # 冲突提示组件 (待创建)
├── hooks/
│   ├── useScheduleQueries.ts        # 日程 API hooks (待创建)
│   └── useSSE.ts                    # SSE 连接（实时通知）(待创建)
└── types/proto/api/v1/
    └── schedule_service_pb.ts       # Protobuf 类型 ✅ (已生成)
```

## 实现进度

### ✅ Phase 1: 数据库迁移
- [x] 创建 schedule 表迁移脚本
- [x] 为 PostgreSQL/MySQL/SQLite 创建兼容版本
- [x] 更新 LATEST.sql

**文件**:
- `store/migration/postgres/0.26/1__add_schedule.sql`
- `store/migration/sqlite/0.26/1__add_schedule.sql`
- `store/migration/mysql/0.26/1__add_schedule.sql` (已废弃，不推荐使用)

### ✅ Phase 2: Store 层实现
- [x] 定义 `Schedule` 结构体
- [x] 实现 CRUD 方法接口
- [x] 实现各数据库驱动

**文件**:
- `store/schedule.go` - 数据模型和接口
- `store/db/postgres/schedule.go` - PostgreSQL 驱动
- `store/db/sqlite/schedule.go` - SQLite 驱动
- `store/db/mysql/schedule.go` - MySQL 驱动

### ✅ Phase 3: Protobuf 和 API 服务
- [x] 定义 Protobuf API
- [x] 生成代码（通过 `buf generate`）
- [x] 实现 ScheduleService
- [x] 实现 ParseAndCreateSchedule API（Phase 5 前置）
- [x] 在 v1.go 中注册服务
- [x] 在 connect_handler.go 中添加 Connect 包装方法

**文件**:
- `proto/api/v1/schedule_service.proto` - Protobuf 定义
- `proto/gen/api/v1/schedule_service*.go` - 生成的代码
- `server/router/api/v1/schedule_service.go` - 服务实现
- `server/router/api/v1/v1.go` - 服务注册（已修改）
- `server/router/api/v1/connect_handler.go` - Connect 包装（已修改）

**实现说明**:
- ✅ 所有基础 CRUD API 已实现
- ✅ CheckConflict API 已实现（基础重叠检测）
- ✅ ParseAndCreateSchedule API 已实现（集成 LLM 自然语言解析）

### ⏳ Phase 4: 前端基础 (待实现)
- [ ] 生成 TypeScript 类型 (已完成，通过 protobuf)
- [ ] 实现 React Query hooks
- [ ] 创建日历组件（基于 ActivityCalendar）
- [ ] 创建日程卡片组件

**待创建文件**:
- `web/src/hooks/useScheduleQueries.ts`
- `web/src/components/AIChat/ScheduleCalendar.tsx`
- `web/src/components/AIChat/ScheduleInput.tsx`
- `web/src/components/AIChat/ScheduleConflictAlert.tsx`

### ✅ Phase 5: 自然语言解析
- [x] 实现时间解析器（支持中文）
- [x] 实现事件提取器
- [x] 集成 LLM 进行意图识别
- [x] 实现 ParseAndCreateSchedule API

**文件**:
- `plugin/ai/schedule/parser.go` - 时间解析器和事件提取器 ✅

**实现说明**:
- 使用 LLM (DeepSeek) 进行自然语言理解
- 支持中文时间表达（"明天下午3点"、"下周三"等）
- 自动提取标题、时间、地点、提醒等信息
- 支持 auto_confirm 模式直接创建日程

### ⏳ Phase 6: AI 聊天集成 (待实现)
- [ ] 扩展 AI 服务支持日程意图识别
- [ ] 前端意图检测
- [ ] 添加确认对话框
- [ ] 流式响应处理

**待修改文件**:
- `server/router/api/v1/ai_service.go` - 添加日程意图识别
- `web/src/pages/AIChat.tsx` - 集成日程功能
- `web/src/hooks/useAIQueries.ts` - 添加日程意图处理

### ⏳ Phase 7: 冲突检测 (待实现)
- [ ] 实现时间冲突检测逻辑
- [ ] 集成到创建流程
- [ ] AI 主动提醒功能

**待创建文件**:
- `plugin/ai/schedule/conflict.go`

### ⏳ Phase 8: 重复日程 (待实现)
- [ ] 实现 RRULE 解析器（简化版）
- [ ] 计算重复实例
- [ ] 更新查询逻辑

**待创建文件**:
- `plugin/ai/schedule/recurrence.go`

### ⏳ Phase 9: 实时通知 (待实现)
- [ ] 实现后台提醒检查任务
- [ ] 实现 SSE 服务端推送
- [ ] 前端 SSE 连接
- [ ] 通知 UI 组件

**待创建文件**:
- `server/runner/schedule/reminder.go`
- `web/src/hooks/useSSE.ts`

### ⏳ Phase 10: 端到端测试 (待实现)
- [ ] 完整流程测试
- [ ] 性能优化
- [ ] 错误处理完善

**技术债务清单**:

#### 1. 单元测试 ⚠️ 高优先级
**状态**: 未实现
**影响**: 代码质量、重构安全性
**工作量**: 2-3 天

**需要测试的模块**:
- `store/schedule.go` - 数据模型和辅助方法
- `store/db/postgres/schedule.go` - PostgreSQL CRUD 操作
- `store/db/sqlite/schedule.go` - SQLite CRUD 操作
- `server/router/api/v1/schedule_service.go` - API 服务层
- `plugin/ai/schedule/parser.go` - 自然语言解析器

**测试覆盖目标**:
- 代码覆盖率 > 80%
- 关键路径（CRUD、冲突检测）100% 覆盖
- 边界条件和错误处理场景

#### 2. 性能测试 ⚠️ 中优先级
**状态**: 未实现
**影响**: 生产环境稳定性
**工作量**: 1-2 天

**性能指标**:
- 单次 CRUD 操作延迟 < 100ms (P95)
- ListSchedules 查询性能（1000 条记录）< 200ms (P95)
- CheckConflict 冲突检测 < 100ms (P95)
- ParseAndCreateSchedule (LLM 调用) < 3s (P95)

**压力测试场景**:
- 100 个并发用户创建日程
- 10000 条日程的查询性能
- 长时间运行的 LLM 调用稳定性

#### 3. 集成测试 ⚠️ 中优先级
**状态**: 未实现
**工作量**: 1-2 天

**测试场景**:
- 端到端日程创建流程
- 自然语言解析准确性测试
- 多用户数据隔离验证
- 跨时区日程处理

**测试数据**:
- 各种中文时间表达
- 边界时间（跨天、跨月、跨年）
- 特殊字符和长文本

#### 4. 错误处理完善 ⚠️ 低优先级
**状态**: 部分完成
**工作量**: 0.5-1 天

**待处理项**:
- LLM 调用失败降级策略
- 数据库连接池耗尽处理
- 并发写入冲突处理
- 更详细的错误信息返回

**建议优先级**:
1. **立即执行**: 单元测试（Phase 10.1）- 保障代码质量
2. **短期执行**: 集成测试（Phase 10.3）- 功能验证
3. **中期规划**: 性能测试（Phase 10.2）- 生产准备
4. **持续优化**: 错误处理（Phase 10.4）- 渐进式改进

---

## 数据库设计

### 主表：schedule

```sql
-- PostgreSQL 版本
CREATE TABLE schedule (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,

  -- 标准字段
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',

  -- 日程核心字段
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  location TEXT DEFAULT '',

  -- 时间字段
  start_ts BIGINT NOT NULL,           -- Unix timestamp (UTC)
  end_ts BIGINT,                      -- 可选，对于全天事件
  all_day BOOLEAN NOT NULL DEFAULT FALSE,

  -- 时区
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',

  -- 重复规则 (JSON 格式)
  recurrence_rule TEXT,               -- {"frequency":"WEEKLY","interval":1}
  recurrence_end_ts BIGINT,           -- 重复结束时间

  -- 提醒设置 (JSON 数组格式)
  reminders TEXT NOT NULL DEFAULT '[]', -- [{"type":"before","value":15,"unit":"minutes"}]

  -- 扩展
  payload JSONB NOT NULL DEFAULT '{}',

  FOREIGN KEY (creator_id) REFERENCES "user"(id) ON DELETE CASCADE
);

-- 索引（性能优化）
CREATE INDEX idx_schedule_creator_start ON schedule(creator_id, start_ts);
CREATE INDEX idx_schedule_creator_status ON schedule(creator_id, row_status);
CREATE INDEX idx_schedule_start_ts ON schedule(start_ts);
```

## API 端点

### 已实现的 API

| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/api/v1/schedules` | 创建日程 |
| GET | `/api/v1/schedules` | 列出日程 |
| GET | `/api/v1/schedules/{uid}` | 获取单个日程 |
| PATCH | `/api/v1/schedules/{uid}` | 更新日程 |
| DELETE | `/api/v1/schedules/{uid}` | 删除日程 |
| POST | `/api/v1/schedules:checkConflict` | 检查冲突 |
| POST | `/api/v1/schedules:parseAndCreate` | 自然语言创建日程 ✅ |

**注意**: `PATCH /api/v1/schedules/{uid}` 更新接口有两种行为模式：

1. **提供 `update_mask`**（推荐）：只更新 `update_mask` 中指定的字段，其他字段保持不变。这是**推荐模式**，可以避免意外覆盖字段。
2. **不提供 `update_mask`**：更新所有非零/非空字段。这种模式可能会导致意外覆盖，**不推荐使用**。

`update_mask` 支持的字段包括：
- `title` - 标题
- `description` - 描述
- `location` - 地点
- `start_ts` - 开始时间
- `end_ts` - 结束时间
- `all_day` - 是否全天
- `timezone` - 时区
- `recurrence_rule` - 重复规则
- `recurrence_end_ts` - 重复结束时间
- `state` - 状态 (NORMAL/ARCHIVED)
- `reminders` - 提醒设置

## 验证计划

### 测试场景

1. **基础 CRUD**
   - 创建日程："明天下午3点开会"
   - 查看日程列表
   - 编辑日程
   - 删除日程

2. **自然语言解析**
   - "明天下午3点开会"
   - "下周三全天参加技术大会"
   - "每周一上午10点站会，持续一个月"
   - "明天下午2点和客户开会，地点在会议室A，提前15分钟提醒"

3. **冲突检测**
   - 创建重叠时间的日程时显示冲突
   - AI 主动提醒有冲突的日程

4. **重复日程**
   - 每日重复
   - 每周重复
   - 工作日重复
   - 月度重复

5. **提醒通知**
   - 提前 15 分钟收到通知
   - 应用内通知正确显示

6. **日历视图**
   - 日历上显示日程标记
   - 点击日程查看详情

---

## 开发记录

### 2026-01-20

#### 上午 (Phase 1-3)
- 完成数据库迁移文件创建（PostgreSQL/SQLite/MySQL）
- 完成 Store 层实现（数据模型 + 接口）
- 完成 Protobuf 定义和代码生成
- 完成 ScheduleService 基础 API 实现（CRUD + CheckConflict）
- 完成服务注册和 Connect 集成
- 后端基础 CRUD 已可用

#### 下午 - Bug 修复分支
- 创建 `fix/schedule-assistant` 分支
- 修复文档版本号错误（0.31 → 0.26）
- 修复时间范围查询逻辑错误
- 修复 parseInt32 错误处理
- 移除 OrderByTimeAsc 冗余字段
- 添加输入验证（title、start_ts、end_ts、reminders）
- 完善冲突检测逻辑（添加参数验证、改进重叠检测）
- 更新 update_mask 文档说明
- 合并修复分支到主分支

#### 下午 - Phase 5 实现
- 实现 `plugin/ai/schedule/parser.go` 自然语言解析器
- 集成 LLM (DeepSeek) 进行中文时间理解
- 完成 ParseAndCreateSchedule API 实现
- 支持自动创建模式（auto_confirm）
- 后端 API 已完整可用

### 已完成功能
✅ Phase 1: 数据库迁移
✅ Phase 2: Store 层实现
✅ Phase 3: Protobuf 和 API 服务
✅ Phase 5: 自然语言解析

### 待完成功能
⏳ Phase 4: 前端基础（React Query hooks、日历组件）
⏳ Phase 6: AI 聊天集成（前端意图识别）
⏳ Phase 7: 冲突检测增强（AI 主动提醒）
⏳ Phase 8: 重复日程（RRULE 解析）
⏳ Phase 9: 实时通知（SSE 推送）
⏳ Phase 10: 端到端测试（技术债务）

### 下一步计划
1. **前端开发**: 实现 Phase 4 前端基础功能
2. **AI 集成**: 在 AIChat 页面集成日程意图识别
3. **测试完善**: 补充单元测试和集成测试（技术债务）

### 技术债务
详见 [Phase 10 技术债务清单](#-phase-10-端到端测试-待实现)

---

*计划版本: 1.2*
*创建时间: 2026-01-20*
*最后更新: 2026-01-20*
*状态: Phase 1-3, 5 已完成，后端 API 完整可用，待前端集成*
