# Docker 开发环境

Memos 提供多种 Docker 开发方案，适应不同的开发需求。

## 方案对比

| 方案 | PostgreSQL | 后端 | 前端 | 适用场景 |
|------|-----------|------|------|---------|
| **全 Docker** | ✅ Docker | ✅ Docker | ✅ Docker | 无需本地安装 Go/Node，快速开始 |
| **半 Docker** | ✅ Docker | ❌ 本地 | ❌ 本地 | 需要本地调试前后端 |
| **生产 Docker** | ✅ Docker | ✅ Docker | ❌ 构建 | 生产部署 |

## 全 Docker 开发环境 (推荐)

### 特点

- **零依赖**: 无需本地安装 Go、Node.js、pnpm
- **热重载**: 后端使用 Air，前端使用 Vite HMR
- **一键启动**: 所有服务通过一个命令启动

### 使用方法

```bash
# 启动所有服务
make docker-full-up

# 查看日志
make docker-full-logs

# 查看特定服务日志
make docker-full-logs backend
make docker-full-logs frontend
make docker-full-logs postgres

# 查看容器状态
make docker-full-ps

# 进入容器调试
make docker-full-exec-backend    # 进入后端容器
make docker-full-exec-frontend   # 进入前端容器
make docker-full-exec-postgres   # 连接 PostgreSQL

# 停止所有服务
make docker-full-down

# 重新构建并启动
make docker-full-rebuild
```

### 服务地址

| 服务 | 地址 |
|------|------|
| 前端 | http://localhost:5173 |
| 后端 API | http://localhost:8081 |
| PostgreSQL | localhost:5432 |

### 目录结构

```
docker/
├── compose/
│   ├── dev.yml        # 仅 PostgreSQL (半 Docker)
│   ├── full-dev.yml   # 全 Docker 开发环境
│   └── prod.yml       # 生产环境
└── dev/
    ├── Dockerfile.backend   # 后端开发镜像
    ├── Dockerfile.frontend  # 前端开发镜像
    └── .air.toml           # Air 热重载配置
```

## 环境变量配置

创建 `.env` 文件配置 AI 功能：

```bash
cp .env.example .env
```

编辑 `.env`：

```bash
# AI 功能开关
MEMOS_AI_ENABLED=true

# Embedding 配置
MEMOS_AI_EMBEDDING_PROVIDER=siliconflow
MEMOS_AI_SILICONFLOW_API_KEY=your_api_key_here
MEMOS_AI_EMBEDDING_MODEL=BAAI/bge-m3

# LLM 配置
MEMOS_AI_LLM_PROVIDER=deepseek
MEMOS_AI_DEEPSEEK_API_KEY=your_api_key_here
MEMOS_AI_LLM_MODEL=deepseek-chat
```

## 常见问题

### 后端热重载不生效？

检查 `docker/dev/.air.toml` 配置，确保监视的目录正确。

### 前端 HMR 不生效？

确保 Vite 开发服务器配置了 `--host 0.0.0.0`，这已在 `full-dev.yml` 中配置。

### 容器构建失败？

```bash
# 清理并重新构建
docker compose -f docker/compose/full-dev.yml down
docker compose -f docker/compose/full-dev.yml build --no-cache
docker compose -f docker/compose/full-dev.yml up -d
```

### 查看 Docker 资源占用

```bash
docker stats
```

## 半 Docker 开发环境

如果需要本地调试，可以使用半 Docker 方案：

```bash
# 仅启动 PostgreSQL
make docker-up

# 本地启动后端
make run

# 本地启动前端
make web
```

## 生产部署

```bash
# 启动生产环境
make docker-prod-up

# 查看日志
make docker-prod-logs

# 停止
make docker-prod-down
```
