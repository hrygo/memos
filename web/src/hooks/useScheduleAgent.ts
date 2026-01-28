import { useMutation, useQueryClient } from "@tanstack/react-query";
import { scheduleAgentServiceClient } from "@/connect";

/**
 * Hook to chat with Schedule Agent (non-streaming)
 */
export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

interface ScheduleAgentChatRequest {
  message: string;
  userTimezone?: string;
  history?: ChatMessage[];
}

/**
 * Parsed event from the agent stream
 */
export interface ParsedEvent {
  type: string;
  data: string;
  uiType?: string;
  uiData?: unknown;
}

/**
 * UI Tool Event types
 */
export interface UIScheduleSuggestionData {
  title: string;
  start_ts: number;
  end_ts: number;
  location?: string;
  description?: string;
  all_day?: boolean;
  confidence?: number;
  reason?: string;
  session_id?: string;
}

export interface UITimeSlotData {
  label: string;
  start_ts: number;
  end_ts: number;
  duration: number;
  reason: string;
}

export interface UITimeSlotPickerData {
  slots: UITimeSlotData[];
  default_idx: number;
  reason: string;
  session_id?: string;
}

export interface UIConflictSchedule {
  uid: string;
  title: string;
  start_time: number;
  end_time: number;
  location?: string;
  all_day: boolean;
}

export interface UIAutoResolvedSlot {
  label: string;
  start_ts: number;
  end_ts: number;
  reason: string;
  score?: number;  // Optional - backend may not always provide
}

export interface UIConflictResolutionData {
  new_schedule: UIScheduleSuggestionData;
  conflicting_schedules: UIConflictSchedule[];
  suggested_slots: UITimeSlotData[];
  actions: string[];
  session_id?: string;
  auto_resolved?: UIAutoResolvedSlot;
}

export interface UIQuickActionData {
  id: string;
  label: string;
  description: string;
  icon?: string;
  prompt: string;
}

export interface UIQuickActionsData {
  title: string;
  description: string;
  actions: UIQuickActionData[];
  session_id?: string;
}

export interface UIMemoPreviewData {
  uid?: string;
  title: string;
  content: string;
  tags?: string[];
  confidence: number;
  reason?: string;
  session_id?: string;
}

export interface UIScheduleItem {
  uid: string;
  title: string;
  start_ts: number;
  end_ts: number;
  all_day: boolean;
  location?: string;
  status?: string;
}

export interface UIScheduleListData {
  title: string;
  query: string;
  count: number;
  schedules: UIScheduleItem[];
  time_range?: string;
  reason?: string;
  session_id?: string;
}

export interface ProgressStep {
  id: string;
  label: string;
  status: "pending" | "in_progress" | "completed" | "failed";
  error?: string;
}

export interface UIProgressTrackerData {
  title: string;
  steps: ProgressStep[];
  current_step: number;
  can_cancel: boolean;
  session_id?: string;
}

/**
 * UI Tool Event wrapper
 */
export interface UIToolEvent {
  type: "schedule_suggestion" | "time_slot_picker" | "conflict_resolution" | "quick_actions" | "memo_preview" | "progress_tracker";
  data: UIScheduleSuggestionData | UITimeSlotPickerData | UIConflictResolutionData | UIQuickActionsData | UIMemoPreviewData | UIProgressTrackerData;
}

/**
 * Hook to chat with Schedule Agent (non-streaming)
 */
export function useScheduleAgentChat() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: ScheduleAgentChatRequest) => {
      // Build context-aware message
      const parts: string[] = [];

      // Conversation History
      if (request.history && request.history.length > 0) {
        parts.push("[Conversation History]");
        request.history.forEach((msg) => {
          parts.push(`${msg.role === "user" ? "User" : "Assistant"}: ${msg.content}`);
        });
      }

      // Current Message
      parts.push(`User: ${request.message}`);

      const fullMessage = parts.join("\n\n");

      const response = await scheduleAgentServiceClient.chat({
        message: fullMessage,
        userTimezone: request.userTimezone || "Asia/Shanghai",
      });
      return response;
    },
    onSuccess: () => {
      // Invalidate schedule lists to refetch
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
    },
  });
}

/**
 * Parse an event JSON string into a ParsedEvent
 */
export function parseEvent(eventJSON: string): ParsedEvent | null {
  try {
    const event = JSON.parse(eventJSON);

    // Check if this is a UI event
    if (event.type && event.type.startsWith("ui_")) {
      let uiData: unknown;
      try {
        uiData = JSON.parse(event.data);
      } catch {
        uiData = event.data;
      }

      return {
        type: event.type,
        data: event.data,
        uiType: event.type,
        uiData,
      };
    }

    return {
      type: event.type,
      data: event.data,
    };
  } catch (e) {
    console.error("Failed to parse event:", eventJSON, e);
    return null;
  }
}

/**
 * Check if an event is a UI tool event
 */
export function isUIToolEvent(event: ParsedEvent): event is ParsedEvent & { uiData: unknown; uiType: string } {
  return event.uiType !== undefined && event.uiData !== undefined;
}

/**
 * Get the UI tool type from an event
 */
export function getUIToolType(event: ParsedEvent): UIToolEvent["type"] | null {
  if (!event.uiType) return null;

  switch (event.uiType) {
    case "ui_schedule_suggestion":
      return "schedule_suggestion";
    case "ui_time_slot_picker":
      return "time_slot_picker";
    case "ui_conflict_resolution":
      return "conflict_resolution";
    case "ui_quick_actions":
      return "quick_actions";
    case "ui_memo_preview":
      return "memo_preview";
    case "ui_progress_tracker":
      return "progress_tracker";
    default:
      return null;
  }
}

/**
 * Hook to chat with Schedule Agent (streaming)
 * Returns an async generator that yields stream events
 */
export async function* scheduleAgentChatStream(
  message: string,
  userTimezone = "Asia/Shanghai",
  onEvent?: (event: { type: string; data: string; uiType?: string; uiData?: unknown }) => void,
): AsyncGenerator<{ type: string; data: string; content?: string; done?: boolean; uiType?: string; uiData?: unknown }, void> {
  const response = await scheduleAgentServiceClient.chatStream({
    message,
    userTimezone,
  });

  for await (const chunk of response) {
    // Parse the event JSON
    if (chunk.event) {
      const parsed = parseEvent(chunk.event);
      if (parsed) {
        const enhancedEvent = {
          type: parsed.type,
          data: parsed.data,
          uiType: parsed.uiType,
          uiData: parsed.uiData,
        };
        onEvent?.(enhancedEvent);
        yield enhancedEvent;
      }
    }

    // Yield the raw chunk for compatibility
    yield {
      type: chunk.event ? "raw" : "data",
      data: chunk.event || "",
      content: chunk.content,
      done: chunk.done,
    };
  }
}
