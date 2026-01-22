import { useCallback, useState } from "react";
import type { FlowMessage, FlowStep, ParsedSchedule, ScheduleTemplate } from "../types";

interface UseScheduleFlowOptions {
  /** Initial step */
  initialStep?: FlowStep;
  /** Callback when flow is completed */
  onComplete?: (data: Partial<ParsedSchedule>) => void;
  /** Callback when flow is cancelled */
  onCancel?: () => void;
  /** Translate function */
  t?: (key: string) => string | unknown;
}

interface UseScheduleFlowReturn {
  /** Current flow step */
  currentStep: FlowStep;
  /** Conversation history */
  conversation: FlowMessage[];
  /** Parsed schedule data being built */
  scheduleData: Partial<ParsedSchedule>;
  /** Whether flow is active */
  isActive: boolean;
  /** Transition to next step */
  nextStep: (step?: FlowStep) => void;
  /** Go back to previous step */
  prevStep: () => void;
  /** Add message to conversation */
  addMessage: (role: "user" | "assistant", content: string) => void;
  /** Update schedule data */
  updateScheduleData: (data: Partial<ParsedSchedule>) => void;
  /** Reset flow */
  reset: () => void;
  /** Start flow with initial data */
  start: (initialData?: Partial<ParsedSchedule>) => void;
  /** Complete flow */
  complete: () => void;
  /** Cancel flow */
  cancel: () => void;
  /** Apply template */
  applyTemplate: (template: ScheduleTemplate) => void;
}

const STEP_ORDER: FlowStep[] = ["initial", "time-selection", "duration", "location", "confirmation"];

/**
 * Hook for managing progressive schedule creation flow.
 * Handles conversation state and step transitions.
 */
export function useScheduleFlow(options: UseScheduleFlowOptions = {}): UseScheduleFlowReturn {
  const { initialStep = "initial", onComplete, onCancel, t } = options;

  const [currentStep, setCurrentStep] = useState<FlowStep>(initialStep);
  const [conversation, setConversation] = useState<FlowMessage[]>([]);
  const [scheduleData, setScheduleData] = useState<Partial<ParsedSchedule>>({});
  const [isActive, setIsActive] = useState(false);

  const nextStep = useCallback((step?: FlowStep) => {
    setCurrentStep((prev) => {
      if (step) {
        return step;
      }
      const currentIndex = STEP_ORDER.indexOf(prev);
      const nextIndex = Math.min(currentIndex + 1, STEP_ORDER.length - 1);
      return STEP_ORDER[nextIndex];
    });
  }, []);

  const prevStep = useCallback(() => {
    setCurrentStep((prev) => {
      const currentIndex = STEP_ORDER.indexOf(prev);
      const prevIndex = Math.max(currentIndex - 1, 0);
      return STEP_ORDER[prevIndex];
    });
  }, []);

  const addMessage = useCallback((role: "user" | "assistant", content: string) => {
    setConversation((prev) => [
      ...prev,
      {
        role,
        content,
        timestamp: Date.now(),
      },
    ]);
  }, []);

  const updateScheduleData = useCallback((data: Partial<ParsedSchedule>) => {
    setScheduleData((prev) => ({ ...prev, ...data }));
  }, []);

  const reset = useCallback(() => {
    setCurrentStep(initialStep);
    setConversation([]);
    setScheduleData({});
    setIsActive(false);
  }, [initialStep]);

  const start = useCallback((initialData?: Partial<ParsedSchedule>) => {
    setIsActive(true);
    setCurrentStep("initial");
    setConversation([]);
    setScheduleData(initialData || {});
  }, []);

  const complete = useCallback(() => {
    if (Object.keys(scheduleData).length > 0) {
      onComplete?.(scheduleData);
    }
    reset();
  }, [scheduleData, onComplete, reset]);

  const cancel = useCallback(() => {
    onCancel?.();
    reset();
  }, [onCancel, reset]);

  const applyTemplate = useCallback(
    (template: ScheduleTemplate) => {
      const now = Math.floor(Date.now() / 1000);
      const durationSeconds = template.duration * 60;

      updateScheduleData({
        title: template.defaultTitle || template.title,
        startTs: BigInt(now),
        endTs: BigInt(now + durationSeconds),
        description: t
          ? (t("schedule.quick-input.template-applied") as string).replace("{title}", template.title)
          : `来自模板：${template.title}`,
      });

      // Add system message
      addMessage(
        "assistant",
        t
          ? (t("schedule.quick-input.template-applied-message") as string)
              .replace("{title}", template.title)
              .replace("{duration}", String(template.duration))
          : `已应用"${template.title}"模板，时长 ${template.duration} 分钟`,
      );

      // Move to confirmation step
      nextStep("confirmation");
    },
    [updateScheduleData, addMessage, nextStep, t],
  );

  return {
    currentStep,
    conversation,
    scheduleData,
    isActive,
    nextStep,
    prevStep,
    addMessage,
    updateScheduleData,
    reset,
    start,
    complete,
    cancel,
    applyTemplate,
  };
}

/**
 * Determine next step based on missing fields
 */
export function determineNextStep(parsedSchedule: Partial<ParsedSchedule>): FlowStep {
  const missingFields = parsedSchedule.missingFields || [];

  if (missingFields.includes("title") || !parsedSchedule.title) {
    return "initial";
  }

  if (missingFields.includes("startTime")) {
    return "time-selection";
  }

  if (missingFields.includes("duration") || missingFields.includes("endTime")) {
    return "duration";
  }

  // If we have all essential fields, go to confirmation
  return "confirmation";
}

/**
 * Generate prompt for missing information
 */
export function generatePromptForStep(step: FlowStep, scheduleData: Partial<ParsedSchedule>, t?: (key: string) => string | unknown): string {
  switch (step) {
    case "initial":
      return (t?.("schedule.flow.ask-title-time") as string) || "您想创建什么日程？请描述标题和时间。";
    case "time-selection":
      return (t?.("schedule.flow.ask-specific-time") as string) || '请告诉我具体时间，例如："明天下午3点" 或 "后天上午10点"';
    case "duration":
      return (t?.("schedule.flow.ask-duration") as string) || '这个日程需要多长时间？例如："30分钟" 或 "2小时"';
    case "location":
      return (t?.("schedule.flow.ask-location") as string) || "需要设置地点吗？（可选）";
    case "confirmation": {
      const title = scheduleData.title || (t?.("schedule.quick-input.default-title") as string) || "新日程";
      const timeStr = scheduleData.startTs
        ? new Date(Number(scheduleData.startTs) * 1000).toLocaleString("zh-CN", {
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          })
        : (t?.("schedule.time.tbd") as string) || "时间待定";
      return `${(t?.("schedule.quick-input.confirm-create") as string) || "确认创建"}：${title}\n${(t?.("schedule.time.label") as string) || "时间"}：${timeStr}`;
    }
    default:
      return (t?.("schedule.flow.ask-continue") as string) || "请继续...";
  }
}
