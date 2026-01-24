import { MessageSquarePlus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import toast from "react-hot-toast";
import { useAIChat } from "@/contexts/AIChatContext";
import { useAvailableParrots } from "@/hooks/useParrots";
import { cn } from "@/lib/utils";
import { PARROT_ICONS, ParrotAgentType } from "@/types/parrot";
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
    addContextSeparator,
    selectConversation,
  } = useAIChat();

  // Track which conversations have been loaded (have non-empty messages)
  const loadedConversationIds = new Set(
    conversations
      .filter(c => c.messages.length > 0)
      .map(c => c.id)
  );

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
                onResetContext={handleResetContext}
                isLoaded={loadedConversationIds.has(conversation.id)}
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
  const availableParrots = useAvailableParrots();

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
          <div className="fixed inset-0 z-10 cursor-pointer" onClick={() => setIsOpen(false)} />
          <div className="absolute bottom-full left-0 mb-2 z-20 bg-white dark:bg-zinc-800 rounded-lg shadow-lg border border-zinc-200 dark:border-zinc-700 py-1.5 min-w-[180px]">
            {availableParrots.map((parrot) => {
              const icon = PARROT_ICONS[parrot.id] || parrot.icon;
              return (
                <button
                  key={parrot.id}
                  onClick={() => {
                    onStartChat(parrot.id);
                    setIsOpen(false);
                  }}
                  className="flex items-center gap-3 w-full px-3 py-2 text-sm text-left hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors"
                >
                  {/* Parrot Icon */}
                  {icon.startsWith("/") ? (
                    <img src={icon} alt={parrot.displayName} className="w-5 h-5 object-contain" />
                  ) : (
                    <span className="text-base">{icon}</span>
                  )}
                  {/* Localized Name */}
                  <span className="text-zinc-900 dark:text-zinc-100 font-medium">{parrot.displayName}</span>
                  {/* Alt Name */}
                  <span className="text-xs text-zinc-400 dark:text-zinc-500 ml-auto">{parrot.displayNameAlt}</span>
                </button>
              );
            })}
          </div>
        </>
      )}
    </div>
  );
}
