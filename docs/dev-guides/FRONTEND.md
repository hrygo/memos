# Frontend Development Guide

## Tech Stack
- **Framework**: React 18 with Vite 7
- **Language**: TypeScript
- **Styling**: Tailwind CSS 4, Radix UI components
- **State**: TanStack Query (React Query)
- **Internationalization**: `web/src/locales/` (i18next)
- **Markdown**: React Markdown with KaTeX, Mermaid, GFM support
- **Calendar**: FullCalendar for schedule visualization

---

## Workflow

### Commands (run in `web/` directory)

```bash
pnpm dev            # Start dev server (port 25173)
pnpm build          # Build for production
pnpm lint           # Run TypeScript and Biome checks
pnpm lint:fix       # Auto-fix linting issues
```

### From Project Root

```bash
make web            # Start frontend dev server
make build-web      # Build frontend for production
make check-i18n     # Verify i18n keys completeness
```

---

## Tailwind CSS 4 Pitfalls

### CRITICAL: Never use semantic `max-w-sm/md/lg/xl`

**Root Cause**: Tailwind CSS 4 redefines these classes to use `--spacing-*` variables (~16px) instead of traditional container widths (384-512px). This causes Dialogs, Sheets, and modals to collapse into unusable "slivers".

| Semantic Class | Tailwind 3 | Tailwind 4 |
|:---------------|:-----------|:-----------|
| `max-w-sm`      | 384px      | ~16px (broken) |
| `max-w-md`      | 448px      | ~16px (broken) |
| `max-w-lg`      | 512px      | ~16px (broken) |

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
| Width   | rem Value      | Use Case                          |
|:--------|:--------------|:----------------------------------|
| 384px   | `max-w-[24rem]` | Small dialogs, sidebars         |
| 448px   | `max-w-[28rem]` | Standard dialogs                 |
| 512px   | `max-w-[32rem]` | Large dialogs, forms             |
| 672px   | `max-w-[42rem]` | Wide content                     |

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
|:-------------------|:-------------------------|
| Dialog/Modal/Popover | Grid containers |
| Tooltip/Alert text   | Flex items that need to fill |
| Sidebar/Drawer       | Cards in responsive layouts |

**Rule**: Grid uses `gap`, not `max-w-*`. If `max-width / column_count < 200px`, don't use `max-w-*`.

---

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

| File | Purpose | Sidebar Type | Responsive |
|:-----|:--------|:-------------|:-----------|
| `RootLayout.tsx` | Global navigation and auth | None | N/A |
| `MainLayout.tsx` | Content-heavy pages | Collapsible `MemoExplorer` | md: fixed |
| `AIChatLayout.tsx` | AI chat interface | Fixed `AIChatSidebar` | Always fixed |
| `ScheduleLayout.tsx` | Schedule/calendar | Fixed `ScheduleCalendar` | Always fixed |

### Feature Layout Template

For new feature pages requiring a dedicated sidebar:

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
      </div>

      {/* Desktop Sidebar */}
      {lg && (
        <div className="fixed top-0 left-16 shrink-0 h-svh border-r border-border bg-background w-72 overflow-auto">
          <FeatureSidebar />
        </div>
      )}

      {/* Main Content */}
      <div className={cn("flex-1 min-h-0 overflow-x-hidden", lg ? "pl-72" : "")}>
        <Outlet />
      </div>
    </section>
  );
};
```

**Sidebar Width Options**:
| Class  | Pixels | Use Case                              |
|:-------|:-------|:---------------------------------------|
| `w-56`  | 224px  | Collapsible sidebar (MainLayout md)    |
| `w-64`  | 256px  | Standard sidebar                       |
| `w-72`  | 288px  | Default feature sidebar (AIChat, etc)  |
| `w-80`  | 320px  | Wide sidebar (Schedule)                |

**Responsive Breakpoints**:
| Breakpoint | Width  | Behavior                    |
|:-----------|:-------|:----------------------------|
| `sm`       | 640px  | Nav bar appears             |
| `md`       | 768px  | Sidebar becomes fixed       |
| `lg`       | 1024px | Full sidebar width          |

---

## Page Components

### Available Pages

| Path | Component | Layout | Purpose |
|:-----|:----------|:-------|:--------|
| `/` | `Home.tsx` | MainLayout | Main timeline with memo composer |
| `/explore` | `Explore.tsx` | MainLayout | Search and explore content |
| `/archived` | `Archived.tsx` | MainLayout | Archived memos |
| `/chat` | `AIChat.tsx` | AIChatLayout | AI chat interface |
| `/schedule` | `Schedule.tsx` | ScheduleLayout | Calendar view |
| `/review` | `Review.tsx` | MainLayout | Daily review |
| `/setting` | `Setting.tsx` | MainLayout | User settings |
| `/u/:username` | `UserProfile.tsx` | MainLayout | Public user profile |
| `/auth/callback` | `AuthCallback.tsx` | None | OAuth callback handler |

### Adding a New Page

1. Create component in `web/src/pages/YourPage.tsx`
2. Add i18n keys to `web/src/locales/en.json` and `zh-Hans.json`
3. Add route in `web/src/router/index.tsx`:
   ```tsx
   {
     path: "/your-page",
     element: <YourPage />,
   }
   ```
4. Run `make check-i18n` to verify translations

---

## Internationalization (i18n)

### File Structure

```
web/src/locales/
    ├── en.json       # English translations
    ├── zh-Hans.json  # Simplified Chinese
    └── zh-Hant.json  # Traditional Chinese
```

### Adding New Translations

1. Add key to `en.json`:
   ```json
   {
     "your": {
       "key": "Your text"
     }
   }
   ```

2. Add key to `zh-Hans.json`:
   ```json
   {
     "your": {
       "key": "您的文本"
     }
   }
   ```

3. Use in component:
   ```tsx
   import { t } from "i18next";

   const text = t("your.key");
   ```

4. Verify: `make check-i18n`

**CRITICAL**: Never hardcode text in components. Always use `t("key")`.

---

## Component Patterns

### MemoCard

Memo cards are used throughout the app for displaying memo content:

```tsx
import MemoCard from "@/components/MemoCard";

<MemoCard
  memo={memo}
  onView={() => navigate(`/m/${memo.id}`)}
  onEdit={() => openEditDialog(memo)}
/>
```

### Dialog/Modal Pattern

Always use explicit rem values for width:

```tsx
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

<DialogContent className="max-w-[28rem]">
  <DialogHeader>
    <DialogTitle>{t("title")}</DialogTitle>
  </DialogHeader>
  {/* Content */}
</DialogContent>
```

---

## State Management

### Data Fetching (TanStack Query)

```tsx
import { useQuery } from "@tanstack/react-query";

const { data, isLoading, error } = useQuery({
  queryKey: ["memos"],
  queryFn: () => api.memo.list(),
});
```

### Mutations

```tsx
import { useMutation } from "@tanstack/react-query";

const mutation = useMutation({
  mutationFn: (memo: MemoCreate) => api.memo.create(memo),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ["memos"] });
  },
});
```
