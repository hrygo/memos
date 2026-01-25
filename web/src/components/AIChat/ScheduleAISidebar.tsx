import { ChevronLeft, ChevronRight, Sparkles, Clock, Lightbulb } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import { StreamingScheduleAssistant } from "./StreamingScheduleAssistant";
import { useScheduleContext } from "@/contexts/ScheduleContext";
import { useQueryClient } from "@tanstack/react-query";

interface ScheduleAISidebarProps {
  className?: string;
}

// Quick suggestion type
interface QuickSuggestion {
  id: string;
  icon: typeof Clock;
  label: string;
  prompt: string;
  category: "time" | "idea" | "history";
}

const QUICK_SUGGESTIONS: QuickSuggestion[] = [
  {
    id: "check-today",
    icon: Clock,
    label: "今天还有什么安排？",
    prompt: "今天还有什么安排？",
    category: "history",
  },
  {
    id: "free-time",
    icon: Clock,
    label: "今天什么时候有空？",
    prompt: "今天什么时候有空？",
    category: "time",
  },
  {
    id: "schedule-meeting",
    icon: Lightbulb,
    label: "帮我安排一个会议",
    prompt: "帮我安排一个会议",
    category: "idea",
  },
];

export function ScheduleAISidebar({ className }: ScheduleAISidebarProps) {
  const t = useTranslate();
  const { selectedDate } = useScheduleContext();
  const queryClient = useQueryClient();

  const [isExpanded, setIsExpanded] = useState(false);
  const [showSuggestions, setShowSuggestions] = useState(true);

  const handleSuccess = () => {
    // Refresh schedules
    queryClient.invalidateQueries({ queryKey: ["schedules"] });
    setShowSuggestions(true);
  };

  const handleError = (error: Error) => {
    console.error("Schedule AI error:", error);
  };

  const handleSuggestionClick = (_prompt: string) => {
    setShowSuggestions(false);
  };

  return (
    <div
      className={cn(
        "relative flex flex-col bg-white dark:bg-zinc-900 border-l border-border/50 transition-all duration-300 ease-in-out",
        isExpanded ? "w-full md:w-[400px]" : "w-0 md:w-[60px]",
        className,
      )}
    >
      {/* Toggle Button */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className={cn(
          "absolute -left-6 top-1/2 -translate-y-1/2 w-6 h-12 rounded-l-lg flex items-center justify-center transition-all duration-200",
          "bg-gradient-to-r from-primary/20 to-primary/10 hover:from-primary/30 hover:to-primary/20",
          "border border-l-0 border-primary/30",
          "group",
        )}
        aria-label={isExpanded ? "Collapse AI Assistant" : "Expand AI Assistant"}
      >
        {isExpanded ? <ChevronLeft className="w-4 h-4 text-primary" /> : <ChevronRight className="w-4 h-4 text-primary" />}
      </button>

      {/* Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex-shrink-0 flex items-center gap-2 p-4 border-b border-border/50">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary/20 to-primary/10 flex items-center justify-center">
            <Sparkles className="w-4 h-4 text-primary" />
          </div>
          {isExpanded && (
            <div className="flex-1">
              <h3 className="font-semibold text-foreground">{t("schedule.ai-sidebar.title") || "AI 助手"}</h3>
              <p className="text-xs text-muted-foreground">{t("schedule.ai-sidebar.subtitle") || "智能日程管理"}</p>
            </div>
          )}
        </div>

        {/* Content Area */}
        {isExpanded ? (
          <div className="flex-1 overflow-y-auto">
            {/* Quick Suggestions */}
            {showSuggestions && (
              <div className="p-4 border-b border-border/50">
                <div className="flex items-center gap-2 mb-3">
                  <Lightbulb className="w-4 h-4 text-yellow-500" />
                  <span className="text-sm font-medium text-foreground">{t("schedule.ai-sidebar.quick-suggestions") || "快捷操作"}</span>
                </div>
                <div className="space-y-2">
                  {QUICK_SUGGESTIONS.map((suggestion) => {
                    const Icon = suggestion.icon;
                    return (
                      <button
                        key={suggestion.id}
                        onClick={() => handleSuggestionClick(suggestion.prompt)}
                        className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg bg-muted/30 hover:bg-muted/50 transition-colors text-left group"
                      >
                        <div className="flex-shrink-0 w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                          <Icon className="w-4 h-4 text-primary" />
                        </div>
                        <span className="text-sm text-foreground">{suggestion.label}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Streaming Assistant */}
            <div className="p-4">
              <StreamingScheduleAssistant
                onSuccess={handleSuccess}
                onError={handleError}
                placeholder={
                  selectedDate
                    ? (t("schedule.ai-sidebar.placeholder-with-date", { date: selectedDate }) as string)
                    : (t("schedule.ai-sidebar.placeholder") as string)
                }
              />
            </div>
          </div>
        ) : (
          <div className="flex-1 flex items-center justify-center py-4">
            <div className="flex flex-col items-center gap-3 text-muted-foreground">
              <Sparkles className="w-6 h-6 text-primary/50" />
            </div>
          </div>
        )}

        {/* Footer Hint */}
        {isExpanded && (
          <div className="flex-shrink-0 p-3 border-t border-border/50 text-center">
            <p className="text-xs text-muted-foreground">{t("schedule.ai-sidebar.hint") || "使用自然语言创建和管理日程"}</p>
          </div>
        )}
      </div>
    </div>
  );
}
