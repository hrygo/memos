import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import copy from "copy-to-clipboard";

import { MemoQueryResult } from "@/components/AIChat/MemoQueryResult";
import { ParrotQuickActions } from "@/components/AIChat/ParrotQuickActions";
import { ChatHeader } from "@/components/AIChat/ChatHeader";
import { ChatMessages } from "@/components/AIChat/ChatMessages";
import { ChatInput } from "@/components/AIChat/ChatInput";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useChatWithMemos } from "@/hooks/useAIQueries";
import useMediaQuery from "@/hooks/useMediaQuery";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import type { MemoQueryResultData } from "@/types/parrot";
import type { ChatItem, ConversationMessage, ContextSeparator } from "@/types/aichat";
import { ParrotAgent, PARROT_AGENTS, PARROT_THEMES, PARROT_ICONS, ParrotAgentType } from "@/types/parrot";

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
function HubView({ onSelectParrot }: { onSelectParrot: (parrot: ParrotAgent) => void }) {
  const { t } = useTranslation();
  const availableParrots = Object.values(PARROT_AGENTS).filter((p) => p.available);

  const handleSuggestedPrompt = (query: string, parrot: ParrotAgent) => {
    onSelectParrot(parrot);
    setTimeout(() => {
      window.dispatchEvent(new CustomEvent('aichat-send-message', { detail: query }));
    }, 100);
  };

  return (
    <div className="w-full h-full flex flex-col bg-[#F8F5F0] dark:bg-zinc-900">
      {/* Header */}
      <div className="px-4 md:px-8 py-4 border-b border-zinc-200/50 dark:border-zinc-800">
        <h1 className="text-lg md:text-xl font-semibold text-zinc-900 dark:text-zinc-100">
          {t("ai.parrot.select-agent")}
        </h1>
      </div>

      {/* Agent Cards Grid */}
      <div className="flex-1 overflow-auto p-3 md:p-6">
        <div className="max-w-4xl mx-auto">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
            {availableParrots.map((parrot) => {
              const parrotTheme = PARROT_THEMES[parrot.id] || PARROT_THEMES.DEFAULT;
              const icon = PARROT_ICONS[parrot.id] || parrot.icon;

              return (
                <button
                  key={parrot.id}
                  onClick={() => handleSuggestedPrompt("", parrot)}
                  className={cn(
                    "group relative w-full text-left rounded-xl border transition-all duration-200",
                    "hover:shadow-md hover:scale-[1.01] active:scale-[0.99]",
                    parrotTheme.cardBg,
                    parrotTheme.cardBorder
                  )}
                >
                  <div className="p-3 md:p-4 flex items-start gap-3">
                    {/* Icon */}
                    <div className={cn(
                      "w-10 h-10 md:w-11 md:h-11 rounded-xl flex items-center justify-center text-xl md:text-2xl shrink-0",
                      parrotTheme.iconBg
                    )}>
                      <span>{icon}</span>
                    </div>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <h3 className="text-sm md:text-base font-semibold text-zinc-900 dark:text-zinc-100 truncate">
                          {parrot.displayName}
                        </h3>
                        <span className={cn(
                          "text-xs px-1.5 py-0.5 rounded-md font-medium shrink-0",
                          parrotTheme.iconBg,
                          parrotTheme.iconText
                        )}>
                          {parrot.id === "MEMO" && t("ai.parrot.memo-tagline")}
                          {parrot.id === "SCHEDULE" && t("ai.parrot.schedule-tagline")}
                          {parrot.id === "AMAZING" && t("ai.parrot.amazing-tagline")}
                          {parrot.id === "CREATIVE" && t("ai.parrot.creative-tagline")}
                          {parrot.id === "DEFAULT" && "RAG"}
                        </span>
                      </div>

                      <p className="text-xs md:text-sm text-zinc-500 dark:text-zinc-400 line-clamp-2">
                        {parrot.description}
                      </p>

                      {/* Suggested Prompts */}
                      <div className="mt-2 space-y-1">
                        {(parrot.examplePrompts || []).slice(0, 2).map((prompt, idx) => (
                          <button
                            key={idx}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleSuggestedPrompt(prompt, parrot);
                            }}
                            className="block w-full text-left px-2 py-1 rounded-lg text-xs border border-zinc-200/50 dark:border-zinc-700/50 hover:bg-white/50 dark:hover:bg-zinc-800/50 transition-colors truncate"
                          >
                            {prompt}
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>
                </button>
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
  currentParrot: ParrotAgent;
  input: string;
  setInput: (value: string) => void;
  onSend: () => void;
  isTyping: boolean;
  clearDialogOpen: boolean;
  setClearDialogOpen: (open: boolean) => void;
  onClearChat: () => void;
  onClearContext: () => void;
  onBackToHub: () => void;
  memoQueryResults: MemoQueryResultData[];
  items: ChatItem[];
  onParrotChange: (parrot: ParrotAgent | null) => void;
}

function ChatView({
  currentParrot,
  input,
  setInput,
  onSend,
  isTyping,
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
      <div className={cn("w-14 h-14 md:w-16 md:h-16 rounded-2xl flex items-center justify-center text-2xl md:text-3xl mb-3", theme.iconBg)}>
        {currentIcon}
      </div>
      <h3 className="text-lg md:text-xl font-semibold text-zinc-900 dark:text-zinc-100 mb-2">
        Hi, I'm {currentParrot.displayName}
      </h3>
      <p className="text-sm text-zinc-500 dark:text-zinc-400 max-w-md mb-4">
        {currentParrot.description}
      </p>

      {currentParrot.examplePrompts && currentParrot.examplePrompts.length > 0 && (
        <div className="flex flex-wrap gap-2 justify-center">
          {currentParrot.examplePrompts.slice(0, 3).map((prompt, idx) => (
            <button
              key={idx}
              onClick={() => {
                setInput(prompt);
                setTimeout(() => {
                  setInput("");
                  onSend();
                }, 100);
              }}
              className={cn(
                "px-3 py-2 rounded-xl text-sm border transition-colors",
                theme.inputBg,
                theme.inputBorder,
                theme.iconText,
                "hover:opacity-80"
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
    <div className={cn(
      "w-full h-full flex flex-col relative",
      theme.bgLight,
      theme.bgDark
    )}>
      {/* Desktop Header */}
      {md && (
        <ChatHeader
          parrot={currentParrot}
          isThinking={false}
          onBack={onBackToHub}
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
  const { t } = useTranslation();
  const md = useMediaQuery("md");
  const chatHook = useChatWithMemos();
  const aiChat = useAIChat();

  // Local state
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [clearDialogOpen, setClearDialogOpen] = useState(false);
  const [memoQueryResults, setMemoQueryResults] = useState<MemoQueryResultData[]>([]);

  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const messageIdRef = useRef(0);

  // Get current conversation from context
  const { currentConversation, createConversation, addMessage, updateMessage, addReferencedMemos, addContextSeparator, clearMessages, setViewMode } = aiChat;

  // Determine current parrot from conversation or default
  const currentParrot = currentConversation
    ? (PARROT_AGENTS[currentConversation.parrotId] || PARROT_AGENTS[ParrotAgentType.DEFAULT])
    : null;

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
  const handleParrotSelect = useCallback((parrot: ParrotAgent) => {
    createConversation(parrot.id, parrot.displayName);
  }, [createConversation]);

  // Handle back to hub
  const handleBackToHub = useCallback(() => {
    setViewMode("hub");
  }, [setViewMode]);

  // Handle parrot chat with callbacks
  const handleParrotChat = useCallback(async (userMessage: string, history: string[]) => {
    if (!currentConversation || !currentParrot) {
      console.warn("[Parrot] No active conversation or parrot");
      return;
    }

    setIsTyping(true);
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
            const lastItem = items[items.length - 1];
            if (lastItem && isConversationMessage(lastItem) && lastItem.id) {
              updateMessage(currentConversation.id, lastItem.id, {
                content: (lastItem.content || "") + content,
              });
            }
          },
          onDone: () => {
            setIsTyping(false);
          },
          onError: (error) => {
            setIsTyping(false);
            console.error("[Parrot Error]", error);
          },
        },
      );
    } catch (error) {
      setIsTyping(false);
      console.error("[Parrot Chat Error]", error);
    }
  }, [currentConversation, currentParrot, chatHook, items, updateMessage, addReferencedMemos]);

  const handleSend = useCallback(async (messageContent?: string) => {
    const userMessage = (messageContent || input).trim();
    if (!userMessage) return;

    if (isTyping) {
      resetTypingState();
    }

    // Ensure we have a conversation
    if (!currentConversation) {
      createConversation(ParrotAgentType.DEFAULT);
      return;
    }

    // Add user message
    addMessage(currentConversation.id, {
      role: "user",
      content: userMessage,
    });

    // Add empty assistant message that will be filled during streaming
    addMessage(currentConversation.id, {
      role: "assistant",
      content: "",
    });

    setInput("");

    const contextMessages = items.filter(isConversationMessage);
    const history = contextMessages.map((m) => m.content);

    if (currentParrot) {
      await handleParrotChat(userMessage, history);
    }
  }, [input, isTyping, currentConversation, currentParrot, addMessage, createConversation, handleParrotChat, items, resetTypingState]);

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

  const handleParrotChange = useCallback((parrot: ParrotAgent | null) => {
    if (!parrot) {
      handleBackToHub();
      return;
    }
    createConversation(parrot.id, parrot.displayName);
  }, [createConversation, handleBackToHub]);

  // Handle custom event for sending messages (from suggested prompts)
  useEffect(() => {
    const handler = (e: CustomEvent<string>) => {
      setInput(e.detail);
      setTimeout(() => {
        setInput("");
        handleSend(e.detail);
      }, 100);
    };

    window.addEventListener('aichat-send-message', handler as EventListener);
    return () => {
      window.removeEventListener('aichat-send-message', handler as EventListener);
    };
  }, [handleSend]);

  // View mode determination
  const viewMode = currentConversation ? "chat" : "hub";

  // ============================================================
  // RENDER
  // ============================================================
  return viewMode === "hub" || !currentParrot ? (
    <HubView onSelectParrot={handleParrotSelect} />
  ) : (
    <ChatView
      currentParrot={currentParrot}
      input={input}
      setInput={setInput}
      onSend={handleSend}
      isTyping={isTyping}
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
