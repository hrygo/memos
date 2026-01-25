import { useTranslation } from "react-i18next";
import { useAIChat } from "@/contexts/AIChatContext";
import { getLocalizedParrot } from "@/hooks/useParrots";
import { cn } from "@/lib/utils";
import { SidebarTab } from "@/types/aichat";
import { PARROT_AGENTS, PARROT_ICONS, PARROT_THEMES, PINNED_PARROT_AGENTS, ParrotAgentType } from "@/types/parrot";
import { ConversationHistoryPanel } from "./ConversationHistoryPanel";
import { ReferencedMemosPanel } from "./ReferencedMemosPanel";
import { SidebarTabs } from "./SidebarTabs";

interface AIChatSidebarProps {
  className?: string;
  onClose?: () => void;
}

export function AIChatSidebar({ className, onClose }: AIChatSidebarProps) {
  const { t } = useTranslation();
  const { state, setSidebarTab, conversations, createConversation, selectConversation } = useAIChat();
  const { sidebarTab, currentConversationId } = state;

  const handleTabChange = (tab: SidebarTab) => {
    setSidebarTab(tab);
  };

  const handleAgentClick = (agentId: ParrotAgentType) => {
    // Find latest conversation for this agent
    const agentConversations = conversations.filter((c) => c.parrotId === agentId);
    if (agentConversations.length > 0) {
      // Sort by updated time desc if needed, but conversations are usually sorted.
      // Assuming index 0 is latest or we just take the first one found.
      // Better to pick the most recent one.
      const latest = agentConversations[0];
      selectConversation(latest.id);
    } else {
      // Create new
      const agent = PARROT_AGENTS[agentId];
      // Need localized name?
      const localizedAgent = getLocalizedParrot(agent, t);
      createConversation(agentId, localizedAgent.displayName);
    }

    // Close sidebar on mobile if needed
    if (onClose && window.innerWidth < 768) {
      onClose();
    }
  };

  return (
    <aside className={cn("flex flex-col h-full bg-zinc-50 dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800", className)}>
      {/* Pinned Agents Section */}
      <div className="flex flex-col gap-2 p-3 border-b border-zinc-200 dark:border-zinc-800 shrink-0">
        <div className="text-xs font-semibold text-zinc-400 dark:text-zinc-500 px-2 mb-1 uppercase tracking-wider">{t("ai.assistants")}</div>
        <div className="flex flex-col gap-1">
          {PINNED_PARROT_AGENTS.map((agentId) => {
            const agent = PARROT_AGENTS[agentId];
            const localizedAgent = getLocalizedParrot(agent, t);
            const icon = PARROT_ICONS[agentId] || "🤖";
            const theme = PARROT_THEMES[agentId] || PARROT_THEMES.DEFAULT;

            // Check if current conversation belongs to this agent
            // This is a bit tricky, we might want to highlight if the *active* conversation is this agent type
            const activeConversation = conversations.find(c => c.id === currentConversationId);
            const isActive = activeConversation?.parrotId === agentId;

            return (
              <button
                key={agentId}
                onClick={() => handleAgentClick(agentId)}
                className={cn(
                  "relative flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-200 group text-left w-full outline-none focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:ring-zinc-400 dark:focus-visible:ring-zinc-600",
                  isActive
                    ? "bg-white dark:bg-zinc-800 shadow-sm ring-1 ring-zinc-900/5 dark:ring-white/5"
                    : "hover:bg-zinc-200/50 dark:hover:bg-zinc-800/50"
                )}
              >
                {/* Active Indicator (Left Bar) */}
                {isActive && (
                  <div className={cn("absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 rounded-r-full", theme.accent)} />
                )}

                <div className={cn(
                  "w-10 h-10 rounded-xl flex items-center justify-center text-lg shrink-0 transition-all shadow-sm border border-zinc-100 dark:border-zinc-700/50",
                  isActive ? theme.iconBg : "bg-white dark:bg-zinc-800 group-hover:scale-105"
                )}>
                  {icon.startsWith("/") ? (
                    <img src={icon} alt={localizedAgent.displayName} className="w-6 h-6 object-contain" />
                  ) : (
                    <span>{icon}</span>
                  )}
                </div>
                <div className="flex-1 min-w-0 flex flex-col justify-center gap-0.5">
                  <div className={cn(
                    "text-sm font-semibold truncate leading-tight",
                    isActive ? "text-zinc-900 dark:text-zinc-50" : "text-zinc-700 dark:text-zinc-300 group-hover:text-zinc-900 dark:group-hover:text-zinc-100"
                  )}>
                    {localizedAgent.displayName}
                  </div>
                  <div className={cn(
                    "text-xs truncate opacity-80 font-medium",
                    isActive ? "text-zinc-500 dark:text-zinc-400" : "text-zinc-400 dark:text-zinc-500"
                  )}>
                    {localizedAgent.description}
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Tabs */}
      <div className="px-2 pt-2 pb-3 flex-shrink-0">
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
