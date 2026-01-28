# DivineSense 单机部署指南 (2C2G)

适用于阿里云/腾讯云 2核2G 服务器的生产环境部署方案。

---

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh | bash
```

**自动完成：**
- ✅ 安装 Docker + Docker Compose
- ✅ 配置国内镜像加速
- ✅ 下载 DivineSense 镜像
- ✅ 生成安全密码
- ✅ 初始化 PostgreSQL + pgvector
- ✅ 启动服务
- ✅ 配置防火墙
- ✅ 设置每日自动备份

**安装完成后：**

1. 配置 AI API Keys：
```bash
vi /opt/divinesense/.env.prod

# 修改以下两项：
DIVINESENSE_AI_SILICONFLOW_API_KEY=sk-xxx
DIVINESENSE_AI_DEEPSEEK_API_KEY=sk-xxx

# 重启服务
cd /opt/divinesense && ./deploy.sh restart
```

2. 访问服务：`http://your-server-ip:5230`

---

## 架构

```
┌─────────────────────────────────────────────────┐
│              2C2G 服务器                         │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │           Docker Network                 │  │
│  │                                          │  │
│  │  ┌──────────────┐  ┌─────────────────┐  │  │
│  │  │  PostgreSQL  │  │   DivineSense   │  │  │
│  │  │  pg16+vector │  │   0.75核/800M  │  │  │
│  │  │ 0.75核/400M  │──│  :5230 ────────►│───┼──► 公网
│  │  │  :5432       │  │                 │  │  │
│  │  └──────────────┘  └─────────────────┘  │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  数据卷: postgres_data, divinesense_data        │
└─────────────────────────────────────────────────┘
```

**资源分配 (2C2G 优化)**

| 服务 | CPU | 内存 | 说明 |
|------|-----|------|------|
| PostgreSQL | 0.75核 | 400M | 数据库 |
| DivineSense | 0.75核 | 800M | 应用服务 |
| 系统预留 | 0.5核 | 512M | OS + Docker |

---

## AI 配置

DivineSense 需要 2 个 API Key（国内推荐）：

| API Key | 用途 | 获取地址 |
|---------|------|----------|
| SiliconFlow | 向量/重排/意图分类 | https://cloud.siliconflow.cn |
| DeepSeek | 对话 LLM | https://platform.deepseek.com |

**其他方案：**
- 纯 SiliconFlow（单一供应商）
- OpenAI（海外用户）
- Ollama（本地离线）

详见 `.env.prod` 文件内注释。

---

## 运维命令

```bash
cd /opt/divinesense

./deploy.sh status     # 查看状态
./deploy.sh logs       # 查看日志
./deploy.sh restart    # 重启服务
./deploy.sh stop       # 停止服务
./deploy.sh backup     # 手动备份
./deploy.sh upgrade    # 升级版本
```

---

## 备份

**自动备份：** 每天凌晨 2 点（安装时已配置）

**手动备份：**
```bash
cd /opt/divinesense && ./deploy.sh backup
```

**恢复备份：**
```bash
cd /opt/divinesense && ./deploy.sh restore backups/divinesense-backup-xxx.gz
```

---

## 常见问题

| 问题 | 解决方案 |
|------|----------|
| 镜像拉取慢 | 一键安装脚本已自动配置国内镜像源 |
| 服务无法启动 | `./deploy.sh logs` 查看日志 |
| 忘记数据库密码 | `cat /opt/divinesense/.db_password` |
| 防火墙问题 | 确保开放 5230 端口 |

---

## 安全建议

1. **修改密码** - 安装后修改数据库密码
2. **备份** - 已配置每日自动备份，建议定期下载到本地
3. **防火墙** - 只开放必要端口 (22, 80, 443, 5230)
4. **HTTPS** - 生产环境建议配置反向代理 + SSL

---

## 文件位置

```
/opt/divinesense/
├── .env.prod          # 环境配置
├── .db_password       # 数据库密码
├── deploy.sh          # 运维脚本
└── backups/           # 备份目录
```
