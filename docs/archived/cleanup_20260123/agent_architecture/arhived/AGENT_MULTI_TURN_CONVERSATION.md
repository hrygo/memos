# Agent 多轮对话交互改进

## 用户反馈的问题

**场景**: 用户在日历界面创建日程时：
1. 输入："明天下午 21 点买些鲜花"
2. Agent 澄清："21点通常是晚上9点，您是指下午还是晚上？活动需要多长时间？"
3. **问题**：用户不知道如何回复 Agent 的问题

## 改进方案

### 1. 保留对话上下文 ✅

**修改**: `handleAgentParse` 函数不再清除 `agentResponse`

```typescript
// 修改前
setAgentResponse(null); // ❌ 清空历史

// 修改后
// Don't clear agentResponse - preserve conversation history
// ✅ 保留对话历史
```

### 2. 智能检测创建结果 ✅

**逻辑**: 检测 Agent 响应中是否包含创建成功的关键词

```typescript
const createdSchedule = result.response.includes("已成功创建") ||
                       result.response.includes("日程已创建") ||
                       result.response.includes("schedule created");

if (createdSchedule) {
  toast.success("日程创建成功");
  queryClient.invalidateQueries({ queryKey: ["schedules"] });
  // Auto-close dialog after 1.5 seconds
  setTimeout(() => handleClose(), 1500);
} else {
  // Agent is asking for clarification
  toast.success("智能助手回复");
  // Clear input for user's response
  setInput("");
}
```

### 3. 优化输入框引导 ✅

**修改**: 动态更新输入框的标签和占位符

```typescript
<Label htmlFor="schedule-input">
  {t("schedule.description") || "Description"}
  {useAgentMode && agentResponse && (
    <span className="text-primary ml-2">💬 请在下方回复助手的问题</span>
  )}
</Label>

<Textarea
  placeholder={
    useAgentMode && agentResponse
      ? "例如：\"晚上9点\" 或 \"大概30分钟\""
      : useAgentMode
        ? "智能模式：\"明天下午3点开会\" 或 \"查看本周日程\""
        : "e.g., \"明天下午3点开会\""
  }
/>
```

### 4. 按钮文案动态切换 ✅

**修改**: 根据对话状态显示不同的按钮文字

```typescript
{useAgentMode ? (
  <>
    <Bot className="mr-2 h-4 w-4" />
    {agentResponse ? "继续对话" : "智能解析"}
  </>
) : (
  t("schedule.create-schedule")
)}
```

### 5. 添加"切换到手动模式"按钮 ✅

**功能**: 当用户不想回答澄清问题时，可以切换到传统模式

```typescript
<Button
  variant="outline"
  size="sm"
  onClick={() => {
    setUseAgentMode(false);
    setAgentResponse(null);
    if (!input.trim()) {
      setInput("明天晚上21点买鲜花 (1小时)");
    }
    toast("已切换到手动模式，请调整时间后点击创建");
  }}
>
  切换到手动模式
</Button>
```

## 用户交互流程

### 方案 A：继续对话（多轮交互）

```
1. 用户输入: "明天下午21点买鲜花"
   ↓
2. 点击"智能解析"
   ↓
3. Agent 回复: "21点通常是晚上9点，您是指下午还是晚上？"
   ↓
4. 输入框提示: 💬 请在下方回复助手的问题
   占位符: 例如："晚上9点" 或 "大概30分钟"
   按钮文字: "继续对话"
   ↓
5. 用户输入: "晚上9点，大概30分钟"
   ↓
6. 点击"继续对话"
   ↓
7. Agent 创建日程 ✅
   ↓
8. 显示"日程创建成功"
   ↓
9. 1.5秒后自动关闭对话框
```

### 方案 B：切换到手动模式

```
1. 用户输入: "明天下午21点买鲜花"
   ↓
2. 点击"智能解析"
   ↓
3. Agent 询问澄清
   ↓
4. 用户点击"切换到手动模式"
   ↓
5. 对话框切换到传统模式
   ↓
6. 输入框预填充: "明天晚上21点买鲜花 (1小时)"
   ↓
7. 用户可以编辑时间
   ↓
8. 点击"创建日程"
   ↓
9. 传统解析器处理并显示表单
   ↓
10. 用户确认并创建
```

## 技术改进

### 状态管理优化

| 状态 | 说明 |
|------|------|
| `agentResponse` | 保留对话历史，不清空 |
| `input` | Agent 回复后清空，准备接收用户回复 |
| `useAgentMode` | 可随时切换到手动模式 |
| `isProcessingAgent` | 防止重复提交 |

### 用户体验提升

1. **视觉引导**
   - 💬 emoji 提示用户可以回复
   - 动态占位符示例
   - 按钮文字从"智能解析"变为"继续对话"

2. **操作便捷**
   - 输入框始终可用
   - 支持多轮对话
   - 提供"切换到手动模式"备选方案

3. **智能判断**
   - 自动检测是否创建成功
   - 创建成功后自动关闭对话框
   - 询问澄清时保持对话框打开

## 编译验证

```bash
✓ built in 9.48s
```

**结果**: 前端编译成功，无错误。

## 使用指南

### 对于用户

当 Agent 询问澄清问题时，你可以：

**选项 1：继续对话**
1. 在输入框中回复，例如："晚上9点"或"大概30分钟"
2. 点击"继续对话"按钮
3. Agent 会根据你的回答创建日程

**选项 2：切换到手动模式**
1. 点击"切换到手动模式"按钮
2. 系统会预填充一个示例
3. 编辑时间信息
4. 点击"创建日程"

**选项 3：重新开始**
1. 点击"清除"按钮
2. 输入更清晰的描述
3. 再次点击"智能解析"

### 示例对话

```
用户: 明天下午21点开会
Agent: 我注意到您说'明天下午21点'，但21点通常是晚上9点。
       您是指明天（2026年1月22日）的：
       1. 下午（比如15:00左右）开会？
       2. 还是晚上21:00开会？
       另外，这个活动大概需要多长时间？

用户: 晚上9点，1小时
Agent: 已成功创建日程：明天（2026年1月22日）晚上21:00-22:00 开会
       [1.5秒后自动关闭对话框]
```

## 总结

**问题**: Agent 询问澄清后，用户不知道如何继续

**解决**:
1. ✅ 保留对话历史
2. ✅ 添加视觉引导（💬 提示）
3. ✅ 动态占位符和按钮文案
4. ✅ 提供备选方案（切换到手动模式）
5. ✅ 智能判断并自动关闭

**效果**: 用户可以轻松进行多轮对话，或随时切换到手动模式

---

**改进完成时间**: 2026-01-21 21:15
**编译状态**: ✅ 前端通过
