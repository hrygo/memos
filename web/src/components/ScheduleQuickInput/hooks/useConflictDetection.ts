import dayjs from "dayjs";
import { useCallback, useMemo } from "react";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import type { ConflictInfo, SuggestedTimeSlot } from "../types";

// ============================================================================
// Constants
// ============================================================================

const DAY_START = 8; // 8 AM
const DAY_END = 22;  // 10 PM
const SLOT_INCREMENT = 30; // 30 minutes
const MAX_SUGGESTIONS = 3;

// ============================================================================
// Types
// ============================================================================

interface UseConflictDetectionOptions {
  startTs?: bigint;
  endTs?: bigint;
  existingSchedules?: Schedule[];
  excludeName?: string;
  t?: (key: string) => string | unknown;
}

interface UseConflictDetectionReturn {
  conflicts: ConflictInfo[];
  hasConflicts: boolean;
  suggestions: SuggestedTimeSlot[];
  findAvailableSlots: (date: Date, duration: number) => SuggestedTimeSlot[];
}

// ============================================================================
// Core Conflict Detection
// ============================================================================

/**
 * Check if two time ranges overlap.
 * Convention: [start, end) - inclusive start, exclusive end.
 * Overlap when: start1 < end2 && end1 > start2
 */
function hasOverlap(start1: number, end1: number, start2: number, end2: number): boolean {
  return start1 < end2 && end1 > start2;
}

/**
 * Find conflicts between a new schedule and existing schedules.
 */
function findConflicts(
  startTs: number,
  endTs: number,
  existingSchedules: Schedule[],
  excludeName?: string
): ConflictInfo[] {
  return existingSchedules
    .filter((s) => {
      if (excludeName && s.name === excludeName) return false;
      return hasOverlap(startTs, endTs, Number(s.startTs), Number(s.endTs));
    })
    .map((schedule) => {
      const sStart = Number(schedule.startTs);
      const sEnd = Number(schedule.endTs);
      const isFull = startTs >= sStart && endTs <= sEnd;

      return {
        conflictingSchedule: schedule,
        type: isFull ? ("full" as const) : ("partial" as const),
        overlapStartTs: BigInt(Math.max(startTs, sStart)),
        overlapEndTs: BigInt(Math.min(endTs, sEnd)),
      };
    });
}

/**
 * Hook for detecting schedule conflicts and suggesting alternatives.
 */
export function useConflictDetection(options: UseConflictDetectionOptions): UseConflictDetectionReturn {
  const { startTs, endTs, existingSchedules = [], excludeName, t } = options;

  // Detect conflicts
  const conflicts = useMemo(() => {
    if (!startTs || !endTs || existingSchedules.length === 0) return [];
    return findConflicts(Number(startTs), Number(endTs), existingSchedules, excludeName);
  }, [startTs, endTs, existingSchedules, excludeName]);

  const hasConflicts = conflicts.length > 0;

  // Find available slots
  const findAvailableSlots = useCallback((date: Date, durationMinutes: number): SuggestedTimeSlot[] => {
    const slots: SuggestedTimeSlot[] = [];
    const dayStart = dayjs(date).hour(DAY_START).minute(0);
    const dayEnd = dayjs(date).hour(DAY_END).minute(0);
    const now = dayjs();

    // Get schedules for this day, sorted by start time
    const daySchedules = existingSchedules
      .filter((s) => dayjs(Number(s.startTs) * 1000).isSame(date, "day"))
      .sort((a, b) => Number(a.startTs) - Number(b.startTs));

    // Start searching from current time (rounded up) or DAY_START
    let searchStart = dayStart;
    if (now.isSame(date, "day") && now.isAfter(dayStart)) {
      const roundedMinute = Math.ceil(now.minute() / SLOT_INCREMENT) * SLOT_INCREMENT;
      searchStart = now.minute(roundedMinute).second(0);
      if (searchStart.isBefore(dayStart)) searchStart = dayStart;
    }

    // Scan through the day looking for gaps
    let currentStart = searchStart.unix();

    for (const schedule of daySchedules) {
      const sStart = Number(schedule.startTs);
      const sEnd = Number(schedule.endTs);
      const gapSeconds = sStart - currentStart;

      // Check if gap fits the required duration
      if (gapSeconds >= durationMinutes * 60 && !hasOverlap(currentStart, currentStart + durationMinutes * 60, sStart, sEnd)) {
        const slotEnd = currentStart + durationMinutes * 60;
        slots.push({
          startTs: BigInt(currentStart),
          endTs: BigInt(slotEnd),
          label: `${dayjs.unix(currentStart).format("HH:mm")} - ${dayjs.unix(slotEnd).format("HH:mm")}`,
          reason: (t?.("schedule.conflict.no-conflict-slot") as string) ?? "Available",
        });
        if (slots.length >= MAX_SUGGESTIONS) break;
      }

      // Move to after this schedule
      currentStart = Math.max(currentStart, sEnd);
    }

    // Check for slot after last schedule
    if (slots.length < MAX_SUGGESTIONS) {
      const remainingSeconds = dayEnd.unix() - currentStart;
      if (remainingSeconds >= durationMinutes * 60) {
        const slotEnd = currentStart + durationMinutes * 60;
        slots.push({
          startTs: BigInt(currentStart),
          endTs: BigInt(slotEnd),
          label: `${dayjs.unix(currentStart).format("HH:mm")} - ${dayjs.unix(slotEnd).format("HH:mm")}`,
          reason: (t?.("schedule.conflict.end-slot") as string) ?? "Available",
        });
      }
    }

    return slots.slice(0, MAX_SUGGESTIONS);
  }, [existingSchedules, t]);

  // Generate suggestions based on current input
  const suggestions = useMemo(() => {
    if (!startTs || !endTs) return [];

    const durationMinutes = (Number(endTs) - Number(startTs)) / 60;
    const startDate = dayjs(Number(startTs) * 1000).toDate();

    let slots = findAvailableSlots(startDate, durationMinutes);

    // If no slots same day, try next day
    if (slots.length === 0) {
      const nextDay = dayjs(startDate).add(1, "day").toDate();
      slots = findAvailableSlots(nextDay, durationMinutes);
      slots = slots.map((s) => ({
        ...s,
        reason: (t?.("schedule.conflict.next-day-available") as string) ?? s.reason,
      }));
    }

    return slots;
  }, [startTs, endTs, findAvailableSlots, t]);

  return {
    conflicts,
    hasConflicts,
    suggestions,
    findAvailableSlots,
  };
}

// ============================================================================
// Utility Functions (exported for testing)
// ============================================================================

export function checkOverlap(start1: bigint | number, end1: bigint | number, start2: bigint | number, end2: bigint | number): boolean {
  return hasOverlap(
    Number(start1), Number(end1),
    Number(start2), Number(end2)
  );
}

export function getOverlapDuration(start1: bigint | number, end1: bigint | number, start2: bigint | number, end2: bigint | number): number {
  if (!checkOverlap(start1, end1, start2, end2)) return 0;

  const s1 = Number(start1), e1 = Number(end1);
  const s2 = Number(start2), e2 = Number(end2);
  const overlapStart = Math.max(s1, s2);
  const overlapEnd = Math.min(e1, e2);

  return Math.max(0, overlapEnd - overlapStart) / 60;
}
