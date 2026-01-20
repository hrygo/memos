import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { aiServiceClient } from "@/connect";
import {
  ChatWithMemosRequestSchema,
  GetRelatedMemosRequestSchema,
  SemanticSearchRequestSchema,
  SuggestTagsRequestSchema,
} from "@/types/proto/api/v1/ai_service_pb";

// Query keys factory for consistent cache management
export const aiKeys = {
  all: ["ai"] as const,
  search: () => [...aiKeys.all, "search"] as const,
  searchQuery: (query: string) => [...aiKeys.search(), query] as const,
  related: (name: string) => [...aiKeys.all, "related", name] as const,
};

/**
 * useSemanticSearch performs semantic search on memos.
 * @param query - Search query string
 * @param options.enabled - Whether the query is enabled
 */
export function useSemanticSearch(query: string, options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: aiKeys.searchQuery(query),
    queryFn: async () => {
      const request = create(SemanticSearchRequestSchema, {
        query,
        limit: 10,
      });
      return await aiServiceClient.semanticSearch(request);
    },
    enabled: (options.enabled ?? true) && query.length > 2,
    staleTime: 60 * 1000, // 1 minute
  });
}

/**
 * useSuggestTags suggests tags for memo content using AI.
 */
export function useSuggestTags() {
  return useMutation({
    mutationFn: async (params: { content: string; limit?: number }) => {
      const request = create(SuggestTagsRequestSchema, {
        content: params.content,
        limit: params.limit ?? 5,
      });
      const response = await aiServiceClient.suggestTags(request);
      return response.tags;
    },
  });
}

/**
 * useRelatedMemos finds memos related to a specific memo.
 * @param name - Memo name in format "memos/{uid}"
 * @param options.enabled - Whether the query is enabled
 * @param options.limit - Maximum number of related memos to return
 */
export function useRelatedMemos(name: string, options: { enabled?: boolean; limit?: number } = {}) {
  return useQuery({
    queryKey: aiKeys.related(name),
    queryFn: async () => {
      const request = create(GetRelatedMemosRequestSchema, {
        name,
        limit: options.limit ?? 5,
      });
      return await aiServiceClient.getRelatedMemos(request);
    },
    enabled: (options.enabled ?? true) && !!name && name.startsWith("memos/"),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * useChatWithMemos streams a chat response using memos as context.
 * Uses Connect RPC streaming to receive responses in real-time.
 *
 * @returns An object with stream function and callbacks
 */
export function useChatWithMemos() {
  const queryClient = useQueryClient();

  return {
    /**
     * Stream chat with memos as context.
     * @param params - Chat parameters
     * @param callbacks - Optional callbacks for streaming events
     * @returns A promise that resolves when streaming completes
     */
    stream: async (
      params: { message: string; history?: string[] },
      callbacks?: {
        onContent?: (content: string) => void;
        onSources?: (sources: string[]) => void;
        onDone?: () => void;
        onError?: (error: Error) => void;
        onScheduleIntent?: (intent: { detected: boolean; scheduleDescription: string }) => void;
        onScheduleQueryResult?: (result: {
          detected: boolean;
          schedules: Array<{
            uid: string;
            title: string;
            startTs: bigint;
            endTs: bigint;
            allDay: boolean;
            location: string;
            recurrenceRule: string;
            status: string;
          }>;
          timeRangeDescription: string;
          queryType: string;
        }) => void;
      },
    ) => {
      const request = create(ChatWithMemosRequestSchema, {
        message: params.message,
        history: params.history ?? [],
      });

      try {
        // Use the streaming method from Connect RPC client
        const stream = aiServiceClient.chatWithMemos(request);

        const sources: string[] = [];
        let fullContent = "";
        let doneCalled = false;

        for await (const response of stream) {
          // Handle sources (sent in first response)
          if (response.sources.length > 0) {
            sources.push(...response.sources);
            callbacks?.onSources?.(response.sources);
          }

          // Handle content chunks
          if (response.content) {
            fullContent += response.content;
            callbacks?.onContent?.(response.content);
          }

          // Handle schedule creation intent (sent in final chunk)
          if (response.scheduleCreationIntent?.detected) {
            callbacks?.onScheduleIntent?.({
              detected: response.scheduleCreationIntent.detected,
              scheduleDescription: response.scheduleCreationIntent.scheduleDescription || "",
            });
          }

          // Handle schedule query result (sent in final chunk)
          if (response.scheduleQueryResult?.detected) {
            const schedules = (response.scheduleQueryResult.schedules || []).map((sched) => ({
              uid: sched.uid || "",
              title: sched.title || "",
              startTs: sched.startTs || BigInt(0),
              endTs: sched.endTs || BigInt(0),
              allDay: sched.allDay || false,
              location: sched.location || "",
              recurrenceRule: sched.recurrenceRule || "",
              status: sched.status || "ACTIVE",
            }));

            callbacks?.onScheduleQueryResult?.({
              detected: response.scheduleQueryResult.detected,
              schedules,
              timeRangeDescription: response.scheduleQueryResult.timeRangeDescription || "",
              queryType: response.scheduleQueryResult.queryType || "",
            });
          }

          // Handle completion
          if (response.done === true) {
            doneCalled = true;
            callbacks?.onDone?.();
            break;
          }
        }

        // Fallback: if stream ended without done signal, call onDone
        if (!doneCalled) {
          callbacks?.onDone?.();
        }

        return { content: fullContent, sources };
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));
        callbacks?.onError?.(err);
        throw err;
      }
    },
    /**
     * Invalidate AI-related queries after chat
     */
    invalidate: () => {
      queryClient.invalidateQueries({ queryKey: aiKeys.all });
    },
  };
}

// Type exports for convenience
export type SemanticSearchResult = Awaited<ReturnType<typeof aiServiceClient.semanticSearch>>;
export type SuggestTagsResult = string[];
export type RelatedMemosResult = Awaited<ReturnType<typeof aiServiceClient.getRelatedMemos>>;
