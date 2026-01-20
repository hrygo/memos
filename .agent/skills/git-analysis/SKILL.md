---
name: git_merge_analysis
description: Performs a comprehensive analysis of git branch comparisons to evaluate merge necessity, risks, and impacts. Generates a detailed report.
---

# Git Merge Analysis Skill

This skill guides you through the process of analyzing changes between two git references to produce a professional merge analysis report.

**Default Upstream**: `https://github.com/usememos/memos`
**Output Language**: Chinese (Simplified)

## 1. Context Gathering

First, understand the repositories and branches involved.

1.  Check current remotes:
    ```bash
    git remote -v
    ```
2.  **Ensure Upstream Exists**:
    If `upstream` does not exist or points to a different URL, configure it:
    ```bash
    git remote remove upstream 2>/dev/null
    git remote add upstream https://github.com/usememos/memos
    ```
3.  Fetch the latest upstream changes:
    ```bash
    git fetch upstream
    ```

## 2. Data Collection

Collect raw data about the divergence between the `SOURCE` (default: `upstream/main`) and `TARGET` (default: `HEAD`).

1.  **Commit Overview**:
    -   Count commits: `git rev-list --count TARGET..SOURCE`
    -   List top commits (focus on conventions like feat/fix/breaking):
        ```bash
        git log --oneline --graph --decorate -n 20 TARGET..SOURCE
        ```
2.  **File Impact**:
    -   Get a summary of changed files:
        ```bash
        git diff --stat TARGET..SOURCE
        ```
    -   Identify "High Risk" files:
        -   Build/Config: `Makefile`, `Dockerfile`, `go.mod`, `package.json`, `vite.config.ts`, `.env*`
        -   Database: `store/migration/**/*.sql`, `store/db/**/*.go`
        -   API/Protocol: `proto/**/*.proto`
        -   Core Logic: `server/server.go`, `server/router/**/*.go`

3.  **Key Changes Inspection**:
    -   If a commit message suggests a breaking change (e.g., "refactor!", "breaking", "remove"), inspect it specifically:
        ```bash
        git show <commit_hash> --stat
        ```

## 3. Analysis Framework

Analyze the collected data using the following dimensions:

### A. Change Categorization
-   **Breaking Changes**: API changes, config flag removals, database schema changes.
-   **Features**: New capabilities added.
-   **Fixes**: Bug fixes, performance improvements, security patches.
-   **Refactoring**: Code structure changes without behavioral changes.

### B. Impact Assessment
-   **Build/Run**: Will the project still compile? Do environment variables or startup flags need changing?
-   **Database**: Are there new migrations? Is manual intervention required?
-   **Conflicts**: Which files are heavily modified in both branches?

### C. Necessity Evaluation
-   **Critical**: Security fixes, crash fixes, unblocking core features.
-   **High**: New stable features, performance boosts.
-   **Medium**: Minor bug fixes.
-   **Low**: Cosmetic changes.

## 4. Report Generation

Output a markdown report in **Chinese** with the following structure:

```markdown
# 🕵️ Git 合并分析报告

**源分支**: `[SOURCE]` | **目标分支**: `[TARGET]`
**差异提交数**: `[COUNT]` | **变更文件数**: `[COUNT]`

## 🚨 执行摘要 (Executive Summary)
[2-3 句话总结是否建议合并，紧迫程度，以及最大的风险点。]

## 🔄 核心变更 (Key Changes)
- **💥 破坏性变更 (Breaking Changes)**: [列出破坏性变更]
- **✨ 新特性 (Features)**: [列出关键新功能]
- **🐛 修复 (Fixes)**: [列出重要修复]

## ⚠️ 影响评估 (Impact Assessment)
| 领域          | 影响等级 | 说明                                  |
| ------------- | -------- | ------------------------------------- |
| **构建/运行** | 高/中/低 | [例如：CLI 参数变更，需更新 Makefile] |
| **数据库**    | 高/中/低 | [例如：包含新迁移脚本]                |
| **代码库**    | 高/中/低 | [例如：核心 User Service 重构]        |

## 💡 建议 (Recommendation)
- **行动**: [立即合并 / 暂缓 / Cherry-pick]
- **后续步骤**:
    1. [步骤 1, 如: 执行 `git merge upstream/main`]
    2. [步骤 2, 如: 更新 `Makefile` 去除旧参数]
    3. [步骤 3, 如: 验证启动]
```
