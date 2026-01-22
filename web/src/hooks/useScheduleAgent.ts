import { useMutation, useQueryClient } from "@tanstack/react-query";
import { scheduleAgentServiceClient } from "@/connect";

/**
 * Hook to chat with Schedule Agent (non-streaming)
 */
export function useScheduleAgentChat() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: { message: string; userTimezone?: string }) => {
      const response = await scheduleAgentServiceClient.chat({
        message: request.message,
        userTimezone: request.userTimezone || "Asia/Shanghai",
      });
      return response;
    },
    onSuccess: () => {
      // Invalidate schedule lists to refetch
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
    },
  });
}

/**
 * Hook to chat with Schedule Agent (streaming)
 * Returns an async generator that yields stream events
 */
export async function* scheduleAgentChatStream(
  message: string,
  userTimezone = "Asia/Shanghai",
  onEvent?: (event: { type: string; data: string }) => void,
): AsyncGenerator<{ type: string; data: string; content?: string; done?: boolean }, void> {
  const response = await scheduleAgentServiceClient.chatStream({
    message,
    userTimezone,
  });

  for await (const chunk of response) {
    // Parse the event JSON
    if (chunk.event) {
      try {
        const event = JSON.parse(chunk.event);
        onEvent?.(event);
        yield event;
      } catch (e) {
        console.error("Failed to parse event:", chunk.event);
      }
    }

    // Yield the raw chunk for compatibility
    yield {
      type: chunk.event ? "raw" : "data",
      data: chunk.event || "",
      content: chunk.content,
      done: chunk.done,
    };
  }
}
