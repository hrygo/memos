# Frontend Development Guide

## Workflow
- **Commands** (run in `web/` directory):
  - `pnpm dev`: Start dev server
  - `pnpm build`: Build for production
  - `pnpm lint`: Run TypeScript and Biome checks
  - `pnpm lint:fix`: Auto-fix linting issues
- **Styling**: Tailwind CSS 4 (primary), Radix UI components
- **State**: TanStack Query (React Query)
- **Internationalization**: `web/src/locales/`
- **Markdown**: React Markdown with KaTeX, Mermaid, GFM support

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
