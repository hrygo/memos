/**
 * ScheduleAI Component Types
 * Types for generative UI components in the schedule page
 */

import type {
  UIConflictResolutionData,
  UIQuickActionsData,
  UIScheduleSuggestionData,
  UITimeSlotData,
  UITimeSlotPickerData,
  UIMemoPreviewData,
  UIProgressTrackerData,
  UIScheduleListData,
} from "@/hooks/useScheduleAgent";

// Re-export types from useScheduleAgent for convenience
export type {
  UIConflictResolutionData,
  UIQuickActionsData,
  UIScheduleSuggestionData,
  UITimeSlotData,
  UITimeSlotPickerData,
  UIMemoPreviewData,
  UIProgressTrackerData,
  UIScheduleListData,
} from "@/hooks/useScheduleAgent";

/**
 * UI Tool event with metadata
 */
export interface UIToolEvent {
  id: string;
  type: "schedule_suggestion" | "time_slot_picker" | "conflict_resolution" | "quick_actions" | "memo_preview" | "progress_tracker" | "schedule_list";
  data: UIScheduleSuggestionData | UITimeSlotPickerData | UIConflictResolutionData | UIQuickActionsData | UIMemoPreviewData | UIProgressTrackerData | UIScheduleListData;
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
  onReject?: () => void;
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

/**
 * UI Quick Action data
 */
export interface UIQuickActionData {
  id: string;
  label: string;
  description: string;
  icon?: string;
  prompt: string;
}

/**
 * Props for MemoPreview
 */
export interface MemoPreviewProps {
  data: UIMemoPreviewData;
  onConfirm: (data: UIMemoPreviewData) => void;
  onDismiss?: () => void;
  isLoading?: boolean;
}

/**
 * Props for MemoSearchResultCard
 */
export interface MemoSearchResultCardProps {
  data: UIMemoPreviewData;
  onDismiss?: () => void;
}

/**
 * Props for ScheduleListCard
 */
export interface ScheduleListCardProps {
  data: UIScheduleListData;
  onDismiss?: () => void;
}

/**
 * Props for ProgressTracker
 */
export interface ProgressTrackerProps {
  data: UIProgressTrackerData;
  onCancel?: () => void;
  onDismiss?: () => void;
}
