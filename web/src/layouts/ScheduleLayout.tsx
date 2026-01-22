import { Outlet } from "react-router-dom";
import { ScheduleCalendar } from "@/components/AIChat/ScheduleCalendar";
import { ScheduleSearchBar } from "@/components/AIChat/ScheduleSearchBar";
import NavigationDrawer from "@/components/NavigationDrawer";
import { useScheduleContext } from "@/contexts/ScheduleContext";
import useMediaQuery from "@/hooks/useMediaQuery";
import { useSchedulesOptimized } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";

const ScheduleSidebar = () => {
  const { selectedDate, setSelectedDate } = useScheduleContext();

  // Anchor date for schedule fetching - use selected date or today
  const anchorDate = selectedDate ? new Date(selectedDate + "T00:00:00") : new Date();
  const { data: schedulesData } = useSchedulesOptimized(anchorDate);
  const schedules = schedulesData?.schedules || [];

  return (
    <div className="h-full overflow-y-auto py-4 px-3">
      <ScheduleCalendar schedules={schedules} selectedDate={selectedDate} onDateClick={setSelectedDate} showMobileHint={false} />
    </div>
  );
};

const ScheduleLayout = () => {
  const lg = useMediaQuery("lg");
  const { setFilteredSchedules, setHasSearchFilter } = useScheduleContext();

  // Fetch schedules for search
  const anchorDate = new Date();
  const { data: schedulesData } = useSchedulesOptimized(anchorDate);
  const allSchedules = schedulesData?.schedules || [];

  return (
    <section className="@container w-full h-screen flex flex-col lg:h-screen overflow-hidden">
      {/* Mobile Header with Search */}
      <div className="lg:hidden flex-none flex items-center gap-2 px-4 py-3 border-b border-border/50 bg-background">
        <NavigationDrawer />
        <ScheduleSearchBar
          schedules={allSchedules}
          onFilteredChange={setFilteredSchedules}
          onHasFilterChange={setHasSearchFilter}
          className="flex-1 min-w-0"
        />
      </div>

      {/* Desktop Sidebar */}
      {lg && (
        <div className="fixed top-0 left-16 shrink-0 h-svh border-r border-border bg-background w-80 overflow-y-auto">
          <ScheduleSidebar />
        </div>
      )}

      {/* Main Content */}
      <div className={cn("flex-1 min-h-0 overflow-x-hidden", lg ? "pl-80" : "")}>
        <Outlet />
      </div>
    </section>
  );
};

export default ScheduleLayout;
