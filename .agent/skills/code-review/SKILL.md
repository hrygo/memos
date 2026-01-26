---
name: code-review
description: |
  Code review expert: audit changes for security, stability, performance, and architecture.
  Output structured report with table summary and detailed analysis.
allowed-tools: Bash, Read, Grep, Glob
disable-model-invocation: false
---

# Code Review

审计代码变更，输出结构化报告。

---

## 执行

1. **获取变更**: `git diff --cached` 或 `git diff`
2. **加载上下文**: 读取 `.code-review.yaml` (如有)
3. **运行检查**: 执行语言对应的 linter
4. **深度审计**: 安全/稳定性/性能/架构/UX/UI
5. **输出报告**: 遵循下方格式

---

## 报告格式

### 通过
```
✅ Code Review 通过
审查文件: 5 个 | 静态检查: 通过
```

### 发现问题
```
🚨 发现 4 个审计项 (严重: 1, 高危: 2, 中等: 1)

┌───────────┬──────────┬─────────────────────┬──────────────┐
│ 级别      │ 类别     │ 位置                │ 问题 (置信度) │
├───────────┼──────────┼─────────────────────┼──────────────┤
│ 🔴 严重   │ 安全     │ pkg/db/query.go:42  │ SQL 注入 (高) │
│ 🟠 高危   │ 性能     │ api/handler.go:78   │ N+1 查询 (中) │
│ 🟠 高危   │ 稳定性   │ store/mem.go:105    │ nil 风险 (高) │
│ 🟡 中等   │ 架构     │ router/routes.go:23 │ 循环依赖 (中) │
└───────────┴──────────┴─────────────────────┴──────────────┘

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[🔴 严重] 安全 - SQL 注入
├─ pkg/db/query.go:42
├─ 证据: `query := "SELECT * FROM users WHERE id = " + userId`
├─ 影响: 攻击者可执行任意 SQL
└─ 修复: db.Query("SELECT ... WHERE id = ?", userId)

[🟠 高危] 性能 - N+1 查询
├─ api/handler.go:78
├─ 证据: 循环中调用 repo.GetPosts(user.ID)
├─ 影响: N 用户 → N+1 次查询
└─ 修复: 使用 WHERE IN 预加载

[🟠 高危] 稳定性 - nil 指针风险
├─ store/mem.go:105
├─ 证据: `return s.cache[id].Value` (未检查 key 存在)
├─ 影响: key 不存在时 panic
└─ 修复: 先检查 `ok := s.cache[id]`

[🟡 中等] 架构 - 循环依赖
├─ router/routes.go:23
├─ 证据: import "myapp/handler" (handler 反向 import router)
├─ 影响: 编译循环依赖
└─ 修复: 提取接口到独立包
```

---

## 审计标准

| 级别   | 判据                           |
| ------ | ------------------------------ |
| 🔴 严重 | 可利用漏洞、确认崩溃、数据丢失 |
| 🟠 高危 | 明确 bug、可复现、重大反模式   |
| 🟡 中等 | 潜在问题、上下文依赖           |
| 🟢 低危 | 风格、次要优化                 |

| 类别 | 覆盖范围                        |
| ---- | ------------------------------- |
| 安全 | 注入、加密、授权、XSS、凭据泄露 |
| 稳定 | 竞态、泄漏、错误处理、边界条件  |
| 性能 | 算法复杂度、N+1 查询、阻塞 I/O  |
| 架构 | 语言惯用法、耦合度、可维护性    |

---

## 约束

- **只读**: 绝不修改代码
- **证据驱动**: 每个问题必须包含 file:line 和代码片段
- **拒绝幻觉**: 无直接证据不报告

---

## 项目上下文

`.code-review.yaml` (可选):

```yaml
overrides:
  security:
    ignore:
      - "demo/* 允许不安全示例"
      - "test/* 允许测试凭证"
  architecture:
    conventions:
      go:
        - "错误绝不静默忽略"
        - "包名小写单词"
      typescript:
        - "组件 PascalCase"
        - "Hooks use 前缀"
```
