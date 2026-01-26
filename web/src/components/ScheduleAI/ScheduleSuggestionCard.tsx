import dayjs from "dayjs";
import { Calendar, Check, Clock, MapPin } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import { type Translations, useTranslate } from "@/utils/i18n";
import type { ScheduleSuggestionCardProps } from "./types";

export function ScheduleSuggestionCard({ data, onConfirm, isLoading = false }: ScheduleSuggestionCardProps) {
  const t = useTranslate();
  const [isCreating, setIsCreating] = useState(false);

  const startTime = dayjs.unix(data.start_ts).format("HH:mm");
  const endTime = dayjs.unix(data.end_ts).format("HH:mm");
  const dateStr = dayjs.unix(data.start_ts).format("YYYY-MM-DD");

  const handleClick = () => {
    if (isLoading || isCreating) return;
    setIsCreating(true);
    onConfirm(data);
  };

  // Get translations with fallback
  const clickToCreateText = t("schedule.suggestion.click-to-create" as Translations) || "Click to create";
  const creatingText = t("schedule.quick-input.creating" as Translations) || "Creating...";

  const showCreating = isLoading || isCreating;

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={showCreating}
      className={cn(
        "w-full text-left rounded-xl border p-4 transition-all duration-200",
        "animate-in fade-in slide-in-from-top-2",
        showCreating
          ? "bg-green-500/10 border-green-500/30 scale-[0.98]"
          : "bg-primary/10 border-primary/20 hover:bg-primary/15 hover:border-primary/30 hover:shadow-md cursor-pointer",
        "disabled:cursor-default",
      )}
    >
      <div className="flex items-start gap-3">
        <div
          className={cn(
            "flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center transition-colors duration-200",
            showCreating ? "bg-green-500/20" : "bg-primary/20",
          )}
        >
          {showCreating ? (
            <Check className="w-5 h-5 text-green-600 dark:text-green-400 animate-in zoom-in duration-200" />
          ) : (
            <Calendar className="w-5 h-5 text-primary" />
          )}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <h4 className="font-semibold text-foreground">{data.title}</h4>
            {!showCreating && (
              <span className="text-xs text-primary/70 hidden sm:inline">{clickToCreateText}</span>
            )}
            {showCreating && (
              <span className="text-xs text-green-600 dark:text-green-400">{creatingText}</span>
            )}
          </div>

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

          {data.reason && <p className="mt-2 text-xs text-muted-foreground">{data.reason}</p>}
        </div>
      </div>
    </button>
  );
}
