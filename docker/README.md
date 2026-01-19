# Docker 配置

Memos 使用 Docker 运行 PostgreSQL 数据库。

## 目录结构

```
docker/
├── compose/
│   ├── dev.yml        # 开发环境 (PostgreSQL)
│   ├── prod.yml       # 生产环境 (PostgreSQL)
│   └── quick.yml      # 快速启动 (SQLite)
└── Dockerfile         # 生产环境镜像构建
```

## 开发环境

```bash
# 启动 PostgreSQL
make docker-up

# 查看日志
make docker-logs

# 停止
make docker-down

# 重置数据 (危险!)
make docker-reset
```

## 生产环境

生产环境使用 PostgreSQL + 二进制部署：

```bash
# 启动生产数据库
make docker-prod-up

# 查看日志
make docker-prod-logs

# 停止
make docker-prod-down
```

## 数据库连接

```bash
# 连接到 PostgreSQL
make db-connect

# 或直接使用 docker
docker exec -it memos-postgres-dev psql -U memos -d memos
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| POSTGRES_DB | memos | 数据库名 |
| POSTGRES_USER | memos | 用户名 |
| POSTGRES_PASSWORD | memos | 密码 |
