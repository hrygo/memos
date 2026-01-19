# Memos 项目目录结构规范

## 目录结构

```
memos/
├── .github/              # GitHub 相关配置
│   └── workflows/        # CI/CD 工作流
│
├── bin/                  # 编译后的二进制文件 (gitignore)
│
├── build/                # 构建缓存目录
│
├── cmd/                  # 命令行入口
│   ├── memos/            # 主程序
│   └── test-ai/          # AI 测试工具
│
├── docker/               # Docker 相关配置
│   ├── Dockerfile        # 生产环境镜像构建
│   └── compose/          # Docker Compose 文件
│       ├── dev.yml       # 开发环境 (PostgreSQL + pgvector)
│       ├── prod.yml      # 生产环境 (PostgreSQL + Memos)
│       └── quick.yml     # 快速体验 (官方镜像)
│
├── docs/                 # 项目文档
│   └── specs/            # 功能规格文档
│
├── internal/             # 内部包 (不对外暴露)
│   ├── base/             # 基础类型和接口
│   ├── profile/          # 配置管理
│   ├── util/             # 工具函数
│   └── version/          # 版本信息
│
├── plugin/               # 插件系统
│   ├── ai/               # AI 功能 (Embedding, LLM, Reranker)
│   ├── cron/             # 定时任务
│   ├── email/            # 邮件发送
│   ├── filter/           # 内容过滤
│   ├── httpgetter/       # HTTP 请求
│   ├── idp/              # 身份提供商
│   ├── markdown/         # Markdown 解析
│   ├── scheduler/        # 任务调度
│   ├── storage/          # 存储适配器
│   └── webhook/          # Webhook
│
├── proto/                # Protobuf 定义
│   ├── api/              # API 接口定义
│   ├── gen/              # 生成的代码 (gitignore)
│   └── store/            # 存储层定义
│
├── scripts/              # 开发和构建脚本
│   ├── build.sh          # 构建脚本
│   ├── dev.sh            # 开发环境管理 (启动/停止/日志)
│   ├── entrypoint.sh     # 容器入口脚本
│   └── entrypoint_test.sh # 测试入口脚本
│
├── server/               # 服务器代码
│   ├── auth/             # 认证授权
│   ├── runner/           # 后台任务运行器
│   │   └── embedding/    # 向量嵌入任务
│   └── router/           # HTTP 路由
│       ├── api/          # API 处理器
│       └── v1/           # v1 API 实现
│
├── store/                # 数据存储层
│   ├── db/               # 数据库实现
│   │   ├── mysql/        # MySQL 实现
│   │   ├── postgres/      # PostgreSQL 实现
│   │   └── sqlite/        # SQLite 实现
│   └── test/             # 测试工具
│
├── web/                  # 前端项目
│   ├── dist/             # 构建产物 (gitignore)
│   ├── docs/             # 前端文档
│   ├── node_modules/     # 依赖 (gitignore)
│   ├── public/           # 静态资源
│   └── src/              # 源代码
│
├── .env                  # 环境变量 (gitignore)
├── .env.example          # 环境变量示例
├── .envrc                # direnv 配置 (gitignore)
├── .gitignore            # Git 忽略配置
├── .golangci.yaml        # Go 代码检查配置
├── .logs/                # 运行时日志 (gitignore)
├── .pids/                # 运行时 PID (gitignore)
├── AGENTS.md             # AI Agent 相关文档
├── go.mod                # Go 模块定义
├── go.sum                # Go 依赖锁定
├── LICENSE               # 许可证
├── Makefile              # 构建和开发命令
├── README.md             # 项目说明
└── SECURITY.md           # 安全策略
```

## 命名规范

### 目录命名
- **小写 + 连字符**: `docker/`, `web/`, `node_modules/`
- **复数形式**: `scripts/`, `docs/`, `workflows/`

### 文件命名
- **Go 源文件**: `snake_case.go` (如: `memo_embedding.go`)
- **脚本文件**: `kebab-case.sh` (如: `dev.sh`, `build.sh`)
- **配置文件**: `kebab-case.yml` (如: `dev.yml`, `prod.yml`)
- **文档文件**: `UPPER_CASE.md` (如: `README.md`, `AGENTS.md`)

### Go 包命名
- **简单小写**: `package store`, `package server`
- **不要下划线**: ✅ `plugin/ai`, ❌ `plugin/ai_service`

## 添加新功能的规范

### 1. 新增 API 端点
```
server/router/api/
  ├── v1/              # 实现
  │   └── ai_service.go
  └── ...
```

### 2. 新增插件
```
plugin/
  └── your-plugin/      # 新插件目录
      ├── plugin.go     # 插件入口
      └── ...
```

### 3. 新增存储层
```
store/
  ├── db/
  │   ├── postgres/
  │   │   └── your_feature.go  # PostgreSQL 实现
  │   ├── mysql/
  │   │   └── your_feature.go  # MySQL 实现
  │   └── sqlite/
  │       └── your_feature.go  # SQLite 实现
  └── your_feature.go          # 接口定义
```

### 4. 新增前端组件
```
web/src/
  ├── components/
  │   └── YourComponent.tsx
  ├── hooks/
  │   └── useYourFeature.ts
  └── pages/
      └── your-page.tsx
```

## Docker 相关规范

所有 Docker 相关文件统一放在 `docker/` 目录：

| 文件/目录 | 位置 | 用途 |
|----------|------|------|
| Dockerfile | `docker/Dockerfile` | 生产镜像构建 |
| Compose 文件 | `docker/compose/*.yml` | 容器编排配置 |
| 容器脚本 | `scripts/entrypoint.sh` | 容器内执行脚本 |

### Compose 文件命名
- `dev.yml` - 开发环境
- `prod.yml` - 生产环境
- `test.yml` - 测试环境
- `quick.yml` - 快速体验/演示

## 运行时文件规范

所有运行时生成的文件应该在 `.gitignore` 中：

| 目录 | 用途 |
|------|------|
| `.logs/` | 服务日志输出 |
| `.pids/` | 后台进程 PID 文件 |
| `bin/` | 编译后的二进制 |
| `build/` | 构建缓存 |

## 环境变量规范

### 环境变量命名
- 统一前缀: `MEMOS_`
- 分组命名: `MEMOS_AI_*`, `MEMOS_DRIVER`, `MEMOS_DSN`

### 配置优先级
1. 系统环境变量 (direnv)
2. `.env` 文件
3. 代码默认值
