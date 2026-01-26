import { MessageSquarePlus } from "lucide-react";
import { useMemo } from "react";
import toast from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import { ConversationSummary } from "@/types/aichat";
import { ParrotAgentType } from "@/types/parrot";
import { ConversationItem } from "./ConversationItem";

interface ConversationHistoryPanelProps {
  className?: string;
  onSelectConversation?: (id: string) => void;
}

/**
 * 会话历史面板 - 统一入口设计
 *
 * 设计原则：
 * - 新建对话直接进入，无需选择助手类型
 * - 会话按时间分组，提升回溯效率
 * - 智能路由由系统自动处理
 */
export function ConversationHistoryPanel({ className, onSelectConversation }: ConversationHistoryPanelProps) {
  const { t } = useTranslation();
  const { conversationSummaries, conversations, state, createConversation, addContextSeparator, selectConversation } = useAIChat();

  const loadedConversationIds = new Set(conversations.filter((c) => c.messages.length > 0).map((c) => c.id));

  // 按时间分组会话
  const groupedConversations = useMemo(() => {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    const yesterday = today - 24 * 60 * 60 * 1000;
    const thisWeek = today - 7 * 24 * 60 * 60 * 1000;

    const groups: { key: string; label: string; conversations: ConversationSummary[] }[] = [
      { key: "today", label: t("ai.aichat.sidebar.time-group-today"), conversations: [] },
      { key: "yesterday", label: t("ai.aichat.sidebar.time-group-yesterday"), conversations: [] },
      { key: "thisWeek", label: t("ai.aichat.sidebar.time-group-this-week"), conversations: [] },
      { key: "earlier", label: t("ai.aichat.sidebar.time-group-earlier"), conversations: [] },
    ];

    conversationSummaries.forEach((conv) => {
      const timestamp = conv.updatedAt;
      if (timestamp >= today) {
        groups[0].conversations.push(conv);
      } else if (timestamp >= yesterday) {
        groups[1].conversations.push(conv);
      } else if (timestamp >= thisWeek) {
        groups[2].conversations.push(conv);
      } else {
        groups[3].conversations.push(conv);
      }
    });

    // 只返回有内容的分组
    return groups.filter((g) => g.conversations.length > 0);
  }, [conversationSummaries, t]);

  const handleSelectConversation = (id: string) => {
    selectConversation(id);
    onSelectConversation?.(id);
  };

  // 统一入口：直接创建会话，使用 AMAZING 作为默认（综合助手，智能路由）
  const handleStartNewChat = () => {
    createConversation(ParrotAgentType.AMAZING);
    onSelectConversation?.("");
  };

  const handleResetContext = (id: string) => {
    addContextSeparator(id, "manual");
    toast.success(t("ai.context-cleared-toast"), {
      duration: 2000,
      icon: "✂️",
      className: "dark:bg-zinc-800 dark:border-zinc-700",
    });
  };

  const hasConversations = conversationSummaries.length > 0;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* 新建对话按钮 - 置顶 */}
      <div className="p-2 border-b border-zinc-200/50 dark:border-zinc-800/50 shrink-0">
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

      {/* 会话列表 */}
      <div className="flex-1 overflow-y-auto">
        {hasConversations ? (
          <div className="flex flex-col py-1">
            {groupedConversations.map((group) => (
              <div key={group.key} className="mb-1">
                {/* 时间分组标签 */}
                <div className="px-3 py-1.5 text-xs font-medium text-zinc-400 dark:text-zinc-500 uppercase tracking-wide">
                  {group.label}
                </div>
                {/* 会话列表 */}
                <div className="flex flex-col gap-0.5 px-2">
                  {group.conversations.map((conversation) => (
                    <ConversationItem
                      key={conversation.id}
                      conversation={conversation}
                      isActive={conversation.id === state.currentConversationId}
                      onSelect={handleSelectConversation}
                      onResetContext={handleResetContext}
                      isLoaded={loadedConversationIds.has(conversation.id)}
                    />
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <EmptyState onStartChat={handleStartNewChat} />
        )}
      </div>
    </div>
  );
}

interface EmptyStateProps {
  onStartChat: () => void;
}

function EmptyState({ onStartChat }: EmptyStateProps) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center justify-center h-full p-4 text-center">
      <div className="w-12 h-12 rounded-2xl bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mb-3">
        <MessageSquarePlus className="w-5 h-5 text-zinc-400" />
      </div>
      <h3 className="text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-1">{t("ai.aichat.sidebar.no-conversations")}</h3>
      <p className="text-xs text-zinc-500 dark:text-zinc-400 mb-4">{t("ai.aichat.sidebar.start-new-chat")}</p>
      <button
        onClick={onStartChat}
        className={cn(
          "flex items-center justify-center gap-2 px-4 py-2",
          "bg-gradient-to-r from-emerald-500 to-green-600 hover:from-emerald-600 hover:to-green-700",
          "text-white text-sm font-medium rounded-lg transition-all",
          "shadow-sm active:scale-[0.98]",
        )}
      >
        <MessageSquarePlus className="w-4 h-4" />
        {t("ai.aichat.sidebar.new-chat")}
      </button>
    </div>
  );
}
