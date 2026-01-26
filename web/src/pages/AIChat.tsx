import copy from "copy-to-clipboard";
import { X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { AmazingInsightCard } from "@/components/AIChat/AmazingInsightCard";
import { ChatHeader } from "@/components/AIChat/ChatHeader";
import { ChatInput } from "@/components/AIChat/ChatInput";
import { ChatMessages } from "@/components/AIChat/ChatMessages";
import { ParrotHub } from "@/components/AIChat/ParrotHub";
import { PartnerGreeting } from "@/components/AIChat/PartnerGreeting";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useAIChat } from "@/contexts/AIChatContext";
import { useChat } from "@/hooks/useAIQueries";
import { useCapabilityRouter } from "@/hooks/useCapabilityRouter";
import useMediaQuery from "@/hooks/useMediaQuery";
import type { ChatItem } from "@/types/aichat";
import { CapabilityStatus, CapabilityType, capabilityToParrotAgent } from "@/types/capability";
import type { MemoQueryResultData, ScheduleQueryResultData } from "@/types/parrot";
import { ParrotAgentType } from "@/types/parrot";

// ============================================================
// UNIFIED CHAT VIEW - 单一对话视图
// ============================================================
interface UnifiedChatViewProps {
  input: string;
  setInput: (value: string) => void;
  onSend: (messageContent?: string) => void;
  onNewChat: () => void;
  isTyping: boolean;
  isThinking: boolean;
  clearDialogOpen: boolean;
  setClearDialogOpen: (open: boolean) => void;
  onClearChat: () => void;
  onClearContext: () => void;
  memoQueryResults: MemoQueryResultData[];
  scheduleQueryResults: ScheduleQueryResultData[];
  items: ChatItem[];
  currentCapability: CapabilityType;
  capabilityStatus: CapabilityStatus;
  recentMemoCount?: number;
  upcomingScheduleCount?: number;
}

function UnifiedChatView({
  input,
  setInput,
  onSend,
  onNewChat,
  isTyping,
  isThinking,
  clearDialogOpen,
  setClearDialogOpen,
  onClearChat,
  onClearContext,
  memoQueryResults,
  scheduleQueryResults,
  items,
  currentCapability,
  capabilityStatus,
  recentMemoCount,
  upcomingScheduleCount,
}: UnifiedChatViewProps) {
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

  return (
    <div className="w-full h-full flex flex-col relative bg-white dark:bg-zinc-900">
      {/* Desktop Header */}
      {md && <ChatHeader currentCapability={currentCapability} capabilityStatus={capabilityStatus} isThinking={isThinking} />}

      {/* Messages Area with Welcome */}
      <ChatMessages
        items={items}
        isTyping={isTyping}
        currentParrotId={ParrotAgentType.AMAZING}
        onCopyMessage={handleCopyMessage}
        onDeleteMessage={handleDeleteMessage}
        amazingInsightCard={
          currentCapability === CapabilityType.AMAZING && (memoQueryResults.length > 0 || scheduleQueryResults.length > 0) ? (
            <AmazingInsightCard memos={memoQueryResults[0]?.memos ?? []} schedules={scheduleQueryResults[0]?.schedules ?? []} />
          ) : undefined
        }
      >
        {/* Welcome message - 统一入口，示例提问直接发送 */}
        {items.length === 0 && (
          <PartnerGreeting recentMemoCount={recentMemoCount} upcomingScheduleCount={upcomingScheduleCount} onSendMessage={onSend} />
        )}
      </ChatMessages>

      {/* Input Area */}
      <ChatInput
        value={input}
        onChange={handleInputChange}
        onSend={onSend}
        onNewChat={onNewChat}
        onClearContext={onClearContext}
        onClearChat={() => setClearDialogOpen(true)}
        disabled={isTyping}
        isTyping={isTyping}
        currentParrotId={ParrotAgentType.AMAZING}
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
// CAPABILITY PANEL VIEW - 能力面板视图
// ============================================================
interface CapabilityPanelViewProps {
  currentCapability: CapabilityType;
  capabilityStatus: CapabilityStatus;
  onCapabilitySelect: (capability: CapabilityType) => void;
  onBack: () => void;
}

function CapabilityPanelView({ currentCapability, capabilityStatus, onCapabilitySelect, onBack }: CapabilityPanelViewProps) {
  const md = useMediaQuery("md");
  const { t } = useTranslation();

  return (
    <div className="w-full h-full flex flex-col relative bg-white dark:bg-zinc-900">
      {/* Mobile Sub-Header */}
      {!md && (
        <header className="flex items-center justify-between px-3 py-2 border-b border-zinc-100 dark:border-zinc-800/60 bg-white/80 dark:bg-zinc-900/80 backdrop-blur-md sticky top-0 z-20">
          <button
            onClick={onBack}
            className="flex items-center gap-1.5 px-2 py-1.5 rounded-lg text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-all"
          >
            <X className="w-4 h-4" />
            <span className="text-xs font-medium">{t("common.close") || "Close"}</span>
          </button>
          <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">{t("ai.capability.title") || "我的能力"}</span>
          <div className="w-16" />
        </header>
      )}

      {/* Capability Panel */}
      <ParrotHub currentCapability={currentCapability} capabilityStatus={capabilityStatus} onCapabilitySelect={onCapabilitySelect} />
    </div>
  );
}

// ============================================================
// MAIN AI CHAT PAGE - 重构为单一对话入口
// ============================================================
const AIChat = () => {
  const chatHook = useChat();
  const aiChat = useAIChat();
  const capabilityRouter = useCapabilityRouter();

  // Local state
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [isThinking, setIsThinking] = useState(false);

  const [clearDialogOpen, setClearDialogOpen] = useState(false);
  const [memoQueryResults, setMemoQueryResults] = useState<MemoQueryResultData[]>([]);
  const [scheduleQueryResults, setScheduleQueryResults] = useState<ScheduleQueryResultData[]>([]);
  const [showCapabilityPanel, setShowCapabilityPanel] = useState(false);

  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const messageIdRef = useRef(0);
  const lastAssistantMessageIdRef = useRef<string | null>(null);
  const streamingContentRef = useRef<string>("");

  // Get current conversation and capability from context
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
    state,
    setCurrentCapability,
    setCapabilityStatus,
  } = aiChat;

  const currentCapability = state.currentCapability || CapabilityType.AUTO;
  const capabilityStatus = state.capabilityStatus || "idle";

  // Get messages from current conversation
  const items = currentConversation?.messages || [];

  const { t } = useTranslation();

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

  // Handle parrot chat with callbacks
  const handleParrotChat = useCallback(
    async (conversationId: string, parrotId: ParrotAgentType, userMessage: string, _conversationIdNum: number) => {
      setIsTyping(true);
      setIsThinking(true);
      setCapabilityStatus("thinking");
      setMemoQueryResults([]);
      setScheduleQueryResults([]);
      const _messageId = ++messageIdRef.current;

      const explicitMessage = userMessage;

      try {
        await chatHook.stream(
          {
            message: explicitMessage,
            agentType: parrotId,
            userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
          },
          {
            onThinking: (msg) => {
              if (lastAssistantMessageIdRef.current) {
                updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                  content: msg,
                });
              }
            },
            onToolUse: (toolName) => {
              setCapabilityStatus("processing");
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
              if (lastAssistantMessageIdRef.current) {
                streamingContentRef.current += content;
                updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                  content: streamingContentRef.current,
                });
              }
            },
            onDone: () => {
              setIsTyping(false);
              setIsThinking(false);
              setCapabilityStatus("idle");
            },
            onError: (error) => {
              setIsTyping(false);
              setIsThinking(false);
              setCapabilityStatus("idle");
              console.error("[Parrot Error]", error);
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
        setCapabilityStatus("idle");
        console.error("[Parrot Chat Error]", error);
      }
    },
    [chatHook, updateMessage, addReferencedMemos, setCapabilityStatus, t],
  );

  const handleSend = useCallback(
    async (messageContent?: string) => {
      const userMessage = (messageContent || input).trim();
      if (!userMessage) return;

      if (isTyping) {
        resetTypingState();
      }

      // 智能路由：根据输入内容自动识别能力
      const intentResult = capabilityRouter.route(userMessage, currentCapability);
      const targetCapability = intentResult.capability;

      // 如果识别出不同的能力，切换能力
      if (targetCapability !== currentCapability && targetCapability !== CapabilityType.AUTO) {
        setCurrentCapability(targetCapability);
        console.debug("[AI Chat] Auto-switching capability", {
          from: currentCapability,
          to: targetCapability,
          confidence: intentResult.confidence,
          reasoning: intentResult.reasoning,
        });
      }

      // 确定使用哪个 Agent
      const targetParrotId = capabilityToParrotAgent(targetCapability);

      // Ensure we have a conversation
      let targetConversationId = currentConversation?.id;

      if (!targetConversationId) {
        // No active conversation - create one with AMAZING agent (综合助手)
        // (会话不再绑定特定Agent，能力可以在会话中动态切换)
        const existingConversation = conversations.find((c) => !c.parrotId || c.parrotId === ParrotAgentType.AMAZING);
        if (existingConversation) {
          targetConversationId = existingConversation.id;
          selectConversation(existingConversation.id);
        } else {
          targetConversationId = createConversation(ParrotAgentType.AMAZING);
        }
      }

      if (!targetConversationId) {
        console.error("[AI Chat] Failed to determine conversation");
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
        const targetConversationIdNum = parseInt(targetConversationId, 10);
        await handleParrotChat(targetConversationId, targetParrotId, userMessage, targetConversationIdNum);
        return;
      }

      // Add empty assistant message
      const newMessage = {
        role: "assistant" as const,
        content: "",
      };
      const assistantMessageId = addMessage(targetConversationId, newMessage);
      lastAssistantMessageIdRef.current = assistantMessageId;

      streamingContentRef.current = "";
      setInput("");

      const targetConversationIdNum = parseInt(targetConversationId, 10);
      const conversationIdNum = isNaN(targetConversationIdNum) ? 0 : targetConversationIdNum;

      await handleParrotChat(targetConversationId, targetParrotId, userMessage, conversationIdNum);
    },
    [
      input,
      isTyping,
      currentConversation,
      currentCapability,
      capabilityRouter,
      setCurrentCapability,
      conversations,
      selectConversation,
      createConversation,
      addMessage,
      handleParrotChat,
      resetTypingState,
    ],
  );

  const handleClearChat = useCallback(() => {
    if (currentConversation) {
      clearMessages(currentConversation.id);
    }
    setClearDialogOpen(false);
  }, [currentConversation, clearMessages]);

  const handleNewChat = useCallback(() => {
    createConversation(ParrotAgentType.AMAZING);
  }, [createConversation]);

  const handleClearContext = useCallback(
    (trigger: "manual" | "auto" | "shortcut" = "manual") => {
      if (currentConversation) {
        addContextSeparator(currentConversation.id, trigger);
        toast.success(t("ai.context-cleared-toast"), {
          duration: 2000,
          icon: "✂️",
          className: "dark:bg-zinc-800 dark:border-zinc-700",
        });
      }
    },
    [currentConversation, addContextSeparator, t],
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

  // Keyboard shortcuts: ⌘K clear context, ⌘N new chat, ⌘L clear chat
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;

      switch (e.key.toLowerCase()) {
        case "k":
          e.preventDefault();
          if (currentConversation) {
            handleClearContext("shortcut");
          }
          break;
        case "n":
          e.preventDefault();
          handleNewChat();
          break;
        case "l":
          e.preventDefault();
          if (currentConversation) {
            setClearDialogOpen(true);
          }
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [currentConversation, handleClearContext, handleNewChat]);

  // ============================================================
  // RENDER
  // ============================================================
  return showCapabilityPanel ? (
    <CapabilityPanelView
      currentCapability={currentCapability}
      capabilityStatus={capabilityStatus}
      onCapabilitySelect={(cap) => {
        setCurrentCapability(cap);
        setShowCapabilityPanel(false);
      }}
      onBack={() => setShowCapabilityPanel(false)}
    />
  ) : (
    <UnifiedChatView
      input={input}
      setInput={setInput}
      onSend={handleSend}
      onNewChat={handleNewChat}
      isTyping={isTyping}
      isThinking={isThinking}
      clearDialogOpen={clearDialogOpen}
      setClearDialogOpen={setClearDialogOpen}
      onClearChat={handleClearChat}
      onClearContext={handleClearContext}
      memoQueryResults={memoQueryResults}
      scheduleQueryResults={scheduleQueryResults}
      items={items}
      currentCapability={currentCapability}
      capabilityStatus={capabilityStatus}
    />
  );
};

export default AIChat;
