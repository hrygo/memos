import { Clock, X } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import { type Translations, useTranslate } from "@/utils/i18n";
import type { TimeSlotPickerProps } from "./types";

export function TimeSlotPicker({ data, onSelect, onDismiss, isLoading = false }: TimeSlotPickerProps) {
  const t = useTranslate();
  const [selectedIdx, setSelectedIdx] = useState(data.default_idx ?? 0);

  const handleSelect = () => {
    onSelect(data.slots[selectedIdx]);
  };

  // Get translations with fallback
  const selectTimeText = t("schedule.ai.select-time" as Translations) || "Select a time";
  const confirmSlotText = t("schedule.ai.confirm-slot" as Translations) || "Confirm";

  return (
    <div className="bg-muted/50 rounded-xl border border-border p-4 animate-in fade-in slide-in-from-top-2 duration-300">
      <div className="flex items-center justify-between mb-3">
        <h4 className="font-semibold text-foreground flex items-center gap-2">
          <Clock className="w-4 h-4 text-primary" />
          {selectTimeText}
        </h4>
        <button
          type="button"
          onClick={onDismiss}
          disabled={isLoading}
          className={cn(
            "p-1 rounded-md transition-colors",
            "hover:bg-muted-foreground/20",
            "disabled:opacity-50 disabled:cursor-not-allowed",
          )}
        >
          <X className="w-4 h-4 text-muted-foreground" />
        </button>
      </div>

      {data.reason && <p className="text-sm text-muted-foreground mb-3">{data.reason}</p>}

      <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 mb-3">
        {data.slots.map((slot, idx) => {
          const isSelected = idx === selectedIdx;
          return (
            <button
              key={idx}
              type="button"
              onClick={() => setSelectedIdx(idx)}
              disabled={isLoading}
              className={cn(
                "p-3 rounded-lg border text-center transition-all",
                "hover:bg-background",
                isSelected ? "bg-primary text-primary-foreground border-primary" : "bg-background border-border",
                "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-background",
              )}
            >
              <div className="font-medium text-sm">{slot.label}</div>
              {slot.reason && <div className="text-xs opacity-70 mt-1">{slot.reason}</div>}
            </button>
          );
        })}
      </div>

      <button
        type="button"
        onClick={handleSelect}
        disabled={isLoading}
        className={cn(
          "w-full py-2 px-4 rounded-lg font-medium text-sm transition-colors",
          "bg-primary text-primary-foreground hover:bg-primary/90",
          "disabled:opacity-50 disabled:cursor-not-allowed",
        )}
      >
        {confirmSlotText}
      </button>
    </div>
  );
}
