import copy from "copy-to-clipboard";
import toast from "react-hot-toast";
import { ChevronLeft } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AmazingInsightCard } from "@/components/AIChat/AmazingInsightCard";
import { ChatHeader } from "@/components/AIChat/ChatHeader";
import { ChatInput } from "@/components/AIChat/ChatInput";
import { ChatMessages } from "@/components/AIChat/ChatMessages";

import { ParrotHub } from "@/components/AIChat/ParrotHub";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useAIChat } from "@/contexts/AIChatContext";
import { useChat } from "@/hooks/useAIQueries";
import useMediaQuery from "@/hooks/useMediaQuery";
import type { ParrotAgentI18n } from "@/hooks/useParrots";
import { getLocalizedParrot } from "@/hooks/useParrots";
import { cn } from "@/lib/utils";
import type { ChatItem } from "@/types/aichat";
import type { MemoQueryResultData, ParrotChatParams, ScheduleQueryResultData } from "@/types/parrot";
import { PARROT_AGENTS, PARROT_ICONS, ParrotAgentType } from "@/types/parrot";



// ============================================================
// CHAT VIEW - Active Conversation
// ============================================================
interface ChatViewProps {
  currentParrot: ParrotAgentI18n;
  input: string;
  setInput: (value: string) => void;
  onSend: (messageContent?: string) => void;
  isTyping: boolean;
  isThinking: boolean;
  clearDialogOpen: boolean;
  setClearDialogOpen: (open: boolean) => void;
  onClearChat: () => void;
  onClearContext: () => void;
  onBackToHub: () => void;
  memoQueryResults: MemoQueryResultData[];
  scheduleQueryResults: ScheduleQueryResultData[];
  items: ChatItem[];
  onParrotChange: (parrot: ParrotAgentI18n | null) => void;
}

function ChatView({
  currentParrot,
  input,
  setInput,
  onSend,
  isTyping,
  isThinking,
  clearDialogOpen,
  setClearDialogOpen,
  onClearChat,
  onClearContext,
  onBackToHub,
  memoQueryResults,
  scheduleQueryResults,
  items,
  onParrotChange,
}: ChatViewProps) {
  const { t } = useTranslation();
  const md = useMediaQuery("md");

  const handleInputChange = (value: string) => {
    setInput(value);
  };

  const handleCopyMessage = (content: string) => {
    copy(content);
  };

  const handleDeleteMessage = () => {
    // TODO: Implement message deletion
  };

  const getParrotIcon = (parrotId: string) => {
    return PARROT_ICONS[parrotId] || "🤖";
  };

  const currentIcon = getParrotIcon(currentParrot.id);

  // Welcome message when no messages
  const welcomeMessage = (
    <div className="flex flex-col items-center justify-center h-full text-center px-4">
      <div className="w-14 h-14 md:w-16 md:h-16 rounded-2xl flex items-center justify-center text-2xl md:text-3xl mb-3 shrink-0">
        {currentIcon.startsWith("/") ? (
          <img src={currentIcon} alt={currentParrot.displayName} className="w-12 h-12 md:w-14 md:h-14 object-contain" />
        ) : (
          currentIcon
        )}
      </div>
      <h3 className="text-lg md:text-xl font-semibold text-zinc-900 dark:text-zinc-100 mb-2">
        {t("ai.welcome-intro", { name: currentParrot.displayName })}
        <span className="text-sm text-zinc-400 dark:text-zinc-500 ml-2">{currentParrot.displayNameAlt}</span>
      </h3>
      {currentParrot.description && (
        <p className="text-sm text-zinc-600 dark:text-zinc-400 mb-6 leading-relaxed">
          {currentParrot.description}
        </p>
      )}

      {currentParrot.examplePrompts && currentParrot.examplePrompts.length > 0 && (
        <div className="flex flex-col sm:flex-row flex-wrap gap-2 justify-center w-full max-w-2xl">
          {currentParrot.examplePrompts.slice(0, 3).map((prompt, idx) => (
            <button
              key={idx}
              onClick={() => {
                onSend(prompt);
              }}
              className={cn(
                "px-4 py-2 rounded-lg text-sm font-medium",
                "border border-zinc-200 dark:border-zinc-700",
                "bg-white dark:bg-zinc-800",
                "text-zinc-700 dark:text-zinc-300",
                "hover:bg-zinc-100 dark:hover:bg-zinc-700",
                "hover:border-zinc-300 dark:hover:border-zinc-600",
                "active:scale-95 cursor-pointer",
                "transition-all duration-200",
                "shadow-sm hover:shadow",
              )}
            >
              {prompt}
            </button>
          ))}
        </div>
      )}
    </div>
  );

  return (
    <div className="w-full h-full flex flex-col relative bg-white dark:bg-zinc-900">
      {/* Mobile Sub-Header - Compact back button only */}
      {!md && (
        <header className="flex items-center px-1.5 py-1 border-b border-zinc-100 dark:border-zinc-800/60 bg-white/80 dark:bg-zinc-900/80 backdrop-blur-md sticky top-0 z-20">
          <button
            onClick={onBackToHub}
            className="flex items-center gap-1.5 px-2 py-1.5 rounded-lg text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-all active:scale-95 group"
            aria-label="Back to hub"
          >
            <ChevronLeft className="w-4 h-4 transition-transform group-hover:-translate-x-0.5" />
            <span className="text-xs font-medium">{t("common.back") || "Back"}</span>
          </button>
        </header>
      )}

      {/* Desktop Header */}
      {md && (
        <ChatHeader
          parrot={currentParrot}
          isThinking={isThinking}
          onBack={onBackToHub}
          onClearContext={onClearContext}
          onClearChat={onClearChat}
        />
      )}

      {/* Messages Area */}
      <ChatMessages
        items={items}
        isTyping={isTyping}
        currentParrotId={currentParrot.id}
        onCopyMessage={handleCopyMessage}
        onDeleteMessage={handleDeleteMessage}
        amazingInsightCard={
          currentParrot.id === ParrotAgentType.AMAZING &&
            (memoQueryResults.length > 0 || scheduleQueryResults.length > 0) ? (
            <AmazingInsightCard
              memos={memoQueryResults[0]?.memos ?? []}
              schedules={scheduleQueryResults[0]?.schedules ?? []}
            />
          ) : undefined
        }
      >
        {/* Welcome message */}
        {items.length === 0 && welcomeMessage}
      </ChatMessages>

      {/* Input Area */}
      <ChatInput
        value={input}
        onChange={handleInputChange}
        onSend={onSend}
        onClearChat={onClearChat}
        onClearContext={onClearContext}
        disabled={isTyping}
        isTyping={isTyping}
        currentParrotId={currentParrot.id}
        onParrotChange={onParrotChange}
      />

      {/* Clear Chat Confirmation Dialog */}
      <ConfirmDialog
        open={clearDialogOpen}
        onOpenChange={setClearDialogOpen}
        title={t("ai.clear-chat")}
        confirmLabel={t("common.confirm")}
        description={t("ai.clear-chat-confirm")}
        cancelLabel={t("common.cancel")}
        onConfirm={onClearChat}
        confirmVariant="destructive"
      />
    </div>
  );
}

// ============================================================
// MAIN AI CHAT PAGE
// ============================================================
const AIChat = () => {
  const chatHook = useChat();
  const aiChat = useAIChat();

  // Local state
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [isThinking, setIsThinking] = useState(false);

  const [clearDialogOpen, setClearDialogOpen] = useState(false);
  const [memoQueryResults, setMemoQueryResults] = useState<MemoQueryResultData[]>([]);
  const [scheduleQueryResults, setScheduleQueryResults] = useState<ScheduleQueryResultData[]>([]);

  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const messageIdRef = useRef(0);
  const lastAssistantMessageIdRef = useRef<string | null>(null);
  const streamingContentRef = useRef<string>("");

  // Get current conversation from context
  const {
    currentConversation,
    conversations,
    createConversation,
    selectConversation,
    addMessage,
    updateMessage,
    addReferencedMemos,
    addContextSeparator,
    clearMessages,
    setViewMode,
  } = aiChat;

  const { t } = useTranslation();

  // Determine current parrot type from conversation
  const currentParrotType = currentConversation?.parrotId;
  // Get i18n version of the current parrot
  const currentParrot = useMemo(() => {
    if (!currentParrotType) return null;
    const agent = PARROT_AGENTS[currentParrotType] || PARROT_AGENTS[ParrotAgentType.DEFAULT];
    return getLocalizedParrot(agent, t);
  }, [currentParrotType, t]);

  // Get messages from current conversation
  const items = currentConversation?.messages || [];

  // Clear timeout on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const resetTypingState = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    setIsTyping(false);
  }, []);



  // Handle back to hub
  const handleBackToHub = useCallback(() => {
    setViewMode("hub");
  }, [setViewMode]);

  // Handle parrot chat with callbacks
  const handleParrotChat = useCallback(
    async (conversationId: string, parrotId: ParrotAgentType, userMessage: string, conversationIdNum: number) => {
      setIsTyping(true);
      setIsThinking(true);
      setMemoQueryResults([]);
      setScheduleQueryResults([]);
      const _messageId = ++messageIdRef.current;

      // Use user message directly - system context is handled by backend
      const explicitMessage = userMessage;

      try {
        await chatHook.stream({
          message: explicitMessage,
          conversationId: conversationIdNum,
          agentType: parrotId,
          userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        } as ParrotChatParams, {
          onThinking: (msg) => {
            if (lastAssistantMessageIdRef.current) {
              updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                content: msg,
              });
            }
          },
          onToolUse: (toolName) => {
            if (lastAssistantMessageIdRef.current) {
              updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                content: toolName,
              });
            }
          },
          onToolResult: (result) => {
            console.log("[Parrot Tool Result]", result);
          },
          onMemoQueryResult: (result) => {
            if (_messageId === messageIdRef.current) {
              setMemoQueryResults((prev) => [...prev, result]);
              addReferencedMemos(conversationId, result.memos);
            }
          },
          onScheduleQueryResult: (result) => {
            if (_messageId === messageIdRef.current) {
              // Transform callback result to ScheduleQueryResultData format
              const transformedResult: ScheduleQueryResultData = {
                schedules: result.schedules.map((s) => ({
                  uid: s.uid,
                  title: s.title,
                  startTimestamp: Number(s.startTs),
                  endTimestamp: Number(s.endTs),
                  allDay: s.allDay,
                  location: s.location || undefined,
                  status: s.status,
                })),
                query: "",
                count: result.schedules.length,
                timeRangeDescription: result.timeRangeDescription,
                queryType: result.queryType,
              };
              setScheduleQueryResults((prev) => [...prev, transformedResult]);
            }
          },
          onContent: (content) => {
            console.debug("[AI Chat] onContent callback", {
              contentLength: content.length,
              contentPreview: content.slice(0, 50),
              conversationId,
              lastMessageId: lastAssistantMessageIdRef.current,
            });
            if (lastAssistantMessageIdRef.current) {
              streamingContentRef.current += content;
              updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                content: streamingContentRef.current,
              });
            } else {
              console.warn("[AI Chat] onContent: lastAssistantMessageIdRef.current is null!");
            }
          },
          onDone: () => {
            setIsTyping(false);
            setIsThinking(false);
          },
          onError: (error) => {
            setIsTyping(false);
            setIsThinking(false);
            console.error("[Parrot Error]", error);
            // Add error message to conversation
            if (lastAssistantMessageIdRef.current) {
              updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                content: streamingContentRef.current || t("ai.error-generic") || "Sorry, something went wrong. Please try again.",
                error: true,
              });
            }
          },
        },
        );
      } catch (error) {
        setIsTyping(false);
        setIsThinking(false);
        console.error("[Parrot Chat Error]", error);
      }
    },
    [chatHook, updateMessage, addReferencedMemos],
  );

  const handleSend = useCallback(
    async (messageContent?: string) => {
      const userMessage = (messageContent || input).trim();
      if (!userMessage) return;

      if (isTyping) {
        resetTypingState();
      }

      // Ensure we have a conversation
      let targetConversationId = currentConversation?.id;
      let targetParrotId = currentConversation?.parrotId;

      if (!targetConversationId) {
        // No active conversation - need to create one
        // Use current parrot if available (user selected agent from hub), otherwise DEFAULT
        const fallbackParrotId = currentParrot?.id || ParrotAgentType.DEFAULT;

        // Check for existing conversation with same parrotId
        const existingConversation = conversations.find((c) => c.parrotId === fallbackParrotId);
        if (existingConversation) {
          targetConversationId = existingConversation.id;
          targetParrotId = existingConversation.parrotId;
          selectConversation(existingConversation.id);
        } else {
          targetConversationId = createConversation(fallbackParrotId);
          targetParrotId = fallbackParrotId;
        }
      }

      // At this point we must have both conversationId and parrotId
      if (!targetConversationId || !targetParrotId) {
        console.error("[AI Chat] Failed to determine conversation or parrot");
        return;
      }

      // Add user message
      addMessage(targetConversationId, {
        role: "user",
        content: userMessage,
      });

      // Special handling for cutting line (context separator)
      if (userMessage === "---") {
        setInput("");
        // Optimization: UI already shows the message, backend will persist SEPARATOR.
        // We trigger handleParrotChat to ensure the backend sees the command.
        const targetConversationIdNum = parseInt(targetConversationId, 10);
        await handleParrotChat(targetConversationId, targetParrotId, userMessage, targetConversationIdNum);
        return;
      }

      // Add empty assistant message that will be filled during streaming
      const newMessage = {
        role: "assistant" as const,
        content: "",
      };
      const assistantMessageId = addMessage(targetConversationId, newMessage);

      // Store the new message ID for streaming updates
      lastAssistantMessageIdRef.current = assistantMessageId;

      console.debug("[AI Chat] handleSend: messages added", {
        targetConversationId,
        targetParrotId,
        assistantMessageId,
        refValue: lastAssistantMessageIdRef.current,
      });

      // Reset streaming content ref
      streamingContentRef.current = "";

      setInput("");

      // Backend now handles context building, including SEPARATOR filtering
      // We only need to pass the conversationId
      const targetConversationIdNum = parseInt(targetConversationId, 10);
      const conversationIdNum = isNaN(targetConversationIdNum) ? 0 : targetConversationIdNum;

      console.debug("[AI Chat] handleSend: calling handleParrotChat", {
        targetConversationId,
        targetConversationIdNum: conversationIdNum,
        targetParrotId,
      });

      // Execute chat with explicit conversationId and parrotId
      // Backend will build history from database, filtering SEPARATOR messages
      await handleParrotChat(targetConversationId, targetParrotId, userMessage, conversationIdNum);
    },
    [
      input,
      isTyping,
      currentConversation,
      currentParrot,
      addMessage,
      conversations,
      selectConversation,
      createConversation,
      handleParrotChat,
      resetTypingState,
      items,
    ],
  );

  const handleClearChat = useCallback(() => {
    if (currentConversation) {
      clearMessages(currentConversation.id);
    }
    setClearDialogOpen(false);
  }, [currentConversation, clearMessages]);

  const handleClearContext = useCallback((trigger: "manual" | "auto" | "shortcut" = "manual") => {
    if (currentConversation) {
      // Only call addContextSeparator - it handles both API call and UI refresh
      // No need to send "---" as a message, that would create a duplicate user message
      addContextSeparator(currentConversation.id, trigger);

      toast.success(t("ai.context-cleared-toast"), {
        duration: 2000,
        icon: "✂️",
        className: "dark:bg-zinc-800 dark:border-zinc-700",
      });
    }
  }, [currentConversation, addContextSeparator, t]);

  const handleParrotChange = useCallback(
    (parrot: ParrotAgentI18n | null) => {
      if (!parrot) {
        handleBackToHub();
        return;
      }
      // Check for existing conversation with same parrotId
      const existingConversation = conversations.find((c) => c.parrotId === parrot.id);
      if (existingConversation) {
        selectConversation(existingConversation.id);
      } else {
        createConversation(parrot.id, parrot.displayName);
      }
    },
    [conversations, createConversation, handleBackToHub, selectConversation],
  );

  // Handle custom event for sending messages (from suggested prompts)
  useEffect(() => {
    const handler = (e: CustomEvent<string>) => {
      setInput(e.detail);
      setTimeout(() => {
        setInput("");
        handleSend(e.detail);
      }, 100);
    };

    window.addEventListener("aichat-send-message", handler as EventListener);
    return () => {
      window.removeEventListener("aichat-send-message", handler as EventListener);
    };
  }, [handleSend]);

  // Keyboard shortcuts: Cmd/Ctrl+K to clear context
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd+K or Ctrl+K
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        // Only clear context if in chat view
        if (currentConversation && viewMode === "chat") {
          handleClearContext("shortcut");
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [currentConversation, handleClearContext]);

  // View mode determination
  const viewMode = aiChat.state.viewMode;

  // ============================================================
  // RENDER
  // ============================================================
  return viewMode === "hub" || !currentParrot ? (
    <ParrotHub />
  ) : (
    <ChatView
      currentParrot={currentParrot}
      input={input}
      setInput={setInput}
      onSend={handleSend}
      isTyping={isTyping}
      isThinking={isThinking}
      clearDialogOpen={clearDialogOpen}
      setClearDialogOpen={setClearDialogOpen}
      onClearChat={handleClearChat}
      onClearContext={handleClearContext}
      onBackToHub={handleBackToHub}
      memoQueryResults={memoQueryResults}
      scheduleQueryResults={scheduleQueryResults}
      items={items}
      onParrotChange={handleParrotChange}
    />
  );
};

export default AIChat;
