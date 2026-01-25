import { useCallback, useState, useRef } from "react";
import type {
  UIToolEvent,
  UIAction,
  UIScheduleSuggestionData,
  UITimeSlotData,
  UIConflictResolutionData,
} from "@/components/ScheduleAI/types";
import type {
  ParsedEvent,
  UIScheduleSuggestionData as HookUIScheduleSuggestionData,
  UITimeSlotPickerData,
  UIConflictResolutionData as HookUIConflictResolutionData,
} from "@/hooks/useScheduleAgent";

/**
 * Hook to manage UI tool events from the AI agent
 * Handles event parsing, state management, and user actions
 */
export function useAITools() {
  const [tools, setTools] = useState<UIToolEvent[]>([]);
  const toolIdCounter = useRef(0);

  /**
   * Process an event from the AI stream and add UI tools if present
   */
  const processEvent = useCallback((event: ParsedEvent) => {
    if (!event.uiType || !event.uiData) return;

    const toolId = `uitool-${++toolIdCounter.current}`;

    let toolType: UIToolEvent["type"];
    let toolData: UIToolEvent["data"];

    switch (event.uiType) {
      case "ui_schedule_suggestion": {
        toolType = "schedule_suggestion";
        toolData = event.uiData as UIScheduleSuggestionData;
        break;
      }
      case "ui_time_slot_picker": {
        toolType = "time_slot_picker";
        toolData = event.uiData as UITimeSlotPickerData;
        break;
      }
      case "ui_conflict_resolution": {
        toolType = "conflict_resolution";
        toolData = event.uiData as UIConflictResolutionData;
        break;
      }
      default:
        return;
    }

    const newTool: UIToolEvent = {
      id: toolId,
      type: toolType,
      data: toolData,
      timestamp: Date.now(),
    };

    setTools((prev) => [...prev, newTool]);
  }, []);

  /**
   * Handle user action from a UI tool
   */
  const handleAction = useCallback((action: UIAction) => {
    console.log("[useAITools] User action:", action);

    // Remove the tool after action
    setTools((prev) => prev.filter((t) => t.id !== action.toolId));

    // The action will be handled by the caller (e.g., sending confirmation to AI)
    return action;
  }, []);

  /**
   * Dismiss a specific tool
   */
  const dismissTool = useCallback((toolId: string) => {
    setTools((prev) => prev.filter((t) => t.id !== toolId));
  }, []);

  /**
   * Clear all tools
   */
  const clearTools = useCallback(() => {
    setTools([]);
  }, []);

  return {
    tools,
    processEvent,
    handleAction,
    dismissTool,
    clearTools,
    hasTools: tools.length > 0,
  };
}
