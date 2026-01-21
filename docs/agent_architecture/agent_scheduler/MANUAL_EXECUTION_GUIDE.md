# 日程智能体 - 手动执行指南

本指南将帮助你手动启动、测试和验证日程智能体功能。

## 📋 前置条件

1. **确保数据库正在运行**
   ```bash
   make docker-up
   ```

2. **配置环境变量**
   创建 `.env` 文件（如果还没有）：
   ```bash
   MEMOS_DRIVER=postgres
   MEMOS_DSN=postgres://memos:memos@localhost:25432/memos?sslmode=disable

   # AI 配置（必需）
   MEMOS_AI_ENABLED=true
   MEMOS_AI_EMBEDDING_PROVIDER=siliconflow
   MEMOS_AI_LLM_PROVIDER=deepseek
   MEMOS_AI_LLM_MODEL=deepseek-chat

   # API Keys（替换为你的实际 key）
   MEMOS_AI_DEEPSEEK_API_KEY=your_deepseek_key
   MEMOS_AI_SILICONFLOW_API_KEY=your_siliconflow_key
   ```

## 🚀 方式 1: 完整启动（推荐）

启动所有服务（PostgreSQL + 后端 + 前端）：

```bash
# 1. 构建最新版本
make build

# 2. 启动所有服务
make start

# 服务将在以下端口运行：
# - 前端: http://localhost:25173
# - 后端: http://localhost:28081
# - PostgreSQL: localhost:25432
```

### 测试步骤

1. **打开前端** - 访问 http://localhost:25173
2. **登录或注册账户**
3. **进入 AI Chat 界面**
4. **测试命令**：
   ```
   查询：下周一我有什么安排？
   创建：明天下午2点开个产品会
   查询：本周有哪些会议？
   ```

## 🔧 方式 2: 分步启动（用于调试）

### 步骤 1: 启动数据库

```bash
make docker-up
```

验证数据库连接：
```bash
make db-connect
# 进入 psql shell 后运行：
SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';
```

### 步骤 2: 启动后端

```bash
# 方式 A: 使用 make
make run

# 方式 B: 直接运行（更灵活）
go run ./cmd/memos --mode dev --port 28081
```

验证后端启动：
```bash
curl http://localhost:28081/memos.api.v1.AIService/ChatWithMemos
```

### 步骤 3: 启动前端（可选）

```bash
cd web
pnpm dev
```

前端将在 http://localhost:25173 启动。

## 🧪 方式 3: 使用 API 直接测试

如果你只想测试智能体 API 而不启动前端：

### 准备工作

1. **获取用户 ID 和 Token**

   先注册/登录获取认证 token：
   ```bash
   curl -X POST http://localhost:28081/api/v1/auth/signin \
     -H "Content-Type: application/json" \
     -d '{
       "username": "your_username",
       "password": "your_password"
     }'
   ```

   保存返回的 `access_token`

2. **创建测试脚本**

   创建 `test_agent.sh`：
   ```bash
   #!/bin/bash

   TOKEN="your_access_token_here"
   API_BASE="http://localhost:28081"

   # 测试 1: 查询日程
   echo "=== 测试 1: 查询明天的日程 ==="
   curl -X POST "$API_BASE/api/v1/ai/chat" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "message": "查看明天有什么安排",
       "user_timezone": "Asia/Shanghai"
     }'

   echo -e "\n\n"

   # 测试 2: 创建日程
   echo "=== 测试 2: 创建新日程 ==="
   curl -X POST "$API_BASE/api/v1/ai/chat" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "message": "后天上午10点开个团队周会",
       "user_timezone": "Asia/Shanghai"
     }'

   echo -e "\n\n"

   # 测试 3: 复杂查询
   echo "=== 测试 3: 本周有哪些日程？ ==="
   curl -X POST "$API_BASE/api/v1/ai/chat" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "message": "本周有哪些日程安排？",
       "user_timezone": "Asia/Shanghai"
     }'
   ```

   赋予执行权限：
   ```bash
   chmod +x test_agent.sh
   ./test_agent.sh
   ```

### 使用 grpcurl 测试 gRPC 端点

安装 grpcurl：
```bash
# macOS
brew install grpcurl

# Linux
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

测试 ChatWithScheduleAgent：
```bash
grpcurl -plaintext \
  -d '{
    "message": "查看明天的日程",
    "user_timezone": "Asia/Shanghai"
  }' \
  -H "Authorization: Bearer your_token" \
  localhost:28081 \
  memos.api.v1.AIService/ChatWithScheduleAgent
```

## 📊 查看日志

### 实时查看所有日志
```bash
make logs
```

### 只查看后端日志
```bash
make logs-backend
```

### 实时跟踪后端日志
```bash
make logs-follow-backend
```

### 查看特定关键词
```bash
make logs-follow-backend | grep -i "agent\|schedule\|llm"
```

## 🐛 调试模式

### 启用详细日志

在 `.env` 中添加：
```bash
# 启用调试日志
LOG_LEVEL=debug

# 或者使用 slog
LOG_LEVEL_DEBUG=true
```

### 使用 Delve 调试器

```bash
# 安装 dlv
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试运行
dlv debug ./cmd/memos -- --mode dev --port 28081
```

### 添加调试输出

在代码中添加：
```go
import "log/slog"

// 在关键位置添加
slog.Info("ScheduleAgent",
    "action", "query",
    "user_input", userInput,
    "iteration", iteration,
)
```

## 🧪 单元测试

### 运行所有测试
```bash
go test ./... -v
```

### 只运行智能体相关测试
```bash
# 测试 Service 层
go test ./server/service/schedule/... -v

# 测试工具层
go test ./plugin/ai/agent/tools/... -v

# 测试智能体
go test ./plugin/ai/agent/... -v
```

### 运行特定测试
```bash
go test ./server/service/schedule/... -run TestFindSchedules -v
go test ./plugin/ai/agent/tools/... -run TestScheduleQueryTool -v
```

### 查看测试覆盖率
```bash
go test ./server/service/schedule/... -cover
go test ./plugin/ai/agent/tools/... -cover
```

## 🌐 使用 Postman 测试

### 1. 导入环境变量
在 Postman 中设置：
- `base_url`: http://localhost:28081
- `token`: 你的 access_token

### 2. 创建请求

**请求 1: 聊天**
```
POST {{base_url}}/api/v1/ai/chat
Authorization: Bearer {{token}}
Content-Type: application/json

{
  "message": "明天下午2点开个会",
  "user_timezone": "Asia/Shanghai"
}
```

**请求 2: 流式聊天（SSE）**
```
POST {{base_url}}/api/v1/ai/chat/stream
Authorization: Bearer {{token}}
Content-Type: application/json

{
  "message": "查看本周日程",
  "user_timezone": "Asia/Shanghai"
}
```

### 3. 保存为 Collection

创建 Postman Collection 并保存以下请求：
1. 登录获取 token
2. 查询日程
3. 创建日程
4. 查询冲突
5. 更新日程

## 📝 验证清单

### 基础功能验证

- [ ] 数据库连接正常
  ```bash
  make db-connect
  ```

- [ ] 后端启动成功
  ```bash
  curl http://localhost:28081
  ```

- [ ] AI 功能已启用
  - 检查环境变量 `MEMOS_AI_ENABLED=true`
  - 验证 API keys 正确配置

### 智能体功能验证

- [ ] 查询日程
  ```
  输入: "明天有什么安排？"
  预期: 返回明天的日程列表
  ```

- [ ] 创建日程
  ```
  输入: "后天下午3点开个会"
  预期: 成功创建日程
  ```

- [ ] 冲突检测
  ```
  输入: "在已有会议的时间创建日程"
  预期: 提示冲突并建议其他时间
  ```

- [ ] 周期性日程
  ```
  输入: "每周一下午2点开例会"
  预期: 创建周期性日程
  ```

### 日志验证

检查日志中是否有以下输出：
```
[ScheduleAgent] Executing with callback
[ScheduleAgent] Iteration: 1
[ScheduleAgent] Tool call: schedule_query
[ScheduleAgent] Tool result: Found 2 schedules
[ScheduleAgent] Iteration: 2
[ScheduleAgent] Final answer generated
```

## ⚠️ 常见问题

### 问题 1: "AI features are disabled"

**解决方法**:
```bash
# 检查环境变量
echo $MEMOS_AI_ENABLED

# 在 .env 中设置
echo "MEMOS_AI_ENABLED=true" >> .env
```

### 问题 2: "failed to create scheduler agent"

**解决方法**:
- 检查 LLM 配置是否正确
- 验证 API key 是否有效
- 查看后端日志获取详细错误

### 问题 3: 日程没有创建成功

**检查步骤**:
```bash
# 1. 查看后端日志
make logs-follow-backend

# 2. 直接查询数据库
make db-connect
# 在 psql 中运行：
SELECT * FROM schedules ORDER BY created_ts DESC LIMIT 5;

# 3. 测试直接 API
curl -X POST http://localhost:28081/api/v1/schedules \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title": "测试会议", "start_ts": 1737465600}'
```

### 问题 4: 智能体循环次数过多

**现象**: 查询时间过长或超时

**解决方法**:
- 检查 `MaxIterations` 设置（默认 5）
- 优化 system prompt
- 检查 LLM 响应时间

## 📚 快速命令参考

```bash
# 一键启动
make start

# 停止所有服务
make stop

# 查看状态
make status

# 重新构建
make build
make start

# 查看日志
make logs-follow-backend

# 重置数据库（危险！）
make db-reset

# 运行测试
go test ./server/service/schedule/... -v
go test ./plugin/ai/agent/... -v
```

## 🎯 下一步

1. **验证基础功能**
   - 启动服务
   - 测试查询和创建
   - 检查日志输出

2. **性能调优**
   - 测量响应时间
   - 优化 prompt
   - 调整迭代限制

3. **集成到生产**
   - 配置生产环境变量
   - 设置监控和告警
   - 进行负载测试

详细文档请查看：
- `docs/agent_architecture/agent_scheduler/COMPLETION_REPORT.md`
- `docs/agent_architecture/agent_scheduler/IMPLEMENTATION_SUMMARY.md`
