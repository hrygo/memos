import { Check, FileText, Hash, X } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import type { MemoPreviewProps } from "./types";

// Confidence thresholds for visual feedback
const CONFIDENCE_HIGH = 0.8;
const CONFIDENCE_MEDIUM = 0.6;

export function MemoPreview({ data, onConfirm, onDismiss, isLoading = false }: MemoPreviewProps) {
  const t = useTranslate();
  const [isCreating, setIsCreating] = useState(false);

  const handleClick = () => {
    if (isLoading || isCreating) return;
    setIsCreating(true);
    onConfirm(data);
  };

  // Get translations (keys must exist in i18n files)
  const createText = t("memo.preview.confirm");
  const cancelText = t("memo.preview.cancel");
  const creatingText = t("schedule.quick-input.creating");
  const titleText = t("memo.preview.title");
  const contentText = t("memo.preview.content");
  const tagsText = t("memo.preview.tags");
  const confidenceText = t("memo.preview.confidence");

  const showCreating = isLoading || isCreating;
  const confidencePercent = Math.round(data.confidence * 100);

  return (
    <div
      className={cn(
        "rounded-xl border p-4 transition-all duration-200",
        "animate-in fade-in slide-in-from-top-2",
        "bg-muted/50 border-border",
      )}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
            <FileText className="w-4 h-4 text-primary" />
          </div>
          <h4 className="font-semibold text-foreground">{titleText}</h4>
        </div>
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

      <div className="space-y-3">
        {/* Title */}
        <div>
          <h5 className="font-medium text-base text-foreground mb-1">{data.title}</h5>
          {data.reason && (
            <p className="text-xs text-muted-foreground">{data.reason}</p>
          )}
        </div>

        {/* Content */}
        <div>
          <label className="text-xs font-medium text-muted-foreground flex items-center gap-1">
            <FileText className="w-3 h-3" />
            {contentText}
          </label>
          <p className="text-sm text-foreground mt-1 whitespace-pre-wrap">{data.content}</p>
        </div>

        {/* Tags */}
        {data.tags && data.tags.length > 0 && (
          <div>
            <label className="text-xs font-medium text-muted-foreground flex items-center gap-1">
              <Hash className="w-3 h-3" />
              {tagsText}
            </label>
            <div className="flex flex-wrap gap-1.5 mt-1.5">
              {data.tags.map((tag, idx) => (
                <span
                  key={idx}
                  className="inline-flex items-center px-2 py-0.5 rounded-md text-xs bg-primary/10 text-primary"
                >
                  #{tag}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Confidence */}
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">{confidenceText}:</span>
          <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
            <div
              className={cn(
                "h-full rounded-full transition-all duration-300",
                confidencePercent >= CONFIDENCE_HIGH * 100 ? "bg-green-500" : confidencePercent >= CONFIDENCE_MEDIUM * 100 ? "bg-yellow-500" : "bg-orange-500",
              )}
              style={{ width: `${confidencePercent}%` }}
            />
          </div>
          <span className="text-xs font-medium text-muted-foreground">{confidencePercent}%</span>
        </div>

        {/* Actions */}
        <div className="flex gap-2 pt-2">
          <button
            type="button"
            onClick={handleClick}
            disabled={showCreating}
            className={cn(
              "flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors",
              showCreating
                ? "bg-green-500/20 text-green-600 dark:text-green-400 cursor-default"
                : "bg-primary text-primary-foreground hover:bg-primary/90",
              "disabled:cursor-default",
            )}
          >
            {showCreating ? (
              <>
                <Check className="w-4 h-4" />
                {creatingText}
              </>
            ) : (
              createText
            )}
          </button>
          {onDismiss && !showCreating && (
            <button
              type="button"
              onClick={onDismiss}
              className="px-4 py-2 rounded-lg text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            >
              {cancelText}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
