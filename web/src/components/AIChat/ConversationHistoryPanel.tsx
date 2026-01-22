import { MessageSquarePlus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import { ParrotAgentType } from "@/types/parrot";
import { ConversationItem } from "./ConversationItem";

interface ConversationHistoryPanelProps {
  className?: string;
  onSelectConversation?: (id: string) => void;
}

export function ConversationHistoryPanel({ className, onSelectConversation }: ConversationHistoryPanelProps) {
  const { t } = useTranslation();
  const {
    conversationSummaries,
    conversations,
    state,
    createConversation,
    deleteConversation,
    selectConversation,
    updateConversationTitle,
    pinConversation,
    unpinConversation,
  } = useAIChat();

  const handleSelectConversation = (id: string) => {
    selectConversation(id);
    onSelectConversation?.(id);
  };

  const handleStartNewChat = (parrotId: ParrotAgentType) => {
    // Check for existing conversation with same parrotId
    const existingConversation = conversations.find((c) => c.parrotId === parrotId);
    if (existingConversation) {
      selectConversation(existingConversation.id);
      onSelectConversation?.(existingConversation.id);
      return;
    }
    // Only create new if no existing conversation found
    createConversation(parrotId);
    onSelectConversation?.("");
  };

  const handleDelete = (id: string) => {
    deleteConversation(id);
  };

  const handleRename = (id: string, newTitle: string) => {
    updateConversationTitle(id, newTitle);
  };

  const handleTogglePin = (id: string) => {
    const conversation = conversationSummaries.find((c) => c.id === id);
    if (conversation?.pinned) {
      unpinConversation(id);
    } else {
      pinConversation(id);
    }
  };

  const hasConversations = conversationSummaries.length > 0;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Header */}
      <div className="px-3 py-2 border-b border-zinc-200 dark:border-zinc-700">
        <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">{t("ai.aichat.sidebar.history")}</h2>
        <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
          {t("ai.aichat.sidebar.conversation-count", { count: conversationSummaries.length })}
        </p>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {hasConversations ? (
          <div className="p-2 space-y-1">
            {conversationSummaries.map((conversation) => (
              <ConversationItem
                key={conversation.id}
                conversation={conversation}
                isActive={conversation.id === state.currentConversationId}
                onSelect={handleSelectConversation}
                onDelete={handleDelete}
                onRename={handleRename}
                onTogglePin={handleTogglePin}
              />
            ))}
          </div>
        ) : (
          <EmptyState onStartChat={handleStartNewChat} />
        )}
      </div>

      {/* New Chat Button */}
      {hasConversations && (
        <div className="p-2 border-t border-zinc-200 dark:border-zinc-700">
          <NewChatButton onStartChat={handleStartNewChat} />
        </div>
      )}
    </div>
  );
}

interface EmptyStateProps {
  onStartChat: (parrotId: ParrotAgentType) => void;
}

function EmptyState({ onStartChat }: EmptyStateProps) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center justify-center h-full p-6 text-center">
      <div className="w-12 h-12 rounded-full bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mb-3">
        <MessageSquarePlus className="w-6 h-6 text-zinc-400" />
      </div>
      <h3 className="text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-1">{t("ai.aichat.sidebar.no-conversations")}</h3>
      <p className="text-xs text-zinc-500 dark:text-zinc-400 mb-4">{t("ai.aichat.sidebar.start-new-chat")}</p>
      <NewChatButton onStartChat={onStartChat} />
    </div>
  );
}

interface NewChatButtonProps {
  onStartChat: (parrotId: ParrotAgentType) => void;
}

function NewChatButton({ onStartChat }: NewChatButtonProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);

  const parrots = [
    { id: ParrotAgentType.DEFAULT, label: "Default", icon: "🤖" },
    { id: ParrotAgentType.MEMO, label: "Memo", icon: "🦜" },
    { id: ParrotAgentType.SCHEDULE, label: "Schedule", icon: "📅" },
    { id: ParrotAgentType.AMAZING, label: "Amazing", icon: "⭐" },
    { id: ParrotAgentType.CREATIVE, label: "Creative", icon: "💡" },
  ];

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-center gap-2 px-3 py-2 bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium rounded-lg transition-colors"
      >
        <MessageSquarePlus className="w-4 h-4" />
        {t("ai.aichat.sidebar.new-chat")}
      </button>

      {isOpen && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setIsOpen(false)} />
          <div className="absolute bottom-full left-0 mb-2 z-20 bg-white dark:bg-zinc-800 rounded-lg shadow-lg border border-zinc-200 dark:border-zinc-700 py-1 min-w-[160px]">
            {parrots.map((parrot) => (
              <button
                key={parrot.id}
                onClick={() => {
                  onStartChat(parrot.id);
                  setIsOpen(false);
                }}
                className="flex items-center gap-2 w-full px-3 py-2 text-sm text-left hover:bg-zinc-100 dark:hover:bg-zinc-700"
              >
                <span>{parrot.icon}</span>
                <span className="text-zinc-900 dark:text-zinc-100">{parrot.label}</span>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
