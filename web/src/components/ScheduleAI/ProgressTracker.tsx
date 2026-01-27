import { AlertCircle, ArrowRight, Check, Clock, LoaderCircle, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { type Translations, useTranslate } from "@/utils/i18n";
import type { ProgressTrackerProps } from "./types";
import type { ProgressStep } from "@/hooks/useScheduleAgent";

export function ProgressTracker({ data, onCancel, onDismiss }: ProgressTrackerProps) {
  const t = useTranslate();

  // Get translations with fallback
  const processingText = t("progress.tracker.title" as Translations) || data.title || "Processing";
  const cancelText = t("progress.tracker.cancel" as Translations) || "Cancel";
  const statusTexts = {
    pending: t("progress.tracker.steps.pending" as Translations) || "Pending",
    in_progress: t("progress.tracker.steps.in_progress" as Translations) || "Processing",
    completed: t("progress.tracker.steps.completed" as Translations) || "Completed",
    failed: t("progress.tracker.steps.failed" as Translations) || "Failed",
  };

  const completedCount = data.steps.filter((s) => s.status === "completed").length;
  const totalCount = data.steps.length;
  const progress = totalCount > 0 ? (completedCount / totalCount) * 100 : 0;

  return (
    <div
      className={cn(
        "rounded-xl border p-4 transition-all duration-200",
        "animate-in fade-in slide-in-from-top-2",
        "bg-muted/50 border-border",
      )}
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
            <LoaderCircle className="w-4 h-4 text-primary animate-spin" />
          </div>
          <div>
            <h4 className="font-semibold text-foreground">{processingText}</h4>
            <p className="text-xs text-muted-foreground">
              {completedCount} / {totalCount} {statusTexts.completed}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {data.can_cancel && onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              {cancelText}
            </button>
          )}
          {onDismiss && (
            <button
              type="button"
              onClick={onDismiss}
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>

      {/* Progress bar */}
      <div className="mb-4">
        <div className="h-1.5 bg-muted rounded-full overflow-hidden">
          <div
            className="h-full bg-primary rounded-full transition-all duration-300"
            style={{ width: `${progress}%` }}
          />
        </div>
      </div>

      {/* Steps */}
      <div className="space-y-2">
        {data.steps.map((step) => (
          <ProgressStep key={step.id} step={step} statusTexts={statusTexts} />
        ))}
      </div>
    </div>
  );
}

interface ProgressStepProps {
  step: ProgressStep;
  statusTexts: Record<string, string>;
}

function ProgressStep({ step, statusTexts }: ProgressStepProps) {
  return (
    <div
      className={cn(
        "flex items-start gap-3 p-2 rounded-lg transition-colors",
        step.status === "in_progress" && "bg-primary/10",
        step.status === "failed" && "bg-destructive/10",
      )}
    >
      <StepIcon status={step.status} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between">
          <span
            className={cn(
              "text-sm font-medium",
              step.status === "completed" && "text-foreground line-through",
              step.status === "in_progress" && "text-foreground",
              step.status === "failed" && "text-destructive",
              step.status === "pending" && "text-muted-foreground",
            )}
          >
            {step.label}
          </span>
          {step.status === "in_progress" && (
            <span className="text-xs text-muted-foreground">{statusTexts.in_progress}</span>
          )}
        </div>
        {step.error && (
          <p className="text-xs text-destructive mt-1">{step.error}</p>
        )}
      </div>
      {step.status === "in_progress" && (
        <ArrowRight className="w-4 h-4 text-primary animate-pulse" />
      )}
    </div>
  );
}

interface StepIconProps {
  status: ProgressStep["status"];
}

function StepIcon({ status }: StepIconProps) {
  switch (status) {
    case "completed":
      return (
        <div className="w-5 h-5 rounded-full bg-green-500/20 flex items-center justify-center flex-shrink-0">
          <Check className="w-3 h-3 text-green-600 dark:text-green-400" />
        </div>
      );
    case "in_progress":
      return (
        <div className="w-5 h-5 rounded-full bg-primary/20 flex items-center justify-center flex-shrink-0">
          <LoaderCircle className="w-3 h-3 text-primary animate-spin" />
        </div>
      );
    case "failed":
      return (
        <div className="w-5 h-5 rounded-full bg-destructive/20 flex items-center justify-center flex-shrink-0">
          <AlertCircle className="w-3 h-3 text-destructive" />
        </div>
      );
    case "pending":
    default:
      return (
        <div className="w-5 h-5 rounded-full bg-muted flex items-center justify-center flex-shrink-0">
          <Clock className="w-3 h-3 text-muted-foreground" />
        </div>
      );
  }
}
