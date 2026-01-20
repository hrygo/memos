import dayjs from "dayjs";
import { Calendar, Clock, MapPin, X, Repeat } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { ScheduleSummary } from "@/types/schedule";

interface ScheduleQueryResultProps {
  title: string;
  schedules: ScheduleSummary[];
  onClose: () => void;
  onScheduleClick?: (schedule: ScheduleSummary) => void;
}

export const ScheduleQueryResult = ({ title, schedules, onClose, onScheduleClick }: ScheduleQueryResultProps) => {
  const formatTime = (schedule: ScheduleSummary) => {
    const startDate = dayjs.unix(schedule.startTs);
    const startTime = startDate.format("HH:mm");

    if (schedule.endTs > 0) {
      const endDate = dayjs.unix(schedule.endTs);
      return `${startTime} - ${endDate.format("HH:mm")}`;
    }

    return startTime;
  };

  const formatDate = (schedule: ScheduleSummary) => {
    const date = dayjs.unix(schedule.startTs);
    const now = dayjs().startOf("day");

    // Today
    if (date.isSame(now, "day")) {
      return "今天";
    }
    // Tomorrow
    if (date.isSame(now.add(1, "day"), "day")) {
      return "明天";
    }
    // This week
    if (date.isBefore(now.add(7, "day"))) {
      return date.format("ddd"); // 周一、周二...
    }
    // Date
    return date.format("MM-DD");
  };

  if (schedules.length === 0) {
    return (
      <div className="mx-4 mb-4 rounded-lg border border-dashed border-orange-300 dark:border-orange-800 bg-orange-50/50 dark:bg-orange-950/20 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Calendar className="h-4 w-4 text-orange-600 dark:text-orange-400" />
            <div>
              <h3 className="font-semibold text-sm text-foreground">{title}</h3>
              <p className="text-xs text-muted-foreground mt-0.5">该时间段暂无日程安排</p>
            </div>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose} className="h-8 w-8 p-0">
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-4 mb-4 rounded-lg border border-orange-200 dark:border-orange-800 bg-gradient-to-br from-orange-50/50 to-amber-50/50 dark:from-orange-950/20 dark:to-amber-950/20 p-4 shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className="rounded-full bg-orange-100 dark:bg-orange-900/50 p-1.5">
            <Calendar className="h-4 w-4 text-orange-600 dark:text-orange-400" />
          </div>
          <div>
            <h3 className="font-semibold text-sm text-foreground">{title}</h3>
            <p className="text-xs text-muted-foreground mt-0.5">找到 {schedules.length} 个日程</p>
          </div>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose} className="h-8 w-8 p-0 hover:bg-orange-100 dark:hover:bg-orange-900/30">
          <X className="h-4 w-4 text-muted-foreground" />
        </Button>
      </div>

      {/* Schedule List */}
      <div className="space-y-2">
        {schedules.map((schedule) => (
          <div
            key={schedule.uid}
            onClick={() => onScheduleClick?.(schedule)}
            className={cn(
              "group flex items-start gap-3 rounded-lg border border-orange-200/50 dark:border-orange-800/50 bg-white/50 dark:bg-black/20 p-3 transition-all cursor-pointer hover:bg-orange-100/50 dark:hover:bg-orange-900/30",
              onScheduleClick && "hover:border-orange-300 dark:hover:border-orange-700"
            )}
          >
            {/* Date Badge */}
            <div className="flex-none">
              <div className="rounded-md bg-orange-100 dark:bg-orange-900/50 px-2.5 py-1.5 text-center min-w-[3.5rem]">
                <div className="text-[10px] font-medium text-orange-700 dark:text-orange-300">
                  {formatDate(schedule)}
                </div>
              </div>
            </div>

            {/* Content */}
            <div className="flex-1 min-w-0">
              <h4 className="font-medium text-sm text-foreground mb-1 group-hover:text-orange-700 dark:group-hover:text-orange-300 transition-colors">
                {schedule.title || "无标题日程"}
              </h4>

              <div className="flex items-center gap-3 text-xs text-muted-foreground">
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
                {schedule.recurrenceRule && (
                  <span className="flex items-center gap-1">
                    <Repeat className="h-3 w-3" />
                    重复
                  </span>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Footer Actions */}
      <div className="mt-3 pt-3 border-t border-orange-200/50 dark:border-orange-800/50 flex justify-end">
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            // Open schedule panel
            onClose();
          }}
          className="text-xs"
        >
          查看完整日程表
        </Button>
      </div>
    </div>
  );
};
