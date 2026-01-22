import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs, { Dayjs } from "dayjs";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

interface ScheduleCalendarProps {
  schedules: Schedule[];
  selectedDate?: string;
  onDateClick?: (date: string) => void;
  className?: string;
  // Show mobile hint about tapping to view details
  showMobileHint?: boolean;
}

export const ScheduleCalendar = ({
  schedules,
  selectedDate,
  onDateClick,
  className = "",
  showMobileHint = false,
}: ScheduleCalendarProps) => {
  const t = useTranslate();
  const [currentMonth, setCurrentMonth] = useState(dayjs());

  // Get days in month
  const getDaysInMonth = (date: Dayjs) => {
    const startOfMonth = date.startOf("month");
    const endOfMonth = date.endOf("month");
    const days = [];

    // Add days from previous month to fill the first week (Monday start)
    const startDayOfWeek = startOfMonth.day(); // 0 is Sunday, 1 is Monday
    // Calculate days to subtract: Mon(1)->0, Tue(2)->1, ..., Sun(0)->6
    const daysFromPrevMonth = (startDayOfWeek + 6) % 7;

    for (let i = daysFromPrevMonth - 1; i >= 0; i--) {
      days.push(startOfMonth.subtract(i + 1, "day"));
    }

    // Add days in current month
    for (let i = 0; i < endOfMonth.date(); i++) {
      days.push(startOfMonth.add(i, "day"));
    }

    // Add days from next month to fill the last week
    // Calculate remaining days to fill the row (row length is 7)
    const remainingSlots = 7 - (days.length % 7);
    if (remainingSlots < 7) {
      for (let i = 1; i <= remainingSlots; i++) {
        days.push(endOfMonth.add(i, "day"));
      }
    }

    return days;
  };

  // Get schedule count for a date
  const getScheduleCount = (date: Dayjs) => {
    const dateStr = date.format("YYYY-MM-DD");
    return schedules.filter((s) => {
      const scheduleDate = dayjs(timestampDate(create(TimestampSchema, { seconds: s.startTs, nanos: 0 }))).format("YYYY-MM-DD");
      return scheduleDate === dateStr;
    }).length;
  };

  // Check if date is today
  const isToday = (date: Dayjs) => {
    return date.format("YYYY-MM-DD") === dayjs().format("YYYY-MM-DD");
  };

  // Check if date is selected
  const isSelected = (date: Dayjs) => {
    return date.format("YYYY-MM-DD") === selectedDate;
  };

  // Check if date is in current month
  const isCurrentMonth = (date: Dayjs) => {
    return date.month() === currentMonth.month();
  };

  // Navigate to previous month
  const goToPreviousMonth = () => {
    setCurrentMonth(currentMonth.subtract(1, "month"));
  };

  // Navigate to next month
  const goToNextMonth = () => {
    setCurrentMonth(currentMonth.add(1, "month"));
  };

  // Go to today
  const goToToday = () => {
    setCurrentMonth(dayjs());
    onDateClick?.(dayjs().format("YYYY-MM-DD"));
  };

  // Handle date click
  const handleDateClick = (date: Dayjs) => {
    onDateClick?.(date.format("YYYY-MM-DD"));
  };

  // Weekday labels (Monday start)
  const weekdays = [t("days.mon"), t("days.tue"), t("days.wed"), t("days.thu"), t("days.fri"), t("days.sat"), t("days.sun")];

  // Days to display
  const days = getDaysInMonth(currentMonth);
  const weeks = Math.ceil(days.length / 7);

  return (
    <div className={cn("flex flex-col gap-1.5", className)} role="region" aria-label={t("schedule.calendar-view") as string}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold" aria-live="polite">
          {currentMonth.format("YYYY MMMM")}
        </h3>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={goToToday}
            className="h-8 min-h-[44px] sm:min-h-0 px-3 text-muted-foreground hover:text-foreground cursor-pointer focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
            aria-label={t("schedule.jump-to-today") as string}
          >
            {t("common.today") || "Today"}
          </Button>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              onClick={goToPreviousMonth}
              className="h-9 w-9 min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0 cursor-pointer focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
              aria-label={t("schedule.previous-month") as string}
            >
              <ChevronLeft className="h-4 w-4" aria-hidden="true" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={goToNextMonth}
              className="h-9 w-9 min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0 cursor-pointer focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
              aria-label={t("schedule.next-month") as string}
            >
              <ChevronRight className="h-4 w-4" aria-hidden="true" />
            </Button>
          </div>
        </div>
      </div>

      {/* Calendar Grid */}
      <div className="flex-1 flex flex-col gap-1 min-h-0" role="grid" aria-label={t("schedule.calendar") as string}>
        {/* Weekday Headers */}
        <div className="grid grid-cols-7 gap-1 flex-none" role="row">
          {weekdays.map((day, index) => (
            <div
              key={index}
              className="py-1 text-center text-xs font-medium text-muted-foreground"
              role="columnheader"
              aria-label={String(day)}
            >
              {day}
            </div>
          ))}
        </div>

        {/* Days */}
        <div
          className="grid grid-cols-7 gap-1 flex-1 min-h-0"
          style={{ gridTemplateRows: `repeat(${weeks}, minmax(0, 1fr))` }}
          role="rowgroup"
        >
          {days.map((date, idx) => {
            const scheduleCount = getScheduleCount(date);
            const isTodayDate = isToday(date);
            const isSelectedDate = isSelected(date);
            const inCurrentMonth = isCurrentMonth(date);
            const dateLabel = date.format("YYYY年M月D日");
            const ariaLabel = `${dateLabel}${isTodayDate ? "，今天" : ""}${inCurrentMonth ? "" : "，非本月"}${scheduleCount > 0 ? `，${scheduleCount} 个日程` : ""}`;

            return (
              <button
                key={idx}
                role="gridcell"
                aria-selected={isSelectedDate}
                aria-label={ariaLabel}
                onClick={() => handleDateClick(date)}
                className={cn(
                  "relative w-full h-full min-h-[3rem] rounded-lg p-1 text-sm transition-colors cursor-pointer flex flex-col items-center justify-start pt-1 gap-1",
                  "focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2 focus-visible:outline-none",
                  !inCurrentMonth && "text-muted-foreground/30",
                  // Hover state for the whole cell
                  "hover:bg-accent/30",
                )}
              >
                <span
                  className={cn(
                    "flex items-center justify-center w-8 h-8 rounded-full transition-colors font-medium",
                    // Selected state
                    isSelectedDate && !isTodayDate && "bg-primary text-primary-foreground shadow-sm",
                    isSelectedDate && isTodayDate && "bg-orange-600 text-white shadow-sm",
                    // Today state (when not selected)
                    !isSelectedDate && isTodayDate && "bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300",
                    // Logic to ensure text color is correct when not selected/today
                    !isSelectedDate && !isTodayDate && "text-foreground",
                  )}
                  aria-hidden="true"
                >
                  {date.format("D")}
                </span>

                {/* Schedule indicator */}
                {scheduleCount > 0 && (
                  <div className="flex justify-center gap-0.5" aria-hidden="true">
                    {Array.from({ length: Math.min(scheduleCount, 3) }).map((_, i) => (
                      <div key={i} className={cn("h-1 w-1 rounded-full bg-primary/70")} />
                    ))}
                  </div>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 text-xs text-muted-foreground" role="legend" aria-label={t("schedule.legend") as string}>
        <div className="flex items-center gap-1">
          <div className="h-2 w-2 rounded-full bg-primary" aria-hidden="true" />
          <span>{t("schedule.has-schedules") || "Has schedules"}</span>
        </div>
        <div className="flex items-center gap-1">
          <div
            className="h-2 w-2 rounded-full bg-orange-200 border border-orange-300 dark:bg-orange-900/50 dark:border-orange-800"
            aria-hidden="true"
          />
          <span>{t("common.today") || "Today"}</span>
        </div>
      </div>

      {/* Mobile hint - shown only on small screens */}
      {showMobileHint && (
        <div className="md:hidden mt-3 pt-3 border-t border-border/50" role="note" aria-label={t("schedule.hint") as string}>
          <p className="text-xs text-center text-muted-foreground flex items-center justify-center gap-1.5">
            <span className="inline-block w-1.5 h-1.5 rounded-full bg-primary animate-pulse" aria-hidden="true" />
            {t("schedule.tap-to-view") || "Tap a date to view schedule details"}
          </p>
        </div>
      )}
    </div>
  );
};
