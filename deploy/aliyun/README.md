# Memos 单机部署指南 (2C2G)

适用于单台 2核2G 服务器的 Memos 生产环境部署方案。

---

## 架构概述

```
┌─────────────────────────────────────────────────┐
│              2C2G 服务器                         │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │           Docker Network                 │  │
│  │                                          │  │
│  │  ┌──────────────┐  ┌─────────────────┐  │  │
│  │  │  PostgreSQL  │  │     Memos      │  │  │
│  │  │  pg16+vector │  │  1核 / 1G       │  │  │
│  │  │  1核 / 512M  │──│  :5230 ────────►│───┼──► 公网
│  │  │  :5432       │  │                 │  │  │
│  │  └──────────────┘  └─────────────────┘  │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  数据卷: postgres-data, memos-data              │
└─────────────────────────────────────────────────┘
```

**资源分配 (2C2G)**

| 服务 | CPU | 内存 | 说明 |
|------|-----|------|------|
| PostgreSQL | 1核 | 512M | 数据库 |
| Memos | 1核 | 1G | 应用服务 |
| 系统预留 | - | 512M | OS + Docker |

---

## 快速开始

### 1. 上传部署文件

```bash
# 上传到服务器
scp -r deploy/aliyun user@your-server:/root/memos-deploy
scp -r docker/compose user@your-server:/root/memos-deploy/docker/
scp -r store/migration user@your-server:/root/memos-deploy/store/

# SSH 登录
ssh user@your-server
cd /root/memos-deploy
```

### 2. 配置环境变量

```bash
cp .env.prod.example .env.prod
vi .env.prod  # 修改密码和 API Keys
```

**必填配置** (详见 `.env.prod.example` 内说明):

```bash
POSTGRES_PASSWORD=your_secure_password        # 数据库密码
MEMOS_INSTANCE_URL=http://your-server-ip:5230 # 公网地址
MEMOS_AI_SILICONFLOW_API_KEY=sk-xxx           # 向量/重排
MEMOS_AI_DEEPSEEK_API_KEY=sk-xxx              # 对话 LLM
```

> **配置方案**: 文件内提供 4 种配置方案 (SiliconFlow+DeepSeek / 纯 SiliconFlow / OpenAI / Ollama 本地)

### 3. 部署

```bash
# 方式 1: 使用预构建镜像 (推荐，无需 Go/Node.js)
./deploy.sh pull       # 拉取镜像
./deploy.sh deploy     # 部署

# 方式 2: 本地构建 (需 Go 1.25+ / pnpm)
./deploy.sh build      # 构建镜像
./deploy.sh deploy     # 部署
```

### 4. 验证

```bash
./deploy.sh status     # 查看服务状态
./deploy.sh logs       # 查看日志
```

浏览器访问: `http://your-server-ip:5230`

---

## 国内用户部署 (阿里云/腾讯云)

### Docker Hub 访问问题

国内服务器访问 Docker Hub 可能较慢或失败，推荐使用以下方案：

**方案一: 使用预构建镜像 (推荐)**

```bash
# 编辑 .env.prod
vi .env.prod

# 添加以下行 (使用官方预构建镜像)
USER_IMAGE=ghcr.io/usememos/memos:latest

# 部署 (无需 Go/Node.js 环境)
./deploy.sh deploy
```

**方案二: 配置 Docker 镜像加速**

```bash
# 配置国内镜像源
./deploy.sh setup

# 重启 Docker
sudo systemctl restart docker

# 验证配置
docker info | grep -A 10 "Registry Mirrors"
```

**方案三: 本地构建**

如果需要在服务器上构建，确保已安装：
- Go 1.25+
- Node.js 18+ / pnpm

```bash
# 构建并部署
./deploy.sh build && ./deploy.sh deploy
```

### 常见问题

| 问题 | 解决方案 |
|------|----------|
| `pgvector/pgvector:pg16` 拉取失败 | 运行 `./deploy.sh setup` 配置镜像加速 |
| `go mod download` 超时 | Dockerfile 已配置 GOPROXY=goproxy.cn |
| `pnpm install` 超时 | 已配置 .npmrc 使用 npmmirror.com |
| 构建缺少依赖 | 使用预构建镜像替代 |

---

## 从旧方案迁移

如果你的服务器正在运行旧版本部署方案（PostgreSQL Docker + Memos host 二进制），迁移步骤如下：

### 迁移前检查

```bash
# 1. 检查当前数据库版本
docker exec memos-postgres psql -U memos -d memos -c "SELECT value FROM system_setting WHERE name = 'schema_version';"

# 2. 备份现有数据库
docker exec memos-postgres pg_dump -U memos memos | gzip > memos-backup-pre-migration.sql.gz
```

### 迁移步骤

```bash
# 1. 停止旧的 Memos 进程
sudo systemctl stop memos  # 或其他方式停止

# 2. 更新部署文件
cp docker/compose/prod.yml /root/memos-deploy/
cp -r store/migration/postgres /root/memos-deploy/

# 3. 更新 .env.prod，添加新配置
vi .env.prod

# 4. 使用新版部署脚本
./deploy.sh upgrade
```

### 迁移后验证

```bash
# 1. 检查服务状态
./deploy.sh status

# 2. 验证数据库连接
docker exec memos psql -U memos -d memos -c "SELECT 1;"

# 3. 检查数据完整性
docker exec memos psql -U memos -d memos -c "SELECT COUNT(*) FROM memo;"
```

### 外部数据库连接

如果需要使用 pgAdmin / DataGrip 等工具连接数据库：

```bash
# 编辑 .env.prod，取消注释以下行
POSTGRES_PORT_MAPPING=127.0.0.1:25432:5432

# 重启服务
./deploy.sh restart

# 连接信息
# Host: 127.0.0.1
# Port: 25432
# Database: memos
# User: memos
```

---

## 运维操作

### 常用命令

```bash
./deploy.sh deploy    # 首次部署
./deploy.sh upgrade   # 升级版本
./deploy.sh restart   # 重启服务
./deploy.sh stop      # 停止服务
./deploy.sh status    # 查看状态
./deploy.sh logs      # 查看日志
./deploy.sh version   # 查看版本
```

### 备份恢复

```bash
./deploy.sh backup                     # 手动备份
./deploy.sh restore backups/xxx.gz     # 恢复备份
./deploy.sh cleanup                    # 清理 7 天前备份
```

---

## 版本管理

### 目录结构

```
store/migration/postgres/
├── VERSION           # 当前代码版本 (0.52.0)
├── LATEST.sql        # 全量 Schema (首次部署使用)
└── V*.sql            # 增量迁移脚本 (升级使用)
```

### 首次部署

PostgreSQL 容器首次启动时自动执行 `LATEST.sql`，初始化数据库并写入版本号 `0.52.0`。

### 版本升级

1. **创建迁移脚本** - 放在 `store/migration/postgres/V{version}__{feature}.sql`
2. **更新 VERSION** - `echo "0.52.0" > store/migration/postgres/VERSION`
3. **执行升级** - `./deploy.sh upgrade`

升级流程：
1. 自动备份数据库
2. 执行增量迁移脚本 (按版本号排序)
3. 重新构建镜像
4. 重启服务

---

## 文件说明

```
deploy/aliyun/
├── .env.prod.example          # 环境变量模板 (含 4 种配置方案)
├── deploy.sh                  # 部署脚本
└── README.md                  # 本文档
```

```
docker/compose/
├── dev.yml                    # 开发环境
├── prod.yml                   # 生产环境 (PG + Memos)
└── quick.yml                  # SQLite 快速启动
```

```
store/migration/postgres/
├── VERSION                   # 当前版本
├── LATEST.sql                # 全量初始化
└── V*.sql                    # 增量迁移
```

---

## 故障排查

### 服务无法启动

```bash
./deploy.sh logs    # 查看日志
docker stats        # 查看资源使用
```

### 数据库问题

```bash
# 检查 pgvector 扩展
docker exec memos-postgres psql -U memos -d memos -c "SELECT extname FROM pg_extension WHERE extname = 'vector';"

# 查看数据库版本
docker exec memos-postgres psql -U memos -d memos -c "SELECT value FROM system_setting WHERE name = 'schema_version';"
```

### 回滚

```bash
./deploy.sh restore backups/memos-backup-xxx.gz
./deploy.sh restart
```

---

## 定时备份

使用 cron 定时备份：

```bash
crontab -e

# 每天凌晨 2 点备份
0 2 * * * cd /root/memos-deploy && ./deploy.sh backup && ./deploy.sh cleanup
```

---

## 安全建议

1. **网络安全** - 只开放 22, 80, 443 端口
2. **数据安全** - 定期备份，使用强密码
3. **访问控制** - 配置防火墙规则
4. **更新** - 定期更新镜像和依赖
