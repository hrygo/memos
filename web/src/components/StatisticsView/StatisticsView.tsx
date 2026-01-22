import dayjs from "dayjs";
import { useMemo, useState } from "react";
import { MonthCalendar } from "@/components/ActivityCalendar";
import type { ScheduleSummary } from "@/components/ActivityCalendar/types";
import { useDateFilterNavigation, useSchedulesByMonthGrouped } from "@/hooks";
import type { StatisticsData } from "@/types/statistics";
import { MonthNavigator } from "./MonthNavigator";

interface Props {
  statisticsData: StatisticsData;
}

const StatisticsView = (props: Props) => {
  const { statisticsData } = props;
  const { activityStats } = statisticsData;
  const navigateToDateFilter = useDateFilterNavigation();
  const [visibleMonthString, setVisibleMonthString] = useState(dayjs().format("YYYY-MM"));

  // Fetch schedules for the current month
  const { data: schedulesResponse } = useSchedulesByMonthGrouped(visibleMonthString);

  // Maximum schedule duration to prevent infinite loops (1 year)
  const MAX_SCHEDULE_DAYS = 365;

  // Group schedules by date, handling multi-day schedules
  const schedulesByDate = useMemo(() => {
    if (!schedulesResponse?.schedules) return {};

    const grouped: Record<string, ScheduleSummary[]> = {};
    for (const schedule of schedulesResponse.schedules) {
      const startDate = dayjs(Number(schedule.startTs));
      const endDate = dayjs(Number(schedule.endTs));

      // Validate date range - skip invalid schedules
      if (!startDate.isValid() || !endDate.isValid()) {
        console.error("[StatisticsView] Invalid schedule dates", schedule.name);
        continue;
      }

      // Check for end before start - skip or swap
      if (endDate.isBefore(startDate)) {
        console.error("[StatisticsView] Schedule endTs before startTs", schedule.name);
        continue;
      }

      let currentDate = startDate;
      let daysProcessed = 0;
      while (currentDate.isBefore(endDate) || currentDate.isSame(endDate, "day")) {
        if (++daysProcessed > MAX_SCHEDULE_DAYS) {
          console.warn("[StatisticsView] Schedule exceeds max duration, truncating", schedule.name);
          break;
        }
        const dateKey = currentDate.format("YYYY-MM-DD");
        if (!grouped[dateKey]) {
          grouped[dateKey] = [];
        }
        grouped[dateKey].push({
          uid: schedule.name,
          title: schedule.title,
          startTs: schedule.startTs,
          endTs: schedule.endTs,
          allDay: schedule.allDay,
          location: schedule.location,
        });
        currentDate = currentDate.add(1, "day");
      }
    }
    return grouped;
  }, [schedulesResponse]);

  const maxCount = useMemo(() => {
    const counts = Object.values(activityStats);
    return Math.max(...counts, 1);
  }, [activityStats]);

  return (
    <div className="group w-full mt-2 flex flex-col text-muted-foreground animate-fade-in">
      <MonthNavigator visibleMonth={visibleMonthString} onMonthChange={setVisibleMonthString} activityStats={activityStats} />

      <div className="w-full animate-scale-in">
        <MonthCalendar
          month={visibleMonthString}
          data={activityStats}
          maxCount={maxCount}
          onClick={navigateToDateFilter}
          schedulesByDate={schedulesByDate}
        />
      </div>
    </div>
  );
};

export default StatisticsView;
