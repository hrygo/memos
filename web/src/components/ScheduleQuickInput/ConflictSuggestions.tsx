import { AlertTriangle, Calendar, Clock, X } from "lucide-react";
import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import type { ConflictInfo, SuggestedTimeSlot } from "./types";

interface ConflictSuggestionsProps {
  /** List of conflicting schedules */
  conflicts: ConflictInfo[];
  /** Suggested alternative time slots */
  suggestions: SuggestedTimeSlot[];
  /** Called when user selects a suggested time slot */
  onSuggestionSelect?: (slot: SuggestedTimeSlot) => void;
  /** Called when user chooses to create anyway */
  onForceCreate?: () => void;
  /** Called when user cancels */
  onCancel?: () => void;
  /** Optional className */
  className?: string;
}

export function ConflictSuggestions({
  conflicts,
  suggestions,
  onSuggestionSelect,
  onForceCreate,
  onCancel,
  className,
}: ConflictSuggestionsProps) {
  const t = useTranslate();
  const panelRef = useRef<HTMLDivElement>(null);

  // Handle Escape key to close
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && onCancel) {
        onCancel();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onCancel]);

  // Auto-focus panel when conflicts appear
  useEffect(() => {
    if (panelRef.current && (conflicts.length > 0 || suggestions.length > 0)) {
      panelRef.current.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
  }, [conflicts.length, suggestions.length]);

  if (conflicts.length === 0 && suggestions.length === 0) {
    return null;
  }

  const formatTime = (ts: bigint) => {
    return new Date(Number(ts) * 1000).toLocaleTimeString("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const formatDate = (ts: bigint) => {
    const date = new Date(Number(ts) * 1000);
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);

    const todayStr = t("schedule.quick-input.today") as string;
    const tomorrowStr = t("schedule.quick-input.tomorrow") as string;

    if (date.toDateString() === today.toDateString()) {
      return todayStr;
    } else if (date.toDateString() === tomorrow.toDateString()) {
      return tomorrowStr;
    }
    return date.toLocaleDateString("zh-CN", { month: "short", day: "numeric" });
  };

  return (
    <div
      ref={panelRef}
      role="alertdialog"
      aria-describedby="conflict-description"
      className={cn(
        "rounded-xl border-2 border-red-200 bg-red-50 dark:border-red-500/50 dark:bg-red-950/40 p-4 space-y-3 shadow-sm",
        className,
      )}
    >
      {/* Header with close button */}
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2 text-red-600 dark:text-red-400">
          <AlertTriangle className="h-5 w-5 mt-0.5 shrink-0" aria-hidden="true" />
          <div>
            <div className="font-semibold text-sm" id="conflict-title">
              {conflicts.length === 1
                ? (t("schedule.conflict.title") as string)
                : `${t("schedule.conflict.title") as string} (${conflicts.length} ${t("schedule.schedules") as string})`}
            </div>
            <div className="text-xs text-red-500/80 dark:text-red-400/80 mt-0.5" id="conflict-description">
              {t("schedule.conflict-suggestion-hint") as string}
            </div>
          </div>
        </div>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            aria-label={t("schedule.close-conflict-alert") as string}
            className={cn(
              "text-red-400 hover:text-red-600 dark:text-red-500 dark:hover:text-red-300 transition-colors",
              "min-h-[36px] min-w-[36px] flex items-center justify-center rounded-full hover:bg-red-100/50 dark:hover:bg-red-900/20",
              "focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2",
            )}
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>

      {/* Conflicting Schedules */}
      {conflicts.length > 0 && (
        <div className="space-y-2">
          {conflicts.map(({ conflictingSchedule, type }) => (
            <div
              key={conflictingSchedule.name}
              className={cn(
                "flex items-center gap-2 rounded-lg px-3 py-2.5 text-xs",
                "bg-white/60 dark:bg-black/20",
                "border border-red-200/50 dark:border-red-500/20",
              )}
            >
              <Calendar className="h-3.5 w-3.5 text-red-500 shrink-0" />
              <div className="min-w-0 flex-1">
                <div className="font-medium text-foreground truncate">{conflictingSchedule.title}</div>
                <div className="text-muted-foreground">
                  {formatDate(conflictingSchedule.startTs)} {formatTime(conflictingSchedule.startTs)} -{" "}
                  {formatTime(conflictingSchedule.endTs)}
                </div>
              </div>
              <div className="text-[10px] px-1.5 py-0.5 rounded bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-300 font-medium">
                {type === "full" ? (t("schedule.conflict.type-full") as string) : (t("schedule.conflict.type-partial") as string)}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Suggestions */}
      {suggestions.length > 0 && (
        <div className="space-y-2">
          <div
            className="grid grid-cols-1 sm:grid-cols-2 gap-2"
            role="listbox"
            aria-label={(t("schedule.quick-input.suggested-times") as string) || "建议时段"}
          >
            {suggestions.map((slot, index) => (
              <button
                key={index}
                type="button"
                role="option"
                onClick={() => onSuggestionSelect?.(slot)}
                className={cn(
                  "flex items-center gap-2 rounded-lg px-3 py-2.5 text-left text-sm transition-all",
                  "bg-white dark:bg-black/30",
                  "border-2 border-emerald-200 dark:border-emerald-500/30",
                  "hover:border-emerald-400 hover:bg-emerald-50 min-h-[44px] sm:min-h-0",
                  "dark:hover:border-emerald-400 dark:hover:bg-emerald-950/30",
                  "focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:ring-offset-2",
                )}
              >
                <Clock className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400 shrink-0" aria-hidden="true" />
                <span className="font-medium text-emerald-700 dark:text-emerald-300">{slot.label}</span>
                {slot.reason && <span className="text-xs text-muted-foreground">({slot.reason})</span>}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between pt-3 border-t border-red-200/50 dark:border-red-500/20">
        <Button
          variant="ghost"
          size="sm"
          onClick={onCancel}
          className="h-9 text-xs text-muted-foreground hover:text-foreground min-h-[44px] sm:min-h-0 px-4"
        >
          {t("common.cancel") as string}
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={onForceCreate}
          className={cn(
            "h-9 text-xs min-h-[44px] sm:min-h-0 px-4",
            "border-red-200 text-red-600 hover:bg-red-50",
            "dark:border-red-500/30 dark:text-red-400 dark:hover:bg-red-950/30",
          )}
        >
          {t("schedule.create-anyway") as string}
        </Button>
      </div>
    </div>
  );
}
