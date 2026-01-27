import { useEffect } from "react";
import type { UIConflictResolutionData, UIScheduleSuggestionData, UITimeSlotData, UITimeSlotPickerData, UIMemoPreviewData, UIQuickActionsData, UIQuickActionData, UIProgressTrackerData, UIScheduleListData } from "@/hooks/useScheduleAgent";
import { cn } from "@/lib/utils";
import { validateAndLog, validateScheduleSuggestion, validateTimeSlot, validateMemoPreview, validateScheduleList } from "./uiTypeValidators";
import { ConflictResolution } from "./ConflictResolution";
import { MemoPreview } from "./MemoPreview";
import { MemoSearchResultCard } from "./MemoSearchResultCard";
import { ProgressTracker } from "./ProgressTracker";
import { QuickActions } from "./QuickActions";
import { ScheduleListCard } from "./ScheduleListCard";
import { ScheduleSuggestionCard } from "./ScheduleSuggestionCard";
import { TimeSlotPicker } from "./TimeSlotPicker";
import type { GenerativeUIContainerProps, UIAction } from "./types";

/**
 * Auto-dismiss timeout for UI tools (5 minutes).
 * Tools are automatically dismissed after this period to prevent stale UI.
 */
const UI_TOOL_AUTO_DISMISS_MS = 5 * 60 * 1000;

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
      const remainingTime = UI_TOOL_AUTO_DISMISS_MS - age;

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

  const handleMemoConfirm = (data: UIMemoPreviewData) => {
    onAction({ type: "confirm", toolId: tool.id, data });
  };

  const handleQuickAction = (action: UIQuickActionData) => {
    onAction({ type: "quick_action", toolId: tool.id, data: action });
  };

  const handleCancel = () => {
    onAction({ type: "cancel", toolId: tool.id });
  };

  switch (tool.type) {
    case "schedule_suggestion": {
      const validatedData = validateAndLog(tool.data, validateScheduleSuggestion, "schedule_suggestion");
      if (!validatedData) return null;
      return <ScheduleSuggestionCard data={validatedData} onConfirm={handleConfirm} />;
    }

    case "time_slot_picker":
      return <TimeSlotPicker data={tool.data as UITimeSlotPickerData} onSelect={handleSlotSelect} onDismiss={onDismiss} />;

    case "conflict_resolution":
      return <ConflictResolution data={tool.data as UIConflictResolutionData} onAction={handleConflictAction} onDismiss={onDismiss} />;

    case "memo_preview": {
      const memoData = validateAndLog(tool.data, validateMemoPreview, "memo_preview");
      if (!memoData) return null;
      // If UID is present, this is a search result (read-only), not a create confirmation
      if (memoData.uid) {
        return <MemoSearchResultCard data={memoData} onDismiss={onDismiss} />;
      }
      // Otherwise, this is a create confirmation (requires user action)
      return <MemoPreview data={memoData} onConfirm={handleMemoConfirm} onDismiss={onDismiss} />;
    }

    case "quick_actions":
      return <QuickActions data={tool.data as UIQuickActionsData} onAction={handleQuickAction} onDismiss={onDismiss} />;

    case "schedule_list": {
      const validatedData = validateAndLog(tool.data, validateScheduleList, "schedule_list");
      if (!validatedData) return null;
      return <ScheduleListCard data={validatedData} onDismiss={onDismiss} />;
    }

    case "progress_tracker":
      return <ProgressTracker data={tool.data as UIProgressTrackerData} onCancel={handleCancel} onDismiss={onDismiss} />;

    default:
      // Unknown tool type - render nothing
      return null;
  }
}
