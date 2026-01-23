import { FileTextIcon, LoaderIcon, SparklesIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { useRelatedMemos } from "@/hooks/useAIQueries";
import { cn } from "@/lib/utils";
import { RELATED_MEMO_CARD } from "@/components/ui/card/constants";

interface Props {
  memoName: string; // format: "memos/{id}"
  className?: string;
}

const MemoRelatedList = ({ memoName, className }: Props) => {
  const { t } = useTranslation();
  const { data: relatedMemos, isLoading } = useRelatedMemos(memoName);

  if (isLoading) {
    return (
      <div className={cn("w-full p-4 flex justify-center", className)}>
        <LoaderIcon className="w-4 h-4 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!relatedMemos?.memos || relatedMemos.memos.length === 0) {
    return null;
  }

  return (
    <div className={cn("w-full space-y-2 mt-4", className)}>
      <h3 className="text-sm font-medium text-muted-foreground flex items-center gap-1">
        <SparklesIcon className="w-3 h-3" />
        {t("common.related-memos") || "Related Memos"}
      </h3>
      <div className="flex flex-col gap-2">
        {relatedMemos.memos.map((memo) => {
          const id = memo.name.split("/").pop();
          return (
            <Link
              key={memo.name}
              to={`/m/${id}`}
              className={RELATED_MEMO_CARD}
            >
              <div className="flex items-center justify-between mb-1">
                <div className="flex items-center gap-1 text-xs text-muted-foreground">
                  <FileTextIcon className="w-3 h-3 text-muted-foreground" />
                  <span>Relevance: {(memo.score * 100).toFixed(0)}%</span>
                </div>
              </div>
              <p className="text-sm text-foreground line-clamp-2 leading-relaxed group-hover:text-blue-600 dark:group-hover:text-blue-400">
                {memo.snippet}
              </p>
            </Link>
          );
        })}
      </div>
    </div>
  );
};

export default MemoRelatedList;
