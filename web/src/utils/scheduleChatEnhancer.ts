import { ParrotAgentType } from "@/types/parrot";

/**
 * Build a schedule chat message with date context.
 * Simply adds the selected date as context prefix - let the backend agent handle the rest.
 *
 * @example
 * ```ts
 * buildScheduleMessage("吃午饭", "2026-01-23")
 * // Returns: "当前选中日期: 2026-01-23\n吃午饭"
 * ```
 */
export function buildScheduleMessage(
  userInput: string,
  selectedDate?: string
): string {
  const trimmedInput = userInput.trim();
  if (!trimmedInput) {
    return "";
  }

  // If date is provided, add it as a simple context prefix
  if (selectedDate) {
    return `当前选中日期: ${selectedDate}\n${trimmedInput}`;
  }

  return trimmedInput;
}

/**
 * Get the agent type for schedule chat (always SCHEDULE)
 */
export function getScheduleAgentType(): ParrotAgentType {
  return ParrotAgentType.SCHEDULE;
}
