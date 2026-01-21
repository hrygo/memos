import copy from "copy-to-clipboard";
import {
  BotIcon,
  Calendar,
  CalendarDays,
  ChevronDown,
  ChevronUp,
  EraserIcon,
  LayoutList,
  Loader2,
  MoreHorizontalIcon,
  PlusIcon,
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
import { ScheduleInput } from "@/components/AIChat/ScheduleInput";
import { ScheduleCalendar } from "@/components/AIChat/ScheduleCalendar";
import { ScheduleSuggestionCard } from "@/components/AIChat/ScheduleSuggestionCard";
import { ScheduleTimeline } from "@/components/AIChat/ScheduleTimeline";
import { ScheduleQueryResult } from "@/components/AIChat/ScheduleQueryResult";
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
import { useParseAndCreateSchedule, useSchedulesOptimized, useCheckConflict } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import type { ScheduleSummary } from "@/types/schedule";

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
  const [scheduleViewMode, setScheduleViewMode] = useState<"timeline" | "calendar">("timeline");
  const [editSchedule, setEditSchedule] = useState<Schedule | null>(null);
  const [hasScheduleQueryResult, setHasScheduleQueryResult] = useState(false);
  const [aiHandledScheduleQuery, setAiHandledScheduleQuery] = useState(false); // 标记AI是否已处理日程查询

  // 使用 useRef 存储消息 ID，避免 React state 异步更新导致的竞态条件
  const messageIdRef = useRef(0);
  // Use optimized schedule hook with 30-day window (±15 days from selected date)
  // Calculate anchor date from selectedDate or use today
  // 使用 useMemo 避免 anchorDate 每次渲染都创建新对象导致重复查询
  const anchorDate = useMemo(() => {
    return selectedDate ? new Date(selectedDate + 'T00:00:00') : new Date();
  }, [selectedDate]);
  const { data: schedulesData } = useSchedulesOptimized(anchorDate);

  const schedules = schedulesData?.schedules || [];

  // Debug logging
  useEffect(() => {
    console.log('[AIChat Debug] Schedule Query Info:');
    console.log('  selectedDate:', selectedDate);
    console.log('  anchorDate:', anchorDate.toISOString());
    console.log('  schedulesData:', schedulesData);
    console.log('  schedules.length:', schedules.length);
    if (schedules.length > 0) {
      console.log('  First 3 schedules:');
      schedules.slice(0, 3).forEach((s, i) => {
        console.log(`    [${i}] ${s.title}: startTs=${s.startTs}, endTs=${s.endTs}`);
      });
    }
  }, [schedulesData, selectedDate, anchorDate, schedules]);

  // Schedule suggestion state
  const [suggestedSchedule, setSuggestedSchedule] = useState<Schedule | null>(null);
  const [showScheduleSuggestion, setShowScheduleSuggestion] = useState(false);
  const [lastScheduleMessage, setLastScheduleMessage] = useState("");
  const [isParsingSchedule, setIsParsingSchedule] = useState(false);
  const [scheduleConflicts, setScheduleConflicts] = useState<Schedule[]>([]);
  const [showScheduleQueryResult, setShowScheduleQueryResult] = useState(false);
  const [queryResultSchedules, setQueryResultSchedules] = useState<ScheduleSummary[]>([]);
  const [queryTitle, setQueryTitle] = useState("");
  const parseAndCreateSchedule = useParseAndCreateSchedule();
  const checkConflict = useCheckConflict();

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

  // Intent detection for schedule query
  const detectScheduleQueryIntent = (text: string): boolean => {
    const queryKeywords = [
      "查询", "有什么", "安排", "看看", "show", "what", "list", "query", "查看",
      "多少", "几个", "search", "find", "list",
      "今天", "明天", "后天", "本周", "下周",
      "tomorrow", "today", "week", "schedule", "日程", "计划"
    ];

    const hasQueryKeyword = queryKeywords.some((keyword) =>
      text.toLowerCase().includes(keyword.toLowerCase())
    );

    // Query patterns: "今天有什么日程", "查询明天安排", "show me my schedule"
    const queryPatterns = [
      /今天.*什么|明天.*什么|后天.*什么|本周.*什么|下周.*什么/,
      /有什么日程|有哪些安排|有多少个/,
      /show.*schedule|list.*schedule|what.*schedule|my.*schedule/i,
      /查询.*日程|查看.*日程|我的.*日程/,
    ];

    const matchesPattern = queryPatterns.some((pattern) => pattern.test(text));

    return hasQueryKeyword && matchesPattern;
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

    // 重置日程查询结果标记（新对话）
    setHasScheduleQueryResult(false);
    setAiHandledScheduleQuery(false);

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
          onScheduleIntent: (intent) => {
            // AI 检测到日程创建意图，触发建议卡片
            // 使用 messageIdRef.current 避免竞态条件
            if (messageId !== messageIdRef.current) {
              console.warn(`[ScheduleIntent] Ignoring stale intent for message ${messageId}, current is ${messageIdRef.current}`);
              return;
            }

            if (intent.detected && !scheduleInputOpen) {
              // 验证 scheduleDescription 不为空
              if (!intent.scheduleDescription || intent.scheduleDescription.trim().length === 0) {
                console.warn("[ScheduleIntent] Intent detected but description is empty");
                return;
              }

              console.log(`[ScheduleIntent] Detected with description: "${intent.scheduleDescription}"`);
              handleScheduleSuggestion(intent.scheduleDescription);
            }
          },
          onScheduleQueryResult: (result) => {
            // AI 检测到日程查询意图，显示查询结果
            // 使用 messageIdRef.current 避免竞态条件
            if (messageId !== messageIdRef.current) {
              console.warn(`[ScheduleQuery] Ignoring stale result for message ${messageId}, current is ${messageIdRef.current}`);
              return;
            }

            console.log(`[ScheduleQuery] AI backend handled query with ${result.schedules.length} schedules: "${result.timeRangeDescription}"`);

            // 标记 AI 已处理日程查询
            setAiHandledScheduleQuery(true);

            if (result.detected && result.schedules.length > 0) {
              // 标记有日程查询结果，用于前端智能处理 AI 回复
              setHasScheduleQueryResult(true);

              // 转换为 ScheduleSummary 格式，将 bigint 转换为 number
              const schedules: ScheduleSummary[] = result.schedules.map((sched) => ({
                uid: sched.uid,
                title: sched.title,
                startTs: Number(sched.startTs),
                endTs: Number(sched.endTs),
                allDay: sched.allDay,
                location: sched.location,
                recurrenceRule: sched.recurrenceRule,
                status: sched.status,
              }));

              setQueryResultSchedules(schedules);
              setQueryTitle(result.timeRangeDescription || "近期日程");
              setShowScheduleQueryResult(true);
            } else if (result.detected && result.schedules.length === 0) {
              // 检测到查询意图但没有日程
              setHasScheduleQueryResult(true);
              // AI 后端返回空结果，不显示前端查询的日程卡片
              setShowScheduleQueryResult(false);
              toast("该时间段暂无日程安排", {
                icon: "📅",
                duration: 3000,
              });
            }
          },
        },
      );
    } catch (_error) {
      streamCompleted = true;
      resetTypingState();
      setErrorMessage(t("ai.error-title"));
    }

    // Check for schedule query intent after AI responds
    // 只有在 AI 没有处理日程查询时，才使用前端自动查询
    if (detectScheduleQueryIntent(userMessage) && !aiHandledScheduleQuery) {
      console.log("[ScheduleQuery] AI did not handle query, using frontend fallback");
      handleScheduleQuery(userMessage);
    }
    // 注意：日程创建意图现在由 AI 在后端检测，不再需要前端检测
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
    setAiHandledScheduleQuery(false);
    setHasScheduleQueryResult(false);
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

  const handleScheduleQuery = (userMessage: string) => {
    import("dayjs").then((dayjsMod) => {
      const dayjs = dayjsMod.default;

      // Determine time range title from query
      let title = "";
      const now = dayjs();

      if (userMessage.includes("今天") || userMessage.toLowerCase().includes("today")) {
        title = "今天的日程";
      } else if (userMessage.includes("明天") || userMessage.toLowerCase().includes("tomorrow")) {
        title = "明天的日程";
      } else if (userMessage.includes("后天")) {
        title = "后天的日程";
      } else if (userMessage.includes("本周") || userMessage.toLowerCase().includes("this week")) {
        title = "本周的日程";
      } else if (userMessage.includes("下周") || userMessage.toLowerCase().includes("next week")) {
        title = "下周的日程";
      } else {
        title = "日程查询结果";
      }

      // Filter schedules based on query (schedules already contains ±15 days data)
      const filteredSchedules = schedules.filter((schedule) => {
        const scheduleStart = dayjs.unix(Number(schedule.startTs));
        const scheduleEnd = schedule.endTs > 0 ? dayjs.unix(Number(schedule.endTs)) : scheduleStart.add(1, "hour");

        // Additional filtering based on query
        if (userMessage.includes("今天") || userMessage.toLowerCase().includes("today")) {
          const todayStart = now.startOf("day");
          const todayEnd = now.endOf("day");
          return scheduleStart.isBefore(todayEnd) && scheduleEnd.isAfter(todayStart);
        } else if (userMessage.includes("明天") || userMessage.toLowerCase().includes("tomorrow")) {
          const tomorrowStart = now.add(1, "day").startOf("day");
          const tomorrowEnd = now.add(1, "day").endOf("day");
          return scheduleStart.isBefore(tomorrowEnd) && scheduleEnd.isAfter(tomorrowStart);
        } else if (userMessage.includes("后天")) {
          const dayAfterTomorrowStart = now.add(2, "day").startOf("day");
          const dayAfterTomorrowEnd = now.add(2, "day").endOf("day");
          return scheduleStart.isBefore(dayAfterTomorrowEnd) && scheduleEnd.isAfter(dayAfterTomorrowStart);
        } else if (userMessage.includes("本周") || userMessage.toLowerCase().includes("this week")) {
          const weekStart = now.startOf("week");
          const weekEnd = now.endOf("week");
          return scheduleStart.isBefore(weekEnd) && scheduleEnd.isAfter(weekStart);
        } else if (userMessage.includes("下周") || userMessage.toLowerCase().includes("next week")) {
          const nextWeekStart = now.add(1, "week").startOf("week");
          const nextWeekEnd = now.add(1, "week").endOf("week");
          return scheduleStart.isBefore(nextWeekEnd) && scheduleEnd.isAfter(nextWeekStart);
        }
        // Default: show all schedules (already filtered by ±15 days window)
        return true;
      });

      // Sort by start time
      const sortedSchedules = filteredSchedules.sort((a, b) =>
        Number(a.startTs) - Number(b.startTs)
      );

      // Map Schedule to ScheduleSummary, converting bigint to number
      const mappedSchedules: ScheduleSummary[] = sortedSchedules.map((s) => {
        // Extract uid from name (format: "schedules/{uid}")
        const uid = s.name.replace("schedules/", "");
        return {
          uid,
          title: s.title,
          startTs: Number(s.startTs),
          endTs: Number(s.endTs),
          allDay: s.allDay,
          location: s.location,
          recurrenceRule: s.recurrenceRule || "",
          status: s.state === "NORMAL" ? "ACTIVE" : "CANCELLED",
        };
      });

      setQueryResultSchedules(mappedSchedules);
      setQueryTitle(title);
      setShowScheduleQueryResult(true);
    });
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

        // Check for conflicts
        const endTs = result.parsedSchedule.endTs > 0 ? result.parsedSchedule.endTs : result.parsedSchedule.startTs + BigInt(3600);

        try {
          const conflictResult = await checkConflict.mutateAsync({
            startTs: result.parsedSchedule.startTs,
            endTs: endTs,
          });

          setScheduleConflicts(conflictResult.conflicts || []);
        } catch (error) {
          console.error("[ScheduleSuggestion] Failed to check conflicts:", error);
          setScheduleConflicts([]);
        }

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
    setScheduleConflicts([]);
  };

  const handleAdjustTime = () => {
    if (suggestedSchedule) {
      // Open schedule input for editing with conflict context
      setScheduleInputText(lastScheduleMessage);
      setScheduleInputOpen(true);
      setShowScheduleSuggestion(false);
      setScheduleConflicts([]);
    }
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

  // 检测 AI 回复是否与日程查询结果矛盾
  const isScheduleResponseContradictory = (content: string): boolean => {
    if (!hasScheduleQueryResult) return false;

    const contradictoryPatterns = [
      /没有.*日程|无.*日程|没找到.*日程|未找到.*日程|找不到.*日程/i,
      /暂时.*没有.*安排|没有.*安排/i,
      /没有.*相关.*信息|未找到.*相关.*信息/i,
      /笔记.*没有.*日程|笔记中.*没有/i,
      /sorry.*no.*schedule|no.*schedules.*found/i,
    ];

    return contradictoryPatterns.some((pattern) => pattern.test(content));
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

            // 如果 AI 回复与日程查询结果矛盾，则不显示（前端智能处理）
            if (msg.role === "assistant" && isScheduleResponseContradictory(msg.content)) {
              console.log("[AIChat] Hiding contradictory AI response:", msg.content);
              return null;
            }

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
              conflicts={scheduleConflicts}
              onConfirm={handleConfirmScheduleSuggestion}
              onDismiss={handleDismissScheduleSuggestion}
              onEdit={handleEditScheduleSuggestion}
              onAdjustTime={handleAdjustTime}
            />
          </div>
        )}

        {showScheduleQueryResult && queryResultSchedules.length > 0 && (
          <ScheduleQueryResult
            title={queryTitle}
            schedules={queryResultSchedules}
            onClose={() => {
              setShowScheduleQueryResult(false);
              setQueryResultSchedules([]);
              setQueryTitle("");
            }}
            onScheduleClick={undefined}
            onOpenSchedulePanel={() => {
              setSchedulePanelOpen(true);
            }}
          />
        )}

        {/* Schedule Panel Toggle Button */}
        <div className="shrink-0 border-t bg-background/95 backdrop-blur-md max-w-3xl mx-auto w-full">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setSchedulePanelOpen(!schedulePanelOpen)}
            className="w-full h-8 rounded-none border-b hover:bg-muted/50 cursor-pointer"
          >
            <Calendar className="w-4 h-4 mr-2" />
            <span className="flex-1 text-left">{t("schedule.title") || "Schedule"}</span>
            {schedulePanelOpen ? <ChevronDown className="w-4 h-4" /> : <ChevronUp className="w-4 h-4" />}
          </Button>

          {/* Schedule Panel Content - NEW TIMELINE LAYOUT */}
          {schedulePanelOpen && (
            <div className="bg-muted/30 animate-in slide-in-from-top-2 duration-300">
              <div className="w-full flex flex-col h-[45vh] md:h-[320px]">
                <div className="flex items-center justify-between px-4 py-2 bg-muted/20 border-b border-border/40">
                  {/* Mobile-Friendly Segmented Control */}
                  <div className="flex items-center bg-muted rounded-lg p-0.5">
                    <Button
                      variant={scheduleViewMode === "timeline" ? "default" : "ghost"}
                      size="sm"
                      className={`h-7 px-3 text-xs font-medium rounded-md cursor-pointer ${scheduleViewMode === "timeline" ? "" : "hover:bg-transparent"}`}
                      onClick={() => setScheduleViewMode("timeline")}
                    >
                      <LayoutList className="w-3.5 h-3.5 mr-1.5" />
                      {t("schedule.your-timeline") || "Timeline"}
                    </Button>
                    <Button
                      variant={scheduleViewMode === "calendar" ? "default" : "ghost"}
                      size="sm"
                      className={`h-7 px-3 text-xs font-medium rounded-md cursor-pointer ${scheduleViewMode === "calendar" ? "" : "hover:bg-transparent"}`}
                      onClick={() => setScheduleViewMode("calendar")}
                    >
                      <CalendarDays className="w-3.5 h-3.5 mr-1.5" />
                      {t("schedule.calendar-view") || "Calendar"}
                    </Button>
                  </div>
                  <Button
                    size="sm"
                    className="h-8 gap-1 cursor-pointer"
                    onClick={() => {
                      setScheduleInputText(input);
                      setScheduleInputOpen(true);
                    }}
                  >
                    <PlusIcon className="w-3.5 h-3.5" />
                    <span className="hidden sm:inline">{t("schedule.add") || "Add"}</span>
                  </Button>
                </div>

                <div className="flex-1 min-h-0 bg-background shadow-none overflow-hidden relative">
                  {scheduleViewMode === "timeline" ? (
                    <ScheduleTimeline
                      schedules={schedules}
                      selectedDate={selectedDate}
                      onDateClick={setSelectedDate}
                      onScheduleEdit={(schedule) => {
                        setEditSchedule(schedule);
                        setScheduleInputOpen(true);
                      }}
                      className="rounded-none bg-transparent"
                    />
                  ) : (
                    <ScheduleCalendar
                      schedules={schedules}
                      selectedDate={selectedDate}
                      onDateClick={(date) => {
                        setSelectedDate(date);
                        // On mobile, automatically switch to timeline view to see the day's schedule
                        // On desktop, stay in calendar view for better browsing experience
                        if (!md) {
                          setScheduleViewMode("timeline");
                        }
                      }}
                      showMobileHint={!md}
                      className="p-4 bg-background/50 h-full overflow-y-auto"
                    />
                  )}
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
              className="shrink-0 h-9 w-9 rounded-lg transition-all"
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
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setScheduleInputText(input);
                    setScheduleInputOpen(true);
                  }}
                  className="h-7 text-xs"
                >
                  解析并创建日程
                </Button>
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
        onOpenChange={(open) => {
          setScheduleInputOpen(open);
          if (!open) {
            setEditSchedule(null);
            setScheduleInputText("");
          }
        }}
        initialText={scheduleInputText}
        editSchedule={editSchedule}
        onSuccess={(schedule) => {
          console.log("Schedule saved:", schedule);
          setEditSchedule(null);
          // Refresh schedules by invalidating cache
          // The query will automatically refetch
        }}
      />
    </section>
  );
};

export default AIChat;
