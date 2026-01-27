import dayjs from "dayjs";
import { AlertTriangle, Calendar, Clock, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import type { ConflictResolutionProps } from "./types";

// Check if auto-resolve is enabled (default: true)
const isAutoResolveEnabled = () => localStorage.getItem("schedule.autoResolve") !== "false";

export function ConflictResolution({ data, onAction, onDismiss, isLoading = false }: ConflictResolutionProps) {
  const t = useTranslate();
  const [selectedSlotIdx, setSelectedSlotIdx] = useState<number | null>(null);
  const [showManualSelection, setShowManualSelection] = useState(false);
  const autoResolveExecuted = useRef(false);

  // Auto-resolve when auto_resolved is available
  useEffect(() => {
    if (autoResolveExecuted.current || !isAutoResolveEnabled()) return;
    if (!data.auto_resolved) return;

    autoResolveExecuted.current = true;
    const autoSlot = data.auto_resolved;

    // Convert auto_resolved to UITimeSlotData format
    const slotData = {
      label: autoSlot.label,
      start_ts: autoSlot.start_ts,
      end_ts: autoSlot.end_ts,
      duration: autoSlot.end_ts - autoSlot.start_ts,
      reason: autoSlot.reason,
    };

    // Auto reschedule
    onAction("reschedule", slotData);

    // Show toast with undo option
    toast(
      (toastInstance) => (
        <div className="flex items-center gap-3">
          <span className="text-sm">
            {t("schedule.conflict.auto-resolved", { time: autoSlot.label }) || `已调整到 ${autoSlot.label}`}
          </span>
          <button
            type="button"
            onClick={() => {
              toast.dismiss(toastInstance.id);
              setShowManualSelection(true);
              // Don't reset autoResolveExecuted to prevent re-triggering auto-resolve
            }}
            className="text-xs font-medium text-primary hover:underline"
          >
            {t("common.undo") || "撤销"}
          </button>
        </div>
      ),
      { duration: 4000 },
    );
  }, [data.auto_resolved, onAction, t]);

  // Auto-select first slot when showing manual selection
  useEffect(() => {
    if (showManualSelection && selectedSlotIdx === null && data.suggested_slots?.length > 0) {
      setSelectedSlotIdx(0);
    }
  }, [showManualSelection, selectedSlotIdx, data.suggested_slots]);

  // If auto-resolved and not showing manual, don't render the panel
  if (data.auto_resolved && !showManualSelection && isAutoResolveEnabled()) {
    return null;
  }

  const handleOverride = () => {
    onAction("override");
  };

  const handleReschedule = () => {
    if (selectedSlotIdx !== null && data.suggested_slots?.[selectedSlotIdx]) {
      onAction("reschedule", data.suggested_slots[selectedSlotIdx]);
    }
  };

  const handleCancel = () => {
    onAction("cancel");
  };

  const conflictingSchedule = data.conflicting_schedules?.[0];
  const conflictCount = data.conflicting_schedules?.length ?? 0;
  const suggestedSlots = data.suggested_slots ?? [];

  return (
    <div className="bg-amber-500/10 rounded-xl border border-amber-500/20 p-4 animate-in fade-in slide-in-from-top-2 duration-300">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <AlertTriangle className="w-5 h-5 text-amber-500" />
          <h4 className="font-semibold text-amber-600 dark:text-amber-400">{t("schedule.conflict.title")}</h4>
        </div>
        <button
          type="button"
          onClick={onDismiss}
          disabled={isLoading}
          className={cn("p-1 rounded-md transition-colors", "hover:bg-amber-500/20", "disabled:opacity-50 disabled:cursor-not-allowed")}
        >
          <X className="w-4 h-4 text-muted-foreground" />
        </button>
      </div>

      {/* Conflicting schedule info */}
      {conflictingSchedule && (
        <div className="mb-3 p-2.5 bg-background/50 rounded-lg">
          <p className="text-xs text-muted-foreground mb-1">
            {conflictCount > 1
              ? t("schedule.conflict.multiple-conflicts", { count: conflictCount })
              : t("schedule.conflict.single-conflict")}
          </p>
          <div className="flex items-center gap-2 text-sm">
            <Calendar className="w-4 h-4 text-amber-500" />
            <span className="font-medium">{conflictingSchedule.title}</span>
            <Clock className="w-4 h-4 text-muted-foreground ml-2" />
            <span className="text-muted-foreground">
              {dayjs.unix(conflictingSchedule.start_time).format("HH:mm")} - {dayjs.unix(conflictingSchedule.end_time).format("HH:mm")}
            </span>
          </div>
        </div>
      )}

      {/* Suggested time slots */}
      {suggestedSlots.length > 0 && (
        <div className="mb-3">
          <p className="text-sm font-medium mb-2">{t("schedule.ai.alternatives")}</p>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
            {suggestedSlots.map((slot, idx) => {
              const isSelected = selectedSlotIdx === idx;
              return (
                <button
                  key={idx}
                  type="button"
                  onClick={() => setSelectedSlotIdx(idx)}
                  disabled={isLoading}
                  className={cn(
                    "p-2.5 rounded-lg border text-center transition-all text-sm",
                    "hover:bg-background",
                    isSelected ? "bg-primary text-primary-foreground border-primary" : "bg-background border-border",
                    "disabled:opacity-50 disabled:cursor-not-allowed",
                  )}
                >
                  {slot.label}
                </button>
              );
            })}
          </div>
        </div>
      )}

      {/* Action buttons */}
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={handleReschedule}
          disabled={isLoading || selectedSlotIdx === null}
          className={cn(
            "py-2 px-3 rounded-lg font-medium text-sm transition-colors",
            "bg-primary text-primary-foreground hover:bg-primary/90",
            "disabled:opacity-50 disabled:cursor-not-allowed",
          )}
        >
          {t("schedule.conflict.manual-resolve")}
        </button>
        <button
          type="button"
          onClick={handleOverride}
          disabled={isLoading}
          className={cn(
            "py-2 px-3 rounded-lg font-medium text-sm transition-colors",
            "bg-amber-500 text-white hover:bg-amber-600",
            "disabled:opacity-50 disabled:cursor-not-allowed",
          )}
        >
          {t("schedule.conflict.override")}
        </button>
        <button
          type="button"
          onClick={handleCancel}
          disabled={isLoading}
          className={cn(
            "py-2 px-3 rounded-lg font-medium text-sm transition-colors",
            "bg-muted text-muted-foreground hover:bg-muted/70",
            "disabled:opacity-50 disabled:cursor-not-allowed",
          )}
        >
          {t("common.cancel")}
        </button>
      </div>
    </div>
  );
}
