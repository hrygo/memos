import { useEffect } from "react";
import type { UIConflictResolutionData, UIScheduleSuggestionData, UITimeSlotData, UITimeSlotPickerData } from "@/hooks/useScheduleAgent";
import { cn } from "@/lib/utils";
import { ConflictResolution } from "./ConflictResolution";
import { ScheduleSuggestionCard } from "./ScheduleSuggestionCard";
import { TimeSlotPicker } from "./TimeSlotPicker";
import type { GenerativeUIContainerProps, UIAction } from "./types";

/**
 * GenerativeUIContainer - Renders AI-generated UI components
 *
 * This container receives UI tool events from the AI agent and renders
 * the appropriate interactive components for user confirmation.
 */
export function GenerativeUIContainer({ tools, onAction, onDismiss, className }: GenerativeUIContainerProps) {
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
  onAction: (action: UIAction) => void;
  onDismiss: () => void;
}

function GenerativeUIComponent({ tool, onAction, onDismiss }: GenerativeUIComponentProps) {
  // Create wrappers that include toolId
  const handleConfirm = (data: UIScheduleSuggestionData) => {
    onAction({ type: "confirm", toolId: tool.id, data });
  };

  const handleReject = () => {
    onAction({ type: "reject", toolId: tool.id });
  };

  const handleSlotSelect = (slot: UITimeSlotData) => {
    onAction({ type: "select_slot", toolId: tool.id, data: slot });
  };

  const handleConflictAction = (action: "override" | "reschedule" | "cancel", slot?: UITimeSlotData) => {
    onAction({ type: action, toolId: tool.id, data: slot });
  };

  switch (tool.type) {
    case "schedule_suggestion":
      return <ScheduleSuggestionCard data={tool.data as UIScheduleSuggestionData} onConfirm={handleConfirm} />;

    case "time_slot_picker":
      return <TimeSlotPicker data={tool.data as UITimeSlotPickerData} onSelect={handleSlotSelect} onDismiss={onDismiss} />;

    case "conflict_resolution":
      return <ConflictResolution data={tool.data as UIConflictResolutionData} onAction={handleConflictAction} onDismiss={onDismiss} />;

    default:
      // Unknown tool type - render nothing
      return null;
  }
}
