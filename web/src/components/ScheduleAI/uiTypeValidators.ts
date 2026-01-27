import type { UIScheduleListData } from "@/hooks/useScheduleAgent";

/**
 * Validates if a value is a non-null object (and not an array)
 */
function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/**
 * Validates if a value is a string
 */
function isString(value: unknown): value is string {
  return typeof value === "string";
}

/**
 * Validates if a value is a number
 */
function isNumber(value: unknown): value is number {
  return typeof value === "number" && !isNaN(value);
}

/**
 * Validates if a value is a boolean
 */
function isBoolean(value: unknown): value is boolean {
  return typeof value === "boolean";
}

/**
 * Base interface for validation result
 */
export interface ValidationResult<T> {
  valid: boolean;
  data?: T;
  errors: string[];
}

/**
 * Validator for UIScheduleSuggestionData
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
}

export function validateScheduleSuggestion(data: unknown): ValidationResult<UIScheduleSuggestionData> {
  const errors: string[] = [];

  if (!isObject(data)) {
    return { valid: false, errors: ["Data must be an object"] };
  }

  if (!isString(data.title)) {
    errors.push("title must be a string");
  }
  if (!isNumber(data.start_ts)) {
    errors.push("start_ts must be a number");
  }
  if (!isNumber(data.end_ts)) {
    errors.push("end_ts must be a number");
  }
  if (data.location !== undefined && !isString(data.location)) {
    errors.push("location must be a string if provided");
  }
  if (data.description !== undefined && !isString(data.description)) {
    errors.push("description must be a string if provided");
  }
  if (data.all_day !== undefined && !isBoolean(data.all_day)) {
    errors.push("all_day must be a boolean if provided");
  }
  if (data.confidence !== undefined && !isNumber(data.confidence)) {
    errors.push("confidence must be a number if provided");
  }
  if (data.reason !== undefined && !isString(data.reason)) {
    errors.push("reason must be a string if provided");
  }

  if (errors.length > 0) {
    return { valid: false, errors };
  }

  return {
    valid: true,
    data: data as UIScheduleSuggestionData,
    errors: [],
  };
}

/**
 * Validator for UITimeSlotData
 */
export interface UITimeSlotData {
  label: string;
  start_ts: number;
  end_ts: number;
  duration: number;
  reason?: string;
}

export function validateTimeSlot(data: unknown): ValidationResult<UITimeSlotData> {
  const errors: string[] = [];

  if (!isObject(data)) {
    return { valid: false, errors: ["Data must be an object"] };
  }

  if (!isString(data.label)) {
    errors.push("label must be a string");
  }
  if (!isNumber(data.start_ts)) {
    errors.push("start_ts must be a number");
  }
  if (!isNumber(data.end_ts)) {
    errors.push("end_ts must be a number");
  }
  if (!isNumber(data.duration)) {
    errors.push("duration must be a number");
  }
  if (data.reason !== undefined && !isString(data.reason)) {
    errors.push("reason must be a string if provided");
  }

  if (errors.length > 0) {
    return { valid: false, errors };
  }

  return {
    valid: true,
    data: data as UITimeSlotData,
    errors: [],
  };
}

/**
 * Validator for UIMemoPreviewData
 */
export interface UIMemoPreviewData {
  uid?: string;
  title: string;
  content: string;
  tags?: string[];
  confidence?: number;
  reason?: string;
}

export function validateMemoPreview(data: unknown): ValidationResult<UIMemoPreviewData> {
  const errors: string[] = [];

  if (!isObject(data)) {
    return { valid: false, errors: ["Data must be an object"] };
  }

  if (data.uid !== undefined && !isString(data.uid)) {
    errors.push("uid must be a string if provided");
  }
  if (!isString(data.title)) {
    errors.push("title must be a string");
  }
  if (!isString(data.content)) {
    errors.push("content must be a string");
  }
  if (data.tags !== undefined && !Array.isArray(data.tags)) {
    errors.push("tags must be an array if provided");
  }
  if (data.confidence !== undefined && !isNumber(data.confidence)) {
    errors.push("confidence must be a number if provided");
  }
  if (data.reason !== undefined && !isString(data.reason)) {
    errors.push("reason must be a string if provided");
  }

  if (errors.length > 0) {
    return { valid: false, errors };
  }

  return {
    valid: true,
    data: data as UIMemoPreviewData,
    errors: [],
  };
}

/**
 * Safe type guard wrapper that logs validation errors
 */
export function validateAndLog<T>(
  data: unknown,
  validator: (data: unknown) => ValidationResult<T>,
  context: string,
): T | null {
  const result = validator(data);

  if (!result.valid) {
    console.error(`[UI Type Validation] ${context} failed:`, result.errors);
    return null;
  }

  return result.data;
}

/**
 * Validator for UIScheduleListData
 */
export function validateScheduleList(data: unknown): ValidationResult<UIScheduleListData> {
  const errors: string[] = [];

  if (!isObject(data)) {
    return { valid: false, errors: ["Data must be an object"] };
  }

  if (!isString(data.title)) {
    errors.push("title must be a string");
  }
  if (!isString(data.query)) {
    errors.push("query must be a string");
  }
  if (!isNumber(data.count)) {
    errors.push("count must be a number");
  }
  if (!Array.isArray(data.schedules)) {
    errors.push("schedules must be an array");
  } else {
    // Validate each schedule item
    for (let i = 0; i < data.schedules.length; i++) {
      const schedule = data.schedules[i];
      if (!isObject(schedule)) {
        errors.push(`schedules[${i}] must be an object`);
        continue;
      }
      if (!isString(schedule.uid)) {
        errors.push(`schedules[${i}].uid must be a string`);
      }
      if (!isString(schedule.title)) {
        errors.push(`schedules[${i}].title must be a string`);
      }
      if (!isNumber(schedule.start_ts)) {
        errors.push(`schedules[${i}].start_ts must be a number`);
      }
      if (!isNumber(schedule.end_ts)) {
        errors.push(`schedules[${i}].end_ts must be a number`);
      }
    }
  }
  if (data.time_range !== undefined && !isString(data.time_range)) {
    errors.push("time_range must be a string if provided");
  }
  if (data.reason !== undefined && !isString(data.reason)) {
    errors.push("reason must be a string if provided");
  }

  if (errors.length > 0) {
    return { valid: false, errors };
  }

  return {
    valid: true,
    data: data as UIScheduleListData,
    errors: [],
  };
}
