import { useEffect } from "react";
import { useTranslate } from "@/utils/i18n";
import { cn } from "@/lib/utils";
import type {
  GenerativeUIContainerProps,
  ScheduleSuggestionCardProps,
  TimeSlotPickerProps,
  ConflictResolutionProps,
} from "./types";
import { ScheduleSuggestionCard } from "./ScheduleSuggestionCard";
import { TimeSlotPicker } from "./TimeSlotPicker";
import { ConflictResolution } from "./ConflictResolution";
import type {
  UIScheduleSuggestionData,
  UITimeSlotPickerData,
  UIConflictResolutionData,
} from "@/hooks/useScheduleAgent";

/**
 * GenerativeUIContainer - Renders AI-generated UI components
 *
 * This container receives UI tool events from the AI agent and renders
 * the appropriate interactive components for user confirmation.
 */
export function GenerativeUIContainer({
  tools,
  onAction,
  onDismiss,
  className,
}: GenerativeUIContainerProps) {
  const t = useTranslate();

  // Auto-dismiss tools after 5 minutes (temporary session)
  useEffect(() => {
    const timers = tools.map((tool) => {
      const age = Date.now() - tool.timestamp;
      const remainingTime = 5 * 60 * 1000 - age; // 5 minutes

      if (remainingTime <= 0) {
        onDismiss?.(tool.id);
        return null;
      }

      return setTimeout(() => {
        onDismiss?.(tool.id);
      }, remainingTime);
    });

    return () => {
      timers.forEach((timer) => {
        if (timer) clearTimeout(timer);
      });
    };
  }, [tools, onDismiss]);

  if (tools.length === 0) {
    return null;
  }

  return (
    <div className={cn("space-y-3", className)}>
      {tools.map((tool) => (
        <GenerativeUIComponent
          key={tool.id}
          tool={tool}
          onAction={(action) =>
            onAction({
              ...action,
              toolId: tool.id,
            })
          }
          onDismiss={() => onDismiss?.(tool.id)}
        />
      ))}
    </div>
  );
}

/**
 * GenerativeUIComponent - Renders a single UI tool component
 */
interface GenerativeUIComponentProps {
  tool: GenerativeUIContainerProps["tools"][number];
  onAction: (action: { type: string; data?: unknown }) => void;
  onDismiss: () => void;
}

function GenerativeUIComponent({
  tool,
  onAction,
  onDismiss,
}: GenerativeUIComponentProps) {
  switch (tool.type) {
    case "schedule_suggestion":
      return (
        <ScheduleSuggestionCard
          data={tool.data as UIScheduleSuggestionData}
          onConfirm={(data) => onAction({ type: "confirm", data })}
          onReject={() => onAction({ type: "reject" })}
        />
      );

    case "time_slot_picker":
      return (
        <TimeSlotPicker
          data={tool.data as UITimeSlotPickerData}
          onSelect={(slot) => onAction({ type: "select_slot", data: slot })}
          onDismiss={onDismiss}
        />
      );

    case "conflict_resolution":
      return (
        <ConflictResolution
          data={tool.data as UIConflictResolutionData}
          onAction={(action, slot) =>
            onAction({ type: action, data: slot })
          }
          onDismiss={onDismiss}
        />
      );

    default:
      // Unknown tool type - render nothing
      return null;
  }
}
