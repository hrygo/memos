import copy from "copy-to-clipboard";
import { ChevronLeft } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ChatHeader } from "@/components/AIChat/ChatHeader";
import { ChatInput } from "@/components/AIChat/ChatInput";
import { ChatMessages } from "@/components/AIChat/ChatMessages";
import { MemoQueryResult } from "@/components/AIChat/MemoQueryResult";
import { ParrotQuickActions } from "@/components/AIChat/ParrotQuickActions";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useAIChat } from "@/contexts/AIChatContext";
import { useChatWithMemos } from "@/hooks/useAIQueries";
import useMediaQuery from "@/hooks/useMediaQuery";
import type { ParrotAgentI18n } from "@/hooks/useParrots";
import { getLocalizedParrot, useAvailableParrots } from "@/hooks/useParrots";
import { cn } from "@/lib/utils";
import type { ChatItem, ContextSeparator, ConversationMessage } from "@/types/aichat";
import type { MemoQueryResultData } from "@/types/parrot";
import { PARROT_AGENTS, PARROT_ICONS, PARROT_THEMES, ParrotAgentType } from "@/types/parrot";

// Helper function to check if item is ContextSeparator
function isContextSeparator(item: ChatItem): item is ContextSeparator {
  return "type" in item && item.type === "context-separator";
}

// Helper function to check if item is ConversationMessage
function isConversationMessage(item: ChatItem): item is ConversationMessage {
  return !isContextSeparator(item);
}

// ============================================================
// HUB VIEW - Agent Selection (Accessible when no conversation)
// ============================================================
interface HubViewProps {
  onSelectParrot: (parrot: ParrotAgentI18n) => void;
  isCreating?: boolean;
}

function HubView({ onSelectParrot, isCreating = false }: HubViewProps) {
  const { t } = useTranslation();
  const availableParrots = useAvailableParrots();

  const handleSuggestedPrompt = (query: string, parrot: ParrotAgentI18n) => {
    if (isCreating) return;
    onSelectParrot(parrot);
    setTimeout(() => {
      window.dispatchEvent(new CustomEvent("aichat-send-message", { detail: query }));
    }, 100);
  };

  return (
    <div className="w-full h-full flex flex-col bg-zinc-50 dark:bg-zinc-900">
      {/* Header */}
      <div className="px-4 md:px-8 py-4 border-b border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900">
        <h1 className="text-lg md:text-xl font-semibold text-zinc-900 dark:text-zinc-100">{t("ai.parrot.select-agent")}</h1>
      </div>

      {/* Agent Cards Grid */}
      <div className="flex-1 overflow-auto p-3 md:p-6">
        <div className="max-w-4xl mx-auto">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
            {availableParrots.map((parrot) => {
              const parrotTheme = PARROT_THEMES[parrot.id] || PARROT_THEMES.DEFAULT;
              const icon = PARROT_ICONS[parrot.id] || parrot.icon;

              return (
                <div
                  key={parrot.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => handleSuggestedPrompt("", parrot)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      handleSuggestedPrompt("", parrot);
                    }
                  }}
                  className={cn(
                    "w-full text-left rounded-xl border transition-all duration-200",
                    "hover:shadow-sm active:shadow-none",
                    "focus:outline-none focus:ring-2 focus:ring-zinc-500 focus:ring-offset-2",
                    "cursor-pointer",
                    isCreating && "opacity-50 cursor-not-allowed",
                    "relative overflow-hidden group",
                    parrotTheme.cardBg,
                    parrotTheme.cardBorder,
                  )}
                >
                  {parrot.backgroundImage && (
                    <>
                      <div
                        className="absolute inset-0 z-0 bg-cover bg-center bg-no-repeat opacity-20 dark:opacity-10 pointer-events-none transition-transform duration-500 group-hover:scale-105"
                        style={{ backgroundImage: `url(${parrot.backgroundImage})` }}
                      />
                    </>
                  )}
                  <div className="p-4 flex items-start gap-3 relative z-10">
                    {/* Icon - transparent background */}
                    <div className="w-11 h-11 rounded-xl flex items-center justify-center text-xl shrink-0">
                      {icon.startsWith("/") ? (
                        <img src={icon} alt={parrot.displayName} className="w-10 h-10 object-contain" />
                      ) : (
                        <span>{icon}</span>
                      )}
                    </div>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-baseline gap-2 mb-1">
                        <h3 className="text-base font-semibold text-zinc-900 dark:text-zinc-100">{parrot.displayName}</h3>
                        <span className="text-xs text-zinc-400 dark:text-zinc-500">{parrot.displayNameAlt}</span>
                      </div>

                      <p className="text-sm text-zinc-500 dark:text-zinc-400 line-clamp-2 mb-2">{parrot.description}</p>

                      {/* Suggested Prompts - clean style */}
                      <div className="space-y-1.5">
                        {(parrot.examplePrompts || []).slice(0, 2).map((prompt, idx) => (
                          <button
                            key={idx}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleSuggestedPrompt(prompt, parrot);
                            }}
                            disabled={isCreating}
                            className={cn(
                              "block w-full text-left px-3 py-2 rounded-lg text-xs border",
                              "hover:bg-zinc-50 dark:hover:bg-zinc-800",
                              "border-zinc-200 dark:border-zinc-700",
                              "text-zinc-700 dark:text-zinc-300",
                              "disabled:opacity-50 disabled:cursor-not-allowed",
                              "transition-colors",
                            )}
                          >
                            {prompt}
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

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
  items,
  onParrotChange,
}: ChatViewProps) {
  const { t } = useTranslation();
  const md = useMediaQuery("md");
  const theme = PARROT_THEMES[currentParrot.id] || PARROT_THEMES.DEFAULT;

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
      <div className="w-14 h-14 md:w-16 md:h-16 rounded-2xl flex items-center justify-center text-2xl md:text-3xl mb-3">
        {currentIcon.startsWith("/") ? (
          <img src={currentIcon} alt={currentParrot.displayName} className="w-12 h-12 md:w-14 md:h-14 object-contain" />
        ) : (
          currentIcon
        )}
      </div>
      <h3 className="text-lg md:text-xl font-semibold text-zinc-900 dark:text-zinc-100 mb-2">
        Hi, I'm {currentParrot.displayName}
        <span className="text-sm text-zinc-400 dark:text-zinc-500 ml-2">{currentParrot.displayNameAlt}</span>
      </h3>
      <p className="text-sm text-zinc-500 dark:text-zinc-400 max-w-md mb-4">{currentParrot.description}</p>

      {currentParrot.examplePrompts && currentParrot.examplePrompts.length > 0 && (
        <div className="flex flex-wrap gap-2 justify-center">
          {currentParrot.examplePrompts.slice(0, 3).map((prompt, idx) => (
            <button
              key={idx}
              onClick={() => {
                onSend(prompt);
              }}
              className={cn(
                "px-3 py-2 rounded-xl text-sm border transition-colors cursor-pointer",
                theme.inputBg,
                theme.inputBorder,
                theme.iconText,
                "hover:opacity-80",
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
      {/* Mobile Header */}
      {!md && (
        <header className="flex items-center gap-2 px-3 py-2 border-b border-zinc-200 dark:border-zinc-800 bg-white/50 dark:bg-zinc-950/50 backdrop-blur-sm">
          <button
            onClick={onBackToHub}
            className="p-2 -ml-2 text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100 transition-colors"
            aria-label="Go back to hub"
          >
            <ChevronLeft className="w-5 h-5" />
          </button>
          {currentIcon.startsWith("/") ? (
            <img src={currentIcon} alt={currentParrot.displayName} className="w-6 h-6 object-contain" />
          ) : (
            <span className="text-xl">{currentIcon}</span>
          )}
          <span className="font-medium text-zinc-900 dark:text-zinc-100">{currentParrot.displayName}</span>
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
      >
        {/* Welcome message */}
        {items.length === 0 && welcomeMessage}

        {/* Memo Query Results */}
        {memoQueryResults.map((result, index) => (
          <div key={index} className="max-w-3xl mx-auto mb-4">
            <MemoQueryResult result={result} />
          </div>
        ))}
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
        showQuickActions={true}
        quickActions={
          <div className="mb-2 md:mb-3">
            <ParrotQuickActions
              currentParrot={currentParrot}
              onParrotChange={(parrot) => {
                if (parrot) {
                  onParrotChange(parrot);
                } else {
                  onBackToHub();
                }
              }}
              disabled={isTyping}
            />
          </div>
        }
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
  const chatHook = useChatWithMemos();
  const aiChat = useAIChat();

  // Local state
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [isThinking, setIsThinking] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [clearDialogOpen, setClearDialogOpen] = useState(false);
  const [memoQueryResults, setMemoQueryResults] = useState<MemoQueryResultData[]>([]);

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

  // Handle starting a new chat with a parrot
  const handleParrotSelect = useCallback(
    async (parrot: ParrotAgentI18n) => {
      if (isCreating) return;

      // Check for existing conversation with same parrotId
      const existingConversation = conversations.find((c) => c.parrotId === parrot.id);
      console.log(
        "[handleParrotSelect] parrotId:",
        parrot.id,
        "existingConversation:",
        existingConversation,
        "all conversations:",
        conversations,
      );
      if (existingConversation) {
        selectConversation(existingConversation.id);
        return;
      }

      setIsCreating(true);
      try {
        createConversation(parrot.id, parrot.displayName);
      } finally {
        // Small delay to ensure state is set
        setTimeout(() => setIsCreating(false), 300);
      }
    },
    [createConversation, isCreating, conversations, selectConversation],
  );

  // Handle back to hub
  const handleBackToHub = useCallback(() => {
    setViewMode("hub");
  }, [setViewMode]);

  // Handle parrot chat with callbacks
  const handleParrotChat = useCallback(
    async (userMessage: string, history: string[]) => {
      if (!currentConversation || !currentParrot) {
        console.warn("[Parrot] No active conversation or parrot");
        return;
      }

      setIsTyping(true);
      setIsThinking(true);
      setMemoQueryResults([]);
      const _messageId = ++messageIdRef.current;

      try {
        await chatHook.stream(
          {
            message: userMessage,
            history,
            agentType: currentParrot.id,
            userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
          },
          {
            onThinking: (msg) => {
              console.log("[Parrot Thinking]", msg);
            },
            onToolUse: (toolName) => {
              console.log("[Parrot Tool Use]", toolName);
            },
            onToolResult: (result) => {
              console.log("[Parrot Tool Result]", result);
            },
            onMemoQueryResult: (result) => {
              if (_messageId === messageIdRef.current) {
                setMemoQueryResults((prev) => [...prev, result]);
                addReferencedMemos(currentConversation.id, result.memos);
              }
            },
            onContent: (content) => {
              if (lastAssistantMessageIdRef.current) {
                streamingContentRef.current += content;
                updateMessage(currentConversation.id, lastAssistantMessageIdRef.current, {
                  content: streamingContentRef.current,
                });
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
                updateMessage(currentConversation.id, lastAssistantMessageIdRef.current, {
                  content: streamingContentRef.current || "Sorry, something went wrong. Please try again.",
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
    [currentConversation, currentParrot, chatHook, updateMessage, addReferencedMemos],
  );

  const handleSend = useCallback(
    async (messageContent?: string) => {
      const userMessage = (messageContent || input).trim();
      if (!userMessage) return;

      if (isTyping) {
        resetTypingState();
      }

      // Ensure we have a conversation
      if (!currentConversation) {
        // Check for existing DEFAULT conversation
        const existingConversation = conversations.find((c) => c.parrotId === ParrotAgentType.DEFAULT);
        if (existingConversation) {
          selectConversation(existingConversation.id);
        } else {
          createConversation(ParrotAgentType.DEFAULT);
        }
        return;
      }

      // Add user message
      addMessage(currentConversation.id, {
        role: "user",
        content: userMessage,
      });

      // Add empty assistant message that will be filled during streaming
      const newMessage = {
        role: "assistant" as const,
        content: "",
      };
      const assistantMessageId = addMessage(currentConversation.id, newMessage);

      // Store the new message ID for streaming updates
      lastAssistantMessageIdRef.current = assistantMessageId;

      // Reset streaming content ref
      streamingContentRef.current = "";

      setInput("");

      // Build history: only include messages after the last context separator
      const lastSeparatorIndex = items.findLastIndex((item) => isContextSeparator(item));
      const messagesAfterSeparator = lastSeparatorIndex === -1 ? items : items.slice(lastSeparatorIndex + 1);
      const contextMessages = messagesAfterSeparator.filter(isConversationMessage);
      const history = contextMessages.map((m) => m.content);

      if (currentParrot) {
        await handleParrotChat(userMessage, history);
      }
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
    ],
  );

  const handleClearChat = useCallback(() => {
    if (currentConversation) {
      clearMessages(currentConversation.id);
    }
    setClearDialogOpen(false);
  }, [currentConversation, clearMessages]);

  const handleClearContext = useCallback(() => {
    if (currentConversation) {
      addContextSeparator(currentConversation.id);
    }
  }, [currentConversation, addContextSeparator]);

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

  // View mode determination
  const viewMode = currentConversation ? "chat" : "hub";

  // ============================================================
  // RENDER
  // ============================================================
  return viewMode === "hub" || !currentParrot ? (
    <HubView onSelectParrot={handleParrotSelect} isCreating={isCreating} />
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
      items={items}
      onParrotChange={handleParrotChange}
    />
  );
};

export default AIChat;
