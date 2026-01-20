import { Clock, MapPin } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useTranslate } from "@/utils/i18n";
import { useDeleteSchedule } from "@/hooks/useScheduleQueries";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import dayjs from "dayjs";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { toast } from "sonner";

interface ScheduleListProps {
  schedules: Schedule[];
  selectedDate?: string;
  onScheduleClick?: (schedule: Schedule) => void;
  className?: string;
}

export const ScheduleList = ({
  schedules,
  selectedDate,
  onScheduleClick,
  className = "",
}: ScheduleListProps) => {
  const t = useTranslate();
  const deleteSchedule = useDeleteSchedule();

  const handleDelete = async (e: React.MouseEvent, name: string) => {
    e.stopPropagation();
    if (confirm(t("schedule.delete-schedule") + "?")) {
      await deleteSchedule.mutateAsync(name);
      toast.success(t("schedule.schedule-deleted"));
    }
  };

  const formatTime = (schedule: Schedule) => {
    if (schedule.allDay) {
      return t("schedule.all-day");
    }

    const startDate = timestampDate({ seconds: schedule.startTs, nanos: 0 });
    const startTime = dayjs(startDate).format("HH:mm");

    if (schedule.endTs > 0) {
      const endDate = timestampDate({ seconds: schedule.endTs, nanos: 0 });
      const endTime = dayjs(endDate).format("HH:mm");
      return `${startTime} - ${endTime}`;
    }

    return startTime;
  };

  const getTodaySchedules = () => {
    const today = dayjs().format("YYYY-MM-DD");
    return schedules.filter((s) => {
      const date = dayjs(timestampDate({ seconds: s.startTs, nanos: 0 })).format("YYYY-MM-DD");
      return date === (selectedDate || today);
    });
  };

  const displaySchedules = selectedDate ? getTodaySchedules() : schedules;

  if (displaySchedules.length === 0) {
    return (
      <div className={`flex flex-col items-center justify-center py-12 text-center ${className}`}>
        <div className="rounded-full bg-muted p-4">
          <Clock className="h-6 w-6 text-muted-foreground" />
        </div>
        <p className="mt-3 text-sm text-muted-foreground">{t("schedule.no-schedules")}</p>
      </div>
    );
  }

  return (
    <ScrollArea className={`h-full ${className}`}>
      <div className="space-y-2 p-1">
        {displaySchedules.map((schedule) => (
          <div
            key={schedule.name}
            onClick={() => onScheduleClick?.(schedule)}
            className="group flex cursor-pointer items-start gap-3 rounded-lg border border-border bg-card p-3 transition-colors hover:bg-accent"
          >
            <div className="flex flex-col items-center">
              <div className="rounded-md bg-primary/10 px-2 py-1 text-center">
                <div className="text-xs font-medium text-primary">
                  {dayjs(timestampDate({ seconds: schedule.startTs, nanos: 0 })).format("ddd")}
                </div>
                <div className="text-lg font-bold text-primary">
                  {dayjs(timestampDate({ seconds: schedule.startTs, nanos: 0 })).format("D")}
                </div>
              </div>
            </div>

            <div className="flex-1 space-y-1">
              <div className="flex items-start justify-between gap-2">
                <h4 className="font-medium text-sm leading-tight">{schedule.title}</h4>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100"
                  onClick={(e) => handleDelete(e, schedule.name)}
                >
                  <span className="text-xs">✕</span>
                </Button>
              </div>

              <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <span className="flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  {formatTime(schedule)}
                </span>
                {schedule.location && (
                  <span className="flex items-center gap-1">
                    <MapPin className="h-3 w-3" />
                    {schedule.location}
                  </span>
                )}
              </div>

              {schedule.description && (
                <p className="line-clamp-2 text-xs text-muted-foreground">{schedule.description}</p>
              )}

              {schedule.reminders.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {schedule.reminders.map((reminder, idx) => (
                    <span
                      key={idx}
                      className="rounded-full bg-secondary px-2 py-0.5 text-xs text-secondary-foreground"
                    >
                      {reminder.type === "before" && t("schedule.reminders")}: {reminder.value} {reminder.unit}
                    </span>
                  ))}
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </ScrollArea>
  );
};
