import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { Calendar, Clock, Loader2, MapPin, X } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useCheckConflict, useCreateSchedule, useDeleteSchedule, useParseAndCreateSchedule, useUpdateSchedule } from "@/hooks/useScheduleQueries";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";
import { ScheduleConflictAlert } from "./ScheduleConflictAlert";
import { ScheduleErrorBoundary } from "./ScheduleErrorBoundary";

interface ScheduleInputProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialText?: string;
  editSchedule?: Schedule | null;
  onSuccess?: (schedule: Schedule) => void;
}

export const ScheduleInput = ({ open, onOpenChange, initialText = "", editSchedule, onSuccess }: ScheduleInputProps) => {
  const t = useTranslate();
  const parseAndCreate = useParseAndCreateSchedule();
  const createSchedule = useCreateSchedule();
  const updateSchedule = useUpdateSchedule();
  const checkConflict = useCheckConflict();
  const deleteSchedule = useDeleteSchedule();
  const [showOverwriteConfirm, setShowOverwriteConfirm] = useState(false);
  const isEditMode = !!editSchedule;

  const [input, setInput] = useState(initialText);
  const [parsedSchedule, setParsedSchedule] = useState<Schedule | null>(editSchedule || null);
  const [isParsing, setIsParsing] = useState(false);
  const [conflicts, setConflicts] = useState<Schedule[]>([]);
  const [showConflictAlert, setShowConflictAlert] = useState(false);

  // Initialize with editSchedule when it changes
  useEffect(() => {
    if (editSchedule) {
      setParsedSchedule(editSchedule);
      setInput(editSchedule.title || "");
    } else {
      setParsedSchedule(null);
      setInput(initialText);
    }
  }, [editSchedule, initialText]);

  const handleParse = async () => {
    if (!input.trim()) return;

    // Validate input length (max 500 characters)
    if (input.length > 500) {
      // biome-ignore lint/suspicious/noExplicitAny: Temporary fix for missing translation key
      toast.error(t("schedule.input-too-long" as any));
      return;
    }

    setIsParsing(true);
    try {
      const result = await parseAndCreate.mutateAsync({
        text: input,
        autoConfirm: false,
      });

      if (result.parsedSchedule) {
        // Ensure endTs has a default duration if missing (1 hour)
        if (result.parsedSchedule.endTs === BigInt(0)) {
          result.parsedSchedule.endTs = result.parsedSchedule.startTs + BigInt(3600);
        }
        setParsedSchedule(result.parsedSchedule);

        // Check for conflicts
        const conflictResult = await checkConflict.mutateAsync({
          startTs: result.parsedSchedule.startTs,
          endTs: result.parsedSchedule.endTs,
          excludeNames: [],
        });

        if (conflictResult.conflicts.length > 0) {
          setConflicts(conflictResult.conflicts);
          setShowConflictAlert(true);
        }
      }
    } catch (error) {
      toast.error(t("schedule.parse-error"));
      console.error("Parse error:", error);
    } finally {
      setIsParsing(false);
    }
  };

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
        const validName =
          parsedSchedule.name && parsedSchedule.name.startsWith("schedules/") && parsedSchedule.name.length > 10
            ? parsedSchedule.name
            : `schedules/${self.crypto.randomUUID()}`;

        const scheduleToCreate = { ...parsedSchedule, name: validName };

        const createdSchedule = await createSchedule.mutateAsync(scheduleToCreate);

        if (createdSchedule) {
          toast.success(t("schedule.schedule-created"));
          onSuccess?.(createdSchedule);
          handleClose();
        }
      }
    } catch (error) {
      // Check if error is due to schedule conflicts
      const isConflictError = error && typeof error === "object" && "message" in error
        ? (error.message as string).includes("conflicts detected")
        : false;

      if (isConflictError) {
        toast.error(t("schedule.conflict-error") || "Schedule conflicts detected. Please check your schedule.", {
          duration: 5000,
          id: "schedule-conflict-error",
        });
      } else {
        toast.error(isEditMode ? "Failed to update schedule" : "Failed to create schedule");
      }
      console.error(isEditMode ? "Update error:" : "Create error:", error);
    }
  };

  const handleCreate = async () => {
    if (!parsedSchedule) return;

    try {
      // Ensure parsedSchedule has a valid endTs (default 1 hour if 0)
      if (parsedSchedule.endTs === BigInt(0)) {
        parsedSchedule.endTs = parsedSchedule.startTs + BigInt(3600);
        setParsedSchedule({ ...parsedSchedule });
      }

      // Check conflict - exclude self if editing
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
      console.error("Conflict check error:", error);
      // If conflict check fails, try to create anyway? Or just show error?
      // For now, proceed to create which might fail if backend enforces strictly, but usually it doesn't.
      await executeCreate();
    }
  };

  const handleIgnore = async () => {
    setShowConflictAlert(false);
    await executeCreate();
  };

  const handleAdjust = () => {
    setShowConflictAlert(false);
    // Keep parsedSchedule to allow editing
  };

  const handleOverwrite = () => {
    setShowConflictAlert(false);
    setShowOverwriteConfirm(true);
  };

  const executeOverwrite = async () => {
    setShowOverwriteConfirm(false);
    try {
      // Filter out the schedule being edited to prevent self-deletion
      const schedulesToDelete = isEditMode && parsedSchedule?.name
        ? conflicts.filter((c) => c.name !== parsedSchedule.name)
        : conflicts;

      for (const conflict of schedulesToDelete) {
        await deleteSchedule.mutateAsync(conflict.name);
      }
      await executeCreate();
    } catch (error) {
      toast.error("Failed to overwrite schedule");
    }
  };

  const handleDiscard = () => {
    handleClose();
  };

  const handleClose = () => {
    setInput("");
    setParsedSchedule(null);
    setConflicts([]);
    setShowConflictAlert(false);
    onOpenChange(false);
  };



  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <ScheduleErrorBoundary>
          <DialogContent className="max-w-md">
            <DialogTitle>{isEditMode ? t("schedule.edit-schedule") : t("schedule.create-schedule")}</DialogTitle>
            <DialogDescription>{isEditMode ? "" : t("schedule.natural-language-hint")}</DialogDescription>

            <div className="space-y-4 mt-4">
              {/* Natural Language Input - Only for create mode */}
              {!isEditMode && !parsedSchedule && (
                <div className="space-y-2">
                  <Label htmlFor="schedule-input">{t("schedule.description") || "Description"}</Label>
                  <Textarea
                    id="schedule-input"
                    placeholder='e.g., "明天下午3点开会"'
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        handleParse();
                      }
                    }}
                    className="min-h-24 resize-none"
                  />
                </div>
              )}

              {/* Parse Button - Only for create mode */}
              {!isEditMode && !parsedSchedule && (
                <Button onClick={handleParse} disabled={!input.trim() || isParsing} className="w-full">
                  {isParsing && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {t("schedule.create-schedule")}
                </Button>
              )}

              {/* Schedule Details Form */}
              {parsedSchedule && (
                <div className="space-y-3 rounded-lg border bg-muted/50 p-4">
                  <div className="flex items-center justify-between">
                    <h4 className="font-medium text-sm">
                      {isEditMode ? t("schedule.edit-schedule") : t("schedule.suggested-schedule") || "解析结果"}
                    </h4>
                    {/* Only show reset button in create mode */}
                    {!isEditMode && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setParsedSchedule(null);
                          setConflicts([]);
                        }}
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    )}
                  </div>

                  <div className="space-y-3 text-sm">
                    {/* Title */}
                    <div className="flex items-center gap-2">
                      <Calendar className="h-4 w-4 text-muted-foreground shrink-0" />
                      <Input
                        value={parsedSchedule.title}
                        onChange={(e) => setParsedSchedule({ ...parsedSchedule, title: e.target.value })}
                        className="h-8 font-medium"
                        placeholder={t("common.title")}
                      />
                    </div>

                    {/* Time */}
                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4 text-muted-foreground shrink-0" />
                      <div className="flex items-center gap-2 w-full">
                        <Input
                          type="datetime-local"
                          value={dayjs(timestampDate(create(TimestampSchema, { seconds: parsedSchedule.startTs, nanos: 0 }))).format("YYYY-MM-DDTHH:mm")}
                          onChange={(e) => {
                            const ts = BigInt(dayjs(e.target.value).unix());
                            setParsedSchedule({ ...parsedSchedule, startTs: ts });
                          }}
                          className="h-8 w-full"
                        />
                        <span className="text-muted-foreground">-</span>
                        <Input
                          type="datetime-local"
                          value={parsedSchedule.endTs > 0 ? dayjs(timestampDate(create(TimestampSchema, { seconds: parsedSchedule.endTs, nanos: 0 }))).format("YYYY-MM-DDTHH:mm") : ""}
                          onChange={(e) => {
                            const ts = BigInt(dayjs(e.target.value).unix());
                            setParsedSchedule({ ...parsedSchedule, endTs: ts });
                          }}
                          className="h-8 w-full"
                        />
                      </div>
                    </div>

                    {/* Location */}
                    <div className="flex items-center gap-2">
                      <MapPin className="h-4 w-4 text-muted-foreground shrink-0" />
                      <Input
                        value={parsedSchedule.location || ""}
                        onChange={(e) => setParsedSchedule({ ...parsedSchedule, location: e.target.value })}
                        className="h-8"
                        placeholder={t("common.location") || "Location"}
                      />
                    </div>

                    {/* Description - Always show in edit mode or when present */}
                    {(isEditMode || parsedSchedule.description) && (
                      <div className="pl-6">
                        <Textarea
                          value={parsedSchedule.description || ""}
                          onChange={(e) => setParsedSchedule({ ...parsedSchedule, description: e.target.value })}
                          className="min-h-[60px] text-xs resize-none"
                          placeholder={t("schedule.description")}
                        />
                      </div>
                    )}

                    {parsedSchedule.reminders.length > 0 && (
                      <div className="flex flex-wrap gap-1 pt-2 pl-6">
                        {parsedSchedule.reminders.map((reminder, idx) => (
                          <span key={idx} className="rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
                            {reminder.type === "before" && "提醒"}: {reminder.value} {reminder.unit}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex justify-end gap-2 pt-2">
                    <Button variant="outline" onClick={handleClose}>
                      {t("common.cancel")}
                    </Button>
                    <Button onClick={handleCreate}>
                      {isEditMode ? t("common.save") : t("schedule.create-schedule")}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </DialogContent>
        </ScheduleErrorBoundary>
      </Dialog>

      {/* Conflict Alert */}
      <ScheduleConflictAlert
        open={showConflictAlert}
        onOpenChange={setShowConflictAlert}
        conflicts={conflicts}
        onConfirm={handleOverwrite}
        onIgnore={handleIgnore}
        onAdjust={handleAdjust}
        onDiscard={handleDiscard}
      />

      <Dialog open={showOverwriteConfirm} onOpenChange={setShowOverwriteConfirm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("schedule.overwrite-confirm-title")}</DialogTitle>
            <DialogDescription>{t("schedule.overwrite-confirm-desc")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowOverwriteConfirm(false)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={executeOverwrite}>
              {t("schedule.overwrite")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
