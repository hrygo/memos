import { AlertTriangle, Calendar, Clock, Check, ChevronDown, ChevronUp, Zap } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";

// Conflict information
export interface ScheduleConflict {
  schedule: Schedule;
  conflictType: "full" | "partial"; // Full overlap or partial overlap
  overlapStart: Date;
  overlapEnd: Date;
}

// Suggested time slot for rescheduling
export interface TimeSlotSuggestion {
  startTime: Date;
  endTime: Date;
  label: string;
  reason: string;
  confidence: "high" | "medium" | "low";
}

interface ConflictResolutionPanelProps {
  newSchedule: {
    title: string;
    startTime: Date;
    endTime: Date;
  };
  conflicts: ScheduleConflict[];
  suggestions?: TimeSlotSuggestion[];
  onResolveWithSuggestion?: (suggestion: TimeSlotSuggestion) => void;
  onResolveManually?: () => void;
  onOverride?: () => void;
  onCancel?: () => void;
  className?: string;
}

export function ConflictResolutionPanel({
  newSchedule,
  conflicts,
  suggestions = [],
  onResolveWithSuggestion,
  onResolveManually,
  onOverride,
  onCancel,
  className,
}: ConflictResolutionPanelProps) {
  const t = useTranslate();
  const [expandedConflicts, setExpandedConflicts] = useState<Set<number>>(new Set([0]));
  const [expandedSuggestions, setExpandedSuggestions] = useState(true);

  const toggleConflict = (index: number) => {
    setExpandedConflicts((prev) => {
      const next = new Set(prev);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
  };

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
  };

  const formatDate = (date: Date) => {
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);

    if (date.toDateString() === today.toDateString()) {
      return t("schedule.quick-input.today") || "今天";
    } else if (date.toDateString() === tomorrow.toDateString()) {
      return t("schedule.quick-input.tomorrow") || "明天";
    }
    return date.toLocaleDateString("zh-CN", { month: "short", day: "numeric" });
  };

  const getConfidenceColor = (confidence: TimeSlotSuggestion["confidence"]) => {
    switch (confidence) {
      case "high":
        return "bg-green-500/20 text-green-600 dark:text-green-400 border-green-500/30";
      case "medium":
        return "bg-blue-500/20 text-blue-600 dark:text-blue-400 border-blue-500/30";
      case "low":
        return "bg-yellow-500/20 text-yellow-600 dark:text-yellow-400 border-yellow-500/30";
    }
  };

  return (
    <div
      className={cn(
        "rounded-2xl border-2 border-orange-500/50 bg-gradient-to-br from-orange-50 to-orange-100/50 dark:from-orange-950/30 dark:to-orange-900/20",
        "p-5 space-y-4",
        className,
      )}
    >
      {/* Header */}
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 w-10 h-10 rounded-full bg-orange-500/20 flex items-center justify-center">
          <AlertTriangle className="w-5 h-5 text-orange-500" />
        </div>
        <div className="flex-1">
          <h3 className="font-semibold text-orange-900 dark:text-orange-100">{t("schedule.conflict.title") || "时间冲突"}</h3>
          <p className="text-sm text-orange-700 dark:text-orange-300 mt-0.5">
            {conflicts.length === 1
              ? t("schedule.conflict.single-conflict") || "发现 1 个日程冲突"
              : t("schedule.conflict.multiple-conflicts", { count: conflicts.length }) || `发现 ${conflicts.length} 个日程冲突`}
          </p>
        </div>
      </div>

      {/* New Schedule Preview */}
      <div className="rounded-xl bg-white/60 dark:bg-zinc-900/60 border border-orange-200 dark:border-orange-800/50 p-4">
        <div className="flex items-center gap-2 mb-2">
          <Calendar className="w-4 h-4 text-orange-500" />
          <span className="text-sm font-medium text-orange-900 dark:text-orange-100">
            {t("schedule.conflict.new-schedule") || "新日程"}
          </span>
        </div>
        <div className="text-base font-semibold text-zinc-900 dark:text-zinc-100">{newSchedule.title}</div>
        <div className="flex items-center gap-3 mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          <div className="flex items-center gap-1">
            <Clock className="w-3.5 h-3.5" />
            <span>
              {formatDate(newSchedule.startTime)} {formatTime(newSchedule.startTime)} - {formatTime(newSchedule.endTime)}
            </span>
          </div>
        </div>
      </div>

      {/* Conflicts List */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-orange-800 dark:text-orange-200">
            {t("schedule.conflict.conflicting-schedules") || "冲突的日程"}
          </span>
        </div>

        {conflicts.map((conflict, index) => {
          const isExpanded = expandedConflicts.has(index);
          const isFullConflict = conflict.conflictType === "full";

          return (
            <div
              key={index}
              className="rounded-lg bg-white/40 dark:bg-zinc-900/40 border border-orange-200/50 dark:border-orange-800/30 overflow-hidden"
            >
              <button
                onClick={() => toggleConflict(index)}
                className="w-full flex items-center gap-3 px-4 py-3 hover:bg-white/60 dark:hover:bg-zinc-800/60 transition-colors"
              >
                <div className={cn("flex-shrink-0 w-2 h-2 rounded-full", isFullConflict ? "bg-red-500" : "bg-orange-500")} />
                <div className="flex-1 text-left">
                  <div className="font-medium text-sm text-zinc-900 dark:text-zinc-100">{conflict.schedule.title}</div>
                  <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                    {formatDate(new Date(Number(conflict.schedule.startTs) * 1000))}{" "}
                    {formatTime(new Date(Number(conflict.schedule.startTs) * 1000))} -{" "}
                    {formatTime(new Date(Number(conflict.schedule.endTs) * 1000))}
                  </div>
                </div>
                <span
                  className={cn(
                    "text-xs px-2 py-0.5 rounded-full",
                    isFullConflict
                      ? "bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400"
                      : "bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400",
                  )}
                >
                  {isFullConflict ? t("schedule.conflict.type-full") || "冲突" : t("schedule.conflict.type-partial") || "重叠"}
                </span>
                {isExpanded ? <ChevronUp className="w-4 h-4 text-zinc-400" /> : <ChevronDown className="w-4 h-4 text-zinc-400" />}
              </button>

              {isExpanded && (
                <div className="px-4 pb-3 text-xs text-zinc-600 dark:text-zinc-400">
                  {conflict.schedule.location && (
                    <div className="flex items-center gap-2 mt-2">
                      <span>{conflict.schedule.location}</span>
                    </div>
                  )}
                  {conflict.schedule.description && <div className="mt-2 line-clamp-2">{conflict.schedule.description}</div>}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* AI Suggestions */}
      {suggestions.length > 0 && (
        <div className="rounded-xl bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20 border border-blue-200 dark:border-blue-800/30 overflow-hidden">
          <button
            onClick={() => setExpandedSuggestions(!expandedSuggestions)}
            className="w-full flex items-center gap-3 px-4 py-3 hover:bg-blue-100/30 dark:hover:bg-blue-900/20 transition-colors"
          >
            <div className="flex-shrink-0 w-8 h-8 rounded-full bg-blue-500/20 flex items-center justify-center">
              <Zap className="w-4 h-4 text-blue-500" />
            </div>
            <div className="flex-1 text-left">
              <div className="font-medium text-sm text-blue-900 dark:text-blue-100">
                {t("schedule.conflict.ai-suggestions") || "AI 建议的解决方案"}
              </div>
              <div className="text-xs text-blue-700 dark:text-blue-300 mt-0.5">
                {suggestions.length} {t("schedule.conflict.suggestions-available") || "个可用建议"}
              </div>
            </div>
            {expandedSuggestions ? <ChevronUp className="w-4 h-4 text-blue-400" /> : <ChevronDown className="w-4 h-4 text-blue-400" />}
          </button>

          {expandedSuggestions && (
            <div className="px-4 pb-4 space-y-2">
              {suggestions.map((suggestion, index) => (
                <button
                  key={index}
                  onClick={() => onResolveWithSuggestion?.(suggestion)}
                  className="w-full flex items-center gap-3 p-3 rounded-lg bg-white/60 dark:bg-zinc-900/60 border border-blue-200/50 dark:border-blue-800/30 hover:bg-white dark:hover:bg-zinc-800 transition-colors text-left"
                >
                  <div
                    className={cn(
                      "flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center border",
                      getConfidenceColor(suggestion.confidence),
                    )}
                  >
                    <Clock className="w-4 h-4" />
                  </div>
                  <div className="flex-1">
                    <div className="font-medium text-sm text-zinc-900 dark:text-zinc-100">{suggestion.label}</div>
                    <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                      {formatDate(suggestion.startTime)} {formatTime(suggestion.startTime)} - {formatTime(suggestion.endTime)}
                    </div>
                  </div>
                  <Check className="w-5 h-5 text-blue-500" />
                </button>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex flex-col sm:flex-row gap-2 pt-2">
        {onResolveManually && (
          <Button variant="outline" onClick={onResolveManually} className="flex-1">
            {t("schedule.conflict.manual-resolve") || "手动调整"}
          </Button>
        )}
        {onOverride && (
          <Button variant="default" onClick={onOverride} className="flex-1 bg-orange-500 hover:bg-orange-600 text-white">
            {t("schedule.conflict.override") || "仍要创建"}
          </Button>
        )}
        {onCancel && (
          <Button
            variant="ghost"
            onClick={onCancel}
            className="text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
          >
            {t("common.cancel") || "取消"}
          </Button>
        )}
      </div>
    </div>
  );
}
