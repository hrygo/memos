# Frontend Development Guide

## Tech Stack
- **Framework**: React 18 with Vite 7
- **Language**: TypeScript
- **Styling**: Tailwind CSS 4, Radix UI components
- **State**: TanStack Query (React Query)
- **Internationalization**: `web/src/locales/` (i18next)
- **Markdown**: React Markdown with KaTeX, Mermaid, GFM support
- **Calendar**: FullCalendar for schedule visualization

## Workflow
- **Commands** (run in `web/` directory):
  - `pnpm dev`: Start dev server (port 25173)
  - `pnpm build`: Build for production
  - `pnpm lint`: Run TypeScript and Biome checks
  - `pnpm lint:fix`: Auto-fix linting issues

---

## Tailwind CSS 4 Pitfalls

### CRITICAL: Never use semantic `max-w-sm/md/lg/xl`

**Root Cause**: Tailwind CSS 4 redefines these classes to use `--spacing-*` variables (~16px) instead of traditional container widths (384-512px). This causes Dialogs, Sheets, and modals to collapse into unusable "slivers".

| Semantic Class | Tailwind 3 | Tailwind 4 |
| :--- | :--- | :--- |
| `max-w-sm` | 384px | ~16px (broken) |
| `max-w-md` | 448px | ~16px (broken) |
| `max-w-lg` | 512px | ~16px (broken) |

**Wrong** (collapses to ~16px):
```tsx
<DialogContent className="max-w-md">
<SheetContent className="sm:max-w-sm">
```

**Correct** (explicit rem values):
```tsx
<DialogContent className="max-w-[28rem]">  {/* 448px */}
<SheetContent className="sm:max-w-[24rem]"> {/* 384px */}
```

**Reference Table**:
| Width | rem Value | Use Case |
| :--- | :--- | :--- |
| 384px | `max-w-[24rem]` | Small dialogs, sidebars |
| 448px | `max-w-[28rem]` | Standard dialogs |
| 512px | `max-w-[32rem]` | Large dialogs, forms |
| 672px | `max-w-[42rem]` | Wide content |

### Avoid `max-w-*` on Grid containers

**Wrong** (causes overlap/squash):
```tsx
<div className="grid grid-cols-2 gap-3 w-full max-w-xs">
  {/* 320px / 2 = 160px per column - content crushed */}
</div>
```

**Correct**:
```tsx
<div className="grid grid-cols-2 gap-3 w-full">
  {/* Let gap and parent padding control width */}
</div>
```

| Use `max-w-*` for | Don't use `max-w-*` for |
| :--- | :--- |
| Dialog/Modal/Popover | Grid containers |
| Tooltip/Alert text | Flex items that need to fill |
| Sidebar/Drawer | Cards in responsive layouts |

**Rule**: Grid uses `gap`, not `max-w-*`. If `max-width / column-count < 200px`, don't use `max-w-*`.

## Layout Architecture

### Layout Hierarchy

```
RootLayout (global Nav + auth)
    │
    ├── MainLayout (collapsible sidebar: MemoExplorer)
    │   └── /, /explore, /archived, /u/:username
    │
    ├── AIChatLayout (fixed sidebar: AIChatSidebar)
    │   └── /chat
    │
    └── ScheduleLayout (fixed sidebar: ScheduleCalendar)
        └── /schedule
```

### Layout Files
- `web/src/layouts/RootLayout.tsx` - Global navigation and auth
- `web/src/layouts/MainLayout.tsx` - Collapsible sidebar for memos
- `web/src/layouts/AIChatLayout.tsx` - Fixed sidebar for AI chat
- `web/src/layouts/ScheduleLayout.tsx` - Fixed sidebar for schedules

### Feature Layout Template

For new feature pages requiring a dedicated sidebar, follow this pattern:

```tsx
import { Outlet } from "react-router-dom";
import NavigationDrawer from "@/components/NavigationDrawer";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

const FeatureLayout = () => {
  const lg = useMediaQuery("lg");

  return (
    <section className="@container w-full h-screen flex flex-col lg:h-screen overflow-hidden">
      {/* Mobile Header */}
      <div className="lg:hidden flex-none flex items-center gap-2 px-4 py-3 border-b border-border/50 bg-background">
        <NavigationDrawer />
        {/* Mobile-specific controls */}
      </div>

      {/* Desktop Sidebar */}
      {lg && (
        <div className="fixed top-0 left-16 shrink-0 h-svh border-r border-border bg-background w-{WIDTH} overflow-{auto|hidden}">
          <FeatureSidebar />
        </div>
      )}

      {/* Main Content */}
      <div className={cn("flex-1 min-h-0 overflow-x-hidden", lg ? "pl-{WIDTH}" : "")}>
        <Outlet />
      </div>
    </section>
  );
};
```

**Sidebar Width Options**:
| Class  | Pixels | Use Case                                        |
| ------ | ------ | ----------------------------------------------- |
| `w-56` | 224px  | Collapsible sidebar (MainLayout md)             |
| `w-64` | 256px  | Standard sidebar                                |
| `w-72` | 288px  | Default feature sidebar (AIChat, MainLayout lg) |
| `w-80` | 320px  | Wide sidebar (Schedule)                         |

**Responsive Breakpoints**:
| Breakpoint | Width  | Behavior              |
| ---------- | ------ | --------------------- |
| `sm`       | 640px  | Nav bar appears       |
| `md`       | 768px  | Sidebar becomes fixed |
| `lg`       | 1024px | Full sidebar width    |

**Layout Selection Guide**:

| Use Case                            | Layout Type      | Sidebar Type               |
| ----------------------------------- | ---------------- | -------------------------- |
| Content-heavy pages (Home, Explore) | `MainLayout`     | Collapsible `MemoExplorer` |
| Feature-specific (Chat, Schedule)   | Dedicated Layout | Fixed feature sidebar      |
| Full-screen modals                  | No layout        | None                       |
| Simple pages                        | `MainLayout`     | Reuse existing             |

**Adding a New Feature Layout**:

1. Create `web/src/layouts/FeatureLayout.tsx` using template above
2. Create sidebar component if needed: `web/src/components/FeatureSidebar.tsx`
3. Add route in `web/src/router/index.tsx`:
   ```tsx
   {
     path: "/feature",
     element: <FeatureLayout />,
     children: [
       { index: true, element: <FeaturePage /> }
     ]
   }
   ```
