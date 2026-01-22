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

## 国际化 (i18n) 规范 ⚠️ 强制执行

### 基本原则

**所有新增的 UI 文本必须同时提供英文和中文翻译。**

这是项目的强制规约，违反此规约的代码将不被接受。

### 双语要求

| 文件 | 强制要求 |
|------|----------|
| `en.json` | 必须 - 英文翻译 |
| `zh-Hans.json` | 必须 - 简体中文翻译 |
| `zh-Hant.json` | 可选 - 繁体中文翻译 |
| 其他语言文件 | 可选 - 社区贡献 |

### 添加新文本的步骤

1. **在 `en.json` 中添加英文翻译**
   ```json
   {
     "your": {
       "new": {
         "key": "Your new feature text",
         "description": "Description here"
       }
     }
   }
   ```

2. **在 `zh-Hans.json` 中添加中文翻译**
   ```json
   {
     "your": {
       "new": {
         "key": "您的新功能文本",
         "description": "描述在这里"
       }
     }
   }
   ```

3. **验证 key 完整性**
   ```bash
   make check-i18n
   ```

### 命名规范

- **使用小写字母和连字符**: `your-feature-name`
- **使用点号分隔命名空间**: `common.save`, `schedule.title`
- **key 应该有意义**: 避免使用 `text1`, `label2`
- **保持一致性**: 相同概念使用相同的 key

### 在代码中使用

```tsx
import { useTranslate } from "@/utils/i18n";

const Component = () => {
  const t = useTranslate();

  return (
    <button>{t("common.save")}</button>
  );
};
```

**禁止硬编码文本**:
```tsx
// ❌ 错误
<button>Save</button>

// ✅ 正确
<button>{t("common.save")}</button>
```

### 检查命令

```bash
# 检查 i18n key 是否同步
make check-i18n

# 检查前端代码是否有硬编码文本
make check-i18n-hardcode
```

### 提交前检查清单

- [ ] `en.json` 和 `zh-Hans.json` 都添加了新的 key
- [ ] key 的路径结构一致
- [ ] 运行 `make check-i18n` 无错误
- [ ] 运行 `make check-i18n-hardcode` 无警告

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
