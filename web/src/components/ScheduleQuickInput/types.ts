import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";

/** Parsing state for schedule input */
export type ParseState =
  | "idle" // No input or empty
  | "typing" // User is typing
  | "parsing" // Local/AI parsing in progress
  | "success" // Successfully parsed
  | "partial" // Partially parsed, needs more info
  | "conflict" // Parsed but has conflicts
  | "error"; // Parsing failed

/** Source of parsing result */
export type ParseSource = "local" | "ai" | "manual";

/** Confidence level of parsing result (0-1) */
export type Confidence = number;

/** Parsed schedule data */
export interface ParsedSchedule {
  title: string;
  description?: string;
  location?: string;
  startTs: bigint;
  endTs: bigint;
  allDay?: boolean;
  reminders?: Array<{ type: "before" | "at"; value: number; unit: "minutes" | "hours" | "days" }>;
  confidence: Confidence;
  source: ParseSource;
  missingFields?: Array<"title" | "startTime" | "endTime" | "duration">;
}

/** Conflict information */
export interface ConflictInfo {
  conflictingSchedule: Schedule;
  type: "full" | "partial";
  overlapStartTs: bigint;
  overlapEndTs: bigint;
}

/** Suggested time slot for rescheduling */
export interface SuggestedTimeSlot {
  startTs: bigint;
  endTs: bigint;
  label: string;
  reason?: string;
}

/** Template for quick schedule creation */
export interface ScheduleTemplate {
  id: string;
  title: string;
  icon: string;
  duration: number; // in minutes
  defaultTitle?: string;
  color?: string;
  /** i18n key for translating the title */
  i18nKey?: string;
}

/** Flow step for progressive schedule creation */
export type FlowStep = "initial" | "time-selection" | "duration" | "location" | "confirmation";

/** Conversation message for flow */
export interface FlowMessage {
  role: "user" | "assistant";
  content: string;
  timestamp: number;
}

/** Parse result with metadata */
export interface ParseResult {
  state: ParseState;
  parsedSchedule?: ParsedSchedule;
  conflicts?: ConflictInfo[];
  suggestions?: SuggestedTimeSlot[];
  message?: string;
}

/** User preferences for quick input */
export interface QuickInputPreferences {
  defaultDuration: number; // in minutes
  defaultAllDay: boolean;
  enableLocalParsing: boolean;
  enableAIParsing: boolean;
  parseDebounceMs: number;
  showTemplates: boolean;
  collapseOnCreate: boolean;
}

/** Quick input state */
export interface QuickInputState {
  input: string;
  parseState: ParseState;
  parseResult: ParseResult | null;
  flowStep: FlowStep;
  conversation: FlowMessage[];
  isExpanded: boolean;
  showTemplates: boolean;
  selectedTemplate?: ScheduleTemplate;
}

/** Cache entry for parse results */
export interface ParseCacheEntry {
  input: string;
  result: ParseResult;
  timestamp: number;
}
