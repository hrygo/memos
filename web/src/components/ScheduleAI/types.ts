/**
 * ScheduleAI Component Types
 * Types for generative UI components in the schedule page
 */

import type {
  UIConflictResolutionData,
  UIConflictSchedule,
  UIQuickActionsData,
  UIScheduleSuggestionData,
  UITimeSlotData,
  UITimeSlotPickerData,
} from "@/hooks/useScheduleAgent";

/**
 * UI Tool event with metadata
 */
export interface UIToolEvent {
  id: string;
  type: "schedule_suggestion" | "time_slot_picker" | "conflict_resolution" | "quick_actions";
  data:
    | UIScheduleSuggestionData
    | UITimeSlotPickerData
    | UIConflictResolutionData
    | UIQuickActionsData;
  timestamp: number;
}

/**
 * Props for GenerativeUIContainer
 */
export interface GenerativeUIContainerProps {
  tools: UIToolEvent[];
  onAction: (action: UIAction) => void;
  onDismiss?: (toolId: string) => void;
  className?: string;
}

/**
 * User action from UI components
 */
export interface UIAction {
  type: "confirm" | "reject" | "select_slot" | "override" | "reschedule" | "cancel" | "quick_action";
  toolId: string;
  data?: unknown;
}

/**
 * Props for ScheduleSuggestionCard
 */
export interface ScheduleSuggestionCardProps {
  data: UIScheduleSuggestionData;
  onConfirm: (data: UIScheduleSuggestionData) => void;
  onReject: () => void;
  isLoading?: boolean;
}

/**
 * Props for TimeSlotPicker
 */
export interface TimeSlotPickerProps {
  data: UITimeSlotPickerData;
  onSelect: (slot: UITimeSlotData) => void;
  onDismiss: () => void;
  isLoading?: boolean;
}

/**
 * Props for ConflictResolution
 */
export interface ConflictResolutionProps {
  data: UIConflictResolutionData;
  onAction: (action: "override" | "reschedule" | "cancel", slot?: UITimeSlotData) => void;
  onDismiss: () => void;
  isLoading?: boolean;
}

/**
 * Props for QuickActions
 */
export interface QuickActionsProps {
  data: UIQuickActionsData;
  onAction: (action: UIQuickActionData) => void;
  onDismiss: () => void;
}
