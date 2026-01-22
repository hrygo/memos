import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useScheduleAgentChat } from "@/hooks/useScheduleQueries";
import { localParser } from "../services/localParser";
import type { Confidence, ParsedSchedule, ParseResult, ParseSource } from "../types";

interface UseScheduleParseOptions {
  /** Debounce delay in milliseconds (default: 800ms) */
  debounceMs?: number;
  /** Minimum input length before parsing */
  minLength?: number;
  /** Whether to enable AI parsing as fallback */
  enableAI?: boolean;
  /** Reference date for relative time calculations */
  referenceDate?: Date;
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
  /** Parse input manually (ignores debounce) */
  parse: (input: string) => Promise<void>;
  /** Reset parse state */
  reset: () => void;
}

/**
 * Hook for parsing schedule input with local-first strategy and AI fallback.
 * Implements debouncing to reduce API calls.
 */
export function useScheduleParse(options: UseScheduleParseOptions = {}): UseScheduleParseReturn {
  const { debounceMs = 800, minLength = 2, enableAI = true, referenceDate = new Date() } = options;

  const [parseResult, setParseResult] = useState<ParseResult | null>(null);
  const [isParsing, setIsParsing] = useState(false);
  const [parseSource, setParseSource] = useState<ParseSource | null>(null);
  const [confidence, setConfidence] = useState<Confidence | null>(null);

  const agentChat = useScheduleAgentChat();
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const abortControllerRef = useRef<AbortController | null>(null);

  // Store reference date as timestamp for stable dependency
  const referenceDateTs = useMemo(() => referenceDate.getTime(), [referenceDate]);

  // Cleanup debounce timer on unmount
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
   * Attempt local parsing first
   */
  const parseLocally = useCallback(
    (input: string): ParseResult => {
      const refDate = new Date(referenceDateTs);
      return localParser.parse(input, refDate);
    },
    [referenceDateTs],
  );

  /**
   * Attempt AI parsing as fallback
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

        // Check if AI successfully parsed and created a schedule
        const createdSchedule =
          response.response?.includes("已成功创建") ||
          response.response?.includes("成功创建日程") ||
          response.response?.includes("successfully created");

        if (createdSchedule) {
          return { state: "success", message: response.response };
        }

        // AI is asking for clarification
        return {
          state: "partial",
          message: response.response || "需要更多信息",
        };
      } catch (error) {
        console.error("[useScheduleParse] AI parse error:", error);
        return {
          state: "error",
          message: "智能解析失败，请重试或手动输入",
        };
      }
    },
    [enableAI, agentChat],
  );

  /**
   * Parse input with debouncing
   */
  const parse = useCallback(
    async (input: string) => {
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

      // Debounce the parse
      debounceTimerRef.current = setTimeout(async () => {
        // Cancel any pending AI request
        if (abortControllerRef.current) {
          abortControllerRef.current.abort();
        }
        abortControllerRef.current = new AbortController();

        try {
          // Try local parsing first
          const localResult = parseLocally(input);

          if (localResult.state === "success" && localResult.parsedSchedule) {
            const localConfidence = localParser.getConfidence(input, localResult.parsedSchedule);

            // High confidence local result - use it directly
            if (localConfidence >= 0.8) {
              setParseResult(localResult);
              setParseSource("local");
              setConfidence(localConfidence);
              setIsParsing(false);
              return;
            }

            // Medium confidence - still use local, but flag as partial
            if (localConfidence >= 0.5) {
              setParseResult({
                ...localResult,
                state: "partial",
              });
              setParseSource("local");
              setConfidence(localConfidence);
              setIsParsing(false);
              return;
            }
          }

          // Low confidence or failed local parse - try AI if enabled
          if (enableAI) {
            const aiResult = await parseWithAI(input);
            setParseResult(aiResult);
            setParseSource("ai");
            setConfidence(aiResult.parsedSchedule?.confidence || null);
          } else {
            // AI disabled, use local result even with low confidence
            setParseResult(
              localResult.state === "idle"
                ? {
                    state: "partial",
                    parsedSchedule: localResult.parsedSchedule,
                    message: "请确认时间信息",
                  }
                : localResult,
            );
            setParseSource("local");
            setConfidence(localResult.parsedSchedule?.confidence || 0.3);
          }
        } catch (error) {
          // Final fallback - show manual input option
          if (error instanceof Error && error.name === "AbortError") {
            return; // Request was cancelled, ignore
          }

          setParseResult({
            state: "error",
            message: "解析失败，请手动输入",
          });
          setParseSource(null);
          setConfidence(null);
        } finally {
          setIsParsing(false);
        }
      }, debounceMs);
    },
    [debounceMs, minLength, enableAI, parseLocally, parseWithAI],
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
 * Extract schedule from parse result for creating
 */
export function extractScheduleFromParse(parseResult: ParseResult | null, defaultTitle?: string): Partial<ParsedSchedule> | null {
  if (!parseResult?.parsedSchedule) {
    return null;
  }

  const parsed = parseResult.parsedSchedule;

  return {
    title: parsed.title || defaultTitle || "新日程",
    startTs: parsed.startTs,
    endTs: parsed.endTs,
    allDay: parsed.allDay || false,
    location: parsed.location,
    description: parsed.description,
    reminders: parsed.reminders,
  };
}
