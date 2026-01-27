import { FileTextIcon } from "lucide-react";
import React from "react";
import { useTranslation } from "react-i18next";
import useNavigateTo from "@/hooks/useNavigateTo";
import { cn } from "@/lib/utils";

// Highlight represents a highlighted match position
export interface Highlight {
  start: number;
  end: number;
  matchedText: string;
}

// HighlightedMemo represents a search result with highlights
export interface HighlightedMemo {
  name: string;
  snippet: string;
  score: number;
  highlights: Highlight[];
  createdTs: number;
}

interface HighlightedResultProps {
  memo: HighlightedMemo;
  onClick?: () => void;
}

/**
 * HighlightedResult renders a search result with keyword highlighting.
 * P1-C001: Search Result Highlighting
 */
export function HighlightedResult({ memo, onClick }: HighlightedResultProps) {
  const { t } = useTranslation();
  const navigateTo = useNavigateTo();

  const handleClick = () => {
    if (onClick) {
      onClick();
    } else {
      const id = memo.name.split("/").pop();
      if (id) {
        navigateTo(`/m/${id}`);
      }
    }
  };

  const renderHighlightedSnippet = () => {
    const { snippet, highlights } = memo;

    if (!highlights?.length) {
      return <span>{snippet}</span>;
    }

    // Convert to code points array for proper Unicode handling (emoji, CJK, etc.)
    // Go returns indices in Unicode code points, JS string.slice uses UTF-16 code units
    const codePoints = Array.from(snippet);

    const parts: React.ReactNode[] = [];
    let lastEnd = 0;

    // Sort highlights by start position
    const sortedHighlights = [...highlights].sort((a, b) => a.start - b.start);

    sortedHighlights.forEach((h, i) => {
      // Add text before this highlight
      if (h.start > lastEnd) {
        parts.push(<span key={`text-${i}`}>{codePoints.slice(lastEnd, h.start).join("")}</span>);
      }

      // Add highlighted text (use matchedText directly as it's already extracted correctly)
      parts.push(
        <mark
          key={`mark-${i}`}
          className={cn("bg-yellow-200 dark:bg-yellow-700/60", "text-yellow-900 dark:text-yellow-100", "rounded px-0.5 font-medium")}
        >
          {h.matchedText}
        </mark>,
      );

      lastEnd = h.end;
    });

    // Add remaining text after last highlight
    if (lastEnd < codePoints.length) {
      parts.push(<span key="text-last">{codePoints.slice(lastEnd).join("")}</span>);
    }

    return <>{parts}</>;
  };

  const formatRelativeTime = (timestamp: number) => {
    const now = Date.now() / 1000;
    const diff = now - timestamp;

    if (diff < 60) return t("ai.aichat.sidebar.time-just-now");
    if (diff < 3600) return t("ai.aichat.sidebar.time-minutes-ago", { count: Math.floor(diff / 60) });
    if (diff < 86400) return t("ai.aichat.sidebar.time-hours-ago", { count: Math.floor(diff / 3600) });
    if (diff < 604800) return t("ai.aichat.sidebar.time-days-ago", { count: Math.floor(diff / 86400) });

    return new Date(timestamp * 1000).toLocaleDateString();
  };

  return (
    <div className={cn("p-3 border-b last:border-0 cursor-pointer group", "hover:bg-muted/50 transition-colors")} onClick={handleClick}>
      <div className="flex items-center justify-between mb-1.5">
        <div className="flex items-center text-xs text-muted-foreground">
          <FileTextIcon className="w-3 h-3 mr-1.5" />
          <span>{formatRelativeTime(memo.createdTs)}</span>
        </div>
        <div className="text-xs text-muted-foreground">
          {t("search.score")}: {(memo.score * 100).toFixed(0)}%
        </div>
      </div>
      <div
        className={cn(
          "text-sm leading-relaxed",
          "text-foreground",
          "group-hover:text-blue-600 dark:group-hover:text-blue-400",
          "line-clamp-3",
        )}
      >
        {renderHighlightedSnippet()}
      </div>
    </div>
  );
}

export default HighlightedResult;
