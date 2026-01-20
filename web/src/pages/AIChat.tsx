import copy from "copy-to-clipboard";
import {
  BotIcon,
  Calendar,
  ChevronDown,
  ChevronUp,
  EraserIcon,
  Loader2,
  MoreHorizontalIcon,
  PlusIcon,
  SendIcon,
  SparklesIcon,
  UserIcon,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import EmptyState from "@/components/AIChat/EmptyState";
import ErrorMessage from "@/components/AIChat/ErrorMessage";
import MessageActions from "@/components/AIChat/MessageActions";
import { ScheduleInput } from "@/components/AIChat/ScheduleInput";
import { ScheduleSuggestionCard } from "@/components/AIChat/ScheduleSuggestionCard";
import { ScheduleTimeline } from "@/components/AIChat/ScheduleTimeline";
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
import { useParseAndCreateSchedule, useSchedules } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";

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

  // Schedule-related state
  const [schedulePanelOpen, setSchedulePanelOpen] = useState(false);
  const [scheduleInputOpen, setScheduleInputOpen] = useState(false);
  const [scheduleInputText, setScheduleInputText] = useState("");
  const [selectedDate, setSelectedDate] = useState<string | undefined>();
  const { data: schedulesData } = useSchedules({});

  const schedules = schedulesData?.schedules || [];

  // Schedule suggestion state
  const [suggestedSchedule, setSuggestedSchedule] = useState<Schedule | null>(null);
  const [showScheduleSuggestion, setShowScheduleSuggestion] = useState(false);
  const [lastScheduleMessage, setLastScheduleMessage] = useState("");
  const [isParsingSchedule, setIsParsingSchedule] = useState(false);
  const parseAndCreateSchedule = useParseAndCreateSchedule();

  // Intent detection for schedule creation (improved to reduce false positives)
  const detectScheduleIntent = (text: string): boolean => {
    // Action keywords: explicit intent to create/arrange
    const actionKeywords = ["schedule", "meeting", "remind", "calendar", "日程", "会议", "提醒", "安排", "计划", "添加", "创建", "新建"];

    // Time keywords: tomorrow, next week, etc.
    const timeKeywords = ["明天", "后天", "下周", "今天", "今晚", "明晚"];

    const hasAction = actionKeywords.some((keyword) => text.toLowerCase().includes(keyword.toLowerCase()));

    const hasTime = timeKeywords.some((keyword) => text.includes(keyword));

    // 1. Has action keyword → directly return true
    if (hasAction) return true;

    // 2. Has time keyword + numbers/time expressions → might be a schedule
    if (hasTime && /\d+[点时]|上午|下午|晚上/.test(text)) {
      return true;
    }

    return false;
  };

  const shouldShowQuickSuggestion = (text: string) => {
    return detectScheduleIntent(text) && !schedulePanelOpen && !showScheduleSuggestion;
  };

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

  const handleSend = async (messageContent?: string) => {
    const userMessage = (messageContent || input).trim();
    if (!userMessage) return;

    // If already typing, reset first
    if (isTyping) {
      resetTypingState();
    }

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

    // Check for schedule creation intent after AI responds
    if (detectScheduleIntent(userMessage) && !scheduleInputOpen) {
      handleScheduleSuggestion(userMessage);
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
    // Clear schedule-related state
    setShowScheduleSuggestion(false);
    setSuggestedSchedule(null);
    setLastScheduleMessage("");
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

  const handleScheduleSuggestion = async (userMessage: string) => {
    // Prevent duplicate parsing
    if (isParsingSchedule) {
      console.log("[ScheduleSuggestion] Already parsing, skipping");
      return;
    }

    setIsParsingSchedule(true);
    try {
      // Parse the user message to extract schedule info
      const result = await parseAndCreateSchedule.mutateAsync({
        text: userMessage,
        autoConfirm: false,
      });

      if (result.parsedSchedule) {
        setSuggestedSchedule(result.parsedSchedule);
        setLastScheduleMessage(userMessage);
        setShowScheduleSuggestion(true);
      }
    } catch (error) {
      console.error("[ScheduleSuggestion] Failed to parse:", {
        message: userMessage.substring(0, 50),
        error: error instanceof Error ? error.message : String(error),
      });
      toast.error(t("schedule.parse-error"), {
        duration: 3000,
        id: "schedule-parse-error",
      });
    } finally {
      setIsParsingSchedule(false);
    }
  };

  const handleConfirmScheduleSuggestion = () => {
    if (suggestedSchedule) {
      // Open schedule input with the original message for editing/confirmation
      setScheduleInputText(lastScheduleMessage);
      setScheduleInputOpen(true);
      setShowScheduleSuggestion(false);
    }
  };

  const handleDismissScheduleSuggestion = () => {
    setShowScheduleSuggestion(false);
    setSuggestedSchedule(null);
    setLastScheduleMessage("");
  };

  const handleEditScheduleSuggestion = () => {
    // Open schedule input with the original message for editing
    if (suggestedSchedule) {
      setScheduleInputText(lastScheduleMessage);
      setScheduleInputOpen(true);
      setShowScheduleSuggestion(false);
    }
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
                    <DropdownMenuItem onClick={handleClearContext}>
                      <EraserIcon className="w-4 h-4 mr-2" />
                      {t("ai.clear-context")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setClearDialogOpen(true)} className="text-destructive focus:text-destructive">
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
                        content={msg.content}
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

        {/* Schedule Suggestion Card */}
        {isParsingSchedule && (
          <div className="px-4 py-2">
            <div className="flex items-center gap-2 text-sm text-muted-foreground bg-muted/50 rounded-lg p-3 max-w-3xl mx-auto">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>{t("schedule.parsing") || "正在识别日程..."}</span>
            </div>
          </div>
        )}
        {showScheduleSuggestion && suggestedSchedule && (
          <div className="px-4 py-2">
            <ScheduleSuggestionCard
              parsedSchedule={suggestedSchedule}
              onConfirm={handleConfirmScheduleSuggestion}
              onDismiss={handleDismissScheduleSuggestion}
              onEdit={handleEditScheduleSuggestion}
            />
          </div>
        )}

        {/* Schedule Panel Toggle Button */}
        <div className="shrink-0 border-t bg-background/95 backdrop-blur-md max-w-3xl mx-auto w-full">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setSchedulePanelOpen(!schedulePanelOpen)}
            className="w-full h-8 rounded-none border-b hover:bg-muted/50"
          >
            <Calendar className="w-4 h-4 mr-2" />
            {t("schedule.title") || "Schedule"}
            {schedulePanelOpen ? <ChevronDown className="w-4 h-4 ml-auto" /> : <ChevronUp className="w-4 h-4 ml-auto" />}
          </Button>

          {/* Schedule Panel Content - NEW TIMELINE LAYOUT */}
          {schedulePanelOpen && (
            <div className="bg-muted/30 animate-in slide-in-from-top-2 duration-300">
              <div className="w-full p-4 flex flex-col h-[60vh] md:h-[50vh]">
                <div className="flex items-center justify-between mb-2 px-1">
                  <h3 className="font-semibold text-lg">{t("schedule.your-timeline") || "Timeline"}</h3>
                  <Button
                    size="sm"
                    className="h-8 gap-1"
                    onClick={() => {
                      setScheduleInputText(input);
                      setScheduleInputOpen(true);
                    }}
                  >
                    <PlusIcon className="w-3.5 h-3.5" />
                    {t("schedule.add") || "Add"}
                  </Button>
                </div>

                <div className="flex-1 min-h-0 bg-background/60 rounded-xl border border-border/50 shadow-sm overflow-hidden">
                  <ScheduleTimeline
                    schedules={schedules}
                    selectedDate={selectedDate}
                    onDateClick={setSelectedDate}
                    className="rounded-none bg-transparent"
                  />
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Input Area */}
      <div className="shrink-0 p-4 border-t bg-background/80 backdrop-blur-md sticky bottom-0 z-10">
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
                <DropdownMenuItem onClick={handleClearContext}>
                  <EraserIcon className="w-4 h-4 mr-2" />
                  <div>
                    <div className="font-medium">{t("ai.clear-context")}</div>
                    <div className="text-xs text-muted-foreground">{t("ai.clear-context-desc")}</div>
                  </div>
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setClearDialogOpen(true)} className="text-destructive focus:text-destructive">
                  <EraserIcon className="w-4 h-4 mr-2" />
                  <div>
                    <div className="font-medium">{t("ai.clear-chat")}</div>
                    <div className="text-xs text-muted-foreground">{t("ai.clear-chat-desc")}</div>
                  </div>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
          <div className="flex items-end gap-2 p-2 bg-muted/50 rounded-xl border focus-within:ring-1 focus-within:ring-ring focus-within:bg-background transition-all">
            <Textarea
              value={input}
              onChange={(e) => {
                setInput(e.target.value);
              }}
              onKeyDown={handleKeyDown}
              placeholder={t("common.ai-placeholder")}
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
              className="shrink-0 h-9 w-9 mb-0.5 rounded-lg transition-all"
              onClick={() => handleSend()}
              disabled={!input.trim() || isTyping}
            >
              <SendIcon className="w-4 h-4" />
            </Button>
          </div>

          {/* Schedule intent suggestion */}
          {shouldShowQuickSuggestion(input) && input.trim() && (
            <div className="mt-2 p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800 animate-in slide-in-from-bottom-2 duration-300">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-sm">
                  <Calendar className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                  <span className="text-blue-700 dark:text-blue-300">
                    创建日程? "{input.length > 30 ? input.slice(0, 30) + "..." : input}"
                  </span>
                </div>
                <div className="flex gap-2">
                  <Button variant="ghost" size="sm" onClick={() => setScheduleInputOpen(true)} className="h-7 text-xs">
                    创建日程
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setScheduleInputText(input);
                      setScheduleInputOpen(true);
                    }}
                    className="h-7 text-xs"
                  >
                    解析
                  </Button>
                </div>
              </div>
            </div>
          )}
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

      {/* Schedule Input Dialog */}
      <ScheduleInput
        open={scheduleInputOpen}
        onOpenChange={setScheduleInputOpen}
        initialText={scheduleInputText}
        onSuccess={(schedule) => {
          console.log("Schedule created:", schedule);
          // Refresh schedules by invalidating cache
          // The query will automatically refetch
        }}
      />
    </section>
  );
};

export default AIChat;
