import { AlertTriangle, Link2, Merge, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { SimilarMemo } from "@/types/proto/api/v1/ai_service_pb";

interface DuplicateWarningProps {
  duplicates: SimilarMemo[];
  related: SimilarMemo[];
  onMerge: (targetName: string) => void;
  onLink: (memoName: string) => void;
  onIgnore: () => void;
}

export function DuplicateWarning({ duplicates, related, onMerge, onLink, onIgnore }: DuplicateWarningProps) {
  const { t } = useTranslation();

  if (duplicates.length === 0 && related.length === 0) {
    return null;
  }

  return (
    <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-800 dark:bg-yellow-900/20">
      {duplicates.length > 0 && (
        <div className="mb-4">
          <h4 className="flex items-center gap-2 font-medium text-yellow-800 dark:text-yellow-200">
            <AlertTriangle className="h-4 w-4" />
            {t("duplicate.found-similar")}
          </h4>
          <div className="mt-2 space-y-2">
            {duplicates.map((memo) => (
              <div key={memo.id} className="flex items-center justify-between rounded bg-white p-2 dark:bg-zinc-800">
                <div className="flex-1 min-w-0">
                  <p className="font-medium truncate">{memo.title || t("duplicate.untitled")}</p>
                  <p className="text-sm text-gray-500 dark:text-gray-400 truncate">{memo.snippet}</p>
                  <p className="text-xs text-yellow-600 dark:text-yellow-400">
                    {t("duplicate.similarity")}: {Math.round(memo.similarity * 100)}%
                  </p>
                </div>
                <div className="flex gap-2 ml-2 shrink-0">
                  <button
                    onClick={() => onMerge(memo.name)}
                    className="flex items-center gap-1 px-2 py-1 text-sm bg-yellow-100 hover:bg-yellow-200 rounded dark:bg-yellow-800 dark:hover:bg-yellow-700"
                    title={t("duplicate.merge")}
                  >
                    <Merge className="h-3 w-3" />
                    {t("duplicate.merge")}
                  </button>
                  <button
                    onClick={() => onLink(memo.name)}
                    className="flex items-center gap-1 px-2 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded dark:bg-zinc-700 dark:hover:bg-zinc-600"
                    title={t("duplicate.link")}
                  >
                    <Link2 className="h-3 w-3" />
                    {t("duplicate.link")}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {related.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-gray-600 dark:text-gray-300">{t("duplicate.related-memos")}</h4>
          <div className="mt-2 flex flex-wrap gap-2">
            {related.map((memo) => (
              <button
                key={memo.id}
                onClick={() => onLink(memo.name)}
                className="rounded-full bg-gray-100 px-3 py-1 text-sm hover:bg-gray-200 dark:bg-zinc-700 dark:hover:bg-zinc-600"
                title={`${Math.round(memo.similarity * 100)}% ${t("duplicate.similarity")}`}
              >
                {memo.title || memo.snippet?.slice(0, 20) || t("duplicate.untitled")}
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="mt-4 flex justify-end">
        <button
          onClick={onIgnore}
          className="flex items-center gap-1 px-3 py-1 text-sm text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
        >
          <X className="h-3 w-3" />
          {t("duplicate.ignore")}
        </button>
      </div>
    </div>
  );
}

export default DuplicateWarning;
