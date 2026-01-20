import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs, { Dayjs } from "dayjs";
import "dayjs/locale/zh-cn";
import { AlertTriangle, ChevronLeft, ChevronRight, Clock, Coffee, MapPin, MoreVertical, Pencil, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useDeleteSchedule } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

// Set dayjs default locale to Chinese
dayjs.locale("zh-cn");

interface ScheduleTimelineProps {
  schedules: Schedule[];
  selectedDate?: string;
  onDateClick?: (date: string) => void;
  onScheduleEdit?: (schedule: Schedule) => void;
  className?: string;
}

export const ScheduleTimeline = ({ schedules, selectedDate, onDateClick, onScheduleEdit, className = "" }: ScheduleTimelineProps) => {
  const t = useTranslate();
  const deleteSchedule = useDeleteSchedule();
  // Initialize with selectedDate or today
  const [currentDate, setCurrentDate] = useState(selectedDate ? dayjs(selectedDate) : dayjs());
  const [scheduleToDelete, setScheduleToDelete] = useState<string | null>(null);

  // Sync internal state if prop changes
  useEffect(() => {
    if (selectedDate) {
      setCurrentDate(dayjs(selectedDate));
    }
  }, [selectedDate]);

  // Generate days for the strip (current week view: -3 to +3 days from current focused date, or fixed week?)
  // Let's do a fixed 7-day strip centered on the "anchor" date, but we allow sliding the anchor.
  // Actually, a sliding window of 5-7 days centered on the selected date feels most modern.
  const datesToShow = Array.from({ length: 7 }, (_, i) => {
    return currentDate.subtract(3 - i, "day");
  });

  const handleDateSelect = (date: Dayjs) => {
    setCurrentDate(date);
    onDateClick?.(date.format("YYYY-MM-DD"));
  };

  const handlePrevDays = () => {
    const newDate = currentDate.subtract(1, "week");
    setCurrentDate(newDate);
    // Optionally trigger select? No, just move view.
  };

  const handleNextDays = () => {
    const newDate = currentDate.add(1, "week");
    setCurrentDate(newDate);
  };

  const handleGoToday = () => {
    const today = dayjs();
    setCurrentDate(today);
    onDateClick?.(today.format("YYYY-MM-DD"));
  };

  const handleDelete = (name: string) => {
    setScheduleToDelete(name);
  };

  const confirmDelete = async () => {
    if (!scheduleToDelete) return;
    try {
      await deleteSchedule.mutateAsync(scheduleToDelete);
      toast.success(t("schedule.schedule-deleted"));
    } catch (_error) {
      toast.error("Failed to delete schedule");
    } finally {
      setScheduleToDelete(null);
    }
  };

  // Filter and Sort Schedules for selected date
  const selectedDateStr = currentDate.format("YYYY-MM-DD");
  const daySchedules = schedules
    .filter((s) => {
      const sDate = dayjs(timestampDate(create(TimestampSchema, { seconds: s.startTs, nanos: 0 }))).format("YYYY-MM-DD");
      return sDate === selectedDateStr;
    })
    .sort((a, b) => {
      // Sort by time
      return Number(a.startTs) - Number(b.startTs);
    });

  // Check if a schedule conflicts with any other schedule
  const hasConflict = (schedule: Schedule): boolean => {
    const scheduleEnd = schedule.endTs > 0 ? schedule.endTs : schedule.startTs + BigInt(3600); // Default 1 hour

    return daySchedules.some((other) => {
      if (other.name === schedule.name) return false; // Skip self

      const otherEnd = other.endTs > 0 ? other.endTs : other.startTs + BigInt(3600);

      // Check for time overlap: two intervals [s1, e1] and [s2, e2] overlap if: s1 < e2 AND s2 < e1
      return schedule.startTs < otherEnd && other.startTs < scheduleEnd;
    });
  };

  // Get conflicting schedule names for a schedule
  const getConflictingSchedules = (schedule: Schedule): Schedule[] => {
    const scheduleEnd = schedule.endTs > 0 ? schedule.endTs : schedule.startTs + BigInt(3600);

    return daySchedules.filter((other) => {
      if (other.name === schedule.name) return false;

      const otherEnd = other.endTs > 0 ? other.endTs : other.startTs + BigInt(3600);

      return schedule.startTs < otherEnd && other.startTs < scheduleEnd;
    });
  };

  return (
    <div className={cn("flex flex-col h-full bg-background/50 rounded-xl overflow-hidden", className)}>
      {/* --- Top Section: Date Strip --- */}
      <div className="flex-none p-4 pb-2 border-b border-border/40 backdrop-blur-sm">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-lg font-semibold tracking-tight text-foreground/90">{currentDate.format("MMMM YYYY")}</h3>
          <div className="flex items-center gap-1">
            <Button variant="ghost" size="sm" onClick={handleGoToday} className="h-7 px-2 text-xs font-medium">
              {t("common.today")}
            </Button>
            <div className="flex items-center bg-muted/50 rounded-md p-0.5">
              <Button variant="ghost" size="icon" className="h-6 w-6 rounded-sm" onClick={handlePrevDays}>
                <ChevronLeft className="h-3 w-3" />
              </Button>
              <Button variant="ghost" size="icon" className="h-6 w-6 rounded-sm" onClick={handleNextDays}>
                <ChevronRight className="h-3 w-3" />
              </Button>
            </div>
          </div>
        </div>

        <div className="flex justify-between items-center gap-1">
          {datesToShow.map((date) => {
            const isSelected = date.format("YYYY-MM-DD") === selectedDateStr;
            const isToday = date.format("YYYY-MM-DD") === dayjs().format("YYYY-MM-DD");

            return (
              <button
                key={date.toString()}
                onClick={() => handleDateSelect(date)}
                className={cn(
                  "relative flex flex-col items-center justify-center min-w-[3rem] py-2 px-1 rounded-2xl transition-all duration-200",
                  isSelected && !isToday && "bg-primary text-primary-foreground shadow-md scale-105",
                  isSelected && isToday && "bg-orange-700 text-white shadow-md scale-105",
                  !isSelected && isToday && "bg-orange-200 text-orange-800 dark:bg-orange-900/50 dark:text-orange-300 font-bold",
                  !isSelected && !isToday && "hover:bg-muted text-muted-foreground hover:text-foreground",
                )}
              >
                <span
                  className={cn(
                    "text-[10px] font-medium uppercase tracking-wider mb-1 opacity-80",
                    isSelected && "text-primary-foreground/90",
                    isSelected && isToday && "text-white/90",
                    !isSelected && isToday && "text-orange-700/80 dark:text-orange-300/80",
                  )}
                >
                  {date.format("ddd")}
                </span>
                <span
                  className={cn(
                    "text-lg font-bold leading-none",
                    isSelected && "text-primary-foreground",
                    isSelected && isToday && "text-white",
                  )}
                >
                  {date.format("D")}
                </span>
                {/* Schedule Indicator Dots */}
                {(() => {
                  const dateStr = date.format("YYYY-MM-DD");
                  const scheduleCount = schedules.filter((s) => {
                    const sDate = dayjs(timestampDate(create(TimestampSchema, { seconds: s.startTs, nanos: 0 }))).format("YYYY-MM-DD");
                    return sDate === dateStr;
                  }).length;

                  if (scheduleCount === 0) return null;

                  return (
                    <div className="absolute -bottom-1 left-1/2 -translate-x-1/2 flex gap-0.5">
                      {Array.from({ length: Math.min(scheduleCount, 3) }).map((_, i) => (
                        <span
                          key={i}
                          className={cn(
                            "w-1 h-1 rounded-full",
                            // Use foreground color for selected state (works in both light/dark themes)
                            isSelected
                              ? "bg-foreground shadow-sm"
                              : "bg-primary"
                          )}
                        />
                      ))}
                    </div>
                  );
                })()}
              </button>
            );
          })}
        </div>
      </div>

      {/* --- Bottom Section: Timeline --- */}
      <div className="flex-1 overflow-y-auto p-0 min-h-[300px]">
        {daySchedules.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-8 space-y-3">
            <div className="p-4 bg-muted/50 rounded-full">
              <Coffee className="w-8 h-8 opacity-50" />
            </div>
            <p className="text-sm font-medium">{t("schedule.no-schedules")}</p>
          </div>
        ) : (
          <div className="py-6 px-4 space-y-0 relative">
            {/* Vertical Line */}
            <div className="absolute left-[3.8rem] top-6 bottom-6 w-px bg-border/60" />

            {daySchedules.map((schedule, idx) => {
              const startTime = dayjs(timestampDate(create(TimestampSchema, { seconds: schedule.startTs, nanos: 0 })));
              const endTime = dayjs(timestampDate(create(TimestampSchema, { seconds: schedule.endTs, nanos: 0 })));

              // Check for conflicts
              const conflict = hasConflict(schedule);
              const conflictingSchedules = getConflictingSchedules(schedule);

              // Color coding based on some hash or simple index cycler
              // Use red for conflicts, otherwise use the normal color cycle
              const colors = ["bg-blue-500", "bg-purple-500", "bg-amber-500", "bg-emerald-500", "bg-rose-500"];
              const accentColor = conflict ? "bg-red-500" : colors[idx % colors.length];

              return (
                <div key={idx} className="group relative flex gap-4 mb-6 last:mb-0">
                  {/* Time Column */}
                  <div className="w-12 flex-none flex flex-col items-end pt-0.5">
                    <span className="text-xs font-bold text-foreground/80">{startTime.format("HH:mm")}</span>
                  </div>

                  {/* Timeline Dot */}
                  <div className="relative flex-none w-3 flex justify-center pt-1.5 z-10">
                    <div className={cn("w-3 h-3 rounded-full border-[2px] border-background shadow-sm ring-1 ring-border", accentColor)} />
                  </div>

                  {/* Card */}
                  <div className="flex-1 min-w-0">
                    <div
                      className={cn(
                        "bg-card hover:bg-accent/50 transition-colors rounded-xl p-3 shadow-sm group-hover:shadow-md relative",
                        // Add red border and background for conflicts
                        conflict && "border-2 border-red-500/50 bg-red-50/50 dark:bg-red-950/20 dark:border-red-500/70"
                      )}
                    >
                      {/* Action Menu (Visible on Hover) */}
                      <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity z-20">
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" className="h-6 w-6 hover:bg-background/80">
                              <MoreVertical className="w-3.5 h-3.5" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem
                              onClick={(e) => {
                                e.stopPropagation();
                                onScheduleEdit?.(schedule);
                              }}
                              className="cursor-pointer"
                            >
                              <Pencil className="mr-2 h-4 w-4" />
                              {t("common.edit")}
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={(e) => {
                                e.stopPropagation();
                                handleDelete(schedule.name);
                              }}
                              className="text-destructive focus:text-destructive cursor-pointer"
                            >
                              <Trash2 className="mr-2 h-4 w-4" />
                              {t("common.delete")}
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>

                      <h4 className={cn(
                        "font-semibold text-sm mb-1 truncate leading-tight pr-6",
                        conflict && "text-red-900 dark:text-red-100"
                      )}>
                        {schedule.title || "Untitled Event"}
                      </h4>
                      <div className="flex items-center h-4 text-xs text-muted-foreground gap-3">
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {startTime.format("HH:mm")} - {endTime.format("HH:mm")}
                        </span>
                        {schedule.location && (
                          <span className="flex items-center gap-1">
                            <MapPin className="w-3 h-3" />
                            {schedule.location}
                          </span>
                        )}
                      </div>
                      {schedule.description && (
                        <p className="mt-2 text-xs text-foreground/70 line-clamp-2 leading-relaxed border-t border-border/50 pt-2">
                          {schedule.description}
                        </p>
                      )}

                      {/* Conflict Details */}
                      {conflict && conflictingSchedules.length > 0 && (
                        <div className="mt-2 pt-2 border-t border-red-200 dark:border-red-800">
                          <p className="text-[10px] text-red-600 dark:text-red-400">
                            {t("schedule.conflict-warning") || "Conflicts with"}: {conflictingSchedules.map(s => s.title).join(", ")}
                          </p>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      <Dialog open={!!scheduleToDelete} onOpenChange={(open) => !open && setScheduleToDelete(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("schedule.delete-schedule")}</DialogTitle>
            <DialogDescription>{t("schedule.delete-confirm")}</DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-2 mt-4">
            <Button variant="outline" onClick={() => setScheduleToDelete(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={confirmDelete}>
              {t("common.delete")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
};
