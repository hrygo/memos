import { useState, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslate } from "@/utils/i18n";
import { Button } from "@/components/ui/button";
import { CalendarDays, LayoutList, PlusIcon } from "lucide-react";
import { ScheduleTimeline } from "@/components/AIChat/ScheduleTimeline";
import { ScheduleCalendar } from "@/components/AIChat/ScheduleCalendar";
import { ScheduleInput } from "@/components/AIChat/ScheduleInput";
import { useSchedulesOptimized } from "@/hooks/useScheduleQueries";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";

const Schedule = () => {
  const t = useTranslate();
  const queryClient = useQueryClient();

  // State
  const [selectedDate, setSelectedDate] = useState<string | undefined>();
  const [scheduleViewMode, setScheduleViewMode] = useState<"timeline" | "calendar">("timeline");
  const [scheduleInputOpen, setScheduleInputOpen] = useState(false);
  const [editSchedule, setEditSchedule] = useState<Schedule | null>(null);

  // Calculate anchor date from selectedDate or use today
  const anchorDate = useMemo(() => {
    return selectedDate ? new Date(selectedDate + 'T00:00:00') : new Date();
  }, [selectedDate]);

  // Fetch schedules
  const { data: schedulesData } = useSchedulesOptimized(anchorDate);
  const schedules = schedulesData?.schedules || [];

  // Handlers
  const handleAddSchedule = () => {
    setEditSchedule(null);
    setScheduleInputOpen(true);
  };

  const handleEditSchedule = (schedule: Schedule) => {
    setEditSchedule(schedule);
    setScheduleInputOpen(true);
  };

  const handleCloseInput = () => {
    setScheduleInputOpen(false);
    setEditSchedule(null);
  };

  return (
    <div className="w-full max-w-6xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">{t("schedule.title") || "Schedule"}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t("schedule.description") || "Manage your schedules and events"}
          </p>
        </div>
        <Button onClick={handleAddSchedule} className="gap-2">
          <PlusIcon className="w-4 h-4" />
          <span>{t("schedule.add") || "Add Schedule"}</span>
        </Button>
      </div>

      {/* View Mode Toggle */}
      <div className="flex items-center gap-2 mb-4">
        <div className="flex items-center bg-muted rounded-lg p-0.5">
          <Button
            variant={scheduleViewMode === "timeline" ? "default" : "ghost"}
            size="sm"
            className="h-8 px-3 text-sm font-medium rounded-md"
            onClick={() => setScheduleViewMode("timeline")}
          >
            <LayoutList className="w-4 h-4 mr-2" />
            {t("schedule.your-timeline") || "Timeline"}
          </Button>
          <Button
            variant={scheduleViewMode === "calendar" ? "default" : "ghost"}
            size="sm"
            className="h-8 px-3 text-sm font-medium rounded-md"
            onClick={() => setScheduleViewMode("calendar")}
          >
            <CalendarDays className="w-4 h-4 mr-2" />
            {t("schedule.calendar-view") || "Calendar"}
          </Button>
        </div>
      </div>

      {/* Content */}
      <div className="border rounded-lg bg-background overflow-hidden min-h-[calc(100vh-300px)]">
        {scheduleViewMode === "timeline" ? (
          <ScheduleTimeline
            schedules={schedules}
            selectedDate={selectedDate}
            onDateClick={setSelectedDate}
            onScheduleEdit={handleEditSchedule}
            className="h-full"
          />
        ) : (
          <ScheduleCalendar
            schedules={schedules}
            selectedDate={selectedDate}
            onDateClick={(date) => {
              setSelectedDate(date);
              setScheduleViewMode("timeline");
            }}
            showMobileHint={true}
            className="h-full p-4"
          />
        )}
      </div>

      {/* Schedule Input Dialog */}
      <ScheduleInput
        open={scheduleInputOpen}
        onOpenChange={handleCloseInput}
        editSchedule={editSchedule}
        onSuccess={() => {
          handleCloseInput();
          // Refetch schedules by invalidating the hook's cache
          queryClient.invalidateQueries({ queryKey: ["schedules"] });
        }}
      />
    </div>
  );
};

export default Schedule;
