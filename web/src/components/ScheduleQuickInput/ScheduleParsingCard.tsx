import { Clock, MapPin, X } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ParsedSchedule, ParseResult } from "./types";
import { useTranslate } from "@/utils/i18n";

interface ScheduleParsingCardProps {
  /** Current parse result */
  parseResult: ParseResult | null;
  /** Pending schedule (e.g., from template selection) */
  pendingSchedule?: Partial<ParsedSchedule> | null;
  /** Whether currently parsing */
  isParsing: boolean;
  /** Called when user wants to edit a field */
  onEditField?: (field: "title" | "startTime" | "endTime" | "location") => void;
  /** Called when user confirms the schedule */
  onConfirm?: () => void;
  /** Called when user dismisses the result */
  onDismiss?: () => void;
  /** Optional className */
  className?: string;
}

export function ScheduleParsingCard({
  parseResult,
  pendingSchedule,
  isParsing,
  onConfirm,
  onDismiss,
  className,
}: ScheduleParsingCardProps) {
  const t = useTranslate();

  if (!parseResult && !pendingSchedule && !isParsing) {
    return null;
  }

  // Loading state
  if (isParsing) {
    return (
      <div className={cn("flex items-center gap-2 px-3 py-2 text-sm text-muted-foreground animate-pulse", className)}>
        <div className="w-2 h-2 rounded-full bg-primary/60 animate-bounce [animation-delay:-0.3s]" />
        <div className="w-2 h-2 rounded-full bg-primary/60 animate-bounce [animation-delay:-0.15s]" />
        <div className="w-2 h-2 rounded-full bg-primary/60 animate-bounce" />
        <span className="ml-1">{t("schedule.quick-input.ai-thinking") as string}</span>
      </div>
    );
  }

  // Use pendingSchedule from template selection, otherwise fall back to parseResult
  const scheduleData = pendingSchedule || parseResult?.parsedSchedule;
  const isTemplateMode = !!pendingSchedule;

  if (!scheduleData) {
    return null;
  }

  const { title, startTs, endTs, allDay, location } = scheduleData;

  // Guard: ensure startTs exists before rendering
  if (!startTs) {
    return null;
  }

  const formatDate = (ts: bigint) => {
    const date = new Date(Number(ts) * 1000);
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);

    const timeStr = date.toLocaleTimeString("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    });

    if (date.toDateString() === today.toDateString()) {
      return { date: t("schedule.quick-input.today") as string, time: timeStr };
    } else if (date.toDateString() === tomorrow.toDateString()) {
      return { date: t("schedule.quick-input.tomorrow") as string, time: timeStr };
    }
    // Simplified date format: month/day, e.g., "1/22"
    return {
      date: `${date.getMonth() + 1}/${date.getDate()}`,
      time: timeStr,
    };
  };

  const startTime = formatDate(startTs);
  const hasEndTime = endTs && Number(endTs) > 0;
  const endTime = hasEndTime ? formatDate(endTs) : null;

  return (
    <div className={cn("flex gap-3 w-full", className)}>
      {/* AI Avatar */}
      <div className="flex-shrink-0 pt-0.5">
        <div
          className={cn(
            "w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-medium shadow-sm",
            isTemplateMode ? "bg-gradient-to-br from-blue-500 to-cyan-600" : "bg-gradient-to-br from-violet-500 to-purple-600",
          )}
        >
          {isTemplateMode ? (t("schedule.quick-input.source-type-template") as string) : (t("schedule.quick-input.source-type-ai") as string)}
        </div>
      </div>

      {/* Message Content */}
      <div className="flex-1 min-w-0">
        {/* AI Message Bubble */}
        <div className="bg-muted/50 rounded-2xl rounded-tl-sm px-3 py-2.5">
          {/* Message Text + Close in header */}
          <div className="flex items-start justify-between gap-2 mb-2">
            <p className="text-sm text-foreground/90 leading-snug">
              {isTemplateMode ? (t("schedule.quick-input.template-confirm-hint") as string) : (t("schedule.quick-input.ai-confirm-hint") as string)}
            </p>
            {onDismiss && (
              <button
                onClick={onDismiss}
                className="flex-shrink-0 p-1 -mt-0.5 -mr-1 text-muted-foreground hover:text-foreground transition-colors rounded-md hover:bg-muted"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>

          {/* Schedule Card */}
          <div className="bg-background rounded-lg border border-border/50 p-2.5 shadow-sm">
            {/* Title + Time in one row for compactness */}
            <div className="flex items-center gap-2 min-w-0">
              {/* Time badge */}
              <div className="flex-shrink-0 flex items-center gap-1 px-2 py-1 rounded-md bg-primary/10 text-primary text-xs font-medium">
                <Clock className="w-3 h-3" />
                <span>{allDay ? (t("schedule.all-day") as string) : startTime.time}</span>
              </div>

              {/* Title */}
              <h4 className="font-medium text-sm truncate">{title}</h4>
            </div>

            {/* Date row (show if not today or has end time) */}
            <div className="flex items-center gap-3 mt-1.5 text-xs text-muted-foreground">
              <span>{startTime.date}</span>
              {endTime && (
                <>
                  <span className="text-muted-foreground/30">→</span>
                  <span>
                    {endTime.date !== startTime.date ? `${endTime.date} ` : ""}
                    {endTime.time}
                  </span>
                </>
              )}
            </div>

            {/* Location */}
            {location && (
              <div className="flex items-center gap-1.5 mt-1.5 text-xs text-muted-foreground">
                <MapPin className="w-3 h-3 flex-shrink-0" />
                <span className="truncate">{location}</span>
              </div>
            )}
          </div>

          {/* Confirm Button - Full width, prominent */}
          {onConfirm && (
            <button
              onClick={onConfirm}
              className="w-full mt-2.5 px-3 py-2 bg-primary text-primary-foreground text-sm font-medium rounded-lg hover:bg-primary/90 transition-colors shadow-sm"
            >
              {t("schedule.quick-input.confirm-create") as string}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
