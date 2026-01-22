import { useCallback, useEffect, useRef, useState } from "react";
import { useScheduleAgentChat } from "@/hooks/useScheduleQueries";
import { useTranslate } from "@/utils/i18n";
import type { Confidence, ParsedSchedule, ParseResult, ParseSource } from "../types";

interface UseScheduleParseOptions {
  /** Debounce delay in milliseconds (default: 800ms) */
  debounceMs?: number;
  /** Minimum input length before parsing */
  minLength?: number;
  /** Whether to enable AI parsing */
  enableAI?: boolean;
  /** Reference date for relative time calculations */
  referenceDate?: Date;
  /** Whether to automatically parse on input changes (default: true) */
  autoParse?: boolean;
  /** Callback when AI successfully creates a schedule */
  onScheduleCreated?: () => void;
}

interface UseScheduleParseReturn {
  /** Current parse result */
  parseResult: ParseResult | null;
  /** Whether currently parsing */
  isParsing: boolean;
  /** Source of the last successful parse */
  parseSource: ParseSource | null;
  /** Confidence of the last successful parse */
  confidence: Confidence | null;
  /** Parse input with AI agent (creates schedule directly) */
  parse: (input: string, forceAI?: boolean) => Promise<void>;
  /** Reset parse state */
  reset: () => void;
}

/**
 * Hook for parsing schedule input using AI Agent.
 * The Agent directly creates the schedule - no separate creation step needed.
 */
export function useScheduleParse(options: UseScheduleParseOptions = {}): UseScheduleParseReturn {
  const { debounceMs = 800, minLength = 2, enableAI = true, autoParse = true, onScheduleCreated } = options;

  const t = useTranslate();
  const [parseResult, setParseResult] = useState<ParseResult | null>(null);
  const [isParsing, setIsParsing] = useState(false);
  const [parseSource, setParseSource] = useState<ParseSource | null>(null);
  const [confidence, setConfidence] = useState<Confidence | null>(null);

  const agentChat = useScheduleAgentChat();
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const abortControllerRef = useRef<AbortController | null>(null);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, []);

  /**
   * Parse input using AI Agent - Agent directly creates the schedule
   */
  const parseWithAI = useCallback(
    async (input: string): Promise<ParseResult> => {
      if (!enableAI) {
        return { state: "error", message: "AI parsing is disabled" };
      }

      try {
        const response = await agentChat.mutateAsync({
          message: input,
          userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Shanghai",
        });

        // Check if AI successfully created the schedule
        const createdSchedule =
          response.response?.includes("已成功创建") ||
          response.response?.includes("成功创建日程") ||
          response.response?.includes("successfully created") ||
          response.response?.includes("已安排") ||
          response.response?.includes("已为您创建");

        if (createdSchedule) {
          // AI already created the schedule - notify parent to refresh
          onScheduleCreated?.();
          return {
            state: "created",
            message: response.response || (t("schedule.quick-input.schedule-created") as string),
          };
        }

        // AI is asking for clarification or suggesting options
        return {
          state: "partial",
          message: response.response || (t("schedule.need-more-info-message") as string),
        };
      } catch (error) {
        console.error("[useScheduleParse] AI parse error:", error);
        return {
          state: "error",
          message: t("schedule.ai-parse-failed") as string,
        };
      }
    },
    [enableAI, agentChat, onScheduleCreated, t],
  );

  /**
   * Parse input with debouncing
   */
  const parse = useCallback(
    async (input: string, forceAI = false) => {
      // Clear previous timer
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }

      // Reset state if input is empty
      if (!input.trim() || input.length < minLength) {
        setParseResult(null);
        setParseSource(null);
        setConfidence(null);
        setIsParsing(false);
        return;
      }

      setIsParsing(true);

      // When forceAI or !autoParse, skip debounce
      const delay = autoParse || forceAI ? debounceMs : 0;

      const executeParse = async () => {
        // Cancel any pending request
        if (abortControllerRef.current) {
          abortControllerRef.current.abort();
        }
        abortControllerRef.current = new AbortController();

        try {
          // Always use AI for schedule creation
          if (enableAI && (autoParse || forceAI)) {
            const aiResult = await parseWithAI(input);
            setParseResult(aiResult);
            setParseSource("ai");
            setConfidence(null);
          } else {
            setParseResult({
              state: "idle",
              message: t("schedule.please-input-schedule-content") as string,
            });
            setParseSource(null);
            setConfidence(null);
          }
        } catch (error) {
          if (error instanceof Error && error.name === "AbortError") {
            return; // Request was cancelled
          }
          setParseResult({
            state: "error",
            message: t("schedule.parse-failed-message") as string,
          });
          setParseSource(null);
          setConfidence(null);
        } finally {
          setIsParsing(false);
        }
      };

      if (delay === 0) {
        await executeParse();
      } else {
        debounceTimerRef.current = setTimeout(executeParse, delay);
      }
    },
    [debounceMs, minLength, enableAI, autoParse, parseWithAI, t],
  );

  /**
   * Reset parse state
   */
  const reset = useCallback(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    setParseResult(null);
    setParseSource(null);
    setConfidence(null);
    setIsParsing(false);
  }, []);

  return {
    parseResult,
    isParsing,
    parseSource,
    confidence,
    parse,
    reset,
  };
}

/**
 * @deprecated No longer needed - AI creates schedules directly
 */
export function extractScheduleFromParse(_parseResult: ParseResult | null, _defaultTitle?: string): Partial<ParsedSchedule> | null {
  return null;
}
