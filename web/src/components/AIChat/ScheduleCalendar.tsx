import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import dayjs, { Dayjs } from "dayjs";
import { timestampDate } from "@bufbuild/protobuf/wkt";

interface ScheduleCalendarProps {
  schedules: Schedule[];
  selectedDate?: string;
  onDateClick?: (date: string) => void;
  className?: string;
}

export const ScheduleCalendar = ({
  schedules,
  selectedDate,
  onDateClick,
  className = "",
}: ScheduleCalendarProps) => {
  const t = useTranslate();
  const [currentMonth, setCurrentMonth] = dayjs();

  // Get days in month
  const getDaysInMonth = (date: Dayjs) => {
    const startOfMonth = date.startOf("month");
    const endOfMonth = date.endOf("month");
    const days = [];

    // Add days from previous month to fill the first week
    const startDayOfWeek = startOfMonth.day();
    for (let i = startDayOfWeek - 1; i >= 0; i--) {
      days.push(startOfMonth.subtract(i + 1, "day"));
    }

    // Add days in current month
    for (let i = 0; i < endOfMonth.date(); i++) {
      days.push(startOfMonth.add(i, "day"));
    }

    // Add days from next month to fill the last week
    const endDayOfWeek = endOfMonth.day();
    for (let i = 1; i < 7 - endDayOfWeek; i++) {
      days.push(endOfMonth.add(i, "day"));
    }

    return days;
  };

  // Get schedule count for a date
  const getScheduleCount = (date: Dayjs) => {
    const dateStr = date.format("YYYY-MM-DD");
    return schedules.filter((s) => {
      const scheduleDate = dayjs(timestampDate({ seconds: s.startTs, nanos: 0 })).format("YYYY-MM-DD");
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

  // Weekday labels
  const weekdays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

  // Days to display
  const days = getDaysInMonth(currentMonth);

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="text-lg font-semibold">{currentMonth.format("MMMM YYYY")}</h3>
          <Button variant="ghost" size="sm" onClick={goToToday}>
            {t("schedule.calendar") || "Today"}
          </Button>
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" onClick={goToPreviousMonth}>
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" onClick={goToNextMonth}>
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Calendar Grid */}
      <div className="flex flex-col gap-1">
        {/* Weekday Headers */}
        <div className="grid grid-cols-7 gap-1">
          {weekdays.map((day) => (
            <div key={day} className="p-2 text-center text-xs font-medium text-muted-foreground">
              {day}
            </div>
          ))}
        </div>

        {/* Days */}
        <div className="grid grid-cols-7 gap-1">
          {days.map((date, idx) => {
            const scheduleCount = getScheduleCount(date);
            const isTodayDate = isToday(date);
            const isSelectedDate = isSelected(date);
            const inCurrentMonth = isCurrentMonth(date);

            return (
              <button
                key={idx}
                onClick={() => handleDateClick(date)}
                className={cn(
                  "relative aspect-square rounded-lg p-1 text-sm transition-colors hover:bg-accent",
                  !inCurrentMonth && "text-muted-foreground/30",
                  isTodayDate && "bg-primary/10 font-semibold",
                  isSelectedDate && "bg-primary text-primary-foreground hover:bg-primary/90",
                )}
              >
                <span className="block text-center">{date.format("D")}</span>

                {/* Schedule indicator */}
                {scheduleCount > 0 && (
                  <div className="mt-1 flex justify-center gap-0.5">
                    {Array.from({ length: Math.min(scheduleCount, 3) }).map((_, i) => (
                      <div
                        key={i}
                        className={cn(
                          "h-1 w-1 rounded-full",
                          isSelectedDate ? "bg-primary-foreground" : "bg-primary",
                        )}
                      />
                    ))}
                  </div>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 text-xs text-muted-foreground">
        <div className="flex items-center gap-1">
          <div className="h-2 w-2 rounded-full bg-primary" />
          <span>Has schedules</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="h-2 w-2 rounded-full bg-primary/10" />
          <span>Today</span>
        </div>
      </div>
    </div>
  );
};
