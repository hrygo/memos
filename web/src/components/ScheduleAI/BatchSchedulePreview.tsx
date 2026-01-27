import dayjs from "dayjs";
import { Calendar, CalendarDays, Check, Clock, Loader2, MapPin, Repeat } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { BatchScheduleInfo, Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { type Translations, useTranslate } from "@/utils/i18n";

export interface BatchSchedulePreviewProps {
  info: BatchScheduleInfo;
  preview: Schedule[];
  totalCount: number;
  confidence: number;
  onConfirm: (info: BatchScheduleInfo) => void;
  onCancel?: () => void;
  isLoading?: boolean;
}

export function BatchSchedulePreview({
  info,
  preview,
  totalCount,
  confidence,
  onConfirm,
  onCancel,
  isLoading = false,
}: BatchSchedulePreviewProps) {
  const t = useTranslate();
  const [isCreating, setIsCreating] = useState(false);
  const [expandPreview, setExpandPreview] = useState(false);

  const showCreating = isLoading || isCreating;

  const handleConfirm = () => {
    if (showCreating) return;
    setIsCreating(true);
    onConfirm(info);
  };

  // Parse recurrence rule for display
  const getRecurrenceText = () => {
    if (!info.recurrenceRule) return "";
    try {
      const rule = JSON.parse(info.recurrenceRule);
      
      if (rule.type === "daily") {
        return t("schedule.batch.recurrence.daily" as Translations) || "\u6bcf\u5929";
      } else if (rule.type === "weekly" && rule.weekdays) {
        const weekdayKeys = ["", "mon", "tue", "wed", "thu", "fri", "sat", "sun"];
        const days = rule.weekdays.map((d: number) => 
          t(`schedule.batch.weekday.${weekdayKeys[d]}` as Translations) || weekdayKeys[d]
        ).join(", ");
        return `${t("schedule.batch.recurrence.weekly" as Translations) || "\u6bcf\u5468"} ${days}`;
      } else if (rule.type === "monthly" && rule.month_day) {
        return `${t("schedule.batch.recurrence.monthly" as Translations) || "\u6bcf\u6708"} ${rule.month_day}${t("schedule.batch.day-suffix" as Translations) || "\u53f7"}`;
      }
      return "";
    } catch {
      return "";
    }
  };

  const startTime = info.startTs ? dayjs.unix(Number(info.startTs)).format("HH:mm") : "";
  const recurrenceText = getRecurrenceText();
  const previewToShow = expandPreview ? preview : preview.slice(0, 3);

  return (
    <div
      className={cn(
        "rounded-xl border p-4 transition-all duration-200",
        "animate-in fade-in slide-in-from-top-2",
        showCreating
          ? "bg-green-500/10 border-green-500/30"
          : "bg-primary/10 border-primary/20",
      )}
    >
      {/* Header */}
      <div className="flex items-start gap-3">
        <div
          className={cn(
            "flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center transition-colors duration-200",
            showCreating ? "bg-green-500/20" : "bg-primary/20",
          )}
        >
          {showCreating ? (
            <Check className="w-5 h-5 text-green-600 dark:text-green-400 animate-in zoom-in duration-200" />
          ) : (
            <CalendarDays className="w-5 h-5 text-primary" />
          )}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <h4 className="font-semibold text-foreground">{info.title}</h4>
            <span className="text-xs text-muted-foreground">
              {t("schedule.batch.total-count" as Translations) || "\u5171"} {totalCount} {t("schedule.batch.items" as Translations) || "\u4e2a"}
            </span>
          </div>

          <div className="flex flex-wrap items-center gap-3 mt-2 text-sm text-muted-foreground">
            <div className="flex items-center gap-1.5">
              <Repeat className="w-4 h-4" />
              <span>{recurrenceText}</span>
            </div>
            {startTime && (
              <div className="flex items-center gap-1.5">
                <Clock className="w-4 h-4" />
                <span>{startTime}</span>
              </div>
            )}
            {info.location && (
              <div className="flex items-center gap-1.5">
                <MapPin className="w-4 h-4" />
                <span>{info.location}</span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Preview List */}
      {preview.length > 0 && (
        <div className="mt-4 space-y-2">
          <div className="text-xs font-medium text-muted-foreground mb-2">
            {t("schedule.batch.preview" as Translations) || "\u9884\u89c8"}:
          </div>
          <div className="space-y-1.5 max-h-[200px] overflow-y-auto">
            {previewToShow.map((schedule, index) => (
              <div
                key={schedule.name || index}
                className="flex items-center gap-2 text-sm py-1.5 px-2 rounded-lg bg-background/50"
              >
                <Calendar className="w-3.5 h-3.5 text-muted-foreground flex-shrink-0" />
                <span className="text-muted-foreground">
                  {dayjs.unix(Number(schedule.startTs)).format("YYYY-MM-DD")}
                </span>
                <span className="text-foreground">
                  {dayjs.unix(Number(schedule.startTs)).format("ddd")}
                </span>
                <span className="text-muted-foreground">
                  {dayjs.unix(Number(schedule.startTs)).format("HH:mm")}
                  {schedule.endTs ? ` - ${dayjs.unix(Number(schedule.endTs)).format("HH:mm")}` : ""}
                </span>
              </div>
            ))}
          </div>
          {preview.length > 3 && (
            <button
              type="button"
              onClick={() => setExpandPreview(!expandPreview)}
              className="text-xs text-primary hover:underline"
            >
              {expandPreview
                ? (t("schedule.batch.show-less" as Translations) || "\u6536\u8d77")
                : `${t("schedule.batch.show-more" as Translations) || "\u663e\u793a\u66f4\u591a"} (${preview.length - 3})`}
            </button>
          )}
        </div>
      )}

      {/* Confidence Indicator */}
      {confidence > 0 && confidence < 1 && (
        <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
          <div className="flex-1 h-1 bg-muted rounded-full overflow-hidden">
            <div
              className={cn(
                "h-full transition-all duration-300",
                confidence >= 0.8 ? "bg-green-500" : confidence >= 0.6 ? "bg-yellow-500" : "bg-orange-500"
              )}
              style={{ width: `${confidence * 100}%` }}
            />
          </div>
          <span>{Math.round(confidence * 100)}%</span>
        </div>
      )}

      {/* Actions */}
      <div className="mt-4 flex gap-2 justify-end">
        {onCancel && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onCancel}
            disabled={showCreating}
          >
            {t("common.cancel" as Translations) || "\u53d6\u6d88"}
          </Button>
        )}
        <Button
          size="sm"
          onClick={handleConfirm}
          disabled={showCreating}
          className={cn(
            showCreating && "bg-green-600 hover:bg-green-600"
          )}
        >
          {showCreating ? (
            <>
              <Loader2 className="w-4 h-4 mr-1 animate-spin" />
              {t("schedule.batch.creating" as Translations) || "\u521b\u5efa\u4e2d..."}
            </>
          ) : (
            <>
              <Check className="w-4 h-4 mr-1" />
              {t("schedule.batch.confirm-create" as Translations) || "\u786e\u8ba4\u521b\u5efa"} ({totalCount})
            </>
          )}
        </Button>
      </div>
    </div>
  );
}
