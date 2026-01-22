import { Calendar, Check, Clock } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

/**
 * Parsed suggestion from AI response
 */
export interface ScheduleSuggestion {
  id: string;
  title: string;
  date: string; // "今天", "明天", or formatted date
  startTime: string; // "15:00"
  endTime?: string; // "16:00"
  rawText: string; // Original text for reference
}

/**
 * Parse AI response to extract schedule suggestions
 * Supports formats like:
 * - "建议创建：明天下午3点开会"
 * - "可以帮您创建：今天15:00-16:00的会议"
 */
export function parseSuggestions(aiResponse: string, todayStr = "今天", tomorrowStr = "明天"): ScheduleSuggestion[] {
  if (!aiResponse) return [];

  const suggestions: ScheduleSuggestion[] = [];
  const lines = aiResponse.split("\n").filter((line) => line.trim());

  // Common patterns
  const patterns = [
    // "建议创建：明天下午3点开会"
    /(?:建议创建|可以创建|为您创建|创建)\s*[:：]?\s*(今天|明天)?(?:上午|下午|晚上)?(\d{1,2})(?:点|时)(?:半)?(?:\d{1,2}分?)?(?:到|-)?(?:\d{1,2})?(?:点|时)?(?:半)?的?(.+)/gi,
    // "今天15:00开会" or "明天 3:00 PM 会议"
    /(今天|明天|\d{1,2}月\d{1,2}日)\s*(\d{1,2}:\d{2}|\d{1,2}\s*(?:AM|PM|am|pm)?)\s*(?:到|-)\s*(\d{1,2}:\d{2}|\d{1,2}\s*(?:AM|PM|am|pm)?)?\s*的?(.+)/gi,
  ];

  for (const line of lines) {
    for (const pattern of patterns) {
      pattern.lastIndex = 0; // Reset regex
      const match = pattern.exec(line);
      if (match) {
        const [, datePart, timePart, endTimePart, titlePart] = match;
        const suggestion = parseMatchToSuggestion(datePart, timePart, endTimePart, titlePart, line, todayStr, tomorrowStr);
        if (suggestion) {
          suggestions.push(suggestion);
          break; // Use first match only per line
        }
      }
    }
  }

  // Also check for numbered list items like "1. 明天3点开会"
  const listPattern = /^\d+[\.\、]\s*(.+)/;
  for (const line of lines) {
    const listMatch = line.match(listPattern);
    if (listMatch) {
      const content = listMatch[1];
      // Try to parse the content
      for (const pattern of patterns) {
        pattern.lastIndex = 0;
        const match = pattern.exec(content);
        if (match) {
          const [, datePart, timePart, endTimePart, titlePart] = match;
          const suggestion = parseMatchToSuggestion(datePart, timePart, endTimePart, titlePart, content, todayStr, tomorrowStr);
          if (suggestion && !suggestions.find((s) => s.rawText === content)) {
            suggestions.push(suggestion);
            break;
          }
        }
      }
    }
  }

  return suggestions.slice(0, 3); // Max 3 suggestions
}

function parseMatchToSuggestion(
  datePart: string | undefined,
  timePart: string | undefined,
  endTimePart: string | undefined,
  titlePart: string | undefined,
  rawText: string,
  todayStr: string,
  tomorrowStr: string
): ScheduleSuggestion | null {
  // Parse date
  let date = todayStr;
  if (datePart) {
    if (datePart.includes("明天")) date = tomorrowStr;
    else if (datePart.includes("今天")) date = todayStr;
    else date = datePart; // Use as-is for specific dates
  }

  // Parse time
  let startTime = "09:00";
  let endTime: string | undefined;

  if (timePart) {
    startTime = normalizeTime(timePart);
  }

  if (endTimePart) {
    endTime = normalizeTime(endTimePart);
  }

  // Parse title
  const title = titlePart?.trim() || "";

  return {
    id: `${Date.now()}-${Math.random()}`,
    title: title || "新日程",
    date,
    startTime,
    endTime,
    rawText,
  };
}

function normalizeTime(timeStr: string): string {
  // Handle "3点" -> "03:00"
  const hourMatch = timeStr.match(/(\d{1,2})\s*(?:点|时)/);
  if (hourMatch) {
    let hour = parseInt(hourMatch[1], 10);
    // Check for 下午/晚上
    if ((timeStr.includes("下午") || timeStr.includes("晚上")) && hour < 12) {
      hour += 12;
    }
    // Check for 上午
    if (timeStr.includes("上午") && hour === 12) {
      hour = 0;
    }
    // Check for "半" (30 minutes)
    const minute = timeStr.includes("半") ? "30" : "00";
    return `${hour.toString().padStart(2, "0")}:${minute}`;
  }

  // Handle "15:00" or "3:00 PM"
  const standardMatch = timeStr.match(/(\d{1,2}):(\d{2})\s*(AM|PM|am|pm)?/);
  if (standardMatch) {
    let hour = parseInt(standardMatch[1], 10);
    const minute = standardMatch[2];
    const meridiem = standardMatch[3]?.toUpperCase();

    if (meridiem === "PM" && hour < 12) hour += 12;
    if (meridiem === "AM" && hour === 12) hour = 0;

    return `${hour.toString().padStart(2, "0")}:${minute}`;
  }

  // Handle "3 PM" or "3pm"
  const simpleMatch = timeStr.match(/(\d{1,2})\s*(AM|PM|am|pm)/);
  if (simpleMatch) {
    let hour = parseInt(simpleMatch[1], 10);
    const meridiem = simpleMatch[2].toUpperCase();

    if (meridiem === "PM" && hour < 12) hour += 12;
    if (meridiem === "AM" && hour === 12) hour = 0;

    return `${hour.toString().padStart(2, "0")}:00`;
  }

  return "09:00"; // Default
}

interface AISuggestionCardsProps {
  /** Parsed suggestions to display */
  suggestions: ScheduleSuggestion[];
  /** Called when user confirms a suggestion */
  onConfirmSuggestion: (suggestion: ScheduleSuggestion) => void;
  /** Optional className */
  className?: string;
}

/**
 * Displays AI-suggested schedules as clickable cards.
 * UX Design:
 * - Desktop: Double-click to create directly
 * - Mobile: Single click to select, click again to confirm
 */
export function AISuggestionCards({ suggestions, onConfirmSuggestion, className }: AISuggestionCardsProps) {
  const t = useTranslate();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [lastClickTime, setLastClickTime] = useState(0);

  if (suggestions.length === 0) {
    return null;
  }

  const handleCardClick = (suggestion: ScheduleSuggestion) => {
    const now = Date.now();
    const timeSinceLastClick = now - lastClickTime;

    // Desktop: Double-click detection (within 300ms)
    const isDoubleClick = timeSinceLastClick < 300 && selectedId === suggestion.id;

    // Mobile: Toggle selection
    if (isDoubleClick) {
      // Double-click on desktop: create directly
      onConfirmSuggestion(suggestion);
      setSelectedId(null);
    } else if (selectedId === suggestion.id) {
      // Second click on same card (mobile): confirm
      onConfirmSuggestion(suggestion);
      setSelectedId(null);
    } else {
      // First click: select the card
      setSelectedId(suggestion.id);
    }

    setLastClickTime(now);
  };

  return (
    <div className={cn("w-full", className)}>
      {/* Header */}
      <div className="flex items-center gap-1.5 px-1 mb-2">
        <div className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse" />
        <span className="text-xs text-muted-foreground">{t("schedule.quick-input.ai-suggestions") as string}</span>
      </div>

      {/* Desktop: horizontal row | Mobile: vertical stack */}
      <div className="flex flex-col sm:flex-row gap-2 sm:gap-3">
        {suggestions.map((suggestion) => {
          const isSelected = selectedId === suggestion.id;

          return (
            <button
              key={suggestion.id}
              onClick={() => handleCardClick(suggestion)}
              className={cn(
                "group flex-1 min-w-0 text-left relative",
                "p-3 rounded-xl border-2 transition-all duration-200",
                "active:scale-[0.98]",
                // Normal state
                !isSelected && "border-blue-500/20 bg-blue-50/50 dark:bg-blue-950/20 hover:border-blue-500/40 hover:bg-blue-50/80 dark:hover:bg-blue-950/30",
                // Selected state
                isSelected && "border-blue-500 bg-blue-100 dark:bg-blue-900/40 ring-2 ring-blue-500/30 shadow-md",
              )}
            >
              {/* Selected indicator */}
              {isSelected && (
                <div className="absolute top-2 right-2 w-5 h-5 rounded-full bg-blue-500 flex items-center justify-center animate-in zoom-in-50">
                  <Check className="w-3 h-3 text-white" />
                </div>
              )}

              {/* Title */}
              <div className={cn(
                "font-medium text-sm truncate mb-2 pr-6",
                !isSelected && "group-hover:text-blue-600 dark:group-hover:text-blue-400"
              )}>
                {suggestion.title || (t("schedule.quick-input.default-title") as string)}
              </div>

              {/* Time */}
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Calendar className="w-3 h-3 flex-shrink-0" />
                <span>{suggestion.date}</span>
                <Clock className="w-3 h-3 flex-shrink-0 ml-1" />
                <span className="font-medium text-foreground">
                  {suggestion.startTime}
                  {suggestion.endTime && ` - ${suggestion.endTime}`}
                </span>
              </div>

              {/* Hint text */}
              <div className="mt-2 text-[10px]">
                {isSelected ? (
                  <span className="text-blue-600 dark:text-blue-400 font-medium">{t("schedule.quick-input.click-again-create") as string}</span>
                ) : (
                  <span className="text-blue-500/70 hidden sm:inline">{t("schedule.quick-input.double-click-create") as string}</span>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
