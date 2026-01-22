import dayjs from "dayjs";
import type { Confidence, ParsedSchedule, ParseResult } from "../types";

/** Default duration in minutes when not specified */
const DEFAULT_DURATION_MINUTES = 60;

/** Minimum confidence threshold for auto-accepting local parse results */
const MIN_CONFIDENCE_THRESHOLD = 0.8;

/** Time patterns for extraction */
const TIME_PATTERNS = [
  // "明天3点" "明天下午3点"
  {
    pattern: /(今天|明日?|昨天|昨日)?(早上|早晨|上午|中午|下午|晚上|夜里)?(\d{1,2})(点|:)(\d{1,2})?/,
    confidence: 0.95,
  },
  // "3点" "下午3点"
  {
    pattern: /^(早上|早晨|上午|中午|下午|晚上|夜里)?(\d{1,2})(点|:)(\d{1,2})?/,
    confidence: 0.9,
  },
  // "30分钟后" "2小时后"
  {
    pattern: /^(\d+)(分钟|小时|天|周)(后|之?后)?/,
    confidence: 0.85,
  },
  // "明天全天"
  {
    pattern: /^(今天|明日?|后天|本周|下周|本周)(全天|一天|整天)$/,
    confidence: 0.95,
  },
];

class LocalParser {
  private cache: Map<string, { result: ParsedSchedule; timestamp: number }> = new Map();
  private cacheExpiry = 5 * 60 * 1000; // 5 minutes

  /**
   * Parse natural language input to schedule
   */
  parse(input: string, referenceDate: Date = new Date()): ParseResult {
    if (!input?.trim()) {
      return { state: "idle" };
    }

    // Check cache first
    const cached = this.getFromCache(input);
    if (cached) {
      return { state: "success", parsedSchedule: cached };
    }

    const trimmedInput = input.trim().toLowerCase();

    // Try each pattern
    for (const { pattern, confidence } of TIME_PATTERNS) {
      const match = trimmedInput.match(pattern);
      if (match) {
        const result = this.extractTimeFromMatch(match, referenceDate);
        if (result) {
          // Extract title from input (everything that's not the time part)
          const titleMatch = input.match(/^(.+?)(?=今天|明天|昨天|早上|上午|下午|晚上|\d+点|\d+:\d+|\d+分钟|\d+小时)/);
          const title = titleMatch ? titleMatch[1].trim() : input.replace(pattern, "").trim() || "新日程";

          const parsedSchedule: ParsedSchedule = {
            title,
            startTs: BigInt(result.startTs),
            endTs: BigInt(result.endTs),
            allDay: result.allDay || false,
            confidence,
            source: "local",
            missingFields: result.missingFields as Array<"title" | "startTime" | "endTime" | "duration">,
          };

          // Cache the result
          this.addToCache(input, parsedSchedule);

          return {
            state: confidence >= MIN_CONFIDENCE_THRESHOLD ? "success" : "partial",
            parsedSchedule,
          };
        }
      }
    }

    // Check for title-only input (no time detected)
    if (trimmedInput.length > 0 && trimmedInput.length < 50) {
      // Use current time rounded to the hour for default timestamps
      const now = new Date();
      const startTs = Math.floor(now.getTime() / 1000);
      const endTs = startTs + DEFAULT_DURATION_MINUTES * 60;

      return {
        state: "partial",
        parsedSchedule: {
          title: input,
          startTs: BigInt(startTs),
          endTs: BigInt(endTs),
          confidence: 0.5,
          source: "local",
          missingFields: ["startTime", "duration"],
        },
        message: '请输入时间，如："明天下午3点" 或 "30分钟后"',
      };
    }

    return { state: "idle" };
  }

  /**
   * Extract time information from regex match
   */
  private extractTimeFromMatch(
    match: RegExpMatchArray,
    referenceDate: Date,
  ): { startTs: number; endTs: number; allDay?: boolean; missingFields?: string[] } | null {
    try {
      const fullMatch = match[0];
      const lowerInput = fullMatch.toLowerCase();

      // Check for duration patterns: "30分钟后" "2小时后"
      const durationMatch = lowerInput.match(/^(\d+)(分钟|小时|天|周)/);
      if (durationMatch) {
        const value = parseInt(durationMatch[1], 10);
        const unit = durationMatch[2];
        let startTs = dayjs(referenceDate);

        if (unit.includes("分钟")) {
          startTs = startTs.add(value, "minute");
        } else if (unit.includes("小时")) {
          startTs = startTs.add(value, "hour");
        } else if (unit.includes("天")) {
          startTs = startTs.add(value, "day");
        } else if (unit.includes("周")) {
          startTs = startTs.add(value, "week");
        }

        return {
          startTs: startTs.unix(),
          endTs: startTs.add(DEFAULT_DURATION_MINUTES, "minute").unix(),
        };
      }

      // Check for all-day patterns: "明天全天"
      if (lowerInput.includes("全天") || lowerInput.includes("一天") || lowerInput.includes("整天")) {
        let targetDate = dayjs(referenceDate);

        if (lowerInput.includes("今天") || lowerInput.includes("今日")) {
          targetDate = dayjs(referenceDate).startOf("day");
        } else if (lowerInput.includes("明天") || lowerInput.includes("明日")) {
          targetDate = dayjs(referenceDate).add(1, "day").startOf("day");
        } else if (lowerInput.includes("后天")) {
          targetDate = dayjs(referenceDate).add(2, "day").startOf("day");
        }

        return {
          startTs: targetDate.unix(),
          endTs: targetDate.endOf("day").unix(),
          allDay: true,
        };
      }

      // Extract time of day
      const dayPart = match[1] ?? match[2] ?? "";
      // For pattern 1: hour at [3], minute at [5]
      // For pattern 2: hour at [2], minute at [4]
      const hourStr = match[3] ?? match[2] ?? "";
      // Minute is at [5] for pattern1, [4] for pattern2, with "0" default
      const minuteStr = match[5] ?? match[4] ?? "0";

      const hour = parseInt(hourStr, 10);
      const minute = parseInt(minuteStr, 10) ?? 0;

      // Validate hour is a valid number
      if (Number.isNaN(hour) || hour < 0 || hour > 23) {
        return null;
      }

      // Calculate date
      let targetDate = dayjs(referenceDate).hour(hour).minute(minute).second(0);

      // Adjust for time of day keywords - use 24-hour format internally
      if (dayPart.includes("早上") || dayPart.includes("早晨") || dayPart.includes("上午")) {
        // Morning: 0-12, keep as-is for user input <= 12
        if (hour > 12) {
          return null; // Invalid input like "早上13点"
        }
      } else if (dayPart.includes("中午")) {
        // Noon: 11-13, convert to 24-hour format
        if (hour <= 12) {
          targetDate = targetDate.hour(hour === 12 ? 12 : hour + 12);
        }
      } else if (dayPart.includes("下午") || dayPart.includes("晚上")) {
        // Afternoon/Evening: add 12 for 1-12pm
        if (hour <= 12) {
          targetDate = targetDate.hour(hour + 12);
        }
        // For evening (晚上), if hour > 18, it's already evening time
        if (dayPart.includes("晚上") && hour > 18 && hour <= 23) {
          targetDate = targetDate.hour(hour);
        }
      }

      // Check for relative day keywords
      const dayKeyword = match[1] || "";
      if (dayKeyword.includes("明天") || dayKeyword.includes("明日")) {
        targetDate = targetDate.add(1, "day");
      } else if (dayKeyword.includes("昨天") || dayKeyword.includes("昨日")) {
        targetDate = targetDate.subtract(1, "day");
      }

      // If time has passed, move to next day
      if (targetDate.isBefore(dayjs(referenceDate))) {
        targetDate = targetDate.add(1, "day");
      }

      return {
        startTs: targetDate.unix(),
        endTs: targetDate.add(DEFAULT_DURATION_MINUTES, "minute").unix(),
      };
    } catch (error) {
      console.error("[LocalParser] Error extracting time:", error);
      return null;
    }
  }

  /**
   * Get confidence score for a parse result
   */
  getConfidence(input: string, parsed: ParsedSchedule): Confidence {
    // Higher confidence for specific time patterns
    if (input.match(/\d+点|\d+:\d+/)) {
      return 0.95;
    }

    // Lower confidence for relative time without specific hour
    if (input.match(/明天|今天/)) {
      return 0.85;
    }

    return parsed.confidence;
  }

  /**
   * Check if parse should be auto-accepted
   */
  shouldAutoAccept(result: ParseResult): boolean {
    if (result.state !== "success" || !result.parsedSchedule) {
      return false;
    }

    return result.parsedSchedule.confidence >= MIN_CONFIDENCE_THRESHOLD;
  }

  /**
   * Get cached result
   */
  private getFromCache(input: string): ParsedSchedule | null {
    const cached = this.cache.get(input);
    if (!cached) return null;

    const now = Date.now();
    if (now - cached.timestamp > this.cacheExpiry) {
      this.cache.delete(input);
      return null;
    }

    return cached.result;
  }

  /**
   * Add result to cache
   */
  private addToCache(input: string, result: ParsedSchedule): void {
    this.cache.set(input, {
      result,
      timestamp: Date.now(),
    });

    // Limit cache size
    if (this.cache.size > 100) {
      const oldestKey = this.cache.keys().next().value;
      if (oldestKey) {
        this.cache.delete(oldestKey);
      }
    }
  }

  /**
   * Clear cache
   */
  clearCache(): void {
    this.cache.clear();
  }
}

// Export singleton instance
export const localParser = new LocalParser();
