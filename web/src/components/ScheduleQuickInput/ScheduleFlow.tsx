import { Bot, Calendar, Clock, MapPin, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { generatePromptForStep } from "./hooks/useScheduleFlow";
import type { FlowMessage, FlowStep, ParsedSchedule } from "./types";
import { useTranslate } from "@/utils/i18n";

interface ScheduleFlowProps {
  /** Current flow step */
  currentStep: FlowStep;
  /** Conversation history */
  conversation: FlowMessage[];
  /** Current schedule data */
  scheduleData: Partial<ParsedSchedule>;
  /** Called when user submits input */
  onSubmit: (input: string) => void;
  /** Called when user confirms creation */
  onConfirm: () => void;
  /** Called when flow is cancelled */
  onCancel: () => void;
  /** Optional className */
  className?: string;
}

export function ScheduleFlow({ currentStep, conversation, scheduleData, onSubmit, onConfirm, onCancel, className }: ScheduleFlowProps) {
  const t = useTranslate();
  const [input, setInput] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Auto-scroll to bottom when conversation updates
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [conversation]);

  // Focus input when step changes
  useEffect(() => {
    if (currentStep !== "confirmation" && inputRef.current) {
      inputRef.current.focus();
    }
  }, [currentStep]);

  const handleSubmit = () => {
    if (input.trim()) {
      onSubmit(input);
      setInput("");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const formatTime = (ts: bigint) => {
    const date = new Date(Number(ts) * 1000);
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);

    const timeStr = date.toLocaleTimeString("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    });

    if (date.toDateString() === today.toDateString()) {
      return `${t("schedule.quick-input.today") as string} ${timeStr}`;
    } else if (date.toDateString() === tomorrow.toDateString()) {
      return `${t("schedule.quick-input.tomorrow") as string} ${timeStr}`;
    }
    return date.toLocaleDateString("zh-CN", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const getStepPrompt = () => {
    return generatePromptForStep(currentStep, scheduleData, t as (key: string) => string | unknown);
  };

  return (
    <div className={cn("flex flex-col", className)}>
      {/* Conversation Area */}
      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto px-4 py-3 space-y-3 max-h-[300px] min-h-[120px]"
        role="log"
        aria-live="polite"
        aria-label="对话历史"
      >
        {/* Conversation Messages */}
        {conversation.map((msg, idx) => (
          <div
            key={idx}
            className={cn("flex gap-2 text-sm", msg.role === "user" ? "justify-end" : "justify-start")}
            role="row"
          >
            {msg.role === "assistant" && (
              <div className="h-6 w-6 rounded-full bg-primary/10 flex items-center justify-center shrink-0 mt-0.5" aria-hidden="true">
                <Bot className="h-3.5 w-3.5 text-primary" />
              </div>
            )}
            <div
              className={cn("max-w-[80%] rounded-2xl px-3 py-2", msg.role === "user" ? "bg-primary text-primary-foreground" : "bg-muted")}
              role={msg.role === "assistant" ? "article" : "status"}
              aria-label={msg.role === "assistant" ? "AI 助手回复" : "您的消息"}
            >
              {msg.role === "assistant" ? (
                <div className="prose prose-sm dark:prose-invert max-w-none">
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm, remarkBreaks]}
                    components={{
                      p: ({ node, ...props }) => <p {...props} className="mb-0 last:mb-0" />,
                    }}
                  >
                    {msg.content}
                  </ReactMarkdown>
                </div>
              ) : (
                <p className="mb-0">{msg.content}</p>
              )}
            </div>
          </div>
        ))}

        {/* Current Step Prompt */}
        {currentStep !== "confirmation" && (
          <div className="flex gap-2 justify-start" role="row">
            <div className="h-6 w-6 rounded-full bg-primary/10 flex items-center justify-center shrink-0 mt-0.5" aria-hidden="true">
              <Bot className="h-3.5 w-3.5 text-primary" />
            </div>
            <div className="max-w-[80%] rounded-2xl px-3 py-2 bg-muted" role="article" aria-label="AI 助手提问">
              <p className="text-sm text-muted-foreground mb-0">{getStepPrompt()}</p>
            </div>
          </div>
        )}
      </div>

      {/* Confirmation Step */}
      {currentStep === "confirmation" && (
        <div role="region" aria-label="确认日程" className="px-4 py-3 border-t border-border/50 bg-muted/30">
          <div className="flex items-start gap-3 p-3 rounded-lg border bg-background" role="group" aria-label="日程详情">
            <div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center shrink-0" aria-hidden="true">
              <Calendar className="h-4 w-4 text-primary" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="font-medium text-sm">{scheduleData.title || (t("schedule.quick-input.default-title") as string)}</div>
              <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                <Clock className="h-3 w-3" aria-hidden="true" />
                <span>{scheduleData.startTs ? formatTime(scheduleData.startTs) : (t("schedule.time.tbd") as string)}</span>
                {(() => {
                  const hasEndTime = scheduleData.endTs && scheduleData.startTs && Number(scheduleData.endTs) > 0;
                  return hasEndTime ? (
                    <>
                      <span>-</span>
                      <span>
                        {new Date(Number(scheduleData.endTs!) * 1000).toLocaleTimeString("zh-CN", {
                          hour: "2-digit",
                          minute: "2-digit",
                        })}
                      </span>
                    </>
                  ) : null;
                })()}
              </div>
              {scheduleData.location && (
                <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                  <MapPin className="h-3 w-3" aria-hidden="true" />
                  <span>{scheduleData.location}</span>
                </div>
              )}
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center justify-between mt-3" role="group" aria-label="操作按钮">
            <Button
              variant="ghost"
              size="sm"
              onClick={onCancel}
              className="h-8 min-h-[44px] sm:min-h-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
              aria-label="取消创建"
            >
              {t("common.cancel") as string}
            </Button>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={onCancel}
                className="h-8 min-h-[44px] sm:min-h-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
                aria-label="修改日程信息"
              >
                {t("schedule.quick-input.modify") as string}
              </Button>
              <Button
                size="sm"
                onClick={onConfirm}
                className="h-8 min-h-[44px] sm:min-h-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
                aria-label="确认创建日程"
              >
                {t("schedule.quick-input.confirm-create") as string}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Input Area - Not shown in confirmation step */}
      {currentStep !== "confirmation" && (
        <div className="flex items-center gap-2 px-4 py-3 border-t border-border/50" role="region" aria-label="输入区域">
          <Input
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t("schedule.quick-input.reply-placeholder") as string}
            className="flex-1 h-9 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
            aria-label="输入回复"
            id="flow-input"
          />
          <Button
            size="sm"
            onClick={handleSubmit}
            disabled={!input.trim()}
            className="h-9 shrink-0 min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
            aria-label="发送回复"
          >
            {t("ai.send-shortcut") as string}
          </Button>
          {conversation.length > 0 && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onCancel}
              className="h-9 px-2 shrink-0 min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
              aria-label="取消对话"
            >
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * Compact inline version for quick input area
 */
interface CompactFlowProps {
  scheduleData: Partial<ParsedSchedule>;
  onEdit: () => void;
  className?: string;
}

export function CompactScheduleFlow({ scheduleData, onEdit, className }: CompactFlowProps) {
  const t = useTranslate();
  const formatTime = (ts: bigint) => {
    const date = new Date(Number(ts) * 1000);
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);

    const timeStr = date.toLocaleTimeString("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    });

    if (date.toDateString() === today.toDateString()) {
      return `${t("schedule.quick-input.today") as string} ${timeStr}`;
    } else if (date.toDateString() === tomorrow.toDateString()) {
      return `${t("schedule.quick-input.tomorrow") as string} ${timeStr}`;
    }
    return date.toLocaleDateString("zh-CN", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  return (
    <div
      className={cn("flex items-center gap-3 p-2 rounded-lg bg-primary/5", className)}
      role="region"
      aria-label="日程预览"
    >
      <div className="flex-1 min-w-0">
        <div className="font-medium text-sm truncate" aria-label="日程标题">
          {scheduleData.title || (t("schedule.quick-input.default-title") as string)}
        </div>
        <div className="text-xs text-muted-foreground truncate" aria-label="时间和地点">
          {scheduleData.startTs ? formatTime(scheduleData.startTs) : (t("schedule.time.tbd") as string)}
          {scheduleData.location && ` · @${scheduleData.location}`}
        </div>
      </div>
      <Button
        size="sm"
        variant="outline"
        onClick={onEdit}
        className="h-7 shrink-0 min-h-[44px] min-w-[44px] sm:min-h-0 sm:min-w-0 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2"
        aria-label="编辑日程"
      >
        {t("schedule.quick-input.edit") as string}
      </Button>
    </div>
  );
}
