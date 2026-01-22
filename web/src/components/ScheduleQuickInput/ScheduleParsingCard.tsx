import { Clock, MapPin, X } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ParsedSchedule, ParseResult } from "./types";

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
  onEditField,
  onConfirm,
  onDismiss,
  className,
}: ScheduleParsingCardProps) {
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
        <span className="ml-1">AI 正在理解您的需求...</span>
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
      return { date: "今天", time: timeStr };
    } else if (date.toDateString() === tomorrow.toDateString()) {
      return { date: "明天", time: timeStr };
    }
    // 简化日期格式：月/日，如 "1/22"
    return {
      date: `${date.getMonth() + 1}/${date.getDate()}`,
      time: timeStr,
    };
  };

  const startTime = formatDate(startTs);
  const hasEndTime = endTs && Number(endTs) > 0;
  const endTime = hasEndTime ? formatDate(endTs) : null;

  return (
    <div className={cn("flex gap-3 w-full overflow-hidden", className)}>
      {/* AI Avatar */}
      <div className="flex-shrink-0">
        <div
          className={cn(
            "w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-medium shadow-sm",
            isTemplateMode ? "bg-gradient-to-br from-blue-500 to-cyan-600" : "bg-gradient-to-br from-violet-500 to-purple-600",
          )}
        >
          {isTemplateMode ? "模板" : "AI"}
        </div>
      </div>

      {/* Message Content */}
      <div className="flex-1 min-w-0 overflow-hidden">
        <div className="w-full max-w-full">
          {/* AI Message Bubble */}
          <div className="bg-muted/50 rounded-2xl rounded-tl-sm px-4 py-3">
            <p className="text-sm text-foreground mb-3">{isTemplateMode ? "快速创建日程，请确认：" : "我为您识别到以下日程，请确认："}</p>

            {/* Schedule Card */}
            <div className="bg-background rounded-xl border border-border/50 p-3 shadow-sm overflow-hidden">
              {/* Title */}
              <div className="flex items-start justify-between gap-2 mb-2 min-w-0">
                <h4 className="font-medium text-base truncate">{title}</h4>
              </div>

              {/* Time */}
              <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1.5 flex-wrap">
                <Clock className="w-3.5 h-3.5 flex-shrink-0" />
                <span className="flex items-center gap-1 flex-wrap">
                  {allDay ? (
                    <span className="text-foreground">全天</span>
                  ) : (
                    <>
                      <span className="text-foreground">{startTime.date}</span>
                      <span className="text-muted-foreground/50">·</span>
                      <span className="font-medium text-foreground">{startTime.time}</span>
                      {endTime && (
                        <>
                          <span className="text-muted-foreground/50">-</span>
                          {endTime.date !== startTime.date && <span className="text-foreground">{endTime.date}</span>}
                          <span className="font-medium text-foreground">{endTime.time}</span>
                        </>
                      )}
                    </>
                  )}
                </span>
              </div>

              {/* Location */}
              {location && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground min-w-0">
                  <MapPin className="w-3.5 h-3.5 flex-shrink-0" />
                  <span className="truncate">{location}</span>
                </div>
              )}
            </div>

            {/* Action Buttons */}
            <div className="flex items-center gap-2 mt-3">
              {onConfirm && (
                <button
                  onClick={onConfirm}
                  className="px-3 py-1.5 bg-primary text-primary-foreground text-sm font-medium rounded-lg hover:bg-primary/90 transition-colors"
                >
                  确认创建
                </button>
              )}
              {onEditField && (
                <button
                  onClick={() => onEditField("title")}
                  className="px-3 py-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
                >
                  修改
                </button>
              )}
              {onDismiss && (
                <button onClick={onDismiss} className="p-1.5 text-muted-foreground hover:text-destructive transition-colors">
                  <X className="w-4 h-4" />
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
