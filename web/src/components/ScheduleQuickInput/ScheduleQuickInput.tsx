import { Loader2, Plus, Send, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useScheduleContext } from "@/contexts/ScheduleContext";
import { useCheckConflict, useCreateSchedule, useSchedulesOptimized } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";
import { AISuggestionCards, type ScheduleSuggestion } from "./AISuggestionCards";
import { ConflictSuggestions } from "./ConflictSuggestions";
import { useConflictDetection } from "./hooks/useConflictDetection";
import { extractScheduleFromParse, useScheduleParse } from "./hooks/useScheduleParse";
import { QuickTemplateDropdown } from "./QuickTemplates";
import { ScheduleParsingCard } from "./ScheduleParsingCard";
import type { ConflictInfo, ParsedSchedule, ScheduleTemplate, SuggestedTimeSlot } from "./types";

interface ScheduleQuickInputProps {
  /** Optional initial date from calendar click */
  initialDate?: string;
  /** Called when schedule is created */
  onScheduleCreated?: () => void;
  /** Optional className */
  className?: string;
}

/** Max input height before scrolling */
const MAX_INPUT_HEIGHT = 120;
/** Line height for auto-resize calculation */
const LINE_HEIGHT = 24;

/**
 * Generate a UUID with fallback for environments where crypto.randomUUID() is unavailable.
 */
function generateUUID(): string {
  try {
    if (typeof crypto !== "undefined" && crypto.randomUUID) {
      return crypto.randomUUID();
    }
  } catch {
    // Fall through to manual generation
  }
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export function ScheduleQuickInput({ initialDate, onScheduleCreated, className }: ScheduleQuickInputProps) {
  const { selectedDate } = useScheduleContext();
  const createSchedule = useCreateSchedule();
  const checkConflict = useCheckConflict();
  const t = useTranslate();

  // Input state
  const [input, setInput] = useState("");
  const [inputHeight, setInputHeight] = useState(LINE_HEIGHT);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // UI state
  const [showTemplates, setShowTemplates] = useState(false);
  const [showConflictPanel, setShowConflictPanel] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [conflicts, setConflicts] = useState<ConflictInfo[]>([]);
  const [pendingSchedule, setPendingSchedule] = useState<Partial<ParsedSchedule> | null>(null);
  const [showParsingCard, setShowParsingCard] = useState(false);
  const [skipParse, setSkipParse] = useState(false);
  const [aiSuggestions, setAiSuggestions] = useState<ScheduleSuggestion[]>([]);

  // Reference date for parsing
  const referenceDate = useMemo(() => {
    const dateStr = initialDate || selectedDate;
    return dateStr ? new Date(dateStr + "T00:00:00") : new Date();
  }, [initialDate, selectedDate]);

  // Get existing schedules for conflict detection
  const { data: schedulesData } = useSchedulesOptimized(referenceDate);
  const existingSchedules = schedulesData?.schedules || [];

  // Parse hook - disabled auto parse, require manual trigger
  const { parseResult, isParsing, parse, reset } = useScheduleParse({
    debounceMs: 600,
    minLength: 2,
    enableAI: true,
    referenceDate,
    autoParse: false,
  });

  // Conflict detection
  const { suggestions } = useConflictDetection({
    startTs: pendingSchedule?.startTs,
    endTs: pendingSchedule?.endTs,
    existingSchedules,
    t: t as (key: string) => string | unknown,
  });

  // Store latest parse/reset in refs for useEffect
  const parseRef = useRef(parse);
  const resetRef = useRef(reset);

  // Update refs when functions change
  useEffect(() => {
    parseRef.current = parse;
    resetRef.current = reset;
  }, [parse, reset]);

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    const resize = () => {
      if (!textareaRef.current) return;
      textareaRef.current.style.height = "auto";
      const newHeight = Math.min(Math.max(textareaRef.current.scrollHeight, LINE_HEIGHT), MAX_INPUT_HEIGHT);
      textareaRef.current.style.height = `${newHeight}px`;
      setInputHeight(newHeight);
    };

    resize();
    window.addEventListener("resize", resize);
    return () => window.removeEventListener("resize", resize);
  }, [input]);

  // Show/hide parsing card based on parse result
  useEffect(() => {
    if (parseResult?.state === "success" && parseResult.parsedSchedule) {
      setShowParsingCard(true);
      setAiSuggestions([]); // Clear suggestions when parsed successfully
    } else if (parseResult?.state === "error" || parseResult?.state === "partial") {
      setShowParsingCard(false);
      // Extract suggestions from AI response when partial
      if (parseResult?.state === "partial" && parseResult?.message) {
        // Import parseSuggestions dynamically to avoid circular dependency
        import("./AISuggestionCards").then(({ parseSuggestions: parse }) => {
          const todayStr = t("schedule.quick-input.today") as string;
          const tomorrowStr = t("schedule.quick-input.tomorrow") as string;
          const suggestions = parse(parseResult.message || "", todayStr, tomorrowStr);
          setAiSuggestions(suggestions);
        });
      } else {
        setAiSuggestions([]);
      }
    }
  }, [parseResult, t]);

  // Parse input on change
  useEffect(() => {
    if (skipParse) {
      setSkipParse(false);
      return;
    }
    if (input.trim()) {
      parseRef.current(input);
    } else {
      resetRef.current();
      setPendingSchedule(null);
      setShowParsingCard(false);
      setShowConflictPanel(false);
    }
  }, [input, skipParse]);

  // Handle input change
  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);
  };

  // Handle keyboard shortcuts
  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSendOrParse();
    }
    if (e.key === "Escape" && input) {
      e.preventDefault();
      handleClear();
    }
  };

  // Handle send button click - parse first if not parsed, otherwise create
  const handleSendOrParse = async () => {
    if (!input.trim()) return;

    // If already has successful parse result, create directly
    if (parseResult?.state === "success" || pendingSchedule) {
      await handleCreateSchedule();
      return;
    }

    // Otherwise, trigger AI parsing with forceAI=true
    await parse(input, true);
  };

  // Handle template selection
  const handleTemplateSelect = (template: ScheduleTemplate) => {
    const durationSeconds = template.duration * 60;
    const now = new Date();

    // Calculate start time based on referenceDate (calendar selection)
    let startDate = new Date(referenceDate);

    // Check if reference date is today
    const isToday =
      startDate.getFullYear() === now.getFullYear() &&
      startDate.getMonth() === now.getMonth() &&
      startDate.getDate() === now.getDate();

    if (isToday) {
      // Today: use current time rounded up to next 30min slot
      const currentMinutes = now.getHours() * 60 + now.getMinutes();
      const roundedUpMinutes = Math.ceil(currentMinutes / 30) * 30;
      const startMinutes = roundedUpMinutes === currentMinutes ? roundedUpMinutes + 30 : roundedUpMinutes;
      startDate.setHours(Math.floor(startMinutes / 60), startMinutes % 60, 0, 0);
    } else {
      // Future date: default to 9:00 AM
      startDate.setHours(9, 0, 0, 0);
    }

    const startTs = Math.floor(startDate.getTime() / 1000);

    const scheduleData: Partial<ParsedSchedule> = {
      title: template.defaultTitle || template.title,
      startTs: BigInt(startTs),
      endTs: BigInt(startTs + durationSeconds),
      confidence: 1,
      source: "local",
    };

    // Skip parsing when setting input from template
    setSkipParse(true);
    setPendingSchedule(scheduleData);
    setInput(template.title);
    setShowParsingCard(true);
    setAiSuggestions([]); // Clear AI suggestions when template is selected
  };

  // Handle AI suggestion selection
  const handleAISuggestionSelect = (suggestion: ScheduleSuggestion) => {
    let startDate = new Date(referenceDate);

    // Parse the suggestion date and time
    const todayStr = t("schedule.quick-input.today") as string;
    const tomorrowStr = t("schedule.quick-input.tomorrow") as string;
    const isToday = suggestion.date === todayStr;
    const isTomorrow = suggestion.date === tomorrowStr;

    if (isToday) {
      // Use today's date with suggested time
      startDate = new Date();
    } else if (isTomorrow) {
      // Use tomorrow's date
      startDate = new Date();
      startDate.setDate(startDate.getDate() + 1);
    } else {
      // Use the reference date (already set to calendar selection)
      startDate = new Date(referenceDate);
    }

    // Parse the start time (format: "HH:mm")
    const [hours, minutes] = suggestion.startTime.split(":").map(Number);
    startDate.setHours(hours, minutes, 0, 0);

    const startTs = Math.floor(startDate.getTime() / 1000);

    // Calculate end time
    let endTs = startTs + 3600; // Default 1 hour
    if (suggestion.endTime) {
      const [endHours, endMinutes] = suggestion.endTime.split(":").map(Number);
      const endDate = new Date(startDate);
      endDate.setHours(endHours, endMinutes, 0, 0);
      endTs = Math.floor(endDate.getTime() / 1000);
    }

    const scheduleData: Partial<ParsedSchedule> = {
      title: suggestion.title,
      startTs: BigInt(startTs),
      endTs: BigInt(endTs),
      confidence: 0.8,
      source: "ai",
    };

    setSkipParse(true);
    setPendingSchedule(scheduleData);
    setInput(suggestion.title);
    setShowParsingCard(true);
    setAiSuggestions([]); // Clear suggestions after selection
  };

  // Check for conflicts
  const checkForConflicts = async (scheduleData: Partial<ParsedSchedule>): Promise<boolean> => {
    // Validate timestamps before API call
    if (!scheduleData.startTs || !scheduleData.endTs) {
      console.error("[ScheduleQuickInput] Missing timestamps:", { startTs: scheduleData.startTs, endTs: scheduleData.endTs });
      return false;
    }
    if (scheduleData.startTs <= 0) {
      console.error("[ScheduleQuickInput] Invalid startTs (must be positive):", scheduleData.startTs);
      toast.error((t as any)("schedule.error.invalid-time") || "无效的时间，请重新输入");
      return false;
    }
    if (scheduleData.endTs <= scheduleData.startTs) {
      console.error("[ScheduleQuickInput] Invalid time range (endTs <= startTs):", { startTs: scheduleData.startTs, endTs: scheduleData.endTs });
      toast.error((t as any)("schedule.error.invalid-time-range") || "结束时间必须晚于开始时间");
      return false;
    }

    try {
      const result = await checkConflict.mutateAsync({
        startTs: scheduleData.startTs,
        endTs: scheduleData.endTs,
      });

      if (result.conflicts.length > 0) {
        const conflictInfos: ConflictInfo[] = result.conflicts.map((s) => ({
          conflictingSchedule: s,
          type: "partial" as const,
          overlapStartTs: scheduleData.startTs!,
          overlapEndTs: scheduleData.endTs!,
        }));
        setConflicts(conflictInfos);
        setShowConflictPanel(true);
        return true;
      } else {
        setConflicts([]);
        setShowConflictPanel(false);
        return false;
      }
    } catch (error) {
      console.error("[ScheduleQuickInput] Conflict check error:", error);
      return false;
    }
  };

  // Handle create schedule
  const handleCreateSchedule = async (overrideScheduleData?: Partial<ParsedSchedule>) => {
    if (isCreating) return;

    let scheduleData = overrideScheduleData || pendingSchedule;
    if (!scheduleData && parseResult?.parsedSchedule) {
      scheduleData = extractScheduleFromParse(parseResult);
    }

    if (!scheduleData) {
      const now = Math.floor(Date.now() / 1000);
      scheduleData = {
        title: input.trim() || t("schedule.untitled") || "Untitled Schedule",
        startTs: BigInt(now),
        endTs: BigInt(now + 3600),
        confidence: 0.5,
        source: "manual",
      };
    }

    setIsCreating(true);

    try {
      const hasConflict = await checkForConflicts(scheduleData);
      if (hasConflict) {
        setIsCreating(false);
        return;
      }

      const createRequest: Partial<Schedule> = {
        name: `schedules/${generateUUID()}`,
        title: scheduleData.title || t("schedule.untitled") || "Untitled Schedule",
        startTs: scheduleData.startTs,
        endTs: scheduleData.endTs,
        allDay: scheduleData.allDay || false,
        location: scheduleData.location || "",
        description: scheduleData.description || "",
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      };

      if (scheduleData.reminders && scheduleData.reminders.length > 0) {
        (createRequest as any).reminders = scheduleData.reminders;
      }

      const createdSchedule = await createSchedule.mutateAsync(createRequest);

      if (createdSchedule) {
        toast.success(t("schedule.schedule-created") || "Schedule created successfully");
        handleClear();
        onScheduleCreated?.();
      }
    } catch (error) {
      console.error("[ScheduleQuickInput] Create error:", error);
      toast.error(t("schedule.parse-error") || "Failed to create, please try again");
    } finally {
      setIsCreating(false);
    }
  };

  // Handle clear
  const handleClear = () => {
    setInput("");
    setPendingSchedule(null);
    reset();
    setShowParsingCard(false);
    setShowConflictPanel(false);
    setConflicts([]);
    if (textareaRef.current) {
      textareaRef.current.style.height = `${LINE_HEIGHT}px`;
      setInputHeight(LINE_HEIGHT);
    }
  };

  // Handle dismiss parsing card
  const handleDismissCard = () => {
    setShowParsingCard(false);
  };

  // Handle conflict resolution
  // Handle conflict time slot suggestion
  const handleSuggestionSelect = async (slot: SuggestedTimeSlot) => {
    if (!pendingSchedule) return;

    // Create updated schedule data immediately (not relying on state update)
    const updatedSchedule: Partial<ParsedSchedule> = {
      ...pendingSchedule,
      startTs: slot.startTs,
      endTs: slot.endTs,
    };

    // Update state for display purposes
    setPendingSchedule(updatedSchedule);
    setShowConflictPanel(false);

    // Create directly with updated data (avoiding async state timing issue)
    await handleCreateSchedule(updatedSchedule);
  };

  const handleForceCreate = async () => {
    setShowConflictPanel(false);
    await handleCreateSchedule();
  };

  const canCreate = parseResult?.state === "success" || pendingSchedule !== null;

  return (
    <div className={cn("w-full flex flex-col gap-3", className)}>
      {/* Conflict Panel */}
      {showConflictPanel && (
        <ConflictSuggestions
          conflicts={conflicts}
          suggestions={suggestions}
          onSuggestionSelect={handleSuggestionSelect}
          onForceCreate={handleForceCreate}
          onCancel={() => setShowConflictPanel(false)}
        />
      )}

      {/* AI Parsing Card - shown above input when parsing succeeds */}
      {showParsingCard && (parseResult || pendingSchedule) && (
        <ScheduleParsingCard
          parseResult={parseResult}
          pendingSchedule={pendingSchedule}
          isParsing={isParsing}
          onConfirm={handleCreateSchedule}
          onDismiss={handleDismissCard}
        />
      )}

      {/* AI Suggestions Cards - shown when AI returns partial result with suggestions */}
      {aiSuggestions.length > 0 && !showParsingCard && (
        <AISuggestionCards
          suggestions={aiSuggestions}
          onConfirmSuggestion={handleAISuggestionSelect}
        />
      )}

      {/* Loading indicator - shown inline when parsing */}
      {isParsing && !showParsingCard && (
        <div className="flex items-center gap-2 px-3 py-2.5 text-sm text-muted-foreground bg-primary/5 rounded-lg border border-primary/10">
          <Loader2 className="h-4 w-4 animate-spin text-primary" />
          <span className="">{t("schedule.quick-input.ai-parsing") || "AI 正在理解您的需求..."}</span>
        </div>
      )}

      {/* Input Bar */}
      <div
        className={cn(
          "flex items-center gap-2 p-2 rounded-xl border-2 transition-all duration-200",
          isParsing && "border-primary/30 bg-primary/5",
          showParsingCard && "border-emerald-500/30 bg-emerald-50/50 dark:bg-emerald-950/20",
          !isParsing && !showParsingCard && "border-border bg-background",
        )}
      >
        {/* Templates Dropdown */}
        <div className="flex-shrink-0">
          <QuickTemplateDropdown open={showTemplates} onToggle={() => setShowTemplates(!showTemplates)} onSelect={handleTemplateSelect} />
        </div>

        {/* Text Input */}
        <div className="flex-1 min-w-0">
          <Textarea
            ref={textareaRef}
            value={input}
            onChange={handleInputChange}
            onKeyDown={handleKeyDown}
            placeholder={t("schedule.quick-input.placeholder") || "例：明天下午3点开会"}
            className={cn(
              "min-h-[24px] max-h-[120px] py-1.5 px-3 resize-none",
              "border-0 bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0",
              "text-sm",
            )}
            style={{ height: `${inputHeight}px` }}
            rows={1}
          />
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-1 flex-shrink-0">
          {isCreating ? (
            <Button
              size="sm"
              variant="ghost"
              className="h-8 w-8 p-0 min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0"
              aria-label="创建中"
            >
              <Loader2 className="h-4 w-4 animate-spin" />
            </Button>
          ) : canCreate ? (
            <Button
              size="sm"
              onClick={handleSendOrParse}
              className="h-9 px-3 gap-1.5 min-h-[44px] sm:min-h-0"
              aria-label="确认创建"
            >
              <Send className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">{t("schedule.quick-input.confirm") || "确认"}</span>
            </Button>
          ) : input.length > 0 ? (
            <>
              <Button
                size="sm"
                variant="ghost"
                onClick={handleClear}
                className="h-8 w-8 p-0 min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0"
                aria-label="清除"
              >
                <X className="h-4 w-4" />
              </Button>
              <Button
                size="sm"
                onClick={handleSendOrParse}
                className="h-8 px-2 gap-1.5 min-h-[44px] sm:min-h-0"
                aria-label="AI 解析"
              >
                <Send className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : (
            <Button
              size="sm"
              variant="ghost"
              onClick={() => setShowTemplates(true)}
              className="h-8 w-8 p-0 min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0"
              aria-label="显示模板"
            >
              <Plus className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
