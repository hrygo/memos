# Schedule 页面布局重构计划

## 需求概述

1. **移除 MainLayout**：将 Schedule 页面从 MainLayout 中独立出来，不再显示左侧的 MemoExplorer（memos 日历）
2. **响应式设计**：PC 端和移动端都要有极佳的 UX/UI 体验
3. **布局要求**：
   - **PC 端**：左右分栏，所有视图同时显示
   - **移动端**：使用 TAB 切换（月视图 + 时间轴）
4. **交互逻辑**：
   - PC 端点击左侧月历日期时，右侧视图同步跳转到该日期

## 当前结构分析

### 路由配置（router/index.tsx）
```typescript
{
  element: <MainLayout />,  // ← Schedule 被包裹在 MainLayout 中
  children: [
    { path: Routes.SCHEDULE, element: <LazyRoute component={Schedule} /> },
  ],
}
```

### MainLayout 结构
- 左侧：固定宽度的 MemoExplorer（包含 memos 日历）
- 右侧：`<Outlet />` 渲染子路由（包括 Schedule）

### Schedule 页面当前组件
- `ScheduleCalendar`：月视图日历
- `ScheduleTimeline`：时间轴视图

## 实现方案

### 步骤 1：修改路由配置
将 Schedule 从 MainLayout 的 children 中移出，放在 RootLayout 下独立渲染：

```typescript
// router/index.tsx
{
  path: Routes.ROOT,
  element: <RootLayout />,
  children: [
    {
      element: <MainLayout />,
      children: [
        { path: "", element: <LazyRoute component={AIChat} /> },
        { path: Routes.HOME, element: <Home /> },
        // ... 其他路由
        // 移除 Schedule
      ],
    },
    // Schedule 独立出来
    { path: Routes.SCHEDULE, element: <LazyRoute component={Schedule} /> },
    // ...
  ],
}
```

### 步骤 2：重构 Schedule 页面

#### PC 端布局（lg 断点以上）
```
┌─────────────────────────────────────────────────────────┐
│ Header: 标题 + 添加按钮                                  │
├─────────────────┬───────────────────────────────────────┤
│ 左侧栏（固定）  │ 右侧主内容区                          │
│ (w-80)         │ (flex-1)                              │
│                 │                                       │
│ ScheduleCalendar│ View Toggle: [Timeline] [Calendar]   │
│ (月视图)       │ ┌─────────────────────────────────┐  │
│                 │ │ 当前视图内容                    │  │
│                 │ │ (ScheduleTimeline 或            │  │
│                 │ │  ScheduleCalendar)              │  │
│                 │ └─────────────────────────────────┘  │
└─────────────────┴───────────────────────────────────────┘
```

#### 移动端布局（lg 断点以下）
```
┌─────────────────────────┐
│ Header: 标题 + 添加     │
├─────────────────────────┤
│ [📅 月视图] [⏱ 时间轴] │ ← TAB 切换
├─────────────────────────┤
│                         │
│   当前 TAB 内容         │
│  (ScheduleCalendar 或   │
│   ScheduleTimeline)     │
│                         │
└─────────────────────────┘
```

**移动端 TAB**：
- TAB 1：月视图（ScheduleCalendar）
- TAB 2：时间轴（ScheduleTimeline）

### 步骤 3：创建响应式组件

#### 新组件：`ScheduleLayout`
- PC 端：左右分栏布局
- 移动端：TAB 切换布局

#### TAB 组件
使用现有的 Tabs 组件（来自 shadcn/ui）

## 关键文件

### 需要修改的文件
1. `web/src/router/index.tsx` - 路由配置
2. `web/src/pages/Schedule.tsx` - 主页面组件
3. `web/src/components/AIChat/ScheduleTimeline.tsx` - 移除内部的迷你日历

### 可复用的现有组件
- `ScheduleCalendar` - 月视图日历（可直接用作左侧栏）
- `ScheduleTimeline` - 时间轴视图（需要移除内部迷你日历）

## 设计细节

### PC 端
- 左侧栏宽度：`w-80` 或 `w-96`
- 左侧栏固定，右侧内容可滚动
- 使用 `sticky` 定位让左侧日历始终可见

### 移动端
- TAB 栏固定在顶部
- 内容区域可滚动
- 平滑的 TAB 切换动画

### 主题兼容
- 使用系统 CSS 变量（`--primary`, `--muted`, `--border` 等）
- 支持深色模式
- 支持所有主题色系

## 验证步骤

1. **路由验证**：访问 `/schedule` 页面，确认左侧不再显示 MemoExplorer（memos 日历）
2. **PC 端验证**：
   - 左右分栏布局正常
   - 点击左侧月历日期，右侧视图同步跳转
   - 右侧视图切换（Timeline/Calendar）正常工作
3. **移动端验证**：
   - TAB 切换正常（月视图 ↔ 时间轴）
   - TAB 之间的状态保持（选中日期等）
4. **主题兼容**：切换不同主题，验证所有颜色和样式正确适配
5. **功能测试**：添加、编辑、删除日程功能正常
