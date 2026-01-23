import { MenuIcon, SparklesIcon } from "lucide-react";
import { useState } from "react";
import { Outlet } from "react-router-dom";
import { AIChatSidebar } from "@/components/AIChat/AIChatSidebar";
import NavigationDrawer from "@/components/NavigationDrawer";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { AIChatProvider, useAIChat } from "@/contexts/AIChatContext";
import useMediaQuery from "@/hooks/useMediaQuery";
import { useParrot } from "@/hooks/useParrots";
import { cn } from "@/lib/utils";
import { ParrotAgentType } from "@/types/parrot";
import { useTranslate } from "@/utils/i18n";

const AIChatLayoutContent = () => {
  const lg = useMediaQuery("lg");
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const t = useTranslate();
  const { currentConversation } = useAIChat();
  const currentParrot = useParrot(currentConversation?.parrotId || ParrotAgentType.DEFAULT);

  return (
    <section className="@container w-full h-screen flex flex-col lg:h-screen overflow-hidden">
      {/* Mobile Header */}
      <div className="lg:hidden flex-none relative flex items-center justify-center px-4 py-3 border-b border-border/50 bg-background">
        {/* Left - Navigation Drawer */}
        <div className="absolute left-0 top-0 bottom-0 px-4 flex items-center">
          <NavigationDrawer />
        </div>

        {/* Center - Title */}
        <div className="flex items-center gap-2 overflow-hidden px-2">
          {currentConversation ? (
            <div className="flex items-center gap-2 animate-in fade-in slide-in-from-top-1 duration-300">
              {currentParrot.icon.startsWith("/") ? (
                <img src={currentParrot.icon} alt="" className="w-5 h-5 object-contain" />
              ) : (
                <span className="text-base">{currentParrot.icon}</span>
              )}
              <span className="font-semibold text-zinc-900 dark:text-zinc-100 text-sm truncate">{currentParrot.displayName}</span>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <SparklesIcon className="w-4 h-4 text-zinc-500" />
              <span className="font-medium text-zinc-900 dark:text-zinc-100 text-sm uppercase tracking-wider">{t("common.ai-assistant")}</span>
            </div>
          )}
        </div>

        {/* Right - Sidebar Toggle */}
        <div className="absolute right-0 top-0 bottom-0 px-4 flex items-center">
          <Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
            <SheetContent side="right" className="w-80 max-w-full bg-background p-0 gap-0">
              <SheetHeader>
                <SheetTitle />
              </SheetHeader>
              <AIChatSidebar className="h-full px-4" onClose={() => setMobileSidebarOpen(false)} />
            </SheetContent>
          </Sheet>
          <Button variant="ghost" size="icon" onClick={() => setMobileSidebarOpen(true)} aria-label="Open sidebar">
            <MenuIcon className="w-5 h-5 text-foreground" />
          </Button>
        </div>
      </div>

      {/* Desktop Sidebar */}
      {lg && (
        <div className="fixed top-0 left-16 shrink-0 h-svh border-r border-border bg-background w-72 overflow-hidden">
          <AIChatSidebar className="h-full" />
        </div>
      )}

      {/* Main Content */}
      <div className={cn("flex-1 min-h-0 overflow-x-auto", lg ? "pl-72" : "")}>
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
