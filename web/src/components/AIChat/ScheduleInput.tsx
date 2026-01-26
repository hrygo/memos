import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQueryClient } from "@tanstack/react-query";
import dayjs from "dayjs";
import { AlertTriangle, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  useCheckConflict,
  useCreateSchedule,
  useDeleteSchedule,
  useUpdateSchedule,
} from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import { ScheduleSchema, type Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";
import { ScheduleConflictAlert } from "./ScheduleConflictAlert";
import { ScheduleErrorBoundary } from "./ScheduleErrorBoundary";

interface ScheduleInputProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialText?: string;
  editSchedule?: Schedule | null;
  onSuccess?: (schedule: Schedule) => void;
  contextDate?: string;
}

export const ScheduleInput = ({ open, onOpenChange, editSchedule, onSuccess }: ScheduleInputProps) => {
  const t = useTranslate();
  const queryClient = useQueryClient();
  const createSchedule = useCreateSchedule();
  const updateSchedule = useUpdateSchedule();
  const deleteSchedule = useDeleteSchedule();
  const checkConflict = useCheckConflict();
  const isEditMode = !!editSchedule;

  const [parsedSchedule, setParsedSchedule] = useState<Schedule | null>(null);
  const [conflicts, setConflicts] = useState<Schedule[]>([]);
  const [showConflictAlert, setShowConflictAlert] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // Initialize/Reset state when dialog opens or editSchedule changes
  useEffect(() => {
    if (open) {
      if (editSchedule) {
        setParsedSchedule({ ...editSchedule });
      } else {
        // Default values for new schedule
        const now = dayjs();
        const start = now.add(10 - (now.minute() % 10) + 10, "minute").startOf("minute");
        const end = start.add(1, "hour");

        // We initialize as a Partial/compatible object and cast since we handle creation later
        setParsedSchedule(create(ScheduleSchema, {
          title: "",
          startTs: BigInt(start.unix()),
          endTs: BigInt(end.unix()),
          location: "",
          description: "",
          reminders: [],
          name: "",
        }));
      }
      setConflicts([]);
      setShowConflictAlert(false);
    }
  }, [open, editSchedule]);

  const executeCreate = async () => {
    if (!parsedSchedule) return;

    try {
      if (isEditMode && parsedSchedule.name) {
        // Update existing schedule
        const updatedSchedule = await updateSchedule.mutateAsync({
          schedule: parsedSchedule,
          updateMask: ["title", "description", "location", "start_ts", "end_ts", "reminders"],
        });

        if (updatedSchedule) {
          toast.success((t("schedule.schedule-updated") as string) || "Schedule updated");
          onSuccess?.(updatedSchedule);
          handleClose();
        }
      } else {
        // Create new schedule
        const validName = `schedules/${Date.now()}`;
        const scheduleToCreate = { ...parsedSchedule, name: validName };
        const createdSchedule = await createSchedule.mutateAsync(scheduleToCreate);

        if (createdSchedule) {
          toast.success(t("schedule.schedule-created"));
          onSuccess?.(createdSchedule);
          handleClose();
        }
      }
    } catch (error) {
      const isConflictError =
        error && typeof error === "object" && "message" in error ? (error.message as string).includes("conflicts detected") : false;

      if (isConflictError) {
        const errorMessage = (error as { message?: string }).message || "";
        toast.error(errorMessage);
      } else {
        toast.error(
          isEditMode ? (t("schedule.quick-input.failed-to-update") as string) : (t("schedule.quick-input.failed-to-create") as string),
        );
      }
      console.error(isEditMode ? "Update error:" : "Create error:", error);
    }
  };

  const handleCreate = async () => {
    if (!parsedSchedule) return;

    if (!parsedSchedule.title?.trim()) {
      toast.error(t("message.fill-all-required-fields") || "Please fill all required fields");
      return;
    }

    try {
      // Check conflict
      const conflict = await checkConflict.mutateAsync({
        startTs: parsedSchedule.startTs,
        endTs: parsedSchedule.endTs,
        excludeNames: isEditMode && parsedSchedule.name ? [parsedSchedule.name] : [],
      });

      if (conflict.conflicts.length > 0) {
        setConflicts(conflict.conflicts);
        setShowConflictAlert(true);
        return;
      }

      await executeCreate();
    } catch (error) {
      await executeCreate();
    }
  };

  const handleAdjust = () => setShowConflictAlert(false);
  const handleDiscard = () => handleClose();

  const handleClose = () => {
    setParsedSchedule(null);
    setConflicts([]);
    setShowConflictAlert(false);
    onOpenChange(false);
  };

  const handleDeleteClick = () => setShowDeleteConfirm(true);

  const confirmDelete = async () => {
    if (!parsedSchedule?.name) return;

    try {
      await deleteSchedule.mutateAsync(parsedSchedule.name);
      toast.success(t("schedule.schedule-deleted"));
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
      handleClose();
    } catch (error) {
      console.error("Delete error:", error);
      toast.error(t("schedule.parse-error") || "Failed to delete schedule");
    } finally {
      setShowDeleteConfirm(false);
    }
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <ScheduleErrorBoundary>
          <DialogContent size="sm" className="overflow-hidden min-w-[360px]">
            <DialogTitle className="text-lg font-semibold">
              {isEditMode ? t("schedule.edit-schedule") : t("schedule.create-schedule")}
            </DialogTitle>
            <DialogDescription className="hidden">Schedule Form</DialogDescription>

            <div className="space-y-4 pt-4">
              {parsedSchedule && (
                <div className="space-y-4">
                  {/* Title */}
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">{t("common.title")}</Label>
                    <Input
                      value={parsedSchedule.title}
                      onChange={(e) => setParsedSchedule({ ...parsedSchedule, title: e.target.value })}
                      className="font-medium"
                      placeholder={t("common.title")}
                      autoFocus
                    />
                  </div>

                  {/* Time Range */}
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">{t("schedule.start-time")}</Label>
                      <Input
                        type="datetime-local"
                        value={dayjs(timestampDate(create(TimestampSchema, { seconds: parsedSchedule.startTs, nanos: 0 }))).format("YYYY-MM-DDTHH:mm")}
                        onChange={(e) => {
                          const val = e.target.value;
                          if (val) setParsedSchedule({ ...parsedSchedule, startTs: BigInt(dayjs(val).unix()) });
                        }}
                        className="bg-muted/30"
                      />
                    </div>
                    <div className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">{t("schedule.end-time")}</Label>
                      <Input
                        type="datetime-local"
                        value={dayjs(timestampDate(create(TimestampSchema, { seconds: parsedSchedule.endTs, nanos: 0 }))).format("YYYY-MM-DDTHH:mm")}
                        onChange={(e) => {
                          const val = e.target.value;
                          if (val) setParsedSchedule({ ...parsedSchedule, endTs: BigInt(dayjs(val).unix()) });
                        }}
                        className="bg-muted/30"
                      />
                    </div>
                  </div>

                  {/* Location */}
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">{t("common.location")}</Label>
                    <Input
                      value={parsedSchedule.location || ""}
                      onChange={(e) => setParsedSchedule({ ...parsedSchedule, location: e.target.value })}
                      placeholder={t("common.location")}
                    />
                  </div>

                  {/* Description */}
                  <div className="space-y-1.5">
                    <Label className="text-xs text-muted-foreground">{t("common.description")}</Label>
                    <Textarea
                      value={parsedSchedule.description || ""}
                      onChange={(e) => setParsedSchedule({ ...parsedSchedule, description: e.target.value })}
                      className="min-h-[80px] text-sm resize-none bg-muted/30"
                      placeholder={t("common.description")}
                    />
                  </div>

                  {/* Footer Actions */}
                  <div className={cn("flex justify-between gap-3 pt-4 border-t", isEditMode ? "" : "justify-end")}>
                    {isEditMode && (
                      <Button
                        variant="ghost"
                        onClick={handleDeleteClick}
                        className="text-destructive hover:bg-destructive/10 px-2"
                      >
                        <Trash2 className="h-4 w-4 mr-2" />
                        {t("common.delete")}
                      </Button>
                    )}
                    <div className="flex gap-2">
                      <Button variant="outline" onClick={handleClose}>
                        {t("common.cancel")}
                      </Button>
                      <Button onClick={handleCreate} className="min-w-[80px]">
                        {isEditMode ? t("common.save") : t("common.create")}
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </DialogContent>
        </ScheduleErrorBoundary>
      </Dialog>

      <ScheduleConflictAlert
        open={showConflictAlert}
        onOpenChange={setShowConflictAlert}
        conflicts={conflicts}
        onAdjust={handleAdjust}
        onDiscard={handleDiscard}
      />

      <Dialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="h-5 w-5" />
              {t("schedule.delete-schedule")}
            </DialogTitle>
            <DialogDescription>{t("schedule.delete-confirm")}</DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-2 mt-4">
            <Button variant="outline" onClick={() => setShowDeleteConfirm(false)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={confirmDelete}>
              {t("common.delete")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
};
