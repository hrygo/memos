import { Calendar, Clock, Search, CheckCircle, Loader2, Sparkles, ChevronDown, ChevronUp, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { scheduleAgentServiceClient } from "@/connect";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

// Stream event types from backend
interface StreamEvent {
  type: string;
  data: string;
}

// Tool call state
interface ToolCall {
  id: string;
  name: string;
  input: string;
  result?: string;
  status: "pending" | "running" | "success" | "error";
  timestamp: Date;
}

// Message in the conversation
interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  toolCalls?: ToolCall[];
  timestamp: Date;
}

interface StreamingScheduleAssistantProps {
  onSuccess?: () => void;
  onError?: (error: Error) => void;
  className?: string;
  placeholder?: string;
  initialQuery?: string;
}

const MAX_INPUT_HEIGHT = 120;
const LINE_HEIGHT = 24;

// Tool metadata for UI
const TOOL_METADATA: Record<string, { icon: typeof Search; label: string; color: string }> = {
  schedule_query: {
    icon: Search,
    label: "查询日程",
    color: "text-blue-500",
  },
  schedule_add: {
    icon: CheckCircle,
    label: "创建日程",
    color: "text-green-500",
  },
  find_free_time: {
    icon: Clock,
    label: "查找空闲时间",
    color: "text-purple-500",
  },
  schedule_update: {
    icon: Calendar,
    label: "更新日程",
    color: "text-orange-500",
  },
};

export function StreamingScheduleAssistant({
  onSuccess,
  onError,
  className,
  placeholder,
  initialQuery = "",
}: StreamingScheduleAssistantProps) {
  const t = useTranslate();

  // State
  const [input, setInput] = useState(initialQuery);
  const [inputHeight, setInputHeight] = useState(LINE_HEIGHT);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isProcessing, setIsProcessing] = useState(false);
  const [currentToolCall, setCurrentToolCall] = useState<ToolCall | null>(null);
  const [isThinking, setIsThinking] = useState(false);
  const [expandedToolCalls, setExpandedToolCalls] = useState<Set<string>>(new Set());

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, currentToolCall]);

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = "auto";
    const newHeight = Math.min(Math.max(textarea.scrollHeight, LINE_HEIGHT), MAX_INPUT_HEIGHT);
    textarea.style.height = `${newHeight}px`;
    setInputHeight(newHeight);
  }, [input]);

  // Parse tool name from event data
  const parseToolName = (data: string): string => {
    try {
      const parsed = JSON.parse(data);
      if (parsed.tool) return parsed.tool;
      if (parsed.name) return parsed.name;
    } catch {
      // Try to extract tool name from string
      const match = data.match(/"tool":\s*"([^"]+)"/);
      if (match) return match[1];
    }
    return "unknown";
  };

  // Parse tool result
  const parseToolResult = (data: string): string => {
    try {
      const parsed = JSON.parse(data);
      if (parsed.result) return parsed.result;
      if (parsed.output) return parsed.output;
    } catch {
      // Return as-is
    }
    return data;
  };

  // Handle send message
  const handleSend = async () => {
    const trimmedInput = input.trim();
    if (!trimmedInput || isProcessing) return;

    // Add user message
    const userMessage: ChatMessage = {
      id: `msg-${Date.now()}`,
      role: "user",
      content: trimmedInput,
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, userMessage]);
    setInput("");

    // Create assistant message placeholder
    const assistantMessageId = `msg-${Date.now() + 1}`;
    const assistantMessage: ChatMessage = {
      id: assistantMessageId,
      role: "assistant",
      content: "",
      timestamp: new Date(),
      toolCalls: [],
    };
    setMessages((prev) => [...prev, assistantMessage]);

    await processStream(trimmedInput, assistantMessageId);
  };

  // Process streaming response
  const processStream = async (userMessage: string, messageId: string) => {
    setIsProcessing(true);
    setIsThinking(true);

    try {
      const stream = scheduleAgentServiceClient.chatStream({
        message: userMessage,
        userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Shanghai",
      });

      let fullContent = "";
      let toolCalls: ToolCall[] = [];

      for await (const chunk of stream) {
        // Parse event JSON
        if (chunk.event) {
          try {
            const event: StreamEvent = JSON.parse(chunk.event);

            switch (event.type) {
              case "thinking":
                setIsThinking(true);
                break;

              case "tool_use": {
                const toolName = parseToolName(event.data);
                const toolCall: ToolCall = {
                  id: `tool-${Date.now()}`,
                  name: toolName,
                  input: event.data,
                  status: "running",
                  timestamp: new Date(),
                };
                setCurrentToolCall(toolCall);
                setIsThinking(false);

                // Add to message tool calls
                toolCalls = [...toolCalls, toolCall];
                setMessages((prev) => prev.map((m) => (m.id === messageId ? { ...m, toolCalls: [...toolCalls] } : m)));
                break;
              }

              case "tool_result": {
                const result = parseToolResult(event.data);
                setIsThinking(false);

                // Update current tool call
                if (currentToolCall) {
                  const updatedToolCall: ToolCall = {
                    ...currentToolCall,
                    result,
                    status: result.toLowerCase().includes("error") ? "error" : "success",
                  };
                  setCurrentToolCall(null);

                  // Update in tool calls array
                  toolCalls = toolCalls.map((tc) => (tc.id === currentToolCall.id ? updatedToolCall : tc));

                  setMessages((prev) => prev.map((m) => (m.id === messageId ? { ...m, toolCalls } : m)));

                  // Auto-expand new tool results
                  setExpandedToolCalls((prev) => new Set([...prev, currentToolCall.id]));
                }
                break;
              }

              case "schedule_updated":
                // Schedule was created/updated successfully
                setIsThinking(false);
                onSuccess?.();
                break;

              case "error":
                setIsThinking(false);
                setIsProcessing(false);
                setCurrentToolCall(null);
                onError?.(new Error(event.data));
                return;
            }
          } catch (e) {
            console.error("Failed to parse event:", chunk.event);
          }
        }

        // Accumulate content
        if (chunk.content) {
          fullContent = chunk.content;
          setMessages((prev) => prev.map((m) => (m.id === messageId ? { ...m, content: fullContent } : m)));
        }

        // Check if done
        if (chunk.done) {
          setIsThinking(false);
          setIsProcessing(false);
          setCurrentToolCall(null);

          // Check if schedule was created
          if (
            fullContent.includes("已成功创建") ||
            fullContent.includes("成功创建日程") ||
            fullContent.includes("successfully created") ||
            fullContent.includes("已安排")
          ) {
            onSuccess?.();
          }
        }
      }
    } catch (error) {
      console.error("Stream error:", error);
      setIsThinking(false);
      setIsProcessing(false);
      setCurrentToolCall(null);
      onError?.(error as Error);
    }
  };

  // Toggle tool call expansion
  const toggleToolCall = (id: string) => {
    setExpandedToolCalls((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // Clear conversation
  const handleClear = () => {
    setMessages([]);
    setInput("");
    setCurrentToolCall(null);
    setIsThinking(false);
  };

  // Get default placeholder
  const getPlaceholder = () => {
    if (placeholder) return placeholder;
    return (t("schedule.streaming-assistant.placeholder") as string) || "告诉我你想安排什么，例如：「明天下午3点开会」";
  };

  return (
    <div className={cn("flex flex-col gap-4", className)}>
      {/* Messages */}
      {messages.length > 0 && (
        <div className="flex-1 overflow-y-auto space-y-4 max-h-[400px] px-2">
          {messages.map((message) => (
            <div key={message.id} className={cn("flex gap-3", message.role === "user" ? "justify-end" : "justify-start")}>
              {message.role === "assistant" && (
                <div className="flex-shrink-0 w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                  <Sparkles className="w-4 h-4 text-primary" />
                </div>
              )}

              <div
                className={cn(
                  "max-w-[80%] rounded-2xl px-4 py-3",
                  message.role === "user" ? "bg-primary text-primary-foreground rounded-br-sm" : "bg-muted/50 rounded-bl-sm",
                )}
              >
                <p className="text-sm whitespace-pre-wrap break-words">{message.content}</p>

                {/* Tool Calls */}
                {message.toolCalls && message.toolCalls.length > 0 && (
                  <div className="mt-3 space-y-2">
                    {message.toolCalls.map((toolCall) => {
                      const meta = TOOL_METADATA[toolCall.name] || {
                        icon: Search,
                        label: toolCall.name,
                        color: "text-zinc-500",
                      };
                      const Icon = meta.icon;
                      const isExpanded = expandedToolCalls.has(toolCall.id);

                      return (
                        <div key={toolCall.id} className="rounded-lg bg-background border border-border/50 overflow-hidden">
                          <button
                            onClick={() => toggleToolCall(toolCall.id)}
                            className="w-full flex items-center gap-2 px-3 py-2 hover:bg-muted/30 transition-colors"
                          >
                            <Icon className={cn("w-4 h-4", meta.color)} />
                            <span className="text-sm font-medium">{meta.label}</span>
                            <span
                              className={cn(
                                "ml-auto text-xs",
                                toolCall.status === "success" && "text-green-500",
                                toolCall.status === "error" && "text-red-500",
                                toolCall.status === "running" && "text-blue-500",
                              )}
                            >
                              {toolCall.status === "running" && "运行中..."}
                              {toolCall.status === "success" && "完成"}
                              {toolCall.status === "error" && "失败"}
                            </span>
                            {isExpanded ? (
                              <ChevronUp className="w-4 h-4 text-muted-foreground" />
                            ) : (
                              <ChevronDown className="w-4 h-4 text-muted-foreground" />
                            )}
                          </button>

                          {isExpanded && (
                            <div className="px-3 pb-3 text-xs space-y-2">
                              {/* Input */}
                              <div className="bg-muted/50 rounded p-2">
                                <div className="font-medium text-muted-foreground mb-1">输入参数</div>
                                <pre className="whitespace-pre-wrap break-words text-muted-foreground">{toolCall.input}</pre>
                              </div>

                              {/* Result */}
                              {toolCall.result && (
                                <div className="bg-muted/50 rounded p-2">
                                  <div className="font-medium text-muted-foreground mb-1">执行结果</div>
                                  <pre className="whitespace-pre-wrap break-words text-muted-foreground">{toolCall.result}</pre>
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          ))}

          {/* Current Tool Call Animation */}
          {currentToolCall && (
            <div className="flex gap-3 justify-start">
              <div className="flex-shrink-0 w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                <Sparkles className="w-4 h-4 text-primary animate-pulse" />
              </div>
              <div className="bg-muted/50 rounded-2xl rounded-bl-sm px-4 py-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>{t("schedule.streaming-assistant.executing-tool") || "正在执行"}</span>
                  <span className="font-medium text-foreground">{currentToolCall.name}</span>
                </div>
              </div>
            </div>
          )}

          {/* Thinking Animation */}
          {isThinking && !currentToolCall && (
            <div className="flex gap-3 justify-start">
              <div className="flex-shrink-0 w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                <Sparkles className="w-4 h-4 text-primary animate-pulse" />
              </div>
              <div className="bg-muted/50 rounded-2xl rounded-bl-sm px-4 py-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>{t("schedule.streaming-assistant.thinking") || "正在思考..."}</span>
                </div>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>
      )}

      {/* Input Area */}
      <div
        className={cn(
          "flex items-end gap-2 p-3 rounded-xl border-2 transition-all duration-300",
          isProcessing && "border-primary/40 bg-primary/5",
          !isProcessing && "border-border bg-background",
        )}
      >
        <div className="flex-1 min-w-0">
          <Textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSend();
              }
              if (e.key === "Escape" && input) {
                e.preventDefault();
                setInput("");
              }
            }}
            placeholder={getPlaceholder()}
            className={cn(
              "min-h-[24px] max-h-[120px] py-2 px-3 resize-none",
              "border-0 bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0",
              "text-sm",
            )}
            style={{ height: `${inputHeight}px` }}
            rows={1}
            disabled={isProcessing}
          />
        </div>

        <div className="flex items-center gap-1.5 flex-shrink-0">
          {messages.length > 0 && !isProcessing && (
            <Button size="sm" variant="ghost" onClick={handleClear} className="h-9 w-9 rounded-full p-0">
              <X className="h-4 w-4" />
            </Button>
          )}
          <Button
            size="sm"
            onClick={handleSend}
            disabled={!input.trim() || isProcessing}
            className={cn(
              "h-9 px-4 rounded-lg transition-all duration-200",
              input.trim() ? "bg-primary text-primary-foreground hover:bg-primary/90" : "bg-transparent text-muted-foreground",
            )}
          >
            {isProcessing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
          </Button>
        </div>
      </div>

      {/* Hint */}
      {!input && !isProcessing && messages.length === 0 && (
        <div className="flex items-center gap-2 px-1 text-xs text-muted-foreground">
          <Sparkles className="h-3 w-3 text-primary/60" />
          <span>{t("schedule.streaming-assistant.hint") || "告诉我你想安排什么，支持自然语言输入"}</span>
        </div>
      )}
    </div>
  );
}
