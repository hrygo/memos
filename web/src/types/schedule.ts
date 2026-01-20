/**
 * Schedule types for AI chat and schedule components.
 */

/**
 * ScheduleSummary represents a simplified schedule for query results and display.
 * Contains only the essential fields needed for showing schedules in lists and query results.
 *
 * Note: startTs and endTs use number instead of bigint for better JSON serialization
 * and React state compatibility. Unix timestamps in seconds fit safely in JavaScript Number.
 */
export interface ScheduleSummary {
  uid: string;
  title: string;
  startTs: number;
  endTs: number;
  allDay: boolean;
  location: string;
  recurrenceRule: string;
  status: string;
}
