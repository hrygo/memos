import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";
import { ReferencedMemo } from "@/types/aichat";

interface MiniMemoCardProps {
  memo: ReferencedMemo;
  rank?: number;
  showRank?: boolean;
  className?: string;
}

export function MiniMemoCard({ memo, rank, showRank = true, className }: MiniMemoCardProps) {
  const scorePercentage = Math.round(memo.score * 100);
  const scoreColor = getScoreColor(memo.score);

  return (
    <Link
      to={`/memos/${memo.uid}`}
      className={cn(
        "block p-2.5 rounded-lg border hover:bg-zinc-50 dark:hover:bg-zinc-700/50 transition-colors group",
        "bg-white dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700",
        className
      )}
    >
      <div className="flex items-start gap-2">
        {showRank && rank !== undefined && (
          <div
            className={cn(
              "flex-shrink-0 w-5 h-5 rounded flex items-center justify-center text-xs font-medium",
              rank <= 3
                ? "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300"
                : "bg-zinc-100 dark:bg-zinc-700 text-zinc-500 dark:text-zinc-400"
            )}
          >
            {rank}
          </div>
        )}

        <div className="flex-1 min-w-0">
          <p className="text-xs text-zinc-700 dark:text-zinc-300 line-clamp-3 leading-relaxed">
            {memo.content}
          </p>
          <div className="flex items-center justify-between mt-1.5">
            <span className="text-xs text-zinc-400 dark:text-zinc-500">
              {memo.timestamp ? formatTime(memo.timestamp) : ""}
            </span>
            <span className={cn("text-xs px-1.5 py-0.5 rounded", scoreColor)}>
              {scorePercentage}%
            </span>
          </div>
        </div>
      </div>
    </Link>
  );
}

function getScoreColor(score: number): string {
  if (score >= 0.9) {
    return "bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300";
  } else if (score >= 0.7) {
    return "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300";
  } else if (score >= 0.5) {
    return "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300";
  } else {
    return "bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-400";
  }
}

function formatTime(timestamp: number): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
}
