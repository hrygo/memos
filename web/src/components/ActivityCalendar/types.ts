export type CalendarSize = "default" | "small";

/** Summary of a schedule for calendar display */
export interface ScheduleSummary {
  /** Unique identifier (schedule.name from API) */
  uid: string;
  title: string;
  startTs: bigint;
  endTs: bigint;
  allDay: boolean;
  location?: string;
}

export interface CalendarDayCell {
  date: string;
  label: number;
  count: number;
  isCurrentMonth: boolean;
  isToday: boolean;
  isSelected: boolean;
  isWeekend: boolean;
  scheduleCount?: number;
  hasSchedule?: boolean;
}

export interface CalendarDayRow {
  days: CalendarDayCell[];
}

export interface CalendarMatrixResult {
  weeks: CalendarDayRow[];
  weekDays: string[];
  maxCount: number;
}

export interface MonthCalendarProps {
  month: string;
  data: Record<string, number>;
  maxCount: number;
  size?: CalendarSize;
  onClick?: (date: string) => void;
  className?: string;
  schedulesByDate?: Record<string, ScheduleSummary[]>;
}

export interface YearCalendarProps {
  selectedYear: number;
  data: Record<string, number>;
  onYearChange: (year: number) => void;
  onDateClick: (date: string) => void;
  className?: string;
}
