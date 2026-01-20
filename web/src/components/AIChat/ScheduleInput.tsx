import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { Calendar, Clock, Loader2, MapPin, X } from "lucide-react";
import { useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useCheckConflict, useCreateSchedule, useParseAndCreateSchedule } from "@/hooks/useScheduleQueries";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";
import { ScheduleConflictAlert } from "./ScheduleConflictAlert";
import { ScheduleErrorBoundary } from "./ScheduleErrorBoundary";

interface ScheduleInputProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialText?: string;
  onSuccess?: (schedule: Schedule) => void;
}

export const ScheduleInput = ({ open, onOpenChange, initialText = "", onSuccess }: ScheduleInputProps) => {
  const t = useTranslate();
  const parseAndCreate = useParseAndCreateSchedule();
  const createSchedule = useCreateSchedule();
  const checkConflict = useCheckConflict();

  const [input, setInput] = useState(initialText);
  const [parsedSchedule, setParsedSchedule] = useState<Schedule | null>(null);
  const [isParsing, setIsParsing] = useState(false);
  const [conflicts, setConflicts] = useState<Schedule[]>([]);
  const [showConflictAlert, setShowConflictAlert] = useState(false);

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
        setParsedSchedule(result.parsedSchedule);

        // Check for conflicts
        // Default to 1 hour duration if endTs is not specified or is 0
        const endTs = result.parsedSchedule.endTs > 0 ? result.parsedSchedule.endTs : result.parsedSchedule.startTs + BigInt(3600);

        const conflictResult = await checkConflict.mutateAsync({
          startTs: result.parsedSchedule.startTs,
          endTs: endTs,
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

  const handleCreate = async () => {
    if (!parsedSchedule) return;

    try {
      // Ensure valid name format required by backend
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
    } catch (error) {
      toast.error("Failed to create schedule");
      console.error("Create error:", error);
    }
  };

  const handleClose = () => {
    setInput("");
    setParsedSchedule(null);
    setConflicts([]);
    setShowConflictAlert(false);
    onOpenChange(false);
  };

  const formatDateTime = (ts: bigint) => {
    // Manually construct Timestamp message to avoid object literal type errors
    // or cast to any if necessary, but ideally use create(TimestampSchema)
    // However, timestampDate accepts Timestamp | PlainMessage<Timestamp> in newer versions
    // but strict type checking flags missing $typeName.
    // Safe approach: create(TimestampSchema, { ... })
    const date = timestampDate(create(TimestampSchema, { seconds: ts, nanos: 0 }));
    return dayjs(date).format("YYYY-MM-DD HH:mm");
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <ScheduleErrorBoundary>
          <DialogContent className="max-w-md">
            <DialogTitle>{t("schedule.create-schedule")}</DialogTitle>
            <DialogDescription>{t("schedule.natural-language-hint")}</DialogDescription>

            <div className="space-y-4 mt-4">
              {/* Input */}
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

              {/* Parse Button */}
              {!parsedSchedule && (
                <Button onClick={handleParse} disabled={!input.trim() || isParsing} className="w-full">
                  {isParsing && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {t("schedule.create-schedule")}
                </Button>
              )}

              {/* Parsed Result */}
              {parsedSchedule && (
                <div className="space-y-3 rounded-lg border bg-muted/50 p-4">
                  <div className="flex items-center justify-between">
                    <h4 className="font-medium text-sm">解析结果</h4>
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
                  </div>

                  <div className="space-y-2 text-sm">
                    <div className="flex items-start gap-2">
                      <Calendar className="h-4 w-4 mt-0.5 text-muted-foreground" />
                      <div>
                        <div className="font-medium">{parsedSchedule.title}</div>
                        {parsedSchedule.description && <div className="text-xs text-muted-foreground">{parsedSchedule.description}</div>}
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4 text-muted-foreground" />
                      <div>
                        {formatDateTime(parsedSchedule.startTs)}
                        {parsedSchedule.endTs > 0 && ` - ${formatDateTime(parsedSchedule.endTs)}`}
                        {parsedSchedule.allDay && ` (${t("schedule.all-day")})`}
                      </div>
                    </div>

                    {parsedSchedule.location && (
                      <div className="flex items-center gap-2">
                        <MapPin className="h-4 w-4 text-muted-foreground" />
                        <div>{parsedSchedule.location}</div>
                      </div>
                    )}

                    {parsedSchedule.reminders.length > 0 && (
                      <div className="flex flex-wrap gap-1 pt-2">
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
                    <Button onClick={handleCreate}>{t("schedule.create-schedule")}</Button>
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
        onConfirm={() => {
          setShowConflictAlert(false);
          handleCreate();
        }}
      />
    </>
  );
};
