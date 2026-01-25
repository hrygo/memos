import { MenuIcon } from "lucide-react";
import { useState } from "react";
import { Outlet } from "react-router-dom";
import { AIChatSidebar } from "@/components/AIChat/AIChatSidebar";
import NavigationDrawer from "@/components/NavigationDrawer";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent } from "@/components/ui/sheet";
import { AIChatProvider } from "@/contexts/AIChatContext";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

// 统一的助手图标
const ASSISTANT_ICON = "🦜";

/**
 * AI Chat Layout - 优化的聊天布局
 *
 * UX/UI 改进：
 * - 优化移动端和桌面端的布局切换
 * - 统一间距和边框样式
 * - 改进侧边栏和主内容的视觉层次
 */
const AIChatLayoutContent = () => {
  const lg = useMediaQuery("lg");
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const t = useTranslate();
  const assistantName = t("ai.assistant-name");

  return (
    <section className="@container w-full h-screen flex flex-col lg:h-screen overflow-hidden bg-zinc-50 dark:bg-zinc-950">
      {/* Mobile Header */}
      <div className="lg:hidden flex-none relative flex items-center justify-center px-4 h-14 shrink-0 border-b border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900">
        {/* Left - Navigation Drawer */}
        <div className="absolute left-0 top-0 bottom-0 px-3 flex items-center">
          <NavigationDrawer />
        </div>

        {/* Center - Title */}
        <div className="flex items-center gap-2 overflow-hidden px-8">
          <span className="text-xl">{ASSISTANT_ICON}</span>
          <span className="font-semibold text-zinc-900 dark:text-zinc-100 text-sm truncate">
            {assistantName}
          </span>
        </div>

        {/* Right - Sidebar Toggle */}
        <div className="absolute right-0 top-0 bottom-0 px-3 flex items-center">
          <Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
            <SheetContent
              side="right"
              className="w-80 max-w-full bg-zinc-50 dark:bg-zinc-900 p-0 gap-0 [&_.absolute.top-4.right-4]:hidden border-l border-zinc-200 dark:border-zinc-800"
            >
              <AIChatSidebar className="h-full" onClose={() => setMobileSidebarOpen(false)} />
            </SheetContent>
          </Sheet>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setMobileSidebarOpen(true)}
            aria-label="Open sidebar"
            className="h-9 w-9"
          >
            <MenuIcon className="w-5 h-5" />
          </Button>
        </div>
      </div>

      {/* Desktop Sidebar */}
      {lg && (
        <div className="fixed top-0 left-16 shrink-0 h-svh border-r border-zinc-200/80 dark:border-zinc-800/80 bg-zinc-50/95 dark:bg-zinc-900/95 backdrop-blur-sm w-72 overflow-hidden">
          <AIChatSidebar className="h-full" />
        </div>
      )}

      {/* Main Content */}
      <div className={cn("flex-1 min-h-0 overflow-hidden bg-white dark:bg-zinc-900", lg ? "pl-72" : "")}>
        <Outlet />
      </div>
    </section>
  );
};

const AIChatLayout = () => {
  return (
    <AIChatProvider>
      <AIChatLayoutContent />
    </AIChatProvider>
  );
};

export default AIChatLayout;
