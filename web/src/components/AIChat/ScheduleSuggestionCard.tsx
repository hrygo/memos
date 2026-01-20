import { Calendar, Clock, MapPin } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTranslate } from "@/utils/i18n";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import dayjs from "dayjs";
import { timestampDate } from "@bufbuild/protobuf/wkt";

interface ScheduleSuggestionCardProps {
  parsedSchedule: Schedule;
  onConfirm: () => void;
  onDismiss: () => void;
  onEdit: () => void;
}

export const ScheduleSuggestionCard = ({
  parsedSchedule,
  onConfirm,
  onDismiss,
  onEdit,
}: ScheduleSuggestionCardProps) => {
  const t = useTranslate();

  const formatDateTime = (ts: bigint) => {
    const date = timestampDate({ seconds: ts, nanos: 0 });
    return dayjs(date).format("MM-DD HH:mm");
  };

  return (
    <div className="my-4 rounded-lg border border-primary/20 bg-primary/5 p-4">
      <div className="mb-3 flex items-start gap-2">
        <Calendar className="h-5 w-5 mt-0.5 text-primary" />
        <div className="flex-1">
          <h4 className="font-medium text-sm">{t("schedule.suggested-schedule") || "建议创建日程"}</h4>
          <p className="text-xs text-muted-foreground">
            {t("schedule.suggested-schedule-hint") || "根据您的对话内容，我建议创建以下日程"}
          </p>
        </div>
      </div>

      <div className="space-y-2 text-sm">
        <div className="flex items-start gap-2">
          <div className="font-medium">{parsedSchedule.title}</div>
        </div>

        <div className="flex items-center gap-2 text-muted-foreground">
          <Clock className="h-3.5 w-3.5" />
          <span>
            {formatDateTime(parsedSchedule.startTs)}
            {parsedSchedule.endTs > 0 && ` - ${formatDateTime(parsedSchedule.endTs)}`}
          </span>
        </div>

        {parsedSchedule.location && (
          <div className="flex items-center gap-2 text-muted-foreground">
            <MapPin className="h-3.5 w-3.5" />
            <span>{parsedSchedule.location}</span>
          </div>
        )}
      </div>

      <div className="mt-3 flex justify-end gap-2">
        <Button variant="ghost" size="sm" onClick={onDismiss}>
          {t("common.dismiss") || "忽略"}
        </Button>
        <Button variant="outline" size="sm" onClick={onEdit}>
          {t("common.edit") || "编辑"}
        </Button>
        <Button size="sm" onClick={onConfirm}>
          {t("schedule.create") || "创建"}
        </Button>
      </div>
    </div>
  );
};
