import { BotIcon, Calendar, EraserIcon, SendIcon, SparklesIcon, UserIcon, MoreHorizontalIcon, ChevronDown, ChevronUp } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import copy from "copy-to-clipboard";
import MobileHeader from "@/components/MobileHeader";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { CodeBlock } from "@/components/MemoContent/CodeBlock";
import ConfirmDialog from "@/components/ConfirmDialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useChatWithMemos } from "@/hooks/useAIQueries";
import { useSchedules } from "@/hooks/useScheduleQueries";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import EmptyState from "@/components/AIChat/EmptyState";
import ErrorMessage from "@/components/AIChat/ErrorMessage";
import MessageActions from "@/components/AIChat/MessageActions";
import ThinkingIndicator from "@/components/AIChat/ThinkingIndicator";
import TypingCursor from "@/components/AIChat/TypingCursor";
import { ScheduleCalendar } from "@/components/AIChat/ScheduleCalendar";
import { ScheduleList } from "@/components/AIChat/ScheduleList";
import { ScheduleInput } from "@/components/AIChat/ScheduleInput";
import { ScheduleSuggestionCard } from "@/components/AIChat/ScheduleSuggestionCard";
import { useParseAndCreateSchedule } from "@/hooks/useScheduleQueries";
import dayjs from "dayjs";

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
  const parseAndCreateSchedule = useParseAndCreateSchedule();

  // Intent detection for schedule creation
  const detectScheduleIntent = (text: string): boolean => {
    const keywords = [
      "schedule", "meeting", "remind", "calendar",
      "日程", "会议", "提醒", "安排", "计划",
      "明天", "后天", "下周", "今天",
    ];
    return keywords.some((keyword) => text.toLowerCase().includes(keyword.toLowerCase()));
  };

  // Get actual messages (excluding separators) for API calls
  const getMessagesForContext = useCallback(() => {
    return items
      .filter((item): item is Message => "role" in item)
      .slice(contextStartIndex) as Message[];
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
    } catch (error) {
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
    try {
      // Parse the user message to extract schedule info
      const result = await parseAndCreateSchedule.mutateAsync({
        text: userMessage,
        autoConfirm: false,
      });

      if (result.parsedSchedule) {
        setSuggestedSchedule(result.parsedSchedule);
        setShowScheduleSuggestion(true);
      }
    } catch (error) {
      console.error("Failed to parse schedule suggestion:", error);
    }
  };

  const handleConfirmScheduleSuggestion = () => {
    if (suggestedSchedule) {
      // Open the schedule input dialog with the suggested schedule pre-filled
      setScheduleInputText("");
      setScheduleInputOpen(true);
      // The ScheduleInput will handle the actual creation
      setShowScheduleSuggestion(false);
    }
  };

  const handleDismissScheduleSuggestion = () => {
    setShowScheduleSuggestion(false);
    setSuggestedSchedule(null);
  };

  const handleEditScheduleSuggestion = () => {
    // Open schedule input with the suggested text for editing
    if (suggestedSchedule) {
      setScheduleInputText(suggestedSchedule.title || "");
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
                  <Button
                    variant="ghost"
                    size="sm"
                    className="ml-auto h-8 px-2 text-muted-foreground hover:text-foreground"
                  >
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
                <span className="text-xs text-muted-foreground whitespace-nowrap">
                  {t("ai.context-cleared")}
                </span>
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
                        : "bg-white dark:bg-zinc-800 border border-border/50 rounded-tl-sm",
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

        {isTyping && (() => {
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
      <div className="shrink-0 border-t bg-background/95 backdrop-blur-md">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setSchedulePanelOpen(!schedulePanelOpen)}
          className="w-full h-8 rounded-none border-b hover:bg-muted/50"
        >
          <Calendar className="w-4 h-4 mr-2" />
          {t("schedule.title") || "Schedule"}
          {schedulePanelOpen ? (
            <ChevronDown className="w-4 h-4 ml-auto" />
          ) : (
            <ChevronUp className="w-4 h-4 ml-auto" />
          )}
        </Button>

        {/* Schedule Panel Content */}
        {schedulePanelOpen && (
          <div className="border-t bg-muted/30 animate-in slide-in-from-top-2 duration-300">
            <div className="max-w-6xl mx-auto p-4">
              <div className="grid md:grid-cols-2 gap-4 h-[50vh] md:h-[40vh]">
                {/* Calendar Column */}
                <div className="flex flex-col overflow-hidden rounded-lg border bg-card">
                  <div className="p-4 border-b flex items-center justify-between">
                    <h3 className="font-semibold">{t("schedule.calendar") || "Calendar"}</h3>
                    <Button
                      size="sm"
                      onClick={() => {
                        setScheduleInputText(input);
                        setScheduleInputOpen(true);
                      }}
                    >
                      + {t("schedule.add-schedule") || "Add Schedule"}
                    </Button>
                  </div>
                  <div className="flex-1 p-4 overflow-y-auto">
                    <ScheduleCalendar
                      schedules={schedules}
                      selectedDate={selectedDate}
                      onDateClick={setSelectedDate}
                    />
                  </div>
                </div>

                {/* Schedule List Column */}
                <div className="flex flex-col overflow-hidden rounded-lg border bg-card">
                  <div className="p-4 border-b">
                    <h3 className="font-semibold">
                      {selectedDate
                        ? dayjs(selectedDate).format("MMMM D, YYYY")
                        : t("schedule.todays-schedules") || "Today's Schedules"}
                    </h3>
                  </div>
                  <div className="flex-1 p-4 overflow-y-auto">
                    <ScheduleList
                      schedules={schedules}
                      selectedDate={selectedDate}
                      onScheduleClick={(schedule) => {
                        console.log("Clicked schedule:", schedule);
                      }}
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
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
          {showScheduleSuggestion(input) && input.trim() && (
            <div className="mt-2 p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800 animate-in slide-in-from-bottom-2 duration-300">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-sm">
                  <Calendar className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                  <span className="text-blue-700 dark:text-blue-300">
                    创建日程? "{input.length > 30 ? input.slice(0, 30) + "..." : input}"
                  </span>
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setScheduleInputOpen(true)}
                    className="h-7 text-xs"
                  >
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
    </div>
    </section>
  );
};

export default AIChat;
