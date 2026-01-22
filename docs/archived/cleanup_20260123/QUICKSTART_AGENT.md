# 🚀 日程智能体 - 快速开始

## 3 种测试方式

### 方式 1️⃣: 交互式测试脚本（最简单）

```bash
# 1. 启动服务
make start

# 2. 运行测试脚本
chmod +x scripts/test_schedule_agent.sh
./scripts/test_schedule_agent.sh
```

脚本会引导你：
- 检查环境配置
- 验证服务状态
- 选择测试项目
- 查看实时结果

---

### 方式 2️⃣: Go 测试程序（推荐）

```bash
# 1. 确保数据库运行
make docker-up

# 2. 配置 .env 文件
cat >> .env << 'EOF'
MEMOS_AI_ENABLED=true
MEMOS_AI_LLM_PROVIDER=deepseek
MEMOS_AI_LLM_MODEL=deepseek-chat
MEMOS_AI_DEEPSEEK_API_KEY=your_key_here
EOF

# 3. 运行测试程序
go run ./cmd/test-agent/main.go
```

测试程序会自动执行：
- ✅ 查询明天的日程
- ✅ 创建新日程
- ✅ 查询本周日程

并显示：
- 📊 执行过程（思考、工具调用）
- ⏱️ 响应时间
- 📝 最终结果

---

### 方式 3️⃣: 手动 API 测试

#### 步骤 1: 启动服务

```bash
# 启动所有服务
make start

# 或分别启动
make docker-up  # 数据库
make run        # 后端（新终端）
make web       # 前端（新终端）
```

#### 步骤 2: 获取 Token

```bash
# 登录获取 token
curl -X POST http://localhost:28081/api/v1/auth/signin \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your_username",
    "password": "your_password"
  }'
```

保存返回的 `data.access_token`

#### 步骤 3: 测试 API

```bash
# 设置 token
export TOKEN="your_access_token_here"

# 测试 1: 查询日程
curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "查看明天有什么安排",
    "user_timezone": "Asia/Shanghai"
  }'

# 测试 2: 创建日程
curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "后天上午10点开个产品会",
    "user_timezone": "Asia/Shanghai"
  }'
```

---

## 📊 验证清单

### 基础验证

```bash
# 1. 数据库连接
make db-connect
# 应该进入 psql shell

# 2. 后端服务
curl http://localhost:28081
# 应该返回 404 或 API 信息

# 3. AI 功能
curl http://localhost:28081/api/v1/status
# 检查 ai.enabled 是否为 true
```

### 智能体验证

#### 测试查询
```
输入: "明天有什么安排？"
预期: 返回日程列表或"暂无日程"
```

#### 测试创建
```
输入: "后天下午2点开个会"
预期:
  - 如果无冲突: "成功创建日程..."
  - 如果有冲突: "发现冲突..."
```

#### 测试周期性日程
```
输入: "每周一下午2点开例会"
预期: 成功创建周期性日程
```

---

## 🐛 常见问题

### ❌ "AI features are disabled"

```bash
# 检查环境变量
echo $MEMOS_AI_ENABLED

# 修复
echo "MEMOS_AI_ENABLED=true" >> .env
make stop && make start
```

### ❌ "Failed to create LLM service"

```bash
# 检查配置
cat .env | grep AI

# 确保 API key 正确
echo $MEMOS_AI_DEEPSEEK_API_KEY
```

### ❌ "Database connection failed"

```bash
# 检查数据库
make docker-up
make db-connect

# 重置数据库（如果需要）
make db-reset
```

### ❌ "Token invalid"

```bash
# 重新登录获取新 token
curl -X POST http://localhost:28081/api/v1/auth/signin \
  -H "Content-Type: application/json" \
  -d '{"username":"your_username","password":"your_password"}'
```

---

## 📝 查看日志

```bash
# 实时查看所有日志
make logs

# 只查看后端日志
make logs-follow-backend

# 过滤智能体相关日志
make logs-follow-backend | grep -i "agent\|schedule"
```

---

## 🧪 运行单元测试

```bash
# 测试 Service 层
go test ./server/service/schedule/... -v

# 测试工具层
go test ./plugin/ai/agent/tools/... -v

# 测试智能体
go test ./plugin/ai/agent/... -v

# 查看覆盖率
go test ./server/service/schedule/... -cover
```

---

## 🎯 下一步

### 验证完成后

1. **查看结果**
   ```bash
   # 直接查询数据库
   make db-connect

   # 在 psql 中运行
   SELECT id, title, start_ts, end_ts
   FROM schedules
   ORDER BY created_ts DESC
   LIMIT 5;
   ```

2. **测试前端**
   - 打开 http://localhost:25173
   - 进入 AI Chat
   - 尝试相同的查询

3. **性能调优**
   - 测量响应时间
   - 优化 prompt
   - 调整迭代限制

---

## 📚 更多文档

- [完整实施报告](docs/agent_architecture/agent_scheduler/COMPLETION_REPORT.md)
- [手动执行详细指南](docs/agent_architecture/agent_scheduler/MANUAL_EXECUTION_GUIDE.md)
- [架构设计文档](docs/agent_architecture/RP_001_schedule_agent_refactor.md)
