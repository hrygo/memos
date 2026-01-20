import { AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useTranslate } from "@/utils/i18n";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import dayjs from "dayjs";
import { timestampDate } from "@bufbuild/protobuf/wkt";

interface ScheduleConflictAlertProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  conflicts: Schedule[];
  onConfirm: () => void;
}

export const ScheduleConflictAlert = ({
  open,
  onOpenChange,
  conflicts,
  onConfirm,
}: ScheduleConflictAlertProps) => {
  const t = useTranslate();

  const formatTime = (ts: bigint) => {
    const date = timestampDate({ seconds: ts, nanos: 0 });
    return dayjs(date).format("HH:mm");
  };

  const formatDate = (ts: bigint) => {
    const date = timestampDate({ seconds: ts, nanos: 0 });
    return dayjs(date).format("YYYY-MM-DD");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <div className="flex items-start gap-3">
          <div className="rounded-full bg-destructive/10 p-2">
            <AlertCircle className="h-5 w-5 text-destructive" />
          </div>
          <div className="flex-1">
            <DialogTitle className="text-lg">{t("schedule.conflict-detected")}</DialogTitle>
            <DialogDescription className="mt-2 text-sm text-muted-foreground">
              {t("schedule.conflict-warning")}
            </DialogDescription>
          </div>
        </div>

        <ScrollArea className="mt-4 max-h-60">
          <div className="space-y-2 p-1">
            {conflicts.map((conflict) => (
              <div
                key={conflict.name}
                className="flex items-start gap-3 rounded-lg border border-destructive/20 bg-destructive/5 p-3"
              >
                <div className="mt-1">
                  <div className="h-2 w-2 rounded-full bg-destructive" />
                </div>
                <div className="flex-1 space-y-1">
                  <p className="font-medium text-sm">{conflict.title}</p>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span>
                      {formatDate(conflict.startTs)} {formatTime(conflict.startTs)}
                      {conflict.endTs > 0 && ` - ${formatTime(conflict.endTs)}`}
                    </span>
                    {conflict.location && <span>• {conflict.location}</span>}
                  </div>
                  {conflict.description && (
                    <p className="line-clamp-2 text-xs text-muted-foreground">{conflict.description}</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>

        <div className="mt-4 flex justify-end gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button variant="destructive" onClick={onConfirm}>
            {t("schedule.create-anyway")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};
