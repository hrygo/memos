/**
 * Schedule types for AI chat and schedule components.
 */

/**
 * ScheduleSummary represents a simplified schedule for query results and display.
 * Contains only the essential fields needed for showing schedules in lists and query results.
 */
export interface ScheduleSummary {
  uid: string;
  title: string;
  startTs: bigint;
  endTs: bigint;
  allDay: boolean;
  location: string;
  recurrenceRule: string;
  status: string;
}
