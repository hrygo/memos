---
description: A robust workflow to analyze upstream changes and generate an "Upstream Alignment Spec". Includes state management to track analyzed commits and supports incremental analysis.
---

# Git Upstream Alignment Analysis Workflow (Incremental)

This workflow helps you plan the synchronization of your local codebase with the upstream repository (`https://github.com/usememos/memos`). It uses a state file to track previously analyzed commits, ensuring you only review new changes ("Pragmatic Filter").

**Output Language**: Chinese (Simplified)

## 1. Setup & State Management

Initialize environment and determine the analysis range.

1.  **Configure Temporary Remote**:
    *Use a temporary remote to avoid polluting local config.*
    ```bash
    REMOTE_NAME="_agent_upstream_temp"
    git remote add $REMOTE_NAME https://github.com/usememos/memos 2>/dev/null || true
    git fetch $REMOTE_NAME --tags --force
    ```

2.  **Determine Analysis Baseline (`START_POINT`)**:
    *We use a local state file to remember the last analyzed commit.*
    ```bash
    STATE_FILE=".agent/upstream-sync-state"
    REMOTE_NAME="_agent_upstream_temp"
    TARGET="$REMOTE_NAME/main"
    
    if [ -f "$STATE_FILE" ]; then
        START_POINT=$(cat "$STATE_FILE")
        echo "🔄 Found last analyzed commit: $START_POINT"
    else
        # Optimize: Find the common ancestor as the baseline
        START_POINT=$(git merge-base HEAD $TARGET)
        echo "🆕 No previous state found. Using common ancestor ($START_POINT) as baseline."
    fi
    ```

## 2. Divergence Check (Quick Scan)

Check the volume of new work.

1.  **Check Commit Count**:
    ```bash
    COUNT=$(git rev-list --count $START_POINT..$TARGET)
    echo "📉 New commits to analyze: $COUNT"
    ```
    *If count is 0, you are up-to-date.*

2.  **Identify Versions**:
    ```bash
    echo "Base: $START_POINT"
    echo "Target: $(git describe --tags $TARGET)"
    ```

## 3. Structured Impact Analysis

Scan `START_POINT..TARGET` and categorize changes for the spec.

1.  **🚨 Breaking Changes (Detailed)**:
    *Shows commit message + file stats to assess impact structure.*
    ```bash
    echo "### 🛑 Breaking Changes"
    git log --no-merges --grep="!" --grep="BREAKING CHANGE" --format="#### %h %s%n- **Author**: %an%n- **Date**: %cd%n%n%b" $START_POINT..$TARGET | head -n 30
    
    # Auto-stat key breaking commits
    for commit in $(git log --no-merges --grep="!" --grep="BREAKING CHANGE" --format="%h" $START_POINT..$TARGET | head -n 5); do
        echo "Stats for $commit:"
        git show --stat $commit | head -n 10
        echo "..."
    done
    ```

2.  **✨ New Features (Markdown List)**:
    ```bash
    echo "### ✨ Features"
    git log --no-merges --grep="feat" --format="- **%h** %s" $START_POINT..$TARGET \
        -- . ":!.github" ":!docs" ":!web/src/locales" ":!*.md" | head -n 20
    ```

3.  **🐛 Core Fixes**:
    ```bash
    echo "### 🐛 Fixes"
    git log --no-merges --grep="fix" --format="- **%h** %s" $START_POINT..$TARGET \
        -- . ":!.github" ":!docs" ":!web/src/locales" ":!*.md" | head -n 20
    ```

4.  **⚡ Performance & Refactor**:
    ```bash
    echo "### ⚡ Perf & Refactor"
    git log --no-merges --grep="perf" --grep="refactor" --format="- **%h** %s" $START_POINT..$TARGET \
        -- . ":!.github" ":!docs" ":!web/src/locales" ":!*.md" | head -n 20
    ```

## 4. Infrastructure & Data Check

1.  **Critical Files**:
    ```bash
    echo "### 🏗️ Infrastructure Changes"
    git diff --stat $START_POINT..$TARGET -- go.mod package.json store/migration proto/
    ```

## 5. Reasoning Framework (The Filter)

**Core Principle: Pragmatism (务求实效)**.
Evaluate every identified change against these criteria:

### ✅ High Priority (Adopt)
*   **Substantial Value**: Solves a known bug or adds a requested feature.
*   **Major Optimization**: Measurable performance gain or security fix.
*   **UX Enhancement**: Clear improvement to user journey.

### ⚠️ Low Priority (Evaluate)
*   **Refactoring**: Adopt only if it simplifies future work.
*   **Minor Fixes**: Skip if the issue doesn't affect your use case.

### 🚫 Ignore (Noise)
*   **Telemetry**: Usage tracking code.
*   **Sponsor/Commercial**: Marketing UI/logic.
*   **Internal Tooling**: CI/CD for upstream org.

## 6. Output Generation & State Update

1.  **Generate Spec**:
    Create `docs/specs/sync-[feature]-[date].md` using the template below.

2.  **Update State (After Analysis)**:
    *Run this ONLY after you have successfully generated the specs to "mark as read".*
    ```bash
    git rev-parse $TARGET > .agent/upstream-sync-state
    echo "✅ Analysis state updated to $(cat .agent/upstream-sync-state)"
    ```
    
## 7. Cleanup

Remove the temporary remote to restore the environment.

```bash
git remote remove _agent_upstream_temp 2>/dev/null
echo "🧹 Temporary remote '_agent_upstream_temp' removed."
```

**Template**:

```markdown
# 📋 上游功能逆向与追齐规范 (Upstream Feature Specs)

**分析时间**: `[YYYY-MM-DD]`
**分析范围**: `[Start Commit]` -> `[Target Commit]`

## 🎯 目标摘要
[简述本次需要移植的核心功能或修复。]

## 🚫 排除项 (Ignored)
[列出本次分析中明确跳过的模块]

## 🗺️ 功能逆向与实现细则

### 1. 🛑 架构与破坏性变更 (Critical / Breaking)
#### [变更名称]
- **来源**: [Commit Hash]
- **必要性**: ⭐⭐⭐⭐⭐
- **原理解析 (Reverse Engineering)**:
    - [分析上游变更的本质：例如将 `HOST` 角色移除，数据迁移至 `ADMIN`]
    - [关键变动文件]: `proto/api/v1/user_service.proto`, `store/migrator.go`
- **本地移植规范**:
    - [指导如何在本地代码库重现此变更，例如：编写新的迁移脚本，修改鉴权中间件]

### 2. ⚡ 核心逻辑与修复 (High Priority)
#### [名称]
- **来源**: [Commit Hash]
- **必要性**: ⭐⭐⭐⭐
- **逻辑分析**:
    - [描述 Bug 原因及上游修复方案]
    - [关键代码片段/差异分析]

### 3. ✨ 有价值的新特性 (Features)
#### [名称]
- **来源**: [Commit Hash]
- **实现机制**:
    - [UI 层]: [新增组件/路由]
    - [API 层]: [新增接口/字段]
    - [数据层]: [Schema 变更]
- **移植建议**:
    - [如何适配本地架构]

## ✅ 验证清单
- [ ] 逻辑正确移植
- [ ] 无引入冗余代码
- [ ] 通过相关测试
```
