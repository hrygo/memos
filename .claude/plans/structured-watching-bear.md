# 移除右上角 ADD 按钮实施计划

## 目标
移除日程页面右上角的 ADD 按钮，同时保留 ScheduleInput 对话框用于编辑日程功能。用户将通过底部的 ScheduleQuickInput 组件创建新日程。

## 修改文件
**文件:** `web/src/pages/Schedule.tsx`

## 具体修改

### 1. 移除桌面版 ADD 按钮 (第 109-112 行)
删除以下代码：
```tsx
<Button onClick={handleAddSchedule} size="sm" className="gap-1.5 h-9 px-3">
  <PlusIcon className="w-4 h-4" />
  <span className="hidden sm:inline">{t("schedule.add") || "Add"}</span>
</Button>
```

### 2. 移除移动版 ADD 按钮 (第 145-147 行)
删除以下代码：
```tsx
<Button onClick={handleAddSchedule} size="sm" className="gap-1 h-8 w-8 p-0">
  <PlusIcon className="w-4 h-4" />
</Button>
```

### 3. 移除 handleAddSchedule 函数 (第 45-48 行)
删除以下代码：
```tsx
const handleAddSchedule = () => {
  setEditSchedule(null);
  setScheduleInputOpen(true);
};
```

### 4. 移除未使用的 import
删除 `PlusIcon` 从 lucide-react 的导入（第 2 行）：
```tsx
// 修改前
import { Calendar, LayoutList, PlusIcon } from "lucide-react";

// 修改后
import { Calendar, LayoutList } from "lucide-react";
```

### 5. 调整桌面版头部布局
由于移除 ADD 按钮后，右侧只剩下搜索框，需要调整布局使其更合理：
```tsx
{/* Right: Search Bar only */}
<div className="flex items-center gap-2 flex-1 justify-end">
  <ScheduleSearchBar
    schedules={allSchedules}
    onFilteredChange={setFilteredSchedules}
    onHasFilterChange={setHasSearchFilter}
    className="max-w-xs"
  />
</div>
```

### 6. 调整移动版头部布局
移除 ADD 按钮后，移动版头部可以保持原样（只有视图切换按钮和搜索结果计数）。

## 验证步骤
1. 启动前端服务：`cd web && pnpm dev`
2. 访问日程页面
3. 确认：
   - 右上角不再显示 ADD 按钮（桌面版和移动版）
   - 底部的 ScheduleQuickInput 仍可正常创建日程
   - 点击时间线上的日程仍可打开编辑对话框
4. 检查控制台无错误

## 影响范围
- **用户界面:** 移除右上角 ADD 按钮
- **功能:** 创建日程功能仍可通过底部 ScheduleQuickInput 使用
- **编辑功能:** 完全保留，通过 ScheduleTimeline 的 onScheduleEdit 回调触发
