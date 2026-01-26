import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs, { Dayjs } from "dayjs";
import { ChevronLeft, ChevronRight, Clock, Coffee, GripVertical, MapPin } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

interface ScheduleTimelineProps {
  schedules: Schedule[];
  selectedDate?: string;
  onDateClick?: (date: string) => void;
  onScheduleEdit?: (schedule: Schedule) => void;
  onScheduleUpdate?: (schedule: Schedule, newStartTs: bigint, newEndTs: bigint) => void;
  className?: string;
}

// Date Strip Component - Horizontal week navigator
interface DateStripProps {
  currentDate: Dayjs;
  selectedDate: string;
  schedules: Schedule[];
  onDateSelect: (date: Dayjs) => void;
  onPrevWeek: () => void;
  onNextWeek: () => void;
}

const DateStrip = ({ currentDate, selectedDate, schedules, onDateSelect, onPrevWeek, onNextWeek }: DateStripProps) => {
  const t = useTranslate();

  const todayStr = dayjs().format("YYYY-MM-DD");
  // Show 3 days before and 3 days after current date
  const startDate = currentDate.subtract(3, "day");

  const toDayjs = (ts: bigint) => dayjs(timestampDate(create(TimestampSchema, { seconds: ts, nanos: 0 })));

  const getScheduleCount = (date: Dayjs) => {
    const dateStr = date.format("YYYY-MM-DD");
    return schedules.filter((s) => toDayjs(s.startTs).format("YYYY-MM-DD") === dateStr).length;
  };

  return (
    <div className="flex items-center gap-2 sm:gap-3">
      <Button
        variant="ghost"
        size="icon"
        className="h-9 w-9 sm:h-8 sm:w-8 shrink-0 rounded-full min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
        onClick={onPrevWeek}
        aria-label={t("schedule.previous-week") as string}
      >
        <ChevronLeft className="h-4 w-4" aria-hidden="true" />
      </Button>

      <div className="flex items-center gap-1 sm:gap-1.5 flex-1" role="tablist" aria-label={t("schedule.select-date") as string}>
        {Array.from({ length: 7 }, (_, i) => {
          const date = startDate.add(i, "day");
          const dateStr = date.format("YYYY-MM-DD");
          const isToday = dateStr === todayStr;
          const isSelected = dateStr === selectedDate;
          const scheduleCount = getScheduleCount(date);
          const dayName = date.format("dd");
          const dayNum = date.format("D");
          const fullDate = date.format("YYYY-MM-DD");

          return (
            <button
              key={i}
              role="tab"
              aria-selected={isSelected}
              aria-label={t("schedule.date-aria-label", {
                date: fullDate,
                isToday: isToday ? t("schedule.today") : "",
                scheduleCount: scheduleCount > 0 ? `${scheduleCount} ${t("schedule.schedules")}` : "",
              }).trim()}
              onClick={() => onDateSelect(date)}
              className={cn(
                "flex-1 min-w-[2.5rem] sm:min-w-[3rem] max-w-[4rem] sm:max-w-[4.5rem] aspect-square flex flex-col items-center justify-center gap-0.5 rounded-xl transition-all duration-200 relative group",
                "hover:scale-105 active:scale-95",
                "focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2",
                isSelected && "bg-primary text-primary-foreground shadow-md",
                !isSelected && isToday && "bg-primary/10 text-primary border border-primary/20",
                !isSelected && !isToday && "hover:bg-muted/50",
                !isSelected && isToday && "ring-2 ring-primary/30 ring-offset-1",
              )}
            >
              <span
                className={cn(
                  "text-[9px] sm:text-[10px] font-medium uppercase tracking-wide",
                  isSelected ? "text-primary-foreground/70" : "text-muted-foreground",
                )}
                aria-hidden="true"
              >
                {dayName}
              </span>
              <span
                className={cn("text-base sm:text-lg font-semibold leading-none", isToday && !isSelected && "text-primary")}
                aria-hidden="true"
              >
                {dayNum}
              </span>
              {scheduleCount > 0 && (
                <div className="flex gap-0.5 mt-0.5" aria-hidden="true">
                  {Array.from({ length: Math.min(scheduleCount, 3) }).map((_, j) => (
                    <span key={j} className={cn("w-1 h-1 rounded-full", isSelected ? "bg-primary-foreground/80" : "bg-primary")} />
                  ))}
                </div>
              )}
            </button>
          );
        })}
      </div>

      <Button
        variant="ghost"
        size="icon"
        className="h-9 w-9 sm:h-8 sm:w-8 shrink-0 rounded-full min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
        onClick={onNextWeek}
        aria-label={t("schedule.next-week") as string}
      >
        <ChevronRight className="h-4 w-4" aria-hidden="true" />
      </Button>
    </div>
  );
};

export const ScheduleTimeline = ({
  schedules,
  selectedDate,
  onDateClick,
  onScheduleEdit,
  onScheduleUpdate,
  className = "",
}: ScheduleTimelineProps) => {
  const t = useTranslate();
  const [currentDate, setCurrentDate] = useState(selectedDate ? dayjs(selectedDate) : dayjs());
  const [draggedSchedule, setDraggedSchedule] = useState<Schedule | null>(null);
  const [dragOverTime, setDragOverTime] = useState<string | null>(null);
  const timelineRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (selectedDate) {
      setCurrentDate(dayjs(selectedDate));
    }
  }, [selectedDate]);

  const handleDateSelect = (date: Dayjs) => {
    setCurrentDate(date);
    onDateClick?.(date.format("YYYY-MM-DD"));
  };

  const handlePrevWeek = () => handleDateSelect(currentDate.subtract(1, "week"));
  const handleNextWeek = () => handleDateSelect(currentDate.add(1, "week"));

  const selectedDateStr = currentDate.format("YYYY-MM-DD");

  const toDayjs = (ts: bigint) => dayjs(timestampDate(create(TimestampSchema, { seconds: ts, nanos: 0 })));

  const daySchedules = schedules
    .filter((s) => toDayjs(s.startTs).format("YYYY-MM-DD") === selectedDateStr)
    .sort((a, b) => Number(a.startTs) - Number(b.startTs));

  const hasConflict = (schedule: Schedule): boolean => {
    const scheduleEnd = schedule.endTs > 0 ? schedule.endTs : schedule.startTs + BigInt(3600);
    return daySchedules.some((other) => {
      if (other.name === schedule.name) return false;
      const otherEnd = other.endTs > 0 ? other.endTs : other.startTs + BigInt(3600);
      return schedule.startTs < otherEnd && other.startTs < scheduleEnd;
    });
  };

  // Drag handlers
  const handleDragStart = useCallback((e: React.DragEvent, schedule: Schedule) => {
    setDraggedSchedule(schedule);
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", schedule.name);

    // Add drag image styling
    const target = e.currentTarget as HTMLElement;
    target.style.opacity = "0.5";
  }, []);

  const handleDragEnd = useCallback((e: React.DragEvent) => {
    const target = e.currentTarget as HTMLElement;
    target.style.opacity = "1";
    setDraggedSchedule(null);
    setDragOverTime(null);
  }, []);

  const handleDragOver = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = "move";

      // Calculate time based on mouse position
      if (timelineRef.current && draggedSchedule) {
        const rect = timelineRef.current.getBoundingClientRect();
        const y = e.clientY - rect.top;
        const totalHeight = rect.height;

        // Map Y position to hour (8:00 - 22:00)
        const hourRange = 14; // 22 - 8
        const hour = Math.floor(8 + (y / totalHeight) * hourRange);
        const clampedHour = Math.max(8, Math.min(21, hour));
        const timeStr = `${clampedHour.toString().padStart(2, "0")}:00`;
        setDragOverTime(timeStr);
      }
    },
    [draggedSchedule],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();

      if (draggedSchedule && dragOverTime && onScheduleUpdate) {
        const [hours] = dragOverTime.split(":").map(Number);
        const originalStart = toDayjs(draggedSchedule.startTs);
        const originalEnd = toDayjs(draggedSchedule.endTs);
        const duration = originalEnd.diff(originalStart, "minute");

        // Calculate new start time on the same date
        const newStart = dayjs(selectedDateStr).hour(hours).minute(0).second(0);
        const newEnd = newStart.add(duration, "minute");

        const newStartTs = BigInt(newStart.unix());
        const newEndTs = BigInt(newEnd.unix());

        onScheduleUpdate(draggedSchedule, newStartTs, newEndTs);
      }

      setDraggedSchedule(null);
      setDragOverTime(null);
    },
    [draggedSchedule, dragOverTime, onScheduleUpdate, selectedDateStr],
  );

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Date Strip */}
      <div className="mb-4 sm:mb-6">
        <DateStrip
          currentDate={currentDate}
          selectedDate={selectedDateStr}
          schedules={schedules}
          onDateSelect={handleDateSelect}
          onPrevWeek={handlePrevWeek}
          onNextWeek={handleNextWeek}
        />
      </div>

      {/* Schedule Count Banner */}
      <div className="mb-3 sm:mb-4" role="status" aria-live="polite">
        <div
          className={cn(
            "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm",
            daySchedules.length > 0 ? "bg-primary/10 text-primary" : "bg-muted/50 text-muted-foreground",
          )}
        >
          {daySchedules.length > 0 ? (
            <>
              <Clock className="h-3.5 w-3.5" aria-hidden="true" />
              <span className="font-medium">{daySchedules.length}</span>
              <span className="text-muted-foreground">{t("schedule.schedules")}</span>
            </>
          ) : (
            <>
              <Coffee className="h-3.5 w-3.5" aria-hidden="true" />
              <span>{t("schedule.no-schedules")}</span>
            </>
          )}
        </div>
      </div>

      {/* Timeline Content */}
      <div
        ref={timelineRef}
        className="flex-1 overflow-y-auto -mx-1 px-1 relative"
        role="list"
        aria-label={t("schedule.schedule-list") as string}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        {/* Drag time indicator */}
        {dragOverTime && draggedSchedule && (
          <div className="absolute left-0 right-0 flex items-center pointer-events-none z-10 px-2">
            <div className="flex-1 h-0.5 bg-primary" />
            <span className="px-2 py-1 text-xs font-medium bg-primary text-primary-foreground rounded-md shadow-lg">{dragOverTime}</span>
            <div className="flex-1 h-0.5 bg-primary" />
          </div>
        )}

        {daySchedules.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 sm:py-20 text-muted-foreground" role="status">
            <div className="w-14 h-14 sm:w-16 sm:h-16 rounded-2xl bg-muted/50 flex items-center justify-center mb-4" aria-hidden="true">
              <Coffee className="w-5 h-5 sm:w-6 sm:h-6 opacity-50" />
            </div>
            <p className="text-sm font-medium">{t("schedule.no-schedules")}</p>
          </div>
        ) : (
          <div className="space-y-2 sm:space-y-3 pb-4">
            {daySchedules.map((schedule, idx) => {
              const startTime = toDayjs(schedule.startTs);
              const endTime = toDayjs(schedule.endTs);
              const conflict = hasConflict(schedule);
              const colors = [
                "bg-blue-500/10 border-blue-500/20 text-blue-700 dark:text-blue-300",
                "bg-purple-500/10 border-purple-500/20 text-purple-700 dark:text-purple-300",
                "bg-amber-500/10 border-amber-500/20 text-amber-700 dark:text-amber-300",
                "bg-emerald-500/10 border-emerald-500/20 text-emerald-700 dark:text-emerald-300",
                "bg-rose-500/10 border-rose-500/20 text-rose-700 dark:text-rose-300",
              ];
              const baseColor = colors[idx % colors.length];
              const conflictStyle = conflict ? "bg-red-500/10 border-red-500/30 text-red-700 dark:text-red-300" : baseColor;
              const ariaLabel = t("schedule.schedule-aria-label", {
                title: schedule.title || t("schedule.untitled"),
                startTime: startTime.format("HH:mm"),
                endTime: endTime.format("HH:mm"),
                hasConflict: conflict ? t("schedule.conflict") : "",
              }).trim();
              const isDragging = draggedSchedule?.name === schedule.name;

              return (
                <div
                  key={idx}
                  role="listitem"
                  draggable={!!onScheduleUpdate}
                  onDragStart={(e) => handleDragStart(e, schedule)}
                  onDragEnd={handleDragEnd}
                  onClick={() => onScheduleEdit?.(schedule)}
                  aria-label={ariaLabel}
                  className={cn(
                    "group relative rounded-xl border p-3 sm:p-4 transition-all duration-200 cursor-pointer w-full text-left",
                    "hover:shadow-md active:scale-[0.99]",
                    "focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2 focus-visible:outline-none",
                    conflictStyle,
                    isDragging && "opacity-50 scale-95",
                    onScheduleUpdate && "cursor-grab active:cursor-grabbing",
                  )}
                >
                  <div className="flex items-start gap-3 sm:gap-4">
                    {/* Drag Handle */}
                    {onScheduleUpdate && (
                      <div className="shrink-0 opacity-0 group-hover:opacity-50 transition-opacity cursor-grab" aria-hidden="true">
                        <GripVertical className="h-4 w-4 text-muted-foreground" />
                      </div>
                    )}

                    {/* Time Column */}
                    <div className="shrink-0 w-14 sm:w-16 text-right" aria-hidden="true">
                      <div className="text-sm font-semibold">{startTime.format("HH:mm")}</div>
                      <div className="text-xs opacity-70">{endTime.format("HH:mm")}</div>
                    </div>

                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <h4 className="font-semibold text-sm sm:text-base mb-1 sm:mb-1.5 truncate">
                        {schedule.title || t("schedule.untitled")}
                      </h4>

                      <div className="flex flex-wrap items-center gap-2 sm:gap-3 text-xs sm:text-sm opacity-80">
                        {schedule.location && (
                          <span className="flex items-center gap-1">
                            <MapPin className="h-3 w-3 sm:h-3.5 sm:w-3.5 shrink-0" aria-hidden="true" />
                            <span className="truncate">{schedule.location}</span>
                          </span>
                        )}
                        {schedule.description && <span className="line-clamp-1 hidden sm:inline">{schedule.description}</span>}
                      </div>
                    </div>
                  </div>

                  {/* Conflict Indicator */}
                  {conflict && (
                    <div className="absolute top-2 right-2" aria-label={t("schedule.schedule-conflict") as string}>
                      <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium bg-red-500/20 text-red-600 dark:text-red-400">
                        {t("schedule.conflict")}
                      </span>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
