# Contributing to Memos

感谢你对 Memos 项目的贡献！请遵循以下规范。

---

## 开发规范

### 分支策略

```
main          # 主分支，保持稳定可发布状态
feature/*     # 功能开发分支
fix/*         # 缺陷修复分支
refactor/*    # 重构分支
```

### 提交规范

#### 原子化提交原则

每个提交必须是一个**独立的、可验证的逻辑单元**：

| ✅ 好的提交 | ❌ 不好的提交 |
|------------|--------------|
| `feat(store): add BM25 search` | `feat: add a lot of features` |
| `fix(server): resolve race condition` | `fix: fix bugs and update docs` |
| `refactor(api): remove MySQL dialect` | `update: various files` |

#### 提交消息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型 (type):**
- `feat`: 新功能
- `fix`: 缺陷修复
- `refactor`: 重构（既不是新功能也不是修复）
- `perf`: 性能优化
- `docs`: 文档变更
- `test`: 测试相关
- `chore`: 构建/工具变更
- `style`: 代码格式（不影响逻辑）

**作用域 (scope):**
- `store`, `server`, `plugin`, `web`, `api`, `db`, `cache` 等

**示例:**
```
feat(store): add BM25 full-text search support

Add BM25Search interface and implementations for PostgreSQL and SQLite.

- Add BM25SearchOptions and BM25Result to store interface
- Implement BM25Search using PostgreSQL's ts_rank
- Add input validation for search options

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

---

## 代码规范

### Go 代码

- 遵循 [Effective Go](https://go.dev/doc/effective_go)
- 使用 `gofmt` 格式化
- 运行 `golangci-lint` 检查
- 添加单元测试覆盖率 > 80%

### React/TypeScript 代码

- 使用 TypeScript 严格模式
- 遵循 React Hooks 规则
- 运行 `pnpm lint` 检查

---

## 测试规范

```bash
# 运行所有测试
make test

# 运行指定包测试
go test ./store/...

# 运行前端测试
cd web && pnpm test
```

---

## Pull Request 流程

1. 创建功能分支：`git checkout -b feature/your-feature`
2. 开发并原子化提交
3. 推送到远程：`git push origin feature/your-feature`
4. 创建 Pull Request
5. 等待 Code Review
6. 根据反馈修改
7. 合并到主分支

---

## 数据库变更

- 添加迁移文件到 `store/migration/postgres/<version>/`
- 包含 up 和 down 迁移脚本
- 更新 `LATEST.sql` 引用
- 测试迁移和回滚
