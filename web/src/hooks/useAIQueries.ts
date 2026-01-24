import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { aiServiceClient } from "@/connect";
import { ParrotAgentType, parrotToProtoAgentType } from "@/types/parrot";
import {
  ChatRequestSchema,
  GetRelatedMemosRequestSchema,
  SemanticSearchRequestSchema,
  SuggestTagsRequestSchema,
} from "@/types/proto/api/v1/ai_service_pb";

// Default timeout for streaming AI requests (5 minutes)
const STREAM_TIMEOUT_MS = 5 * 60 * 1000;

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
 * useChat streams a chat response using AI agents.
 * Uses Connect RPC streaming to receive responses in real-time.
 *
 * @returns An object with stream function and callbacks
 */
export function useChat() {
  const queryClient = useQueryClient();

  return {
    /**
     * Stream chat with memos as context.
     * @param params - Chat parameters
     * @param callbacks - Optional callbacks for streaming events
     * @returns A promise that resolves when streaming completes
     */
    stream: async (
      params: { message: string; history?: string[]; agentType?: ParrotAgentType; userTimezone?: string },
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
        // Parrot-specific callbacks
        onThinking?: (message: string) => void;
        onToolUse?: (toolName: string) => void;
        onToolResult?: (result: string) => void;
        onMemoQueryResult?: (result: {
          memos: Array<{ uid: string; content: string; score: number }>;
          query: string;
          count: number;
        }) => void;
      },
    ) => {
      const request = create(ChatRequestSchema, {
        message: params.message,
        history: params.history ?? [],
        agentType: params.agentType !== undefined ? parrotToProtoAgentType(params.agentType) : undefined,
        userTimezone: params.userTimezone,
      });

      console.debug("[AI Chat] Starting stream", {
        messageLength: params.message.length,
        agentType: params.agentType,
        historyCount: params.history?.length ?? 0,
      });

      // Set up timeout for the entire stream operation
      const timeoutController = new AbortController();
      const timeoutId = setTimeout(() => {
        timeoutController.abort();
        console.warn("[AI Chat] Stream timeout exceeded", { timeoutMs: STREAM_TIMEOUT_MS });
      }, STREAM_TIMEOUT_MS);

      const startTime = Date.now();

      try {
        // Use the streaming method from Connect RPC client
        const stream = aiServiceClient.chat(request);

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
              startTs: sched.startTs ? BigInt(sched.startTs) : BigInt(0),
              endTs: sched.endTs ? BigInt(sched.endTs) : BigInt(0),
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

          // Handle parrot-specific events
          if (response.eventType && response.eventData) {
            console.debug("[AI Chat] Parrot event", {
              eventType: response.eventType,
              eventDataLength: response.eventData.length,
              eventDataPreview: response.eventData.slice(0, 100),
            });
            switch (response.eventType) {
              case "thinking":
                callbacks?.onThinking?.(response.eventData);
                break;
              case "tool_use":
                callbacks?.onToolUse?.(response.eventData);
                break;
              case "tool_result":
                callbacks?.onToolResult?.(response.eventData);
                break;
              case "answer":
                // Handle final answer from agent (when no tool is used)
                fullContent += response.eventData;
                callbacks?.onContent?.(response.eventData);
                break;
              case "memo_query_result":
                try {
                  const result = JSON.parse(response.eventData);
                  callbacks?.onMemoQueryResult?.(result);
                } catch (e) {
                  console.error("Failed to parse memo_query_result:", e);
                }
                break;
              case "schedule_query_result":
                try {
                  const result = JSON.parse(response.eventData);
                  // Transform to the expected format with bigint conversion
                  const transformedResult = {
                    detected: true,
                    schedules: (result.schedules || []).map((s: { uid: string; title: string; start_ts: number; end_ts: number; all_day: boolean; location?: string; status: string }) => ({
                      uid: s.uid || "",
                      title: s.title || "",
                      startTs: BigInt(s.start_ts || 0),
                      endTs: BigInt(s.end_ts || 0),
                      allDay: s.all_day || false,
                      location: s.location || "",
                      recurrenceRule: "",
                      status: s.status || "ACTIVE",
                    })),
                    timeRangeDescription: result.time_range_description || "",
                    queryType: result.query_type || "range",
                  };
                  callbacks?.onScheduleQueryResult?.(transformedResult);
                } catch (e) {
                  console.error("Failed to parse schedule_query_result:", e);
                }
                break;
            }
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

        // Clear timeout on successful completion
        clearTimeout(timeoutId);

        const duration = Date.now() - startTime;
        console.debug("[AI Chat] Stream completed successfully", {
          durationMs: duration,
          contentLength: fullContent.length,
          sourcesCount: sources.length,
        });

        return { content: fullContent, sources };
      } catch (error) {
        // Clear timeout on error
        clearTimeout(timeoutId);

        const duration = Date.now() - startTime;

        // Check if it's an abort error (timeout)
        if (error instanceof Error && error.name === "AbortError") {
          console.error("[AI Chat] Stream timeout", { durationMs: duration, timeoutMs: STREAM_TIMEOUT_MS });
          const timeoutErr = new Error(`AI chat timeout after ${STREAM_TIMEOUT_MS}ms`);
          callbacks?.onError?.(timeoutErr);
          throw timeoutErr;
        }

        console.error("[AI Chat] Stream error", {
          error,
          durationMs: duration,
          errorMessage: error instanceof Error ? error.message : String(error),
        });

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
