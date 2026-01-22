import { FileText, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { MiniMemoCard } from "./MiniMemoCard";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import { useState } from "react";

interface ReferencedMemosPanelProps {
  className?: string;
}

export function ReferencedMemosPanel({ className }: ReferencedMemosPanelProps) {
  const { t } = useTranslation();
  const { currentConversation } = useAIChat();
  const [searchQuery, setSearchQuery] = useState("");

  const referencedMemos = currentConversation?.referencedMemos || [];

  // Filter memos by search query
  const filteredMemos = referencedMemos.filter((memo) =>
    memo.content.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Sort by score (descending)
  const sortedMemos = [...filteredMemos].sort((a, b) => b.score - a.score);

  const hasMemos = sortedMemos.length > 0;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Header */}
      <div className="px-3 py-2 border-b border-zinc-200 dark:border-zinc-700">
        <div className="flex items-center gap-2">
          <FileText className="w-4 h-4 text-zinc-500" />
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
            {t("ai.aichat.sidebar.memos")}
          </h2>
        </div>
        <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5 ml-6">
          {referencedMemos.length} {referencedMemos.length === 1 ? "memo" : "memos"}
        </p>
      </div>

      {/* Search */}
      {(hasMemos || searchQuery) && (
        <div className="p-2 border-b border-zinc-200 dark:border-zinc-700">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={t("ai.aichat.sidebar.search-placeholder")}
              className="w-full pl-8 pr-3 py-1.5 text-sm bg-zinc-100 dark:bg-zinc-800 border-0 rounded-md outline-none focus:ring-2 focus:ring-blue-500 text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400"
            />
          </div>
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {hasMemos ? (
          <div className="p-2 space-y-2">
            {sortedMemos.map((memo, index) => (
              <MiniMemoCard
                key={memo.uid}
                memo={memo}
                rank={index + 1}
                showRank={!!searchQuery}
              />
            ))}
          </div>
        ) : (
          <EmptyState searchQuery={searchQuery} hasMemos={referencedMemos.length > 0} />
        )}
      </div>
    </div>
  );
}

interface EmptyStateProps {
  searchQuery: string;
  hasMemos: boolean;
}

function EmptyState({ searchQuery, hasMemos }: EmptyStateProps) {
  const { t } = useTranslation();

  if (searchQuery) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6 text-center">
        <Search className="w-8 h-8 text-zinc-300 dark:text-zinc-600 mb-3" />
        <h3 className="text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-1">
          {t("ai.aichat.sidebar.no-results")}
        </h3>
        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          {t("ai.aichat.sidebar.try-different-search")}
        </p>
      </div>
    );
  }

  if (!hasMemos) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6 text-center">
        <FileText className="w-8 h-8 text-zinc-300 dark:text-zinc-600 mb-3" />
        <h3 className="text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-1">
          {t("ai.aichat.sidebar.no-referenced-memos")}
        </h3>
        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          {t("ai.aichat.sidebar.search-memos-hint")}
        </p>
      </div>
    );
  }

  return null;
}
