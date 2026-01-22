import dayjs from "dayjs";
import { useMemo } from "react";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import type { ConflictInfo, SuggestedTimeSlot } from "../types";

/**
 * 区间约定 Convention:
 * - 日程时间区间：[start, end) 左闭右开
 * - 即：包含开始时间，不包含结束时间
 * - 例如：9:00-10:00 的日程占用 [9:00, 10:00)
 * - 两个日程冲突当且仅当：start1 < end2 && end1 > start2
 */

interface UseConflictDetectionOptions {
  /** Start timestamp for the schedule to check */
  startTs?: bigint;
  /** End timestamp for the schedule to check */
  endTs?: bigint;
  /** Existing schedules to check against */
  existingSchedules?: Schedule[];
  /** Schedule name to exclude from conflict check (for editing) */
  excludeName?: string;
  /** Translate function for i18n */
  t?: (key: string) => string | unknown;
}

interface UseConflictDetectionReturn {
  /** List of conflicting schedules */
  conflicts: ConflictInfo[];
  /** Whether there are any conflicts */
  hasConflicts: boolean;
  /** Suggested alternative time slots */
  suggestions: SuggestedTimeSlot[];
  /** Find available time slots on a specific day */
  findAvailableSlots: (date: Date, duration: number) => SuggestedTimeSlot[];
}

/**
 * Hook for detecting schedule conflicts and suggesting alternative time slots.
 * Works on the frontend for instant feedback before server validation.
 */
export function useConflictDetection(options: UseConflictDetectionOptions): UseConflictDetectionReturn {
  const { startTs, endTs, existingSchedules = [], excludeName, t } = options;

  const conflicts = useMemo(() => {
    if (!startTs || !endTs || existingSchedules.length === 0) {
      return [];
    }

    const start = Number(startTs);
    const end = Number(endTs);

    return existingSchedules
      .filter((s) => {
        // Exclude the schedule being edited
        if (excludeName && s.name === excludeName) {
          return false;
        }

        const sStart = Number(s.startTs);
        const sEnd = Number(s.endTs);

        // Check for overlap using [start, end) convention
        // Overlap when: new.start < existing.end AND new.end > existing.start
        return start < sEnd && end > sStart;
      })
      .map((schedule) => {
        const sStart = Number(schedule.startTs);
        const sEnd = Number(schedule.endTs);

        // Calculate overlap
        const overlapStart = BigInt(Math.max(start, sStart));
        const overlapEnd = BigInt(Math.min(end, sEnd));

        // Determine conflict type
        const type: "full" | "partial" = start >= sStart && end <= sEnd ? "full" : "partial";

        return {
          conflictingSchedule: schedule,
          type,
          overlapStartTs: overlapStart,
          overlapEndTs: overlapEnd,
        } as ConflictInfo;
      });
  }, [startTs, endTs, existingSchedules, excludeName]);

  const hasConflicts = conflicts.length > 0;

  /**
   * Find available time slots on a specific day
   */
  const findAvailableSlots = useMemo(() => {
    return (date: Date, duration: number): SuggestedTimeSlot[] => {
      const slots: SuggestedTimeSlot[] = [];
      const dayStart = dayjs(date).startOf("day");
      const dayEnd = dayjs(date).endOf("day");

      // Get all schedules for this day, sorted by start time
      const daySchedules = existingSchedules
        .filter((s) => {
          const sStart = dayjs(Number(s.startTs) * 1000);
          return sStart.isSame(dayStart, "day") || (sStart.isAfter(dayStart) && sStart.isBefore(dayEnd));
        })
        .sort((a, b) => Number(a.startTs) - Number(b.startTs));

      // Start from 8 AM or current time if today
      const now = dayjs();
      let searchStart = dayStart.hour(8).minute(0);
      if (now.isSame(dayStart, "day") && now.isAfter(searchStart)) {
        // Round up to next 30 minutes
        searchStart = now.minute(Math.ceil(now.minute() / 30) * 30).second(0);
      }

      const durationMinutes = duration;
      const searchEnd = dayEnd.hour(22).minute(0); // Search until 10 PM

      // Find gaps between schedules
      let currentStart = searchStart;

      for (const schedule of daySchedules) {
        const sStart = dayjs(Number(schedule.startTs) * 1000);
        const sEnd = dayjs(Number(schedule.endTs) * 1000);

        // Skip schedules that end before or at our search start
        // Using [start, end) convention: if sEnd <= currentStart, no conflict
        if (sEnd.isBefore(currentStart) || sEnd.isSame(currentStart)) {
          continue;
        }

        // Skip schedules that start at or after our search end
        if (sStart.isSame(searchEnd) || sStart.isAfter(searchEnd)) {
          continue;
        }

        // Check if there's a gap before this schedule
        // Gap exists when: currentStart < sStart
        if (currentStart.isBefore(sStart)) {
          const gapMinutes = sStart.diff(currentStart, "minute");
          if (gapMinutes >= durationMinutes) {
            const slotEnd = currentStart.add(durationMinutes, "minute");
            slots.push({
              startTs: BigInt(Math.floor(currentStart.unix())),
              endTs: BigInt(Math.floor(slotEnd.unix())),
              label: `${currentStart.format("HH:mm")} - ${slotEnd.format("HH:mm")}`,
              reason: (t?.("schedule.conflict.no-conflict-slot") as string) || "无冲突时段",
            });
          }
        } else {
          // currentStart >= sStart: there's a conflict, skip this schedule
          // Move to after this schedule ends
        }

        // Move current start to after this schedule ends
        // Using [start, end) convention: next available time is sEnd
        currentStart = sEnd;
      }

      // Check if there's a gap after the last schedule
      const remainingMinutes = searchEnd.diff(currentStart, "minute");
      if (remainingMinutes >= durationMinutes) {
        const slotEnd = currentStart.add(durationMinutes, "minute");
        slots.push({
          startTs: BigInt(Math.floor(currentStart.unix())),
          endTs: BigInt(Math.floor(slotEnd.unix())),
          label: `${currentStart.format("HH:mm")} - ${slotEnd.format("HH:mm")}`,
          reason: remainingMinutes === durationMinutes
            ? (t?.("schedule.conflict.end-slot") as string || "末尾时段")
            : (t?.("schedule.conflict.no-conflict-slot") as string || "无冲突时段"),
        });
      }

      return slots.slice(0, 3); // Return max 3 suggestions
    };
  }, [existingSchedules, t]);

  const suggestions = useMemo(() => {
    if (!startTs || !endTs) {
      return [];
    }

    const duration = dayjs(Number(endTs) * 1000).diff(dayjs(Number(startTs) * 1000), "minute");
    const startDate = dayjs(Number(startTs) * 1000);

    // Find slots on the same day
    const sameDaySlots = findAvailableSlots(startDate.toDate(), duration);

    // If no slots on same day, try next day
    if (sameDaySlots.length === 0) {
      const nextDaySlots = findAvailableSlots(startDate.add(1, "day").toDate(), duration);
      return nextDaySlots.map((slot) => ({
        ...slot,
        reason: (t?.("schedule.conflict.next-day-available") as string) || "次日可用",
      }));
    }

    return sameDaySlots;
  }, [startTs, endTs, findAvailableSlots, t]);

  return {
    conflicts,
    hasConflicts,
    suggestions,
    findAvailableSlots,
  };
}

/**
 * Check if two time ranges overlap
 */
export function checkOverlap(start1: bigint | number, end1: bigint | number, start2: bigint | number, end2: bigint | number): boolean {
  const s1 = typeof start1 === "bigint" ? Number(start1) : start1;
  const e1 = typeof end1 === "bigint" ? Number(end1) : end1;
  const s2 = typeof start2 === "bigint" ? Number(start2) : start2;
  const e2 = typeof end2 === "bigint" ? Number(end2) : end2;

  return s1 < e2 && e1 > s2;
}

/**
 * Calculate the overlap duration in minutes
 */
export function getOverlapDuration(start1: bigint | number, end1: bigint | number, start2: bigint | number, end2: bigint | number): number {
  if (!checkOverlap(start1, end1, start2, end2)) {
    return 0;
  }

  const s1 = typeof start1 === "bigint" ? Number(start1) : start1;
  const e1 = typeof end1 === "bigint" ? Number(end1) : end1;
  const s2 = typeof start2 === "bigint" ? Number(start2) : start2;
  const e2 = typeof end2 === "bigint" ? Number(end2) : end2;

  const overlapStart = Math.max(s1, s2);
  const overlapEnd = Math.min(e1, e2);

  return Math.max(0, overlapEnd - overlapStart) / 60; // Convert to minutes
}
