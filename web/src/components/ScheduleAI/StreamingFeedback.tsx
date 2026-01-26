import { Calendar, CheckCircle2, Clock, Loader2, Search, Sparkles, Zap } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import { type Translations, useTranslate } from "@/utils/i18n";

/**
 * Streaming event from the Agent
 */
export interface StreamingEvent {
  type: "thinking" | "tool_use" | "tool_result" | "answer" | "error" | "ui_schedule_suggestion";
  data: string;
  timestamp?: number;
}

/**
 * Props for StreamingFeedback component
 */
interface StreamingFeedbackProps {
  events: StreamingEvent[];
  isStreaming: boolean;
  className?: string;
}

/**
 * Icon mapping for different event types
 */
const eventIcons: Record<string, React.ReactNode> = {
  thinking: <Sparkles className="h-3.5 w-3.5" />,
  tool_use: <Zap className="h-3.5 w-3.5" />,
  tool_result: <CheckCircle2 className="h-3.5 w-3.5" />,
  schedule_query: <Search className="h-3.5 w-3.5" />,
  schedule_add: <Calendar className="h-3.5 w-3.5" />,
  find_free_time: <Clock className="h-3.5 w-3.5" />,
};

/**
 * Parse tool name from tool_use event data
 */
function parseToolName(data: string): string | null {
  // Format: "tool_name:{json}" or "tool_name"
  const match = data.match(/^(\w+)(?::|$)/);
  return match ? match[1] : null;
}

// Helper to cast translation keys
const tr = (key: string) => key as Translations;

/**
 * Format event message for display
 */
function formatEventMessage(event: StreamingEvent, t: ReturnType<typeof useTranslate>): string {
  switch (event.type) {
    case "thinking":
      return event.data || t(tr("schedule.ai.thinking"));
    case "tool_use": {
      const toolName = parseToolName(event.data);
      if (toolName === "schedule_query") {
        return t(tr("schedule.ai.checking-schedule"));
      }
      if (toolName === "schedule_add") {
        return t(tr("schedule.ai.creating-schedule"));
      }
      if (toolName === "find_free_time") {
        return t(tr("schedule.ai.finding-free-time"));
      }
      if (toolName === "schedule_update") {
        return t(tr("schedule.ai.updating-schedule"));
      }
      return `${t(tr("schedule.ai.using-tool"))}: ${toolName}`;
    }
    case "tool_result":
      return t(tr("schedule.ai.tool-completed"));
    case "answer":
      return event.data;
    case "error":
      return event.data || t(tr("schedule.ai.error"));
    default:
      return event.data;
  }
}

/**
 * Get icon for event
 */
function getEventIcon(event: StreamingEvent): React.ReactNode {
  if (event.type === "tool_use") {
    const toolName = parseToolName(event.data);
    if (toolName && eventIcons[toolName]) {
      return eventIcons[toolName];
    }
  }
  return eventIcons[event.type] || <Sparkles className="h-3.5 w-3.5" />;
}

/**
 * StreamingFeedback - Real-time display of AI thinking process
 *
 * Shows the user what the AI is doing:
 * - Parsing input
 * - Checking conflicts
 * - Creating schedule
 */
export function StreamingFeedback({ events, isStreaming, className }: StreamingFeedbackProps) {
  const t = useTranslate();
  const containerRef = useRef<HTMLDivElement>(null);
  const [visibleEvents, setVisibleEvents] = useState<StreamingEvent[]>([]);

  // Auto-scroll to bottom when new events arrive
  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [visibleEvents]);

  // Animate events appearing
  useEffect(() => {
    // Only show the last 3 events for compact display
    const recentEvents = events.slice(-3);
    setVisibleEvents(recentEvents);
  }, [events]);

  // Don't render if no events and not streaming
  if (!isStreaming && events.length === 0) {
    return null;
  }

  return (
    <div
      ref={containerRef}
      className={cn("overflow-hidden rounded-lg border border-border/50 bg-muted/30", "transition-all duration-300 ease-out", className)}
    >
      <div className="p-3 space-y-2">
        {/* Event list */}
        {visibleEvents.map((event, idx) => {
          const isLatest = idx === visibleEvents.length - 1;
          const message = formatEventMessage(event, t);
          const icon = getEventIcon(event);

          return (
            <div
              key={`${event.type}-${idx}`}
              className={cn(
                "flex items-start gap-2 text-sm",
                "animate-in fade-in slide-in-from-bottom-2 duration-300",
                isLatest ? "text-foreground" : "text-muted-foreground",
              )}
            >
              {/* Icon with animation */}
              <div
                className={cn(
                  "flex-shrink-0 mt-0.5 p-1 rounded-md",
                  event.type === "tool_result" && "text-green-600 dark:text-green-400 bg-green-500/10",
                  event.type === "tool_use" && "text-blue-600 dark:text-blue-400 bg-blue-500/10",
                  event.type === "thinking" && "text-primary bg-primary/10",
                  event.type === "error" && "text-red-600 dark:text-red-400 bg-red-500/10",
                  isLatest && isStreaming && event.type !== "tool_result" && "animate-pulse",
                )}
              >
                {icon}
              </div>

              {/* Message */}
              <span className={cn("flex-1 leading-relaxed", isLatest && "font-medium")}>{message}</span>
            </div>
          );
        })}

        {/* Loading indicator when streaming with no recent events */}
        {isStreaming && visibleEvents.length === 0 && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            <span>{t(tr("schedule.ai.processing"))}</span>
          </div>
        )}
      </div>

      {/* Progress bar */}
      {isStreaming && (
        <div className="h-0.5 bg-muted overflow-hidden">
          <div className="h-full bg-primary/60 animate-pulse w-full origin-left" style={{ animation: "pulse 1.5s ease-in-out infinite" }} />
        </div>
      )}
    </div>
  );
}

/**
 * Compact streaming indicator for inline use
 */
interface CompactStreamingIndicatorProps {
  isStreaming: boolean;
  currentStep?: string;
  className?: string;
}

export function CompactStreamingIndicator({ isStreaming, currentStep, className }: CompactStreamingIndicatorProps) {
  const t = useTranslate();

  if (!isStreaming) {
    return null;
  }

  return (
    <div className={cn("flex items-center gap-2 text-xs text-muted-foreground", className)}>
      <Loader2 className="h-3 w-3 animate-spin text-primary" />
      <span>{currentStep || t(tr("schedule.ai.processing"))}</span>
    </div>
  );
}

export default StreamingFeedback;
