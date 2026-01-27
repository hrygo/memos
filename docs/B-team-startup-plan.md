# B 团队启动工作规划

> **生成时间**: 2026-01-27  
> **状态**: ✅ Sprint 1 完成，可启动 Sprint 2

---

## Sprint 0 验收状态

| 检查项 | 状态 | 说明 |
|:---|:---:|:---|
| 接口定义 (7个) | ✅ | `plugin/ai/*/interface.go` 全部完成 |
| Mock 实现 (7个) | ✅ | `plugin/ai/*/mock.go` 全部完成 |
| 契约测试 | ✅ | 4 个核心服务测试全部通过 |
| 现有代码分析 | ✅ | 发现可复用组件 |

---

## Sprint 1 完成状态 ✅

| Spec ID | 任务 | 状态 | 交付物 |
|:---|:---|:---:|:---|
| **P1-B002** | 错误恢复机制 | ✅ | `errors.go`, `recovery.go`, `recovery_test.go` |
| **P1-B001** | 工具可靠性增强 | ✅ | `tools/executor.go`, `tools/fallback.go`, `tools/executor_test.go` |

### Code Review 修复项

| Issue | 优先级 | 修复状态 |
|:---|:---:|:---:|
| ctx取消时重试循环未正确退出 | HIGH | ✅ |
| ExecuteDetailed fallback错误被丢弃 | HIGH | ✅ |
| 删除未使用的 maxRetries 字段 | MEDIUM | ✅ |
| fallbackRules 复制避免并发风险 | MEDIUM | ✅ |
| 预编译正则表达式 | LOW | ✅ |
| WithTimezone 返回新实例 | LOW | ✅ |

---

## Sprint 2 (待启动)

| Spec ID | 任务 | 人天 | 依赖状态 | 执行策略 |
|:---|:---|:---:|:---|:---|
| **P1-B003** | 时间解析加固 | 2 | ⏳ P1-A004 | Mock 已可用 |
| **P1-B004** | 规则分类器扩展 | 2 | ⏳ P1-A003 | Mock 已可用 |

---

## 代码组织结构 (实际)

```
plugin/ai/
├── agent/
│   ├── tools/
│   │   ├── executor.go      [P1-B001] ✅
│   │   ├── fallback.go      [P1-B001] ✅
│   │   └── executor_test.go [P1-B001] ✅
│   ├── recovery.go          [P1-B002] ✅
│   ├── recovery_test.go     [P1-B002] ✅
│   ├── errors.go            [P1-B002] ✅
│   ├── intent_classifier.go [P1-B004] PENDING
│   └── ...
├── schedule/
│   └── time_hardener.go     [P1-B003] PENDING
└── ...
```

---

## 开发进度清单

- [x] Sprint 0 接口契约验证通过
- [x] Mock 实现可用于并行开发
- [x] 现有代码分析完成
- [x] **P1-B002**: 错误恢复机制 ✅
- [x] **P1-B001**: 工具可靠性增强 ✅
- [x] Code Review + 修复 ✅
- [ ] 等待 A 团队 Sprint 1 交付后启动 Sprint 2

---

## 风险与缓解

| 风险 | 缓解措施 | 状态 |
|:---|:---|:---:|
| Mock 与真实实现行为差异 | 契约测试约束 + 集成测试验证 | 监控中 |
| A 团队 Sprint 1 延期 | Sprint 2 任务使用 Mock 先行开发 | 准备就绪 |
| 时间解析边界情况 | TimeService Mock 已覆盖中文场景 | 已验证 |

---

## 下一步

B 团队 Sprint 1 已完成，可启动 **Sprint 2** 任务：
- P1-B003 时间解析加固
- P1-B004 规则分类器扩展

等待 A 团队 Sprint 1 (P1-A003 LLM路由优化、P1-A004 时间服务) 交付后正式集成。
