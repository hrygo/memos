import { Outlet } from "react-router-dom";
import { AIChatProvider } from "@/contexts/AIChatContext";
import { AIChatSidebar } from "@/components/AIChat/AIChatSidebar";
import NavigationDrawer from "@/components/NavigationDrawer";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

const AIChatLayoutContent = () => {
  const lg = useMediaQuery("lg");

  return (
    <section className="@container w-full h-screen flex flex-col lg:h-screen overflow-hidden">
      {/* Mobile Header */}
      <div className="lg:hidden flex-none flex items-center gap-2 px-4 py-3 border-b border-border/50 bg-background">
        <NavigationDrawer />
        <div className="flex items-center gap-2 font-medium text-foreground">
          <span>AI Chat</span>
        </div>
      </div>

      {/* Desktop Sidebar */}
      {lg && (
        <div className="fixed top-0 left-16 shrink-0 h-svh border-r border-border bg-background w-72 overflow-hidden">
          <AIChatSidebar className="h-full" />
        </div>
      )}

      {/* Main Content */}
      <div className={cn("flex-1 min-h-0 overflow-x-hidden", lg ? "pl-72" : "")}>
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
