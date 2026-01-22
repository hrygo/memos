import { SidebarTabs } from "./SidebarTabs";
import { ConversationHistoryPanel } from "./ConversationHistoryPanel";
import { ReferencedMemosPanel } from "./ReferencedMemosPanel";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import { SidebarTab } from "@/types/aichat";

interface AIChatSidebarProps {
  className?: string;
}

export function AIChatSidebar({ className }: AIChatSidebarProps) {
  const { state, setSidebarTab } = useAIChat();
  const { sidebarTab } = state;

  const handleTabChange = (tab: SidebarTab) => {
    setSidebarTab(tab);
  };

  return (
    <aside className={cn("flex flex-col h-full", className)}>
      {/* Tabs */}
      <div className="px-2 pb-3">
        <SidebarTabs activeTab={sidebarTab} onTabChange={handleTabChange} />
      </div>

      {/* Panel Content */}
      <div className="flex-1 overflow-hidden">
        {sidebarTab === "history" && <ConversationHistoryPanel className="h-full" />}
        {sidebarTab === "memos" && <ReferencedMemosPanel className="h-full" />}
      </div>
    </aside>
  );
}
