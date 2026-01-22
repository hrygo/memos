# 原子化提交技能

执行原子化 Git 提交，将变更按功能模块分组提交。

## 执行流程

1. **收集变更文件**：获取所有修改和新增的文件
2. **分类变更**：按功能模块分组变更
3. **预检查**：编译验证、测试验证
4. **生成提交消息**：为每组变更生成规范的提交消息
5. **执行提交**：按顺序执行 git add 和 git commit

## 提交消息格式

```
<type>(<scope>): <subject>

<body>

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

## 类型说明

| 类型 | 说明 | 示例 |
|------|------|------|
| `feat` | 新功能 | 添加 BM25 搜索 |
| `fix` | 缺陷修复 | 修复并发竞态 |
| `refactor` | 重构 | 移除 MySQL 支持 |
| `perf` | 性能优化 | 优化查询缓存 |
| `docs` | 文档 | 更新 API 文档 |
| `test` | 测试 | 添加单元测试 |
| `chore` | 构建/工具 | 更新依赖 |

## 作用域说明

| 作用域 | 说明 |
|--------|------|
| `store` | 数据存储层 |
| `server` | HTTP/gRPC 服务器 |
| `plugin` | 插件系统 |
| `web` | 前端应用 |
| `api` | API 路由 |
| `db` | 数据库实现 |
| `cache` | 缓存层 |
| `runner` | 后台任务 |
| `scheduler` | 调度服务 |

## 原子化原则

每个提交必须：
- ✅ 只包含一个功能或修复
- ✅ 能够独立编译通过
- ✅ 可以独立回滚
- ✅ 提交消息清晰描述变更

## 排除文件

不提交以下文件：
- `.claude/settings.local.json` (本地配置)
- `.env` (环境变量)
- `node_modules/`, `dist/`, `bin/` (构建产物)
- `*.log`, `*.tmp` (临时文件)

## 使用示例

```bash
# 执行原子化提交
/commit

# 带参数
/commit "feat(store): add BM25 search"
```
