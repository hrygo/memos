---
name: code-review
description: |
  Pre-commit/Pre-merge Code Review. Analyzes code changes to detect security 
  vulnerabilities, bugs, and anti-patterns. Outputs structured Markdown report.
  Read-only analysis, never modifies code.
allowed-tools: Bash, Read, Grep, View
disable-model-invocation: false
---

# 🔍 Code Review Skill

> **职责**: 只读审查 → 结构化报告 → 开发者决策

---

## 🧠 AI 科学原则

本 Skill 应用以下技术提升精准度、降低幻觉：

| 技术                  | 应用                          |
| --------------------- | ----------------------------- |
| **Chain-of-Thought**  | 分步推理，先分析后结论        |
| **Grounding**         | 基于实际 diff 内容，引用行号  |
| **Self-Verification** | 输出前自检，确认问题真实存在  |
| **Structured Output** | 强制表格格式，减少自由发挥    |
| **Confidence Score**  | 标注置信度，区分确定/疑似问题 |

---

## 执行流程

```
1. 范围 → 2. 规则 → 3. Diff → 4. 推理 → 5. 验证 → 6. 自检 → 7. 报告
```

### Step 1-3: 数据收集

```bash
git diff --cached --name-only || git diff --name-only
```

### Step 4: Chain-of-Thought 推理

对每个文件：
```
思考过程：
1. 这是什么语言/框架？
2. 变更了什么功能？
3. 逐行检查：
   - 安全性：是否涉及用户输入、数据库、认证？
   - Bug：是否有空值、资源、边界问题？
   - 规范：是否符合语言惯用法？
4. 只有当我**确信**存在问题时才报告
```

### Step 5: 执行验证

| 语言       | 验证命令                         |
| ---------- | -------------------------------- |
| Go         | `go build ./... && go vet ./...` |
| TypeScript | `npx tsc --noEmit`               |

### Step 6: Self-Verification (关键!)

输出前自问：
```
□ 我报告的每个问题，是否在 diff 中有对应代码？
□ 行号是否准确？
□ 问题描述是否基于事实而非推测？
□ 建议是否可行？
```

**原则**: 宁可漏报，不可误报。

### Step 7: 生成报告

---

## 输出格式

```markdown
# 📋 Code Review Report

**时间**: YYYY-MM-DD HH:mm
**范围**: staged | working | branch
**文件数**: N

## � 发现问题

### 🔒 Critical [置信度: 高/中]
| 文件    | 行  | 问题     | 证据                     | 建议         |
| ------- | --- | -------- | ------------------------ | ------------ |
| file.go | 42  | SQL 注入 | `query := "..." + input` | 用参数化查询 |

### ⚠️ High [置信度: 高/中]
| 文件 | 行  | 问题 | 证据 | 建议 |
| ---- | --- | ---- | ---- | ---- |

### 💡 Medium
| 文件 | 行  | 问题 | 建议 |
| ---- | --- | ---- | ---- |

## ✅ 验证通过
- [x] `go build` ✓
- [x] `go vet` ✓

## 📊 统计
Critical: X | High: Y | Medium: Z

## 结论
✅ 可提交 / 🚨 需修复
```

---

## 置信度标准

| 级别   | 定义         | 条件                     |
| ------ | ------------ | ------------------------ |
| **高** | 确定存在问题 | 代码模式明确匹配已知漏洞 |
| **中** | 可能存在问题 | 需开发者确认上下文       |

> **只报告"高"置信度的 Critical 问题**

---

## 降幻觉策略

| 策略           | 实现                         |
| -------------- | ---------------------------- |
| **引用证据**   | 每个问题必须引用实际代码片段 |
| **标注行号**   | 必须提供准确行号             |
| **不推测**     | 只分析 diff 中实际存在的代码 |
| **不假设**     | 不假设 diff 外的代码逻辑     |
| **承认不确定** | 用"可能"而非"一定"           |

---

## 项目规则 (可选)

如存在 `.code-review.yaml`：

```yaml
exclude: [vendor/, node_modules/]
checks:
  i18n: "make check-i18n"
```

---

## 协作

| 场景     | 协作              |
| -------- | ----------------- |
| 审查通过 | → `atomic-commit` |
| 需修复   | → 开发者手动      |

---

## 不做什么

- ❌ 修改代码
- ❌ 推测 diff 外的问题
- ❌ 报告不确定的问题为 Critical
