import { useQueryClient } from "@tanstack/react-query";
import { Calendar, LayoutList } from "lucide-react";
import { useMemo, useState } from "react";
import { ScheduleCalendar } from "@/components/AIChat/ScheduleCalendar";
import { ScheduleInput } from "@/components/AIChat/ScheduleInput";
import { ScheduleSearchBar } from "@/components/AIChat/ScheduleSearchBar";
import { ScheduleTimeline } from "@/components/AIChat/ScheduleTimeline";
import { ScheduleQuickInput } from "@/components/ScheduleQuickInput/ScheduleQuickInput";
import { Button } from "@/components/ui/button";
import { useScheduleContext } from "@/contexts/ScheduleContext";
import { useSchedulesOptimized } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

type ViewTab = "calendar" | "timeline";

const Schedule = () => {
  const t = useTranslate();
  const queryClient = useQueryClient();
  const { selectedDate, setSelectedDate, filteredSchedules, hasSearchFilter, setFilteredSchedules, setHasSearchFilter } =
    useScheduleContext();

  // State - default to timeline
  const [viewTab, setViewTab] = useState<ViewTab>("timeline");
  const [scheduleInputOpen, setScheduleInputOpen] = useState(false);
  const [editSchedule, setEditSchedule] = useState<Schedule | null>(null);

  // Calculate anchor date from selectedDate or use today
  const anchorDate = useMemo(() => {
    return selectedDate ? new Date(selectedDate + "T00:00:00") : new Date();
  }, [selectedDate]);

  // Fetch schedules for search (desktop only)
  const { data: schedulesData } = useSchedulesOptimized(anchorDate);
  const allSchedules = schedulesData?.schedules || [];

  // Use filtered schedules when searching, otherwise use all schedules
  const displaySchedules = hasSearchFilter ? filteredSchedules : allSchedules;

  // Hide calendar view when filtering
  const effectiveViewTab = hasSearchFilter ? "timeline" : viewTab;

  // Handlers
  const handleEditSchedule = (schedule: Schedule) => {
    setEditSchedule(schedule);
    setScheduleInputOpen(true);
  };

  const handleCloseInput = () => {
    setScheduleInputOpen(false);
    setEditSchedule(null);
  };

  // Handle date click - switch to timeline tab
  const handleDateClick = (date: string) => {
    setSelectedDate(date);
    setViewTab("timeline");
  };

  return (
    <div className="w-full h-full flex flex-col overflow-hidden">
      {/* Header with View Tabs and Search (desktop) */}
      <div className="hidden lg:flex flex-none px-4 py-3 border-b border-border/50 overflow-hidden">
        <div className="flex items-center justify-between gap-4 w-full">
          {/* Left: View Tabs */}
          {!hasSearchFilter ? (
            <div className="flex items-center gap-1.5 p-1 bg-muted/50 rounded-lg" role="tablist" aria-label="视图切换">
              <Button
                role="tab"
                aria-selected={viewTab === "timeline"}
                aria-controls="schedule-panel"
                variant={viewTab === "timeline" ? "default" : "ghost"}
                size="sm"
                className={cn("h-9 px-3 text-sm font-medium rounded-md", viewTab === "timeline" ? "shadow-sm" : "hover:bg-transparent", "focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2")}
                onClick={() => setViewTab("timeline")}
              >
                <LayoutList className="w-4 h-4 mr-1.5" aria-hidden="true" />
                {t("schedule.timeline") || "Timeline"}
              </Button>
              <Button
                role="tab"
                aria-selected={viewTab === "calendar"}
                aria-controls="schedule-panel"
                variant={viewTab === "calendar" ? "default" : "ghost"}
                size="sm"
                className={cn("h-9 px-3 text-sm font-medium rounded-md", viewTab === "calendar" ? "shadow-sm" : "hover:bg-transparent", "focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2")}
                onClick={() => setViewTab("calendar")}
              >
                <Calendar className="w-4 h-4 mr-1.5" aria-hidden="true" />
                {t("schedule.month-view") || "Month"}
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2" role="status" aria-live="polite">
              <span className="text-sm text-muted-foreground">
                {filteredSchedules.length} {t("schedule.search-results") || "results"}
              </span>
            </div>
          )}

          {/* Right: Search Bar */}
          <div className="flex items-center gap-2 flex-1 justify-end">
            <ScheduleSearchBar
              schedules={allSchedules}
              onFilteredChange={setFilteredSchedules}
              onHasFilterChange={setHasSearchFilter}
              className="max-w-xs"
            />
          </div>
        </div>
      </div>

      {/* Mobile: View Tabs */}
      <div className="lg:hidden flex-none px-3 py-2 flex items-center justify-between gap-2 border-b border-border/50">
        {!hasSearchFilter ? (
          <div className="flex items-center gap-1 p-1 bg-muted/50 rounded-lg" role="tablist" aria-label="视图切换">
            <Button
              role="tab"
              aria-selected={viewTab === "timeline"}
              aria-label="时间线视图"
              variant={viewTab === "timeline" ? "default" : "ghost"}
              size="sm"
              className={cn("h-10 w-10 p-0 rounded-md min-h-[44px] min-w-[44px]", viewTab === "timeline" ? "shadow-sm" : "hover:bg-transparent", "focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2")}
              onClick={() => setViewTab("timeline")}
            >
              <LayoutList className="w-4 h-4" aria-hidden="true" />
            </Button>
            <Button
              role="tab"
              aria-selected={viewTab === "calendar"}
              aria-label="月历视图"
              variant={viewTab === "calendar" ? "default" : "ghost"}
              size="sm"
              className={cn("h-10 w-10 p-0 rounded-md min-h-[44px] min-w-[44px]", viewTab === "calendar" ? "shadow-sm" : "hover:bg-transparent", "focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2")}
              onClick={() => setViewTab("calendar")}
            >
              <Calendar className="w-4 h-4" aria-hidden="true" />
            </Button>
          </div>
        ) : (
          <div className="flex items-center gap-2" role="status" aria-live="polite">
            <span className="text-xs text-muted-foreground">
              {filteredSchedules.length} {t("schedule.search-results") || "results"}
            </span>
          </div>
        )}
      </div>

      {/* Content */}
      <div
        id="schedule-panel"
        role="tabpanel"
        aria-label={effectiveViewTab === "calendar" ? "月历视图" : "时间线视图"}
        tabIndex={0}
        className="flex-1 overflow-y-auto p-4 pb-4 overflow-x-hidden focus-visible:outline-none"
      >
        {effectiveViewTab === "calendar" ? (
          <ScheduleCalendar schedules={displaySchedules} selectedDate={selectedDate} onDateClick={handleDateClick} showMobileHint={false} />
        ) : (
          <ScheduleTimeline
            schedules={displaySchedules}
            selectedDate={selectedDate}
            onDateClick={handleDateClick}
            onScheduleEdit={handleEditSchedule}
          />
        )}
      </div>

      {/* Quick Input Bar - at bottom of content area */}
      <div className="flex-none p-4 bg-background/95 backdrop-blur-sm border-t border-border/50 overflow-visible">
        <ScheduleQuickInput
          initialDate={selectedDate}
          onScheduleCreated={() => {
            queryClient.invalidateQueries({ queryKey: ["schedules"] });
          }}
        />
      </div>

      {/* Schedule Input Dialog */}
      <ScheduleInput
        open={scheduleInputOpen}
        onOpenChange={handleCloseInput}
        editSchedule={editSchedule}
        onSuccess={() => {
          handleCloseInput();
          queryClient.invalidateQueries({ queryKey: ["schedules"] });
        }}
      />
    </div>
  );
};

export default Schedule;
