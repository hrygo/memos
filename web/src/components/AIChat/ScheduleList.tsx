import dayjs from "dayjs";
import { AlertTriangle, Clock, MapPin } from "lucide-react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useDeleteSchedule } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

// Helper function to convert int64 timestamp to Date
const toDate = (seconds: bigint): Date => {
  return new Date(Number(seconds) * 1000);
};

interface ScheduleListProps {
  schedules: Schedule[];
  selectedDate?: string;
  onScheduleClick?: (schedule: Schedule) => void;
  className?: string;
}

export const ScheduleList = ({ schedules, selectedDate, onScheduleClick, className = "" }: ScheduleListProps) => {
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

    const startDate = toDate(schedule.startTs);
    const startTime = dayjs(startDate).format("HH:mm");

    if (schedule.endTs > 0) {
      const endDate = toDate(schedule.endTs);
      const endTime = dayjs(endDate).format("HH:mm");
      return `${startTime} - ${endTime}`;
    }

    return startTime;
  };

  const getTodaySchedules = () => {
    const today = dayjs().format("YYYY-MM-DD");
    return schedules.filter((s) => {
      const date = dayjs(toDate(s.startTs)).format("YYYY-MM-DD");
      return date === (selectedDate || today);
    });
  };

  const displaySchedules = selectedDate ? getTodaySchedules() : schedules;

  // Check if a schedule conflicts with any other schedule
  const hasConflict = (schedule: Schedule): boolean => {
    const scheduleEnd = schedule.endTs > 0 ? schedule.endTs : schedule.startTs + BigInt(3600);

    return displaySchedules.some((other) => {
      if (other.name === schedule.name) return false;

      const otherEnd = other.endTs > 0 ? other.endTs : other.startTs + BigInt(3600);

      // Check for time overlap
      return schedule.startTs < otherEnd && other.startTs < scheduleEnd;
    });
  };

  // Get conflicting schedule names
  const getConflictingSchedules = (schedule: Schedule): Schedule[] => {
    const scheduleEnd = schedule.endTs > 0 ? schedule.endTs : schedule.startTs + BigInt(3600);

    return displaySchedules.filter((other) => {
      if (other.name === schedule.name) return false;

      const otherEnd = other.endTs > 0 ? other.endTs : other.startTs + BigInt(3600);

      return schedule.startTs < otherEnd && other.startTs < scheduleEnd;
    });
  };

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
        {displaySchedules.map((schedule) => {
          const conflict = hasConflict(schedule);
          const conflictingSchedules = getConflictingSchedules(schedule);

          return (
            <div
              key={schedule.name}
              onClick={() => onScheduleClick?.(schedule)}
              className={cn(
                "group flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors hover:bg-accent",
                conflict
                  ? "border-red-500/50 bg-red-50/50 dark:bg-red-950/20 dark:border-red-500/70"
                  : "border-border bg-card"
              )}
            >
              <div className="flex flex-col items-center">
                <div className={cn(
                  "rounded-md px-2 py-1 text-center",
                  conflict ? "bg-red-100 dark:bg-red-900/50" : "bg-primary/10"
                )}>
                  <div className={cn(
                    "text-xs font-medium",
                    conflict ? "text-red-700 dark:text-red-300" : "text-primary"
                  )}>
                    {dayjs(toDate(schedule.startTs)).format("ddd")}
                  </div>
                  <div className={cn(
                    "text-lg font-bold",
                    conflict ? "text-red-700 dark:text-red-300" : "text-primary"
                  )}>
                    {dayjs(toDate(schedule.startTs)).format("D")}
                  </div>
                </div>
              </div>

              <div className="flex-1 space-y-1">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1">
                    <h4 className={cn(
                      "font-medium text-sm leading-tight",
                      conflict && "text-red-900 dark:text-red-100"
                    )}>
                      {schedule.title}
                    </h4>
                  </div>
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

                {schedule.description && <p className="line-clamp-2 text-xs text-muted-foreground">{schedule.description}</p>}

                {conflict && conflictingSchedules.length > 0 && (
                  <div className="mt-1 pt-1 border-t border-red-200 dark:border-red-800">
                    <p className="text-[10px] text-red-600 dark:text-red-400">
                      {t("schedule.conflict-warning") || "Conflicts with"}: {conflictingSchedules.map(s => s.title).join(", ")}
                    </p>
                  </div>
                )}

                {schedule.reminders.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {schedule.reminders.map((reminder, idx) => (
                      <span key={idx} className="rounded-full bg-secondary px-2 py-0.5 text-xs text-secondary-foreground">
                        {reminder.type === "before" && t("schedule.reminders")}: {reminder.value} {reminder.unit}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </ScrollArea>
  );
};
