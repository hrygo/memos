import { SearchIcon, XIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

interface ScheduleSearchBarProps {
  schedules: Schedule[];
  onFilteredChange?: (filtered: Schedule[]) => void;
  onHasFilterChange?: (hasFilter: boolean) => void;
  className?: string;
}

/** Search and filter schedules by title, location, and description */
export const ScheduleSearchBar = ({ schedules, onFilteredChange, onHasFilterChange, className }: ScheduleSearchBarProps) => {
  const t = useTranslate();
  const [searchQuery, setSearchQuery] = useState("");

  // Filter schedules based on search query
  const filteredSchedules = useMemo(() => {
    if (!searchQuery.trim()) {
      return schedules;
    }
    const query = searchQuery.toLowerCase().trim();
    return schedules.filter((schedule) => {
      const title = schedule.title.toLowerCase();
      const location = (schedule.location || "").toLowerCase();
      const description = (schedule.description || "").toLowerCase();
      return title.includes(query) || location.includes(query) || description.includes(query);
    });
  }, [schedules, searchQuery]);

  // Use refs to track previous values and avoid unnecessary callbacks
  const prevFilteredLengthRef = useRef(0);
  const prevFilteredNamesRef = useRef<string>("");

  // Notify parent of filtered results (only when content actually changes)
  useEffect(() => {
    const currentLength = filteredSchedules.length;
    const currentNames = filteredSchedules
      .map((s) => s.name)
      .sort()
      .join(",");

    // Only notify if the actual filtered content changed
    if (currentLength !== prevFilteredLengthRef.current || currentNames !== prevFilteredNamesRef.current) {
      prevFilteredLengthRef.current = currentLength;
      prevFilteredNamesRef.current = currentNames;
      onFilteredChange?.(filteredSchedules);
    }
  }, [filteredSchedules, onFilteredChange]);

  // Notify parent of filter state
  const hasFilter = searchQuery.trim().length > 0;
  const prevHasFilterRef = useRef(false);

  useEffect(() => {
    if (hasFilter !== prevHasFilterRef.current) {
      prevHasFilterRef.current = hasFilter;
      onHasFilterChange?.(hasFilter);
    }
  }, [hasFilter, onHasFilterChange]);

  const handleClear = useCallback(() => {
    setSearchQuery("");
  }, []);

  const resultCount = filteredSchedules.length;
  const totalCount = schedules.length;

  return (
    <div className={cn("relative w-full", className)}>
      <div className="relative flex items-center">
        <SearchIcon className="absolute left-2.5 w-4 h-4 text-muted-foreground pointer-events-none" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder={t("schedule.search-placeholder") || "Search schedules..."}
          className={cn(
            "h-9 w-full pl-9 pr-20 rounded-lg border border-border bg-background text-sm",
            "focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary",
            "transition-colors",
          )}
        />
        {hasFilter && (
          <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
            <span className="text-xs text-muted-foreground">
              {resultCount}/{totalCount}
            </span>
            <button
              type="button"
              onClick={handleClear}
              className="p-1 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
              aria-label="Clear search"
            >
              <XIcon className="w-3.5 h-3.5" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default ScheduleSearchBar;
