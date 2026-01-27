# AI 产品研究与规划文档

> 本目录包含 Memos AI 能力的调研报告与实施路线图。

---

## 文档索引

### 主文档

| 文档 | 说明 |
|:---|:---|
| [00-master-roadmap.md](./00-master-roadmap.md) | **统一实施路线图** - 整合三个领域的协同实施方案 |

### 智能助理 (Team B)

| 文档 | 类型 | 说明 |
|:---|:---|:---|
| [assistant-research.md](./assistant-research.md) | 调研 | 智能助理架构设计与实现细节分析 |
| [assistant-roadmap.md](./assistant-roadmap.md) | 路线图 | 私人版智能助理升级路径 (实际采用) |
| [assistant-roadmap-industry.md](./assistant-roadmap-industry.md) | 参考 | 行业最佳实践版路线图 (参考存档) |

### 日程管理 (Team B)

| 文档 | 类型 | 说明 |
|:---|:---|:---|
| [schedule-research.md](./schedule-research.md) | 调研 | 日程 Agent 系统架构与实现分析 |
| [schedule-roadmap.md](./schedule-roadmap.md) | 路线图 | 日程管理升级路径 |

### 笔记 AI 增强 (Team C)

| 文档 | 类型 | 说明 |
|:---|:---|:---|
| [memo-research.md](./memo-research.md) | 调研 | 笔记 AI 增强功能竞品分析与方案调研 |
| [memo-roadmap.md](./memo-roadmap.md) | 路线图 | 笔记 AI 增强升级路径 |

---

## 文档关系

```
                    ┌─────────────────────────┐
                    │  00-master-roadmap.md   │  ◄── 统一实施方案
                    │    (整合路线图)          │
                    └───────────┬─────────────┘
                                │
          ┌─────────────────────┼─────────────────────┐
          │                     │                     │
          ▼                     ▼                     ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│  智能助理        │   │  日程管理        │   │  笔记增强        │
├─────────────────┤   ├─────────────────┤   ├─────────────────┤
│ assistant-      │   │ schedule-       │   │ memo-           │
│ research.md     │   │ research.md     │   │ research.md     │
│       ↓         │   │       ↓         │   │       ↓         │
│ assistant-      │   │ schedule-       │   │ memo-           │
│ roadmap.md      │   │ roadmap.md      │   │ roadmap.md      │
└─────────────────┘   └─────────────────┘   └─────────────────┘
```

---

## 命名规范

| 后缀 | 含义 | 示例 |
|:---|:---|:---|
| `-research.md` | 调研报告 | `assistant-research.md` |
| `-roadmap.md` | 实施路线图 | `memo-roadmap.md` |
| `-roadmap-industry.md` | 行业参考版 (存档) | `assistant-roadmap-industry.md` |
| `00-` 前缀 | 主文档/入口 | `00-master-roadmap.md` |

---

> **维护说明**: 新增文档请遵循上述命名规范，并在本索引中登记。
