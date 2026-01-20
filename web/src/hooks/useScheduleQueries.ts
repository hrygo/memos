import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { scheduleServiceClient } from "@/connect";
import type {
  CheckConflictRequest,
  ListSchedulesRequest,
  ParseAndCreateScheduleRequest,
  Schedule,
} from "@/types/proto/api/v1/schedule_service_pb";
import {
  CheckConflictRequestSchema,
  ListSchedulesRequestSchema,
  ParseAndCreateScheduleRequestSchema,
} from "@/types/proto/api/v1/schedule_service_pb";

// Query keys factory for consistent cache management
export const scheduleKeys = {
  all: ["schedules"] as const,
  lists: () => [...scheduleKeys.all, "list"] as const,
  list: (filters: Partial<ListSchedulesRequest>) => [...scheduleKeys.lists(), filters] as const,
  details: () => [...scheduleKeys.all, "detail"] as const,
  detail: (name: string) => [...scheduleKeys.details(), name] as const,
  conflicts: () => [...scheduleKeys.all, "conflicts"] as const,
};

/**
 * Hook to fetch schedules with optional filters
 */
export function useSchedules(request: Partial<ListSchedulesRequest> = {}) {
  return useQuery({
    queryKey: scheduleKeys.list(request),
    queryFn: async () => {
      const response = await scheduleServiceClient.listSchedules(create(ListSchedulesRequestSchema, request as Record<string, unknown>));
      return response;
    },
    staleTime: 1000 * 30, // 30 seconds (optimized for multi-user sync)
  });
}

/**
 * Hook to fetch schedules with time range optimization (today ± 15 days)
 */
export function useSchedulesOptimized(anchorDate?: Date) {
  // Calculate time range: anchorDate ± 15 days
  const now = anchorDate || new Date();
  const startOfRange = new Date(now);
  startOfRange.setDate(now.getDate() - 15);
  startOfRange.setHours(0, 0, 0, 0);

  const endOfRange = new Date(now);
  endOfRange.setDate(now.getDate() + 15);
  endOfRange.setHours(23, 59, 59, 999);

  // Convert to Unix timestamps (seconds)
  const startTs = BigInt(Math.floor(startOfRange.getTime() / 1000));
  const endTs = BigInt(Math.floor(endOfRange.getTime() / 1000));

  return useSchedules({
    startTs,
    endTs,
  });
}

/**
 * Hook to fetch a single schedule by name
 */
export function useSchedule(name: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: scheduleKeys.detail(name),
    queryFn: async () => {
      const schedule = await scheduleServiceClient.getSchedule({ name });
      return schedule;
    },
    enabled: options?.enabled ?? true,
    staleTime: 1000 * 30, // 30 seconds (should be >= list cache time for consistency)
  });
}

/**
 * Hook to create a new schedule
 */
export function useCreateSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (scheduleToCreate: Partial<Schedule>) => {
      // API expects a Schedule message, usually we should construct it properly if nested messages are involved
      const response = await scheduleServiceClient.createSchedule({
        schedule: scheduleToCreate as Schedule, // Cast is safe here as connect-web handles partials mostly, or we should use create(ScheduleSchema, ...)
      });
      return response;
    },
    onSuccess: (newSchedule) => {
      // Invalidate schedule lists to refetch
      queryClient.invalidateQueries({ queryKey: scheduleKeys.lists() });
      // Add new schedule to cache
      queryClient.setQueryData(scheduleKeys.detail(newSchedule.name), newSchedule);
    },
  });
}

/**
 * Hook to update a schedule
 */
export function useUpdateSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ schedule, updateMask }: { schedule: Partial<Schedule>; updateMask: string[] }) => {
      const updated = await scheduleServiceClient.updateSchedule({
        schedule: schedule as Schedule,
        updateMask: { paths: updateMask },
      });
      return updated;
    },
    onMutate: async ({ schedule }) => {
      if (!schedule.name) {
        return { previousSchedule: undefined };
      }

      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: scheduleKeys.detail(schedule.name) });

      // Snapshot previous value
      const previousSchedule = queryClient.getQueryData<Schedule>(scheduleKeys.detail(schedule.name));

      // Optimistically update
      if (previousSchedule) {
        // We can't easily merge partial schedule locally without logic
        // For now just setting it directly assuming it's substantial, or better just invalidate on success
        // queryClient.setQueryData(scheduleKeys.detail(schedule.name), schedule as Schedule);
      }

      return { previousSchedule };
    },
    onError: (_err, { schedule }, context) => {
      // Rollback on error
      if (context?.previousSchedule && schedule.name) {
        queryClient.setQueryData(scheduleKeys.detail(schedule.name), context.previousSchedule);
      }
    },
    onSuccess: (updatedSchedule) => {
      // Update cache with server response
      queryClient.setQueryData(scheduleKeys.detail(updatedSchedule.name), updatedSchedule);
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: scheduleKeys.lists() });
    },
  });
}

/**
 * Hook to delete a schedule
 */
export function useDeleteSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (name: string) => {
      await scheduleServiceClient.deleteSchedule({ name });
      return name;
    },
    onSuccess: (name) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: scheduleKeys.detail(name) });
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: scheduleKeys.lists() });
    },
  });
}

/**
 * Hook to check for schedule conflicts
 */
export function useCheckConflict() {
  return useMutation({
    mutationFn: async (request: Partial<CheckConflictRequest>) => {
      const response = await scheduleServiceClient.checkConflict(create(CheckConflictRequestSchema, request as Record<string, unknown>));
      return response;
    },
  });
}

/**
 * Hook to parse natural language and optionally create a schedule
 */
export function useParseAndCreateSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: Partial<ParseAndCreateScheduleRequest>) => {
      const response = await scheduleServiceClient.parseAndCreateSchedule(
        create(ParseAndCreateScheduleRequestSchema, request as Record<string, unknown>),
      );
      return response;
    },
    onSuccess: (response) => {
      // If a schedule was created, invalidate cache
      if (response.createdSchedule) {
        queryClient.invalidateQueries({ queryKey: scheduleKeys.lists() });
        queryClient.setQueryData(scheduleKeys.detail(response.createdSchedule.name), response.createdSchedule);
      }
    },
  });
}
