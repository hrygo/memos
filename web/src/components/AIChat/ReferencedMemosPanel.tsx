import { FileText, Search } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import { MiniMemoCard } from "./MiniMemoCard";

interface ReferencedMemosPanelProps {
  className?: string;
}

export function ReferencedMemosPanel({ className }: ReferencedMemosPanelProps) {
  const { t } = useTranslation();
  const { currentConversation } = useAIChat();
  const [searchQuery, setSearchQuery] = useState("");

  const referencedMemos = currentConversation?.referencedMemos || [];

  // Filter memos by search query
  const filteredMemos = referencedMemos.filter((memo) => memo.content.toLowerCase().includes(searchQuery.toLowerCase()));

  // Sort by score (descending)
  const sortedMemos = [...filteredMemos].sort((a, b) => b.score - a.score);

  const hasMemos = sortedMemos.length > 0;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Header */}
      <div className="px-3 py-2 border-b border-border">
        <div className="flex items-center gap-2">
          <FileText className="w-4 h-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold text-foreground">{t("ai.aichat.sidebar.memos")}</h2>
        </div>
        <p className="text-xs text-muted-foreground mt-0.5 ml-6">
          {referencedMemos.length} {referencedMemos.length === 1 ? "memo" : "memos"}
        </p>
      </div>

      {/* Search */}
      {(hasMemos || searchQuery) && (
        <div className="p-2 border-b border-border">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={t("ai.aichat.sidebar.search-placeholder")}
              className="w-full pl-8 pr-3 py-1.5 text-sm bg-muted border-0 rounded-md outline-none focus:ring-2 focus:ring-blue-500 text-foreground placeholder:text-muted-foreground"
            />
          </div>
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {hasMemos ? (
          <div className="p-2 space-y-2">
            {sortedMemos.map((memo, index) => (
              <MiniMemoCard key={memo.uid} memo={memo} rank={index + 1} showRank={!!searchQuery} />
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
        <Search className="w-8 h-8 text-muted-foreground/50 mb-3" />
        <h3 className="text-sm font-medium text-foreground mb-1">{t("ai.aichat.sidebar.no-results")}</h3>
        <p className="text-xs text-muted-foreground">{t("ai.aichat.sidebar.try-different-search")}</p>
      </div>
    );
  }

  if (!hasMemos) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6 text-center">
        <FileText className="w-8 h-8 text-muted-foreground/50 mb-3" />
        <h3 className="text-sm font-medium text-foreground mb-1">{t("ai.aichat.sidebar.no-referenced-memos")}</h3>
        <p className="text-xs text-muted-foreground">{t("ai.aichat.sidebar.search-memos-hint")}</p>
      </div>
    );
  }

  return null;
}
