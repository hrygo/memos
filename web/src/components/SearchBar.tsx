import { FileTextIcon, LoaderIcon, SearchIcon, SparklesIcon } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMemoFilterContext } from "@/contexts/MemoFilterContext";
import { useSemanticSearch } from "@/hooks/useAIQueries";
import useNavigateTo from "@/hooks/useNavigateTo";
import { cn } from "@/lib/utils";
import MemoDisplaySettingMenu from "./MemoDisplaySettingMenu";

const SearchBar = () => {
  const { t } = useTranslation();
  const { addFilter } = useMemoFilterContext();
  const [queryText, setQueryText] = useState("");
  const [isSemantic, setIsSemantic] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const navigateTo = useNavigateTo();

  const { data: semanticResults, isLoading } = useSemanticSearch(queryText, {
    enabled: isSemantic && queryText.length > 1,
  });

  const onTextChange = (event: React.FormEvent<HTMLInputElement>) => {
    setQueryText(event.currentTarget.value);
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      if (!isSemantic) {
        const trimmedText = queryText.trim();
        if (trimmedText !== "") {
          const words = trimmedText.split(/\s+/);
          words.forEach((word) => {
            addFilter({
              factor: "contentSearch",
              value: word,
            });
          });
          setQueryText("");
        }
      }
    }
  };

  const onMemoClick = (memoId: string) => {
    // Extract ID from "memos/{id}"
    const id = memoId.split("/").pop();
    if (id) {
      navigateTo(`/m/${id}`);
      setQueryText("");
      setIsSemantic(false);
    }
  };

  return (
    <div className="relative w-full h-auto flex flex-col z-20">
      <div className="relative w-full flex flex-row justify-start items-center">
        <SearchIcon className="absolute left-2 w-4 h-auto opacity-40 text-sidebar-foreground" />
        <input
          className={cn(
            "w-full text-sidebar-foreground leading-6 bg-sidebar border border-border text-sm rounded-lg p-1 pl-8 pr-16 outline-0",
            isSemantic && "border-blue-400 ring-1 ring-blue-400",
          )}
          placeholder={isSemantic ? t("common.search") + " (AI)..." : t("memo.search-placeholder")}
          value={queryText}
          onChange={onTextChange}
          onKeyDown={onKeyDown}
          ref={inputRef}
        />
        <div className="absolute right-8 top-1 flex items-center">
          <button
            onClick={() => setIsSemantic(!isSemantic)}
            className={cn(
              "p-1 rounded-md transition-colors",
              isSemantic ? "text-blue-500 bg-blue-100 dark:bg-blue-900" : "text-muted-foreground hover:bg-muted",
            )}
            title="Toggle Semantic Search"
          >
            <SparklesIcon className="w-4 h-4" />
          </button>
        </div>
        <MemoDisplaySettingMenu className="absolute right-2 top-2 text-sidebar-foreground" />
      </div>

      {isSemantic && queryText.length > 1 && (
        <div className="absolute top-full mt-2 w-full bg-background border border-border rounded-lg shadow-xl overflow-hidden max-h-80 overflow-y-auto">
          {isLoading && (
            <div className="p-4 flex items-center justify-center text-muted-foreground">
              <LoaderIcon className="w-4 h-4 animate-spin mr-2" />
              Searching...
            </div>
          )}
          {!isLoading && semanticResults?.results?.length === 0 && (
            <div className="p-4 text-center text-muted-foreground text-sm">No relevant memos found.</div>
          )}
          {!isLoading &&
            semanticResults?.results?.map((result) => (
              <div
                key={result.name}
                className="p-3 border-b last:border-0 hover:bg-muted/50 cursor-pointer group"
                onClick={() => onMemoClick(result.name)}
              >
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center text-xs text-muted-foreground">
                    <FileTextIcon className="w-3 h-3 mr-1" />
                    Score: {(result.score * 100).toFixed(0)}%
                  </div>
                </div>
                <div className="text-sm line-clamp-2 text-foreground group-hover:text-blue-600 dark:group-hover:text-blue-400">
                  {result.snippet}
                </div>
              </div>
            ))}
        </div>
      )}
    </div>
  );
};

export default SearchBar;
