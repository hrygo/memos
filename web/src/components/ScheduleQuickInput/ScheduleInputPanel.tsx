import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQueryClient } from "@tanstack/react-query";
import dayjs from "dayjs";
import { Bot, Calendar, Clock, Loader2, MapPin, Send, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useCheckConflict, useCreateSchedule, useScheduleAgentChat } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import { generateUUID } from "@/utils/uuid";
import { ResizablePanel } from "./ResizablePanel";
import type { ConflictInfo, ParsedSchedule } from "./types";

interface ScheduleInputPanelProps {
  /** Whether panel is open */
  open: boolean;
  /** Called when open state changes */
  onOpenChange: (open: boolean) => void;
  /** Called when schedule is created */
  onSuccess?: () => void;
}

type ConversationRole = "user" | "assistant";

interface ConversationMessage {
  role: ConversationRole;
  content: string;
}

const MAX_INPUT_LENGTH = 500;

export function ScheduleInputPanel({ open, onOpenChange, onSuccess }: ScheduleInputPanelProps) {
  const t = useTranslate();
  const queryClient = useQueryClient();
  const createSchedule = useCreateSchedule();
  const checkConflict = useCheckConflict();
  const agentChat = useScheduleAgentChat();

  // Input state
  const [input, setInput] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);

  // Parsed schedule state
  const [parsedSchedule, setParsedSchedule] = useState<Partial<ParsedSchedule> | null>(null);

  // Conversation state
  const [conversationHistory, setConversationHistory] = useState<ConversationMessage[]>([]);
  const [agentResponse, setAgentResponse] = useState<string | null>(null);

  // Conflict state
  const [conflicts, setConflicts] = useState<ConflictInfo[]>([]);
  const [showConflictPanel, setShowConflictPanel] = useState(false);

  // Refs
  const scrollRef = useRef<HTMLDivElement>(null);
  const closeTimeoutRef = useRef<ReturnType<typeof setTimeout>>();

  // Cleanup
  useEffect(() => {
    return () => {
      if (closeTimeoutRef.current) {
        clearTimeout(closeTimeoutRef.current);
      }
    };
  }, []);

  // Auto-scroll when conversation updates
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [conversationHistory, agentResponse]);

  // Reset when opening
  useEffect(() => {
    if (open) {
      setInput("");
      setParsedSchedule(null);
      setConversationHistory([]);
      setAgentResponse(null);
      setConflicts([]);
      setShowConflictPanel(false);
    }
  }, [open]);

  // Handle AI parse
  const handleAgentParse = useCallback(async () => {
    if (!input.trim()) return;

    if (input.length > MAX_INPUT_LENGTH) {
      toast.error(t("schedule.input-too-long") as string);
      return;
    }

    setIsProcessing(true);

    const newHistory: ConversationMessage[] = [...conversationHistory, { role: "user", content: input }];

    try {
      const parts: string[] = [];
      for (const msg of newHistory) {
        parts.push(`${msg.role}: ${msg.content}`);
      }
      const conversationContext = parts.join("\n");

      const result = await agentChat.mutateAsync({
        message: conversationContext,
        userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Shanghai",
      });

      if (result.response) {
        const updatedHistory: ConversationMessage[] = [...newHistory, { role: "assistant", content: result.response }];
        setConversationHistory(updatedHistory);
        setAgentResponse(result.response);
        setInput("");

        // Check if schedule was created
        const createdRegex = new RegExp(t("schedule.quick-input.created-regex") as string, "i");
        const createdSchedule = createdRegex.test(result.response);
        if (createdSchedule) {
          toast.success(t("schedule.quick-input.schedule-created-success") as string);
          queryClient.invalidateQueries({ queryKey: ["schedules"] });
          if (closeTimeoutRef.current) clearTimeout(closeTimeoutRef.current);
          closeTimeoutRef.current = setTimeout(() => {
            onOpenChange(false);
            onSuccess?.();
          }, 1000);
        }
      }
    } catch (error) {
      console.error("Agent error:", error);
      toast.error((t("schedule.quick-input.parse-failed") as string) + (t("schedule.quick-input.retry-manual-mode") as string));
    } finally {
      setIsProcessing(false);
    }
  }, [input, conversationHistory, agentChat, queryClient, onOpenChange, onSuccess, t]);

  // Handle manual schedule edit
  const handleScheduleUpdate = useCallback((field: keyof ParsedSchedule, value: any) => {
    setParsedSchedule((prev) => ({ ...prev, [field]: value }));
  }, []);

  // Check for conflicts
  const checkForConflicts = useCallback(
    async (scheduleData: Partial<ParsedSchedule>): Promise<boolean> => {
      if (!scheduleData.startTs || !scheduleData.endTs) return false;

      // Validate timestamps before API call
      if (scheduleData.startTs <= 0) {
        console.error("[ScheduleInputPanel] Invalid startTs (must be positive):", scheduleData.startTs);
        toast.error(t("schedule.error.invalid-time") as string);
        return false;
      }
      if (scheduleData.endTs <= scheduleData.startTs) {
        console.error("[ScheduleInputPanel] Invalid time range (endTs <= startTs):", {
          startTs: scheduleData.startTs,
          endTs: scheduleData.endTs,
        });
        toast.error(t("schedule.error.invalid-time-range") as string);
        return false;
      }

      // Extract validated timestamps for type safety (no non-null assertion needed)
      const validatedStartTs = scheduleData.startTs;
      const validatedEndTs = scheduleData.endTs;

      try {
        const result = await checkConflict.mutateAsync({
          startTs: validatedStartTs,
          endTs: validatedEndTs,
        });

        if (result.conflicts.length > 0) {
          const conflictInfos: ConflictInfo[] = result.conflicts.map((s) => ({
            conflictingSchedule: s,
            type: "partial" as const,
            overlapStartTs: validatedStartTs,
            overlapEndTs: validatedEndTs,
          }));
          setConflicts(conflictInfos);
          setShowConflictPanel(true);
          return true;
        }
        return false;
      } catch (error) {
        console.error("[ScheduleInputPanel] Conflict check error:", error);
        return false;
      }
    },
    [checkConflict, t],
  );

  // Handle create schedule
  const handleCreate = useCallback(async () => {
    if (!parsedSchedule?.startTs || !parsedSchedule?.endTs) {
      toast.error(t("schedule.error.set-time-range") as string);
      return;
    }

    // Additional validation for timestamp validity
    if (parsedSchedule.startTs <= 0) {
      toast.error(t("schedule.error.invalid-time") as string);
      return;
    }
    if (parsedSchedule.endTs <= parsedSchedule.startTs) {
      toast.error(t("schedule.error.invalid-time-range") as string);
      return;
    }

    const hasConflict = await checkForConflicts(parsedSchedule);
    if (hasConflict) return;

    try {
      const scheduleName = `schedules/${generateUUID()}`;
      await createSchedule.mutateAsync({
        name: scheduleName,
        title: parsedSchedule.title || (t("schedule.quick-input.default-title") as string),
        startTs: parsedSchedule.startTs,
        endTs: parsedSchedule.endTs,
        allDay: parsedSchedule.allDay || false,
        location: parsedSchedule.location || "",
        description: parsedSchedule.description || "",
        reminders: parsedSchedule.reminders || [],
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      } as any);

      toast.success(t("schedule.quick-input.schedule-created-success") as string);
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
      onOpenChange(false);
      onSuccess?.();
    } catch (error) {
      console.error("Create error:", error);
      toast.error(t("schedule.error.create-failed") as string);
    }
  }, [parsedSchedule, checkForConflicts, createSchedule, queryClient, onOpenChange, onSuccess, t]);

  // Format time for display
  const formatTime = (ts: bigint) => {
    const date = new Date(Number(ts) * 1000);
    return dayjs(date).format("MM/DD HH:mm");
  };

  return (
    <ResizablePanel open={open} onOpenChange={onOpenChange} position="bottom" initialSize={30} minSize={20} maxSize={30}>
      <div className="h-full flex flex-col relative">
        {/* Scrollable content area */}
        <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-4" role="log" aria-live="polite" aria-label={t("schedule.dialog-history") as string}>
          {/* Conversation History */}
          {conversationHistory.map((msg, idx) => (
            <div key={idx} className={cn("flex gap-2 text-sm", msg.role === "user" ? "justify-end" : "justify-start")} role="row">
              {msg.role === "assistant" && (
                <div className="h-6 w-6 rounded-full bg-primary/10 flex items-center justify-center shrink-0 mt-0.5" aria-hidden="true">
                  <Bot className="h-3.5 w-3.5 text-primary" />
                </div>
              )}
              <div
                className={cn("max-w-[80%] rounded-2xl px-3 py-2", msg.role === "user" ? "bg-primary text-primary-foreground" : "bg-muted")}
                role={msg.role === "assistant" ? "article" : "status"}
                aria-label={msg.role === "assistant" ? "AI 助手回复" : "您的消息"}
              >
                {msg.role === "assistant" ? (
                  <div className="prose prose-sm dark:prose-invert max-w-none">
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm, remarkBreaks]}
                      components={{
                        p: ({ node, ...props }) => <p {...props} className="mb-0 last:mb-0" />,
                      }}
                    >
                      {msg.content}
                    </ReactMarkdown>
                  </div>
                ) : (
                  <p className="mb-0">{msg.content}</p>
                )}
              </div>
            </div>
          ))}

          {/* Agent Response */}
          {agentResponse && (
            <div className="flex gap-2 justify-start" role="row">
              <div className="h-6 w-6 rounded-full bg-primary/10 flex items-center justify-center shrink-0 mt-0.5" aria-hidden="true">
                <Bot className="h-3.5 w-3.5 text-primary" />
              </div>
              <div className="max-w-[80%] rounded-2xl px-3 py-2 bg-muted" role="article" aria-label={t("schedule.ai-assistant-reply") as string}>
                <div className="prose prose-sm dark:prose-invert max-w-none">
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm, remarkBreaks]}
                    components={{
                      p: ({ node, ...props }) => <p {...props} className="mb-0 last:mb-0" />,
                    }}
                  >
                    {agentResponse}
                  </ReactMarkdown>
                </div>
              </div>
            </div>
          )}

          {/* Conflict Panel */}
          {showConflictPanel && conflicts.length > 0 && (
            <div
              role="alert"
              aria-live="assertive"
              aria-describedby="conflict-description"
              className="rounded-lg border border-amber-500/30 bg-amber-50/50 dark:bg-amber-950/20 p-3"
            >
              <div className="flex items-start gap-2 text-sm">
                <Bot className="h-4 w-4 text-amber-500 mt-0.5" aria-hidden="true" />
                <div className="flex-1">
                  <p className="font-medium text-amber-700 dark:text-amber-400" id="conflict-description">
                    {t("schedule.quick-input.conflicts-detected") as string} {conflicts.length}{" "}
                    {t("schedule.schedule-count", { count: conflicts.length }) as string}
                  </p>
                  <div className="mt-2 space-y-1" role="list" aria-label={t("schedule.conflicting-schedules") as string}>
                    {conflicts.map((conflict, idx) => (
                      <div key={idx} className="text-xs text-amber-600 dark:text-amber-500" role="listitem">
                        · {conflict.conflictingSchedule.title} ({formatTime(conflict.conflictingSchedule.startTs)})
                      </div>
                    ))}
                  </div>
                  <div className="mt-3 flex gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => setShowConflictPanel(false)}
                      className="h-7 text-xs min-h-[44px] sm:min-h-0 focus-visible:ring-2 focus-visible:ring-amber-500 focus-visible:ring-offset-2"
                      aria-label={t("schedule.cancel-create") as string}
                    >
                      {t("common.cancel") as string}
                    </Button>
                    <Button
                      size="sm"
                      onClick={handleCreate}
                      className="h-7 text-xs min-h-[44px] sm:min-h-0 focus-visible:ring-2 focus-visible:ring-amber-500 focus-visible:ring-offset-2"
                      aria-label={t("schedule.force-create") as string}
                    >
                      {t("schedule.quick-input.force-create") as string}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Manual Schedule Edit Form */}
          {parsedSchedule && (
            <div role="region" aria-label={t("schedule.edit-schedule") as string} className="rounded-lg border bg-muted/50 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h4 className="font-medium text-sm">{t("schedule.edit-schedule") as string}</h4>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setParsedSchedule(null)}
                  className="h-7 w-7 p-0 min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
                  aria-label={t("schedule.edit-form") as string}
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>

              <form
                className="space-y-3 text-sm"
                onSubmit={(e) => {
                  e.preventDefault();
                  handleCreate();
                }}
              >
                {/* Title */}
                <div className="flex items-center gap-2">
                  <Calendar className="h-4 w-4 text-muted-foreground shrink-0" aria-hidden="true" />
                  <Input
                    value={parsedSchedule.title || ""}
                    onChange={(e) => handleScheduleUpdate("title", e.target.value)}
                    className="h-8 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
                    placeholder={t("schedule.quick-input.title-placeholder") as string}
                    aria-label={t("schedule.schedule-title-label") as string}
                    id="schedule-title"
                  />
                </div>

                {/* Time */}
                <div className="flex items-center gap-2">
                  <Clock className="h-4 w-4 text-muted-foreground shrink-0" aria-hidden="true" />
                  <div className="flex items-center gap-2 w-full">
                    <Input
                      type="datetime-local"
                      value={
                        parsedSchedule.startTs && parsedSchedule.startTs > 0
                          ? dayjs(timestampDate(create(TimestampSchema, { seconds: parsedSchedule.startTs, nanos: 0 }))).format(
                              "YYYY-MM-DDTHH:mm",
                            )
                          : ""
                      }
                      onChange={(e) => {
                        if (!e.target.value) {
                          // Clear the timestamp if input is empty
                          handleScheduleUpdate("startTs", undefined);
                          return;
                        }
                        const unix = dayjs(e.target.value).unix();
                        if (Number.isNaN(unix) || unix < 0) {
                          console.warn("[ScheduleInputPanel] Invalid datetime value:", e.target.value);
                          return;
                        }
                        handleScheduleUpdate("startTs", BigInt(unix));
                      }}
                      className="h-8 flex-1 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
                      aria-label={t("schedule.start-time") as string}
                      id="schedule-start-time"
                    />
                    <span className="text-muted-foreground" aria-hidden="true">
                      -
                    </span>
                    <Input
                      type="datetime-local"
                      value={
                        parsedSchedule.endTs && parsedSchedule.endTs > 0
                          ? dayjs(timestampDate(create(TimestampSchema, { seconds: parsedSchedule.endTs, nanos: 0 }))).format(
                              "YYYY-MM-DDTHH:mm",
                            )
                          : ""
                      }
                      onChange={(e) => {
                        if (!e.target.value) {
                          // Clear the timestamp if input is empty
                          handleScheduleUpdate("endTs", undefined);
                          return;
                        }
                        const unix = dayjs(e.target.value).unix();
                        if (Number.isNaN(unix) || unix < 0) {
                          console.warn("[ScheduleInputPanel] Invalid datetime value:", e.target.value);
                          return;
                        }
                        handleScheduleUpdate("endTs", BigInt(unix));
                      }}
                      className="h-8 flex-1 focus-visible:ring-border focus-visible:ring-offset-2"
                      aria-label={t("schedule.end-time") as string}
                      id="schedule-end-time"
                    />
                  </div>
                </div>

                {/* Location */}
                <div className="flex items-center gap-2">
                  <MapPin className="h-4 w-4 text-muted-foreground shrink-0" aria-hidden="true" />
                  <Input
                    value={parsedSchedule.location || ""}
                    onChange={(e) => handleScheduleUpdate("location", e.target.value)}
                    className="h-8 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
                    placeholder={t("schedule.quick-input.location-placeholder") as string}
                    aria-label={t("schedule.location-label") as string}
                    id="schedule-location"
                  />
                </div>
              </form>

              <div className="flex justify-end gap-2 pt-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setParsedSchedule(null)}
                  className="h-8 min-h-[44px] sm:min-h-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
                  aria-label={t("schedule.cancel-edit") as string}
                >
                  {t("common.cancel") as string}
                </Button>
                <Button
                  size="sm"
                  onClick={handleCreate}
                  className="h-8 min-h-[44px] sm:min-h-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
                  aria-label={t("schedule.confirm-create-schedule") as string}
                >
                  {t("schedule.create-schedule") as string}
                </Button>
              </div>
            </div>
          )}
        </div>

        {/* Fixed Input Area */}
        <div className="flex-none px-4 py-3 border-t border-border/50" role="region" aria-label={t("schedule.input-area") as string}>
          <div className="flex items-end gap-2">
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  handleAgentParse();
                }
              }}
              placeholder={t("schedule.quick-input.input-hint") as string}
              className="min-h-[40px] max-h-[120px] resize-none focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
              rows={1}
              aria-label={t("schedule.input-reply") as string}
              id="schedule-input"
            />
            <Button
              onClick={handleAgentParse}
              disabled={!input.trim() || isProcessing}
              className="h-10 px-3 shrink-0 min-h-[44px] min-w-[44px] focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
              aria-label={isProcessing ? "处理中" : "发送消息"}
            >
              {isProcessing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
            </Button>
            <Button
              variant="outline"
              onClick={() => setParsedSchedule({ title: input })}
              disabled={!input.trim()}
              className="h-10 shrink-0 min-h-[44px] min-w-[44px] focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
              aria-label={t("schedule.manual-parse") as string}
            >
              {t("schedule.quick-input.local-parse") as string}
            </Button>
          </div>
        </div>
      </div>
    </ResizablePanel>
  );
}
