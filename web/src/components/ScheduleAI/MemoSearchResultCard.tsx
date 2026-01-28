import { ExternalLink, FileText, Hash, X } from "lucide-react";
import { memo } from "react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

import type { MemoSearchResultCardProps } from "./types";

// Confidence thresholds for visual feedback
const CONFIDENCE_HIGH = 0.8;
const CONFIDENCE_MEDIUM = 0.6;

export const MemoSearchResultCard = memo(function MemoSearchResultCard({ data, onDismiss }: MemoSearchResultCardProps) {
  const t = useTranslate();

  const contentText = t("memo.search.content");
  const tagsText = t("memo.search.tags");
  const confidenceText = t("memo.search.confidence");
  const viewMemoText = t("memo.search.view-memo");

  const confidencePercent = Math.round(data.confidence * 100);

  // Build memo URL - format is /m/{uid}
  const memoUrl = `/m/${data.uid}`;

  return (
    <div
      className={cn(
        "rounded-xl border p-4 transition-all duration-200",
        "animate-in fade-in slide-in-from-top-2",
        "bg-muted/30 border-border hover:bg-muted/50",
        "relative group",
      )}
    >
      {/* Dismiss button */}
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className={cn(
            "absolute top-2 right-2 p-1 rounded-md",
            "text-muted-foreground hover:text-foreground",
            "hover:bg-muted transition-colors",
            "opacity-0 group-hover:opacity-100 focus:opacity-100",
          )}
          aria-label="Dismiss"
        >
          <X className="w-4 h-4" />
        </button>
      )}
      <div className="space-y-3 pr-6">
        {/* Header with title and link */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
              <FileText className="w-4 h-4 text-primary" />
            </div>
            <div>
              <h5 className="font-medium text-base text-foreground">{data.title}</h5>
              {data.reason && (
                <p className="text-xs text-muted-foreground mt-0.5">{data.reason}</p>
              )}
            </div>
          </div>
          <a
            href={memoUrl}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-primary hover:bg-primary/10 transition-colors"
          >
            <ExternalLink className="w-3.5 h-3.5" />
            {viewMemoText}
          </a>
        </div>

        {/* Content */}
        <div>
          <label className="text-xs font-medium text-muted-foreground flex items-center gap-1">
            <FileText className="w-3 h-3" />
            {contentText}
          </label>
          <p className="text-sm text-foreground mt-1 whitespace-pre-wrap line-clamp-4">{data.content}</p>
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
                confidencePercent >= CONFIDENCE_HIGH * 100
                  ? "bg-green-500"
                  : confidencePercent >= CONFIDENCE_MEDIUM * 100
                    ? "bg-yellow-500"
                    : "bg-orange-500",
              )}
              style={{ width: `${confidencePercent}%` }}
            />
          </div>
          <span className="text-xs font-medium text-muted-foreground">{confidencePercent}%</span>
        </div>
      </div>
    </div>
  );
});
