import { Calendar, Clock, MapPin } from "lucide-react";
import { useTranslate } from "@/utils/i18n";
import { cn } from "@/lib/utils";
import type { ScheduleSuggestionCardProps } from "./types";
import dayjs from "dayjs";

export function ScheduleSuggestionCard({
  data,
  onConfirm,
  onReject,
  isLoading = false,
}: ScheduleSuggestionCardProps) {
  const t = useTranslate();

  const startTime = dayjs.unix(data.start_ts).format("HH:mm");
  const endTime = dayjs.unix(data.end_ts).format("HH:mm");
  const dateStr = dayjs.unix(data.start_ts).format("YYYY-MM-DD");

  const handleConfirm = () => {
    onConfirm(data);
  };

  return (
    <div className="bg-primary/10 rounded-xl border border-primary/20 p-4 animate-in fade-in slide-in-from-top-2 duration-300">
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0 w-10 h-10 rounded-full bg-primary/20 flex items-center justify-center">
          <Calendar className="w-5 h-5 text-primary" />
        </div>
        <div className="flex-1 min-w-0">
          <h4 className="font-semibold text-foreground">{data.title}</h4>

          <div className="flex flex-wrap items-center gap-3 mt-2 text-sm text-muted-foreground">
            <div className="flex items-center gap-1.5">
              <Clock className="w-4 h-4" />
              <span>
                {dateStr} {startTime} - {endTime}
              </span>
            </div>
            {data.location && (
              <div className="flex items-center gap-1.5">
                <MapPin className="w-4 h-4" />
                <span>{data.location}</span>
              </div>
            )}
          </div>

          {data.reason && (
            <p className="mt-2 text-xs text-muted-foreground">{data.reason}</p>
          )}
        </div>
      </div>

      <div className="flex gap-2 mt-4">
        <button
          type="button"
          onClick={handleConfirm}
          disabled={isLoading}
          className={cn(
            "flex-1 py-2 px-4 rounded-lg font-medium text-sm transition-colors",
            "bg-primary text-primary-foreground hover:bg-primary/90",
            "disabled:opacity-50 disabled:cursor-not-allowed"
          )}
        >
          {isLoading ? t("schedule.quick-input.creating") : t("schedule.quick-input.confirm-create")}
        </button>
        <button
          type="button"
          onClick={onReject}
          disabled={isLoading}
          className={cn(
            "py-2 px-4 rounded-lg font-medium text-sm transition-colors",
            "bg-muted text-muted-foreground hover:bg-muted/70",
            "disabled:opacity-50 disabled:cursor-not-allowed"
          )}
        >
          {t("common.cancel")}
        </button>
      </div>
    </div>
  );
}
