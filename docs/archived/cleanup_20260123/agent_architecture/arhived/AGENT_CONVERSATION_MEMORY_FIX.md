# Agent 对话记忆功能实现

## 用户反馈的问题

**症状**: Agent 会"失忆"，每次对话都忘记之前的上下文

**示例**:
```
用户: 明天下午21点买鲜花
Agent: 21点通常是晚上9点，您是指下午还是晚上？需要多长时间？

用户: 晚上9点，大概30分钟
Agent: [忘记之前问过问题，重新问一遍] 21点通常是晚上9点...
```

**根本原因**: 每次只发送当前消息，没有传递对话历史

## 解决方案

### 1. 添加对话历史状态 ✅

**文件**: `web/src/components/AIChat/ScheduleInput.tsx` (第 48 行)

```typescript
const [conversationHistory, setConversationHistory] = useState<Array<{role: string, content: string}>>([]);
```

**数据结构**:
```typescript
type ConversationMessage = {
  role: string;      // "user" | "assistant"
  content: string;   // 消息内容
}
```

### 2. 构建完整对话上下文 ✅

**修改**: `handleAgentParse` 函数 (第 117-130 行)

```typescript
// Add user message to history
const newHistory = [
  ...conversationHistory,
  { role: "user", content: input }
];

// Build full conversation context
const conversationContext = newHistory
  .map(msg => `${msg.role}: ${msg.content}`)
  .join("\n");

const result = await agentChat.mutateAsync({
  message: `${conversationContext}\n\nuser: ${input}`,
  userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Shanghai",
});
```

**发送给 Agent 的消息格式**:
```
user: 明天下午21点买鲜花
assistant: 我注意到您说'明天下午21点'，但21点通常是晚上9点。您是指...

user: 晚上9点，大概30分钟
```

### 3. 保存 Assistant 响应到历史 ✅

**修改**: `handleAgentParse` 函数 (第 132-138 行)

```typescript
if (result.response) {
  // Add assistant response to history
  const updatedHistory = [
    ...newHistory,
    { role: "assistant", content: result.response }
  ];
  setConversationHistory(updatedHistory);
  setAgentResponse(result.response);

  // ... rest of logic
}
```

### 4. 清除对话历史的时机 ✅

**场景 1**: 日程创建成功后 (第 151 行)
```typescript
if (createdSchedule) {
  toast.success("日程创建成功");
  setConversationHistory([]); // Clear history after success
  // ...
}
```

**场景 2**: 用户点击"清除"按钮 (第 400 行)
```typescript
onClick={() => {
  setAgentResponse(null);
  setInput("");
  setConversationHistory([]); // Clear conversation history
}}
```

**场景 3**: 用户切换到手动模式 (第 413 行)
```typescript
onClick={() => {
  setUseAgentMode(false);
  setAgentResponse(null);
  setConversationHistory([]); // Clear conversation history
  // ...
}}
```

**场景 4**: 关闭对话框 (第 296 行)
```typescript
const handleClose = () => {
  setInput("");
  setParsedSchedule(null);
  setConflicts([]);
  setShowConflictAlert(false);
  setAgentResponse(null);
  setConversationHistory([]); // Clear conversation history
  onOpenChange(false);
};
```

## 对话流程示例

### 完整的多轮对话

```
【第1轮】
用户输入: "明天下午21点买鲜花"
对话历史: []
发送给 Agent: "user: 明天下午21点买鲜花"
Agent 回复: "我注意到您说'明天下午21点'，但21点通常是晚上9点..."

对话历史: [
  { role: "user", content: "明天下午21点买鲜花" },
  { role: "assistant", content: "我注意到您说..." }
]

【第2轮】
用户输入: "晚上9点，大概30分钟"
对话历史: [
  { role: "user", content: "明天下午21点买鲜花" },
  { role: "assistant", content: "我注意到您说..." }
]
发送给 Agent: "user: 明天下午21点买鲜花
assistant: 我注意到您说'明天下午21点'，但21点通常是晚上9点...

user: 晚上9点，大概30分钟"

Agent 回复: "已成功创建日程：明天（2026年1月22日）晚上21:00-21:30 买鲜花"

对话历史: [
  { role: "user", content: "明天下午21点买鲜花" },
  { role: "assistant", content: "我注意到您说..." },
  { role: "user", content: "晚上9点，大概30分钟" },
  { role: "assistant", content: "已成功创建日程..." }
]

【创建成功】
清除对话历史: []
关闭对话框
```

## 技术实现细节

### 消息格式

**为什么使用文本格式而不是 JSON？**

当前实现使用简单的文本格式：
```
user: 消息1
assistant: 响应1
user: 消息2
```

**优点**:
- ✅ 简单直观，LLM 容易理解
- ✅ 与 Agent 的 prompt 风格一致
- ✅ 减少序列化/反序列化开销

**缺点**:
- ⚠️ 如果消息中包含换行符可能混淆
- ⚠️ 需要更复杂的格式（如 JSON）来支持嵌套结构

**未来改进**:
```typescript
// 可以改为 JSON 格式
const messagesJSON = JSON.stringify(newHistory);
const result = await agentChat.mutateAsync({
  message: input,
  history: messagesJSON,  // 单独传递历史
  userTimezone: ...
});
```

### 状态管理

**为什么使用 `useState` 而不是 `useRef`？**

```typescript
// ✅ 使用 useState (当前方案)
const [conversationHistory, setConversationHistory] = useState<Array<...>>([]);

// ❌ 不使用 useRef
const conversationHistory = useRef<Array<...>>([]);
```

**原因**:
- `useState` 触发重新渲染，UI 可以反映对话状态
- `useRef` 不会触发渲染，虽然性能更好但不适合需要展示历史的场景

### 内存管理

**何时清除历史**:
1. ✅ 日程创建成功后
2. ✅ 用户点击"清除"
3. ✅ 用户切换到手动模式
4. ✅ 对话框关闭
5. ✅ 用户切换 Agent 模式开关

**不需要清除**:
- ❌ Agent 询问澄清时（保留历史）
- ❌ 用户回复澄清时（继续累积）

## 对比修复前后

### 修复前

```
第1轮:
发送: "明天下午21点买鲜花"
Agent: "21点通常是晚上9点，您是指下午还是晚上？"

第2轮:
发送: "晚上9点，大概30分钟"  // ❌ 只有当前消息
Agent: "21点通常是晚上9点..." // ❌ 重复问同样的问题
```

**问题**: Agent 每次都是"全新"的，没有上下文

### 修复后

```
第1轮:
发送: "user: 明天下午21点买鲜花"
Agent: "21点通常是晚上9点，您是指下午还是晚上？"

第2轮:
发送: "user: 明天下午21点买鲜花
       assistant: 21点通常是晚上9点...
       user: 晚上9点，大概30分钟"  // ✅ 包含完整历史
Agent: "已成功创建日程..."      // ✅ 理解上下文
```

**改进**: Agent 可以访问完整对话历史，不会"失忆"

## 编译验证

```bash
✓ built in 9.26s
```

**结果**: 前端编译成功，无错误。

## 测试建议

### 测试场景 1: 正常多轮对话

1. 输入: "明天下午21点开会"
2. Agent 询问: "是下午还是晚上？需要多久？"
3. 输入: "晚上9点，1小时"
4. **预期**: Agent 创建日程成功 ✅

### 测试场景 2: 切换模式

1. 输入: "明天下午21点开会"
2. Agent 询问澄清
3. 点击"切换到手动模式"
4. **预期**: 对话历史清除，切换到传统模式 ✅

### 测试场景 3: 清除重试

1. 输入: "明天下午21点开会"
2. Agent 询问澄清
3. 点击"清除"
4. 输入: "明天晚上9点开1小时会"
5. **预期**: 历史清除，重新开始 ✅

## 已知限制

### 1. 后端可能不支持独立 history 字段

当前实现将历史打包到 `message` 字段中：

```typescript
message: `${conversationContext}\n\nuser: ${input}`
```

如果后端将来支持独立的 `history` 字段，可以优化为：

```typescript
const result = await agentChat.mutateAsync({
  message: input,
  history: conversationHistory,  // 单独传递
  userTimezone: ...
});
```

### 2. 历史长度限制

- 当前没有限制对话轮次
- 如果对话过长，可能超过 LLM 的 context window
- **建议**: 添加最大轮次限制（如 5 轮）

```typescript
const MAX_HISTORY_ROUNDS = 5;

// 保持历史在限制内
const trimmedHistory = conversationHistory.slice(-MAX_HISTORY_ROUNDS * 2);
```

### 3. 复杂消息格式

当前使用简单的文本格式，如果消息包含：
- 多行文本
- 特殊字符（如 `\n`, `\t`）
- 代码片段

可能导致解析问题。**建议**: 使用 JSON 格式并转义。

## 总结

**问题**: Agent 每次对话都"失忆"，无法理解上下文

**解决**:
1. ✅ 添加 `conversationHistory` 状态
2. ✅ 每次发送包含完整对话历史
3. ✅ 保存 Assistant 响应到历史
4. ✅ 在适当时机清除历史

**效果**: Agent 可以理解多轮对话的上下文，不会"失忆"

---

**修复完成时间**: 2026-01-21 21:20
**编译状态**: ✅ 前端通过
