import { timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { AlertCircle, AlertTriangle, Calendar, Clock, Globe, MapPin } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

interface ScheduleSuggestionCardProps {
  parsedSchedule: Schedule;
  conflicts?: Schedule[];
  onConfirm: () => void;
  onDismiss: () => void;
  onEdit: () => void;
  onAdjustTime?: () => void;
}

export const ScheduleSuggestionCard = ({
  parsedSchedule,
  conflicts = [],
  onConfirm,
  onDismiss,
  onEdit,
  onAdjustTime,
}: ScheduleSuggestionCardProps) => {
  const t = useTranslate();

  const hasConflicts = conflicts.length > 0;

  const formatDateTime = (ts: bigint) => {
    const date = timestampDate({ seconds: ts, nanos: 0 });
    const targetDate = dayjs(date);
    const now = dayjs();

    // Smart year display: show year only if different from current year
    if (targetDate.year() !== now.year()) {
      return targetDate.format("YYYY-MM-DD HH:mm");
    }
    return targetDate.format("MM-DD HH:mm");
  };

  return (
    <div className={cn("my-4 rounded-lg border p-4", hasConflicts ? "border-destructive/50 bg-destructive/5" : "border-primary/20 bg-primary/5")}>
      <div className="mb-3 flex items-start gap-2">
        {hasConflicts ? (
          <AlertTriangle className="h-5 w-5 mt-0.5 text-destructive" />
        ) : (
          <Calendar className="h-5 w-5 mt-0.5 text-primary" />
        )}
        <div className="flex-1">
          <h4 className="font-medium text-sm">
            {hasConflicts ? (t("schedule.conflict-detected") || "检测到时间冲突" ) : t("schedule.suggested-schedule")}
          </h4>
          <p className="text-xs text-muted-foreground">
            {hasConflicts
              ? t("schedule.conflict-suggestion-hint") || "该时间段与其他日程冲突，建议调整"
              : t("schedule.suggested-schedule-hint")}
          </p>
        </div>
      </div>

      <div className="space-y-2 text-sm">
        <div className="flex items-start gap-2">
          <div className="font-medium">{parsedSchedule.title || t("schedule.untitled") || "无标题日程"}</div>
        </div>

        {parsedSchedule.allDay ? (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Calendar className="h-3.5 w-3.5" />
            <span>{t("schedule.all-day") || "全天"}</span>
          </div>
        ) : (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <span>
              {formatDateTime(parsedSchedule.startTs)}
              {parsedSchedule.endTs > 0 && ` - ${formatDateTime(parsedSchedule.endTs)}`}
            </span>
          </div>
        )}

        {parsedSchedule.location && (
          <div className="flex items-center gap-2 text-muted-foreground">
            <MapPin className="h-3.5 w-3.5" />
            <span>{parsedSchedule.location}</span>
          </div>
        )}

        {parsedSchedule.timezone && parsedSchedule.timezone !== "UTC" && (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Globe className="h-3.5 w-3.5" />
            <span className="text-xs">{parsedSchedule.timezone}</span>
          </div>
        )}

        {/* Conflict List */}
        {hasConflicts && (
          <div className="mt-3 space-y-2">
            <div className="flex items-center gap-2 text-xs font-medium text-destructive">
              <AlertCircle className="h-3.5 w-3.5" />
              <span>{t("schedule.conflicting-schedules") || "冲突的日程"} ({conflicts.length})</span>
            </div>
            <div className="space-y-1.5 pl-5">
              {conflicts.slice(0, 3).map((conflict) => (
                <div key={conflict.name} className="text-xs text-muted-foreground border-l-2 border-destructive/30 pl-2">
                  <div className="font-medium text-foreground">{conflict.title}</div>
                  <div className="flex items-center gap-1 text-[10px] mt-0.5">
                    <Clock className="h-2.5 w-2.5" />
                    <span>
                      {formatDateTime(conflict.startTs)}
                      {conflict.endTs > 0 && ` - ${formatDateTime(conflict.endTs)}`}
                    </span>
                  </div>
                </div>
              ))}
              {conflicts.length > 3 && (
                <div className="text-xs text-muted-foreground pl-2">
                  +{conflicts.length - 3} {t("schedule.more-conflicts") || "更多冲突"}
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      <div className="mt-3 flex justify-end gap-2 flex-wrap">
        <Button variant="ghost" size="sm" onClick={onDismiss}>
          {t("common.dismiss")}
        </Button>
        {hasConflicts && onAdjustTime && (
          <Button variant="outline" size="sm" onClick={onAdjustTime}>
            {t("schedule.adjust-time") || "调整时间"}
          </Button>
        )}
        <Button variant="outline" size="sm" onClick={onEdit}>
          {t("common.edit")}
        </Button>
        <Button
          size="sm"
          onClick={onConfirm}
          className={hasConflicts ? "bg-destructive hover:bg-destructive/90" : ""}
        >
          {hasConflicts ? (t("schedule.create-anyway") || "仍要创建") : t("schedule.create")}
        </Button>
      </div>
    </div>
  );
};
