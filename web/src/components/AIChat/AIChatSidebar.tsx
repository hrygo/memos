import { useTranslation } from "react-i18next";
import { useAIChat } from "@/contexts/AIChatContext";
import { CapabilityType } from "@/types/capability";
import { cn } from "@/lib/utils";
import { SidebarTab } from "@/types/aichat";
import { ConversationHistoryPanel } from "./ConversationHistoryPanel";
import { ReferencedMemosPanel } from "./ReferencedMemosPanel";
import { SidebarTabs } from "./SidebarTabs";

interface AIChatSidebarProps {
  className?: string;
  onClose?: () => void;
}

export function AIChatSidebar({ className, onClose }: AIChatSidebarProps) {
  const { t } = useTranslation();
  const { state, setSidebarTab, setCurrentCapability } = useAIChat();
  const { sidebarTab, currentCapability, capabilityStatus } = state;

  const handleTabChange = (tab: SidebarTab) => {
    setSidebarTab(tab);
  };

  // 能力配置 - 简化
  const capabilities = [
    { type: CapabilityType.AUTO, icon: "🤖", labelKey: "ai.capability.auto.name" },
    { type: CapabilityType.MEMO, icon: "🦜", labelKey: "ai.capability.memo.name" },
    { type: CapabilityType.SCHEDULE, icon: "⏰", labelKey: "ai.capability.schedule.name" },
    { type: CapabilityType.AMAZING, icon: "🌟", labelKey: "ai.capability.amazing.name" },
  ] as const;

  return (
    <aside className={cn("flex flex-col h-full bg-zinc-50 dark:bg-zinc-900", className)}>
      {/* Header */}
      <div className="flex flex-col gap-3 p-3 border-b border-zinc-200 dark:border-zinc-800 shrink-0">
        {/* 助手卡片 */}
        <div className="flex items-center gap-2.5">
          <div className="relative shrink-0">
            <div className="w-11 h-11 rounded-xl bg-gradient-to-br from-emerald-500 to-green-600 flex items-center justify-center text-xl shadow-sm">
              🦜
            </div>
            <div className={cn(
              "absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-white dark:border-zinc-900",
              capabilityStatus === "idle" ? "bg-green-500" : "bg-amber-500 animate-pulse"
            )} />
          </div>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100 truncate">
              {t("ai.assistant-name")}
            </div>
            <div className="text-xs text-zinc-500 dark:text-zinc-400">
              {capabilityStatus === "thinking" ? t("ai.thinking") : t("ai.ready")}
            </div>
          </div>
        </div>

        {/* 能力切换 */}
        <div className="grid grid-cols-4 gap-1.5">
          {capabilities.map((cap) => {
            const isActive = currentCapability === cap.type;
            return (
              <button
                key={cap.type}
                onClick={() => setCurrentCapability(cap.type)}
                className={cn(
                  "flex items-center justify-center py-1.5 rounded-lg transition-all",
                  isActive
                    ? "bg-white dark:bg-zinc-800 shadow-sm border border-zinc-200 dark:border-zinc-700"
                    : "hover:bg-white/50 dark:hover:bg-zinc-800/50",
                )}
                title={t(cap.labelKey as any)}
              >
                <span className="text-lg">{cap.icon}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Tabs */}
      <div className="px-2 pt-2 pb-1.5 flex-shrink-0">
        <SidebarTabs activeTab={sidebarTab} onTabChange={handleTabChange} />
      </div>

      {/* Panel Content */}
      <div className="flex-1 overflow-hidden min-h-0">
        {sidebarTab === "history" && <ConversationHistoryPanel className="h-full" onSelectConversation={onClose} />}
        {sidebarTab === "memos" && <ReferencedMemosPanel className="h-full" />}
      </div>
    </aside>
  );
}
