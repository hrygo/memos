import { MessageSquarePlus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import { SidebarTab } from "@/types/aichat";
import { ParrotAgentType } from "@/types/parrot";
import { ConversationHistoryPanel } from "./ConversationHistoryPanel";
import { ReferencedMemosPanel } from "./ReferencedMemosPanel";
import { SidebarTabs } from "./SidebarTabs";

interface AIChatSidebarProps {
  className?: string;
  onClose?: () => void;
}

/**
 * AI Chat Sidebar - 统一入口设计
 *
 * 设计原则：
 * - 新建对话置顶，作为最重要的操作
 * - 移除能力切换按钮，完全依赖智能路由
 * - 用户无需理解系统内部能力边界
 */
export function AIChatSidebar({ className, onClose }: AIChatSidebarProps) {
  const { t } = useTranslation();
  const { state, setSidebarTab, createConversation } = useAIChat();
  const { sidebarTab, capabilityStatus } = state;

  const handleTabChange = (tab: SidebarTab) => {
    setSidebarTab(tab);
  };

  // 统一入口：直接创建会话，使用 AMAZING 作为默认（综合助手，智能路由）
  const handleStartNewChat = () => {
    createConversation(ParrotAgentType.AMAZING);
    onClose?.();
  };

  return (
    <aside className={cn("flex flex-col h-full bg-zinc-50 dark:bg-zinc-900", className)}>
      {/* Header - 简化设计，移除能力切换 */}
      <div className="flex items-center gap-2.5 p-3 border-b border-zinc-200 dark:border-zinc-800 shrink-0">
        <div className="relative shrink-0">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-emerald-500 to-green-600 flex items-center justify-center text-lg shadow-sm">
            🦜
          </div>
          <div
            className={cn(
              "absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-white dark:border-zinc-900",
              capabilityStatus === "idle" ? "bg-green-500" : "bg-amber-500 animate-pulse",
            )}
          />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100 truncate">{t("ai.assistant-name")}</div>
          <div className="text-xs text-zinc-500 dark:text-zinc-400">
            {capabilityStatus === "thinking" ? t("ai.thinking") : t("ai.ready")}
          </div>
        </div>
      </div>

      {/* 新建对话按钮 - 置顶，在 Tab 之上 */}
      <div className="p-2 shrink-0">
        <button
          onClick={handleStartNewChat}
          className={cn(
            "w-full flex items-center justify-center gap-2 px-3 py-2",
            "bg-gradient-to-r from-emerald-500 to-green-600 hover:from-emerald-600 hover:to-green-700",
            "text-white text-sm font-medium rounded-lg transition-all",
            "shadow-sm active:scale-[0.98]",
          )}
        >
          <MessageSquarePlus className="w-4 h-4" />
          {t("ai.aichat.sidebar.new-chat")}
        </button>
      </div>

      {/* Tabs */}
      <div className="px-2 pb-1.5 flex-shrink-0">
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
