import { AlertCircle, Bot, CheckCircle2, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import type { ParseResult, ParseSource } from "./types";

interface ScheduleParsingIndicatorProps {
  /** Current parse result */
  parseResult: ParseResult | null;
  /** Whether currently parsing */
  isParsing: boolean;
  /** Source of parsing (local or AI) */
  parseSource?: ParseSource | null;
  /** Optional className */
  className?: string;
}

export function ScheduleParsingIndicator({ parseResult, isParsing, parseSource, className }: ScheduleParsingIndicatorProps) {
  const t = useTranslate();

  if (!parseResult && !isParsing) {
    return null;
  }

  const formatTime = (ts: bigint) => {
    const date = new Date(Number(ts) * 1000);
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);

    const timeStr = date.toLocaleTimeString("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    });

    const todayStr = t("schedule.quick-input.today") as string;
    const tomorrowStr = t("schedule.quick-input.tomorrow") as string;

    if (date.toDateString() === today.toDateString()) {
      return `${todayStr} ${timeStr}`;
    } else if (date.toDateString() === tomorrow.toDateString()) {
      return `${tomorrowStr} ${timeStr}`;
    }
    return date.toLocaleDateString("zh-CN", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  // Loading state
  if (isParsing) {
    return (
      <div
        className={cn("flex items-center gap-2 text-xs text-muted-foreground", className)}
        role="status"
        aria-live="polite"
        aria-label={parseSource === "ai" ? "AI 正在解析" : "正在解析"}
      >
        <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
        <span>{parseSource === "ai" ? (t("schedule.quick-input.ai-parsing") as string) : (t("schedule.quick-input.parsing") as string)}</span>
      </div>
    );
  }

  if (!parseResult) {
    return null;
  }

  // Success state
  if (parseResult.state === "success" && parseResult.parsedSchedule) {
    const { title, startTs, endTs, allDay, location } = parseResult.parsedSchedule;
    const hasEndTime = endTs && Number(endTs) > 0;
    const endTimeStr = hasEndTime
      ? new Date(Number(endTs) * 1000).toLocaleTimeString("zh-CN", {
          hour: "2-digit",
          minute: "2-digit",
        })
      : null;

    return (
      <div
        className={cn("flex items-center gap-2 text-xs", className)}
        role="status"
        aria-live="polite"
        aria-label={`解析成功：${title}`}
      >
        <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" aria-hidden="true" />
        <div className="min-w-0 flex-1">
          <span className="font-medium text-emerald-700 dark:text-emerald-400">{title}</span>
          <span className="text-muted-foreground mx-1" aria-hidden="true">·</span>
          <span className="text-muted-foreground">
            {allDay ? (t("schedule.all-day") as string) : formatTime(startTs)}
            {!allDay && endTimeStr && (
              <>
                <span className="mx-0.5" aria-hidden="true">-</span>
                <span>{endTimeStr}</span>
              </>
            )}
          </span>
          {location && (
            <>
              <span className="text-muted-foreground mx-1" aria-hidden="true">·</span>
              <span className="text-muted-foreground">@{location}</span>
            </>
          )}
        </div>
        {parseSource === "local" && <span className="text-[10px] px-1 py-0.5 rounded bg-primary/10 text-primary">{t("schedule.quick-input.local-parse") as string}</span>}
      </div>
    );
  }

  // Partial state - needs more info
  if (parseResult.state === "partial") {
    return (
      <div
        className={cn("flex items-center gap-2 text-xs", className)}
        role="status"
        aria-live="polite"
        aria-label="需要更多信息"
      >
        <AlertCircle className="h-3.5 w-3.5 text-amber-500 shrink-0" aria-hidden="true" />
        <span className="text-amber-700 dark:text-amber-400">{parseResult.message || (t("schedule.quick-input.parse-partial") as string)}</span>
      </div>
    );
  }

  // Error state
  if (parseResult.state === "error") {
    return (
      <div
        className={cn("flex items-center gap-2 text-xs", className)}
        role="alert"
        aria-live="assertive"
        aria-label="解析失败"
      >
        <AlertCircle className="h-3.5 w-3.5 text-destructive shrink-0" aria-hidden="true" />
        <span className="text-destructive">{parseResult.message || (t("schedule.quick-input.parse-error") as string)}</span>
      </div>
    );
  }

  return null;
}

/**
 * Compact version that shows just an icon
 */
interface CompactIndicatorProps {
  parseResult: ParseResult | null;
  isParsing: boolean;
  className?: string;
}

export function CompactParsingIndicator({ parseResult, isParsing, className }: CompactIndicatorProps) {
  if (isParsing) {
    return (
      <div className={cn("relative", className)} role="status" aria-live="polite" aria-label="正在解析">
        <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" aria-hidden="true" />
      </div>
    );
  }

  if (parseResult?.state === "success") {
    return (
      <div className={cn("relative", className)} role="status" aria-live="polite" aria-label="解析成功">
        <CheckCircle2 className="h-4 w-4 text-emerald-500" aria-hidden="true" />
      </div>
    );
  }

  if (parseResult?.state === "partial") {
    return (
      <div className={cn("relative", className)} role="status" aria-live="polite" aria-label="需要更多信息">
        <Bot className="h-4 w-4 text-amber-500" aria-hidden="true" />
      </div>
    );
  }

  if (parseResult?.state === "error") {
    return (
      <div className={cn("relative", className)} role="alert" aria-live="assertive" aria-label="解析失败">
        <AlertCircle className="h-4 w-4 text-destructive" aria-hidden="true" />
      </div>
    );
  }

  return null;
}
