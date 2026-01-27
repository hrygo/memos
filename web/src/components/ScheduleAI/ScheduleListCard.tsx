import { Calendar, Clock, MapPin, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import dayjs from "dayjs";
import type { ScheduleListCardProps } from "./types";

export function ScheduleListCard({ data, onDismiss }: ScheduleListCardProps) {
  const t = useTranslate();

  // Group schedules by date
  const groupedSchedules = data.schedules.reduce((acc, schedule) => {
    const date = dayjs.unix(schedule.start_ts).format("YYYY-MM-DD");
    if (!acc[date]) {
      acc[date] = [];
    }
    acc[date].push(schedule);
    return acc;
  }, {} as Record<string, typeof data.schedules>);

  // Sort dates
  const sortedDates = Object.keys(groupedSchedules).sort();

  return (
    <div
      className={cn(
        "rounded-xl border p-4 transition-all duration-200",
        "animate-in fade-in slide-in-from-top-2",
        "bg-muted/50 border-border",
      )}
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
            <Calendar className="w-4 h-4 text-primary" />
          </div>
          <div>
            <h4 className="font-semibold text-foreground">{data.title}</h4>
            {data.time_range && (
              <p className="text-xs text-muted-foreground">{data.time_range}</p>
            )}
          </div>
        </div>
        {onDismiss && (
          <button
            type="button"
            onClick={onDismiss}
            className="text-muted-foreground hover:text-foreground transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Summary */}
      <div className="mb-3 text-sm text-muted-foreground">
        {t("schedule.list.summary", { count: data.count }) || `共找到 ${data.count} 个日程`}
      </div>

      {/* Schedule list grouped by date */}
      <div className="space-y-3">
        {sortedDates.map((date) => {
          const schedules = groupedSchedules[date];
          const formattedDate = dayjs(date).format("MM月DD日");
          const isToday = dayjs(date).isSame(dayjs(), "day");
          const isTomorrow = dayjs(date).isSame(dayjs().add(1, "day"), "day");

          let dateLabel = formattedDate;
          if (isToday) dateLabel = `今天 (${formattedDate})`;
          else if (isTomorrow) dateLabel = `明天 (${formattedDate})`;

          return (
            <div key={date}>
              <div className="text-xs font-medium text-muted-foreground mb-1">
                {dateLabel}
              </div>
              <div className="space-y-2">
                {schedules.map((schedule) => (
                  <div
                    key={schedule.uid}
                    className={cn(
                      "p-3 rounded-lg border transition-colors",
                      "bg-background border-border hover:border-primary/50",
                    )}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex-1 min-w-0">
                        <h5 className="font-medium text-foreground truncate">
                          {schedule.title}
                        </h5>
                        <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                          <div className="flex items-center gap-1">
                            <Clock className="w-3 h-3" />
                            <span>
                              {dayjs.unix(schedule.start_ts).format("HH:mm")}
                              {" - "}
                              {dayjs.unix(schedule.end_ts).format("HH:mm")}
                            </span>
                          </div>
                          {schedule.location && (
                            <div className="flex items-center gap-1">
                              <MapPin className="w-3 h-3" />
                              <span className="truncate">{schedule.location}</span>
                            </div>
                          )}
                        </div>
                      </div>
                      {schedule.status === "CANCELLED" && (
                        <span className="text-xs text-muted-foreground line-through">
                          已取消
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>

      {/* Empty state */}
      {data.count === 0 && (
        <div className="text-center py-6 text-muted-foreground">
          <Calendar className="w-8 h-8 mx-auto mb-2 opacity-50" />
          <p className="text-sm">
            {t("schedule.list.empty") || "该时间段没有找到日程"}
          </p>
        </div>
      )}
    </div>
  );
}
