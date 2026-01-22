import { useCallback, useRef, useState } from "react";
import { Loader2, Send, Sparkles, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useScheduleContext } from "@/contexts/ScheduleContext";
import { useChatWithMemos } from "@/hooks/useAIQueries";
import { ParrotAgentType } from "@/types/parrot";
import { cn } from "@/lib/utils";
import { buildScheduleMessage, getScheduleAgentType } from "@/utils/scheduleChatEnhancer";

const MAX_INPUT_HEIGHT = 120;
const LINE_HEIGHT = 24;

interface ScheduleChatInputProps {
  onResponse?: (content: string) => void;
  className?: string;
}

/**
 * ScheduleChatInput - AI-powered chat input for schedule operations
 *
 * Simply builds the message with date context and sends to SCHEDULE agent.
 * The backend agent handles all the parsing and scheduling logic.
 */
export function ScheduleChatInput({ onResponse, className }: ScheduleChatInputProps) {
  const { selectedDate } = useScheduleContext();
  const chatHook = useChatWithMemos();

  const [input, setInput] = useState("");
  const [inputHeight, setInputHeight] = useState(LINE_HEIGHT);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleResize = useCallback(() => {
    if (!textareaRef.current) return;
    textareaRef.current.style.height = "auto";
    const newHeight = Math.min(
      Math.max(textareaRef.current.scrollHeight, LINE_HEIGHT),
      MAX_INPUT_HEIGHT
    );
    textareaRef.current.style.height = `${newHeight}px`;
    setInputHeight(newHeight);
  }, []);

  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);
    handleResize();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
    if (e.key === "Escape" && input) {
      e.preventDefault();
      handleClear();
    }
  };

  const handleClear = useCallback(() => {
    setInput("");
    if (textareaRef.current) {
      textareaRef.current.style.height = `${LINE_HEIGHT}px`;
      setInputHeight(LINE_HEIGHT);
    }
  }, []);

  const handleSend = useCallback(async () => {
    const trimmedInput = input.trim();
    if (!trimmedInput || isLoading) return;

    setIsLoading(true);

    // Build message with date context (simple prefix)
    const message = buildScheduleMessage(trimmedInput, selectedDate);

    console.log("[ScheduleChatInput] Sending to SCHEDULE agent:", { message, selectedDate });

    try {
      let fullResponse = "";

      await chatHook.stream(
        {
          message,
          agentType: getScheduleAgentType(), // Always SCHEDULE
          userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        },
        {
          onContent: (content) => {
            fullResponse += content;
          },
          onDone: () => {
            setIsLoading(false);
            handleClear();
            onResponse?.(fullResponse);
          },
          onError: (error) => {
            console.error("[ScheduleChatInput] Error:", error);
            setIsLoading(false);
          },
        }
      );
    } catch (error) {
      console.error("[ScheduleChatInput] Stream error:", error);
      setIsLoading(false);
    }
  }, [input, isLoading, selectedDate, chatHook, handleClear, handleResize, onResponse]);

  const getPlaceholder = () => {
    if (selectedDate) {
      return `输入 "吃午饭" 或 "今天有什么安排" (${selectedDate})`;
    }
    return '输入 "吃午饭" 或 "今天有什么安排"';
  };

  return (
    <div className={cn("w-full flex flex-col gap-2", className)}>
      {isLoading && (
        <div className="flex items-center gap-2 px-3 py-2 text-sm text-muted-foreground bg-primary/5 rounded-lg border border-primary/10">
          <Loader2 className="h-4 w-4 animate-spin text-primary" />
          <span>正在处理...</span>
        </div>
      )}

      <div
        className={cn(
          "flex items-center gap-2 p-2 rounded-xl border-2 transition-all duration-200",
          isLoading && "border-primary/30 bg-primary/5",
          !isLoading && "border-border bg-background"
        )}
      >
        <div className="flex-shrink-0">
          <div className="w-8 h-8 rounded-lg flex items-center justify-center bg-orange-100 dark:bg-orange-900/40 text-orange-600 dark:text-orange-400">
            <Sparkles className="w-4 h-4" />
          </div>
        </div>

        <div className="flex-1 min-w-0">
          <Textarea
            ref={textareaRef}
            value={input}
            onChange={handleInputChange}
            onKeyDown={handleKeyDown}
            placeholder={getPlaceholder()}
            className={cn(
              "min-h-[24px] max-h-[120px] py-1.5 px-3 resize-none",
              "border-0 bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0",
              "text-sm"
            )}
            style={{ height: `${inputHeight}px` }}
            rows={1}
            disabled={isLoading}
          />
        </div>

        <div className="flex items-center gap-1 flex-shrink-0">
          {isLoading ? (
            <Button size="sm" variant="ghost" className="h-8 w-8 p-0">
              <Loader2 className="h-4 w-4 animate-spin" />
            </Button>
          ) : input.length > 0 ? (
            <>
              <Button size="sm" variant="ghost" onClick={handleClear} className="h-8 w-8 p-0">
                <X className="h-4 w-4" />
              </Button>
              <Button size="sm" onClick={handleSend} className="h-9 px-3 gap-1.5">
                <Send className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">发送</span>
              </Button>
            </>
          ) : (
            <Button size="sm" variant="ghost" className="h-8 w-8 p-0 opacity-50" disabled>
              <Send className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      </div>

      {!input && !isLoading && (
        <p className="text-xs text-muted-foreground px-2">
          💡 自动使用当前选中的日期
        </p>
      )}
    </div>
  );
}
