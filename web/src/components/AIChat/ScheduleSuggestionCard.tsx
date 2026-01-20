import { timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { Calendar, Clock, Globe, MapPin } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

interface ScheduleSuggestionCardProps {
  parsedSchedule: Schedule;
  onConfirm: () => void;
  onDismiss: () => void;
  onEdit: () => void;
}

export const ScheduleSuggestionCard = ({ parsedSchedule, onConfirm, onDismiss, onEdit }: ScheduleSuggestionCardProps) => {
  const t = useTranslate();

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
    <div className="my-4 rounded-lg border border-primary/20 bg-primary/5 p-4">
      <div className="mb-3 flex items-start gap-2">
        <Calendar className="h-5 w-5 mt-0.5 text-primary" />
        <div className="flex-1">
          <h4 className="font-medium text-sm">{t("schedule.suggested-schedule")}</h4>
          <p className="text-xs text-muted-foreground">{t("schedule.suggested-schedule-hint")}</p>
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
      </div>

      <div className="mt-3 flex justify-end gap-2">
        <Button variant="ghost" size="sm" onClick={onDismiss}>
          {t("common.dismiss")}
        </Button>
        <Button variant="outline" size="sm" onClick={onEdit}>
          {t("common.edit")}
        </Button>
        <Button size="sm" onClick={onConfirm}>
          {t("schedule.create")}
        </Button>
      </div>
    </div>
  );
};
