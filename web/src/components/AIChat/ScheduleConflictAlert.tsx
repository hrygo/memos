import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { AlertCircle, Calendar, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

interface ScheduleConflictAlertProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  conflicts: Schedule[];
  onConfirm: () => void;
  onIgnore: () => void;
  onAdjust: () => void;
  onDiscard: () => void;
}

export const ScheduleConflictAlert = ({ open, onOpenChange, conflicts, onConfirm, onIgnore, onAdjust, onDiscard }: ScheduleConflictAlertProps) => {
  const t = useTranslate();

  const formatTime = (ts: bigint) => {
    const date = timestampDate(create(TimestampSchema, { seconds: ts, nanos: 0 }));
    return dayjs(date).format("HH:mm");
  };

  const formatDate = (ts: bigint) => {
    const date = timestampDate(create(TimestampSchema, { seconds: ts, nanos: 0 }));
    return dayjs(date).format("YYYY-MM-DD");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md sm:max-w-lg">
        {/* Header */}
        <div className="flex items-start gap-3">
          <div className="rounded-full bg-orange-100 dark:bg-orange-900/30 p-2.5">
            <AlertCircle className="h-5 w-5 text-orange-600 dark:text-orange-400" />
          </div>
          <div className="flex-1">
            <DialogTitle className="text-lg">时间冲突</DialogTitle>
            <DialogDescription className="mt-1 text-sm">
              该时间段与 {conflicts.length} 个现有日程冲突，是否继续创建？
            </DialogDescription>
          </div>
        </div>

        {/* Conflict List */}
        <div className="mt-4 max-h-60 overflow-y-auto">
          <div className="space-y-2 p-1">
            {conflicts.map((conflict) => (
              <div key={conflict.name} className="flex items-start gap-3 rounded-lg border border-orange-200 dark:border-orange-800 bg-orange-50/50 dark:bg-orange-950/20 p-3">
                <div className="mt-1">
                  <Calendar className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                </div>
                <div className="flex-1 space-y-1">
                  <p className="font-medium text-sm text-foreground">{conflict.title}</p>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="font-medium text-orange-700 dark:text-orange-300">
                      {formatDate(conflict.startTs)} {formatTime(conflict.startTs)}
                      {conflict.endTs > 0 && ` - ${formatTime(conflict.endTs)}`}
                    </span>
                    {conflict.location && <span>• {conflict.location}</span>}
                  </div>
                  {conflict.description && <p className="line-clamp-2 text-xs text-muted-foreground">{conflict.description}</p>}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Action Buttons */}
        <div className="mt-6 flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2 gap-2">
          {/* Cancel - Secondary */}
          <Button
            variant="outline"
            onClick={onDiscard}
            className="w-full sm:w-auto"
          >
            <X className="h-4 w-4 mr-2" />
            取消创建
          </Button>

          {/* Force Create - Destructive Action */}
          <Button
            variant="default"
            onClick={onIgnore}
            className="w-full sm:w-auto bg-orange-600 hover:bg-orange-700 text-white"
          >
            <Calendar className="h-4 w-4 mr-2" />
            仍然创建
          </Button>
        </div>

        {/* Hint Text */}
        <p className="mt-3 text-xs text-center text-muted-foreground">
          提示：选择"仍然创建"可忽略冲突继续创建，或选择"取消创建"返回调整时间
        </p>
      </DialogContent>
    </Dialog>
  );
};
