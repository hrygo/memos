# Memos AI 开发任务清单

## 依赖库

```bash
go get github.com/tmc/langchaingo
go get github.com/go-deepseek/deepseek
```

---

## Phase 1: 基础设施 (2天)

- [ ] `proto/api/v1/ai_service.proto` - gRPC 服务定义
- [ ] `internal/profile/profile.go` - 添加 AI 配置字段
- [ ] `store/migration/postgres/0.30/1__add_pgvector.sql` - 向量表迁移
- [ ] `cd proto && buf generate` - 生成代码

---

## Phase 2: AI 插件 (3天)

- [ ] `plugin/ai/config.go` - 配置解析
- [ ] `plugin/ai/embedding.go` - Embedding 服务 (langchaingo)
- [ ] `plugin/ai/reranker.go` - Reranker 服务
- [ ] `plugin/ai/llm.go` - LLM 服务 (langchaingo)
- [ ] `plugin/ai/ai.go` - 主入口

---

## Phase 3: 数据层 (2天)

- [ ] `store/memo_embedding.go` - MemoEmbedding 模型
- [ ] `store/driver.go` - 扩展 Driver 接口
- [ ] `store/db/postgres/memo_embedding.go` - 向量搜索实现

---

## Phase 4: 服务层 (2天)

- [ ] `server/router/api/v1/ai_service.go` - AI gRPC 服务
- [ ] `server/router/api/v1/v1.go` - 注册服务
- [ ] `server/router/api/v1/acl_config.go` - 配置权限

---

## Phase 5: 后台任务 (1天)

- [ ] `server/runner/embedding/runner.go` - 向量生成任务
- [ ] `server/server.go` - 启动后台任务

---

## Phase 6: 前端 (2天)

- [ ] `web/src/hooks/useAIQueries.ts` - AI React Query Hooks
- [ ] 语义搜索 UI
- [ ] AI 对话组件

---

## Phase 7: 验证 (1天)

- [ ] 单元测试
- [ ] 端到端测试
- [ ] 更新 README.md

---

## 快速配置

```bash
export MEMOS_AI_ENABLED=true
export MEMOS_AI_SILICONFLOW_API_KEY=sk-xxx
export MEMOS_AI_DEEPSEEK_API_KEY=sk-xxx
```
