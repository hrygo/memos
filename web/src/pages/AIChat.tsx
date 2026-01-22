import copy from "copy-to-clipboard";
import {
  BotIcon,
  EraserIcon,
  Loader2,
  MoreHorizontalIcon,
  SendIcon,
  SparklesIcon,
  UserIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import EmptyState from "@/components/AIChat/EmptyState";
import ErrorMessage from "@/components/AIChat/ErrorMessage";
import MessageActions from "@/components/AIChat/MessageActions";
import { ParrotSelector } from "@/components/AIChat/ParrotSelector";
import { ParrotQuickActions } from "@/components/AIChat/ParrotQuickActions";
import { ParrotStatus } from "@/components/AIChat/ParrotStatus";
import { MemoQueryResult } from "@/components/AIChat/MemoQueryResult";
import ThinkingIndicator from "@/components/AIChat/ThinkingIndicator";

import TypingCursor from "@/components/AIChat/TypingCursor";
import ConfirmDialog from "@/components/ConfirmDialog";
import { CodeBlock } from "@/components/MemoContent/CodeBlock";
import MobileHeader from "@/components/MobileHeader";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Textarea } from "@/components/ui/textarea";
import { useChatWithMemos } from "@/hooks/useAIQueries";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { ParrotAgent } from "@/types/parrot";
import type { MemoQueryResultData } from "@/types/parrot";

const STREAM_TIMEOUT = 60000; // 60 seconds timeout

interface Message {
  role: "user" | "assistant";
  content: string;
  error?: boolean;
}

interface ContextSeparator {
  type: "context-separator";
}

type ChatItem = Message | ContextSeparator;

const AIChat = () => {
  const { t } = useTranslation();
  const md = useMediaQuery("md");
  const [input, setInput] = useState("");
  const [items, setItems] = useState<ChatItem[]>([]);
  const [isTyping, setIsTyping] = useState(false);
  const [clearDialogOpen, setClearDialogOpen] = useState(false);
  const [, setErrorMessage] = useState<string | null>(null);
  const [lastUserMessage, setLastUserMessage] = useState("");
  const [contextStartIndex, setContextStartIndex] = useState(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const chatHook = useChatWithMemos();

  // Parrot-related state (Milestone 1)
  const [currentParrot, setCurrentParrot] = useState<ParrotAgent | null>(null);
  const [showParrotSelector, setShowParrotSelector] = useState(false);
  const [parrotSelectorPosition, setParrotSelectorPosition] = useState<{ x: number; y: number } | null>(null);
  const [isParrotThinking, setIsParrotThinking] = useState(false);
  const [memoQueryResults, setMemoQueryResults] = useState<MemoQueryResultData[]>([]);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  // Use ref to optimize frequent content updates
  const currentParrotContentRef = useRef("");
  const updateTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Get actual messages (excluding separators) for API calls
  const getMessagesForContext = useCallback(() => {
    return items.filter((item): item is Message => "role" in item).slice(contextStartIndex) as Message[];
  }, [items, contextStartIndex]);

  const scrollToBottom = () => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  };

  // Clear timeout on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      // Clear content update timer
      if (updateTimerRef.current) {
        clearTimeout(updateTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [items, isTyping]);

  const resetTypingState = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    setIsTyping(false);
  }, []);

  // ============================================================
  // Parrot-related handlers (Milestone 1)
  // ============================================================

  // Handle @ symbol to trigger parrot selector
  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    setInput(value);

    // Only trigger if @ is at the start of the input (to avoid interfering with normal typing)
    if (value === "@" && textareaRef.current) {
      const rect = textareaRef.current.getBoundingClientRect();
      const x = rect.left;
      const y = rect.bottom + window.scrollY;
      setParrotSelectorPosition({ x, y });
      setShowParrotSelector(true);
    } else if (value !== "@" && showParrotSelector) {
      // Close selector if @ symbol is deleted or modified
      setShowParrotSelector(false);
    }
  };

  // Handle parrot selection
  const handleParrotSelect = (parrot: ParrotAgent) => {
    setCurrentParrot(parrot);
    // Remove @ symbol from input
    setInput((prev) => prev.slice(0, -1));
    setShowParrotSelector(false);
  };

  // Handle parrot chat with callbacks
  const handleParrotChat = async (userMessage: string, history: string[]) => {
    if (!currentParrot) {
      // Should not happen, but fallback to default
      console.warn("[Parrot] No parrot selected, using default chat");
      return handleSend(userMessage);
    }

    setIsParrotThinking(true);
    setMemoQueryResults([]);
    const messageId = ++messageIdRef.current;

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
            // Check if this result is for the current message
            if (messageId === messageIdRef.current) {
              setMemoQueryResults((prev) => [...prev, result]);
            }
          },
          onContent: (content) => {
            // Update message content
            setItems((prev) => {
              const newItems = [...prev];
              const lastMessageIndex = newItems.findLastIndex((item) => "role" in item && item.role === "assistant");
              if (lastMessageIndex !== -1 && "content" in newItems[lastMessageIndex]) {
                (newItems[lastMessageIndex] as Message).content += content;
              }
              return newItems;
            });
          },
          onDone: () => {
            setIsParrotThinking(false);
            setIsTyping(false);
          },
          onError: (error) => {
            setIsParrotThinking(false);
            setIsTyping(false);
            console.error("[Parrot Error]", error);
            setErrorMessage(error.message || t("ai.parrot.error-processing"));
          },
        }
      );
    } catch (error) {
      setIsParrotThinking(false);
      setIsTyping(false);
      console.error("[Parrot Chat Error]", error);
      setErrorMessage(t("ai.parrot.error-chat-failed"));
    }
  };

  const handleSend = async (messageContent?: string) => {
    const userMessage = (messageContent || input).trim();
    if (!userMessage) return;

    // If already typing, reset first
    if (isTyping) {
      resetTypingState();
    }

    // ============================================================
    // Parrot routing (Milestone 1)
    // ============================================================
    // Check if a parrot is selected and route to parrot chat
    if (currentParrot) {
      console.log("[Parrot] Routing to", currentParrot.displayName, "for message:", userMessage);
      // Add user message to items
      setItems((prev) => [...prev, { role: "user" as const, content: userMessage }]);
      // Add placeholder for assistant response
      setItems((prev) => [...prev, { role: "assistant" as const, content: "" }]);
      setInput("");
      setIsTyping(true);
      setLastUserMessage(userMessage);

      // Get context messages for history
      const contextMessages = getMessagesForContext();
      const history = contextMessages.map((m) => m.content);

      // Handle with parrot
      await handleParrotChat(userMessage, history);
      return;
    }

    // ============================================================
    // Default chat flow (original logic)
    // ============================================================

    // 原子操作递增消息 ID，避免竞态条件
    const messageId = ++messageIdRef.current;

    setInput("");
    setLastUserMessage(userMessage);
    setErrorMessage(null);
    setItems((prev) => [...prev, { role: "user" as const, content: userMessage }]);
    setIsTyping(true);

    // Track if stream has completed
    let streamCompleted = false;

    // Set timeout to auto-finish if stream doesn't complete
    timeoutRef.current = setTimeout(() => {
      if (!streamCompleted) {
        console.warn("Stream timeout, forcing completion");
        setIsTyping(false);
      }
    }, STREAM_TIMEOUT);

    try {
      const contextMessages = getMessagesForContext();
      const history = contextMessages.map((m) => m.content);
      let currentAssistantMessage = "";
      setItems((prev) => [...prev, { role: "assistant" as const, content: "" }]);

      await chatHook.stream(
        { message: userMessage, history },
        {
          onContent: (content) => {
            currentAssistantMessage += content;
            setItems((prev) => {
              const newItems = [...prev];
              const lastMessageIndex = newItems.findLastIndex((item) => "role" in item && item.role === "assistant");
              if (lastMessageIndex !== -1 && "content" in newItems[lastMessageIndex]) {
                (newItems[lastMessageIndex] as Message).content = currentAssistantMessage;
              }
              return newItems;
            });
          },
          onDone: () => {
            streamCompleted = true;
            resetTypingState();
          },
          onError: (err) => {
            streamCompleted = true;
            console.error("Chat error:", err);
            resetTypingState();
            setErrorMessage(err.message || t("ai.error-title"));
            setItems((prev) => {
              const newItems = [...prev];
              const lastMessageIndex = newItems.findLastIndex((item) => "role" in item && item.role === "assistant");
              if (lastMessageIndex !== -1) {
                (newItems[lastMessageIndex] as Message).content = t("ai.error-title");
                (newItems[lastMessageIndex] as Message).error = true;
              }
              return newItems;
            });
          },
        },
      );
    } catch (_error) {
      streamCompleted = true;
      resetTypingState();
      setErrorMessage(t("ai.error-title"));
    }
  };

  const handleRetry = () => {
    if (lastUserMessage) {
      setItems((prev) => prev.filter((item) => !("role" in item && item.role === "assistant" && item.error)));
      setErrorMessage(null);
      handleSend(lastUserMessage);
    }
  };

  const handleCopyMessage = (content: string) => {
    copy(content);
  };

  const handleRegenerate = () => {
    if (lastUserMessage) {
      // Reset typing state before regenerating
      resetTypingState();
      setItems((prev) => prev.slice(0, -1));
      handleSend(lastUserMessage);
    }
  };

  const handleDeleteMessage = (index: number) => {
    setItems((prev) => prev.filter((_, i) => i !== index));
  };

  const handleClearChat = () => {
    setItems([]);
    setLastUserMessage("");
    setContextStartIndex(0);
    setErrorMessage(null);
    setClearDialogOpen(false);
  };

  const handleClearContext = () => {
    // Add a separator and update context start index
    const messageCount = items.filter((item) => "role" in item).length;
    setItems((prev) => [...prev, { type: "context-separator" }]);
    setContextStartIndex(messageCount);
  };

  const handleSuggestedPrompt = (query: string) => {
    setInput(query);
    setTimeout(() => handleSend(query), 100);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <section className="w-full h-[calc(100vh-4rem)] md:h-[calc(100vh-2rem)] flex flex-col relative">
      {/* Schedule Panel Toggle */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {!md && (
          <MobileHeader>
            <div className="flex flex-row items-center w-full">
              {/* Centered title - absolute positioned to visual center */}
              <div className="absolute left-1/2 -translate-x-1/2 flex items-center gap-1 font-medium text-foreground">
                <SparklesIcon className="w-5 h-5 text-blue-500" />
                {t("common.ai-assistant")}
              </div>
              {/* Right action button - dropdown with clear options */}
              {items.length > 0 && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="sm" className="ml-auto h-8 px-2 text-muted-foreground hover:text-foreground">
                      <EraserIcon className="w-4 h-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={handleClearContext} className="cursor-pointer">
                      <EraserIcon className="w-4 h-4 mr-2" />
                      {t("ai.clear-context")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setClearDialogOpen(true)} className="text-destructive focus:text-destructive cursor-pointer">
                      <EraserIcon className="w-4 h-4 mr-2" />
                      {t("ai.clear-chat")}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </div>
          </MobileHeader>
        )}

        {/* Messages Area */}
        <div className="flex-1 overflow-y-auto px-4 py-6 space-y-6" ref={scrollRef}>
          {/* Current Parrot Status (Milestone 1) */}
          {currentParrot && (
            <div className="max-w-3xl mx-auto mb-4">
              <ParrotStatus
                parrot={currentParrot}
                thinking={isParrotThinking}
              />
            </div>
          )}

          {/* Memo Query Results (Milestone 1) */}
          {memoQueryResults.map((result, index) => (
            <div key={index} className="max-w-3xl mx-auto mb-4">
              <MemoQueryResult result={result} />
            </div>
          ))}

          {items.length === 0 && <EmptyState onSuggestedPrompt={handleSuggestedPrompt} />}

          {items.map((item, index) => {
            // Render context separator
            if ("type" in item && item.type === "context-separator") {
              return (
                <div key={index} className="flex items-center gap-4 max-w-3xl mx-auto py-2">
                  <div className="flex-1 h-px bg-border" />
                  <span className="text-xs text-muted-foreground whitespace-nowrap">{t("ai.context-cleared")}</span>
                  <div className="flex-1 h-px bg-border" />
                </div>
              );
            }

            // Render regular message
            const msg = item as Message;

            return (
              <div
                key={index}
                className={cn(
                  "group flex gap-4 max-w-3xl mx-auto",
                  msg.role === "user"
                    ? "animate-in slide-in-from-right-4 fade-in-0 duration-300 flex-row-reverse"
                    : "animate-in slide-in-from-left-4 fade-in-0 duration-300 flex-row",
                )}
              >
                <div
                  className={cn(
                    "w-8 h-8 rounded-full flex items-center justify-center shrink-0 mt-1 shadow-sm",
                    msg.role === "user"
                      ? "bg-primary text-primary-foreground"
                      : "bg-blue-100 text-blue-600 dark:bg-blue-900 dark:text-blue-300",
                  )}
                >
                  {msg.role === "user" ? <UserIcon size={16} /> : <BotIcon size={16} />}
                </div>

                <div className="flex-1 min-w-0">
                  {msg.role === "assistant" && !msg.error && index === items.length - 1 && (
                    <div className="flex items-start gap-2">
                      <MessageActions
                        onCopy={() => handleCopyMessage(msg.content)}
                        onRegenerate={handleRegenerate}
                        onDelete={() => handleDeleteMessage(index)}
                      />
                    </div>
                  )}

                  {msg.error ? (
                    <ErrorMessage error={msg.content} onRetry={handleRetry} />
                  ) : (
                    <div
                      className={cn(
                        "rounded-2xl p-4 text-sm leading-relaxed shadow-sm",
                        msg.role === "user"
                          ? "bg-primary text-primary-foreground rounded-tr-sm"
                          : "bg-white dark:bg-zinc-800 dark:text-zinc-100 border border-border/50 rounded-tl-sm",
                      )}
                    >
                      {msg.role === "assistant" ? (
                        <div className="prose dark:prose-invert prose-sm max-w-none break-words">
                          <ReactMarkdown
                            remarkPlugins={[remarkGfm, remarkBreaks]}
                            components={{
                              a: ({ node, ...props }) => (
                                <a {...props} className="text-blue-500 hover:underline" target="_blank" rel="noopener noreferrer" />
                              ),
                              p: ({ node, ...props }) => <p {...props} className="mb-2 last:mb-0" />,
                              pre: ({ node, ...props }) => <CodeBlock {...props} />,
                              // biome-ignore lint/suspicious/noExplicitAny: complex react-markdown props
                              code: ({ node, className, children, ...props }: any) =>
                                props.inline ? (
                                  <code className={cn("px-1.5 py-0.5 rounded bg-muted text-sm", className)} {...props}>
                                    {children}
                                  </code>
                                ) : (
                                  <code className={className} {...props}>
                                    {children}
                                  </code>
                                ),
                            }}
                          >
                            {msg.content || "..."}
                          </ReactMarkdown>
                          {isTyping && !msg.error && index === items.length - 1 && <TypingCursor active={true} />}
                        </div>
                      ) : (
                        <div className="whitespace-pre-wrap break-words">{msg.content}</div>
                      )}
                    </div>
                  )}
                </div>
              </div>
            );
          })}

          {isTyping &&
            (() => {
              const lastItem = items[items.length - 1] as ChatItem | undefined;
              if (!lastItem) return true;
              if ("type" in lastItem) return true; // ContextSeparator
              return lastItem.role !== "assistant"; // Message
            })() && (
              <div className="flex gap-4 max-w-3xl mx-auto animate-in fade-in-0 duration-300">
                <div className="w-8 h-8 rounded-full bg-blue-100 text-blue-600 dark:bg-blue-900 dark:text-blue-300 flex items-center justify-center shrink-0 shadow-sm mt-1">
                  <BotIcon size={16} />
                </div>
                <ThinkingIndicator />
              </div>
            )}
        </div>
      </div>

      {/* Input Area */}
      <div className="shrink-0 p-4 border-t bg-background/80 backdrop-blur-md sticky bottom-0 z-10">
        {/* Parrot Quick Actions (Milestone 1) */}
        <div className="max-w-3xl mx-auto mb-3">
          <ParrotQuickActions
            currentParrot={currentParrot}
            onParrotChange={setCurrentParrot}
            disabled={isTyping}
          />
        </div>

        <div className="max-w-3xl mx-auto relative">
          {/* Desktop clear button dropdown */}
          {md && items.length > 0 && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="absolute -top-11 right-0 h-7 px-2 text-xs text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                >
                  <EraserIcon className="w-3.5 h-3.5 mr-1" />
                  {t("ai.clear")}
                  <MoreHorizontalIcon className="w-3 h-3 ml-1" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem onClick={handleClearContext} className="cursor-pointer">
                  <EraserIcon className="w-4 h-4 mr-2" />
                  <div>
                    <div className="font-medium">{t("ai.clear-context")}</div>
                    <div className="text-xs text-muted-foreground">{t("ai.clear-context-desc")}</div>
                  </div>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setClearDialogOpen(true)} className="text-destructive focus:text-destructive cursor-pointer">
                  <EraserIcon className="w-4 h-4 mr-2" />
                  <div>
                    <div className="font-medium">{t("ai.clear-chat")}</div>
                    <div className="text-xs text-muted-foreground">{t("ai.clear-chat-desc")}</div>
                  </div>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
          <div className="flex items-center gap-2 p-2 bg-muted/50 rounded-xl border focus-within:ring-1 focus-within:ring-ring focus-within:bg-background transition-all">
            <Textarea
              ref={textareaRef}
              value={input}
              onChange={handleInputChange}
              onKeyDown={handleKeyDown}
              placeholder={currentParrot
                ? t("ai.parrot.chat-placeholder", { name: currentParrot.displayName })
                : t("ai.parrot.chat-default-placeholder")}
              className="min-h-[44px] max-h-[150px] w-full resize-none border-0 bg-transparent focus-visible:ring-0 px-3 py-2.5 shadow-none"
              rows={1}
              style={{ height: "auto" }}
              onInput={(e) => {
                const target = e.target as HTMLTextAreaElement;
                target.style.height = "auto";
                target.style.height = `${Math.min(target.scrollHeight, 150)}px`;
              }}
            />
            <Button
              size="icon"
              className="shrink-0 h-9 w-9 rounded-lg transition-all"
              onClick={() => handleSend()}
              disabled={!input.trim() || isTyping}
            >
              <SendIcon className="w-4 h-4" />
            </Button>
          </div>
        </div>
      </div>

      {/* Clear Chat Confirmation Dialog */}
      <ConfirmDialog
        open={clearDialogOpen}
        onOpenChange={setClearDialogOpen}
        title={t("ai.clear-chat")}
        confirmLabel={t("common.confirm")}
        description={t("ai.clear-chat-confirm")}
        cancelLabel={t("common.cancel")}
        onConfirm={handleClearChat}
        confirmVariant="destructive"
      />

      {/* Parrot Selector (Milestone 1) */}
      {showParrotSelector && parrotSelectorPosition && (
        <ParrotSelector
          onSelect={handleParrotSelect}
          onClose={() => setShowParrotSelector(false)}
          position={parrotSelectorPosition}
        />
      )}
    </section>
  );
};

export default AIChat;
