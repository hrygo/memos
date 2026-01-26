import { MessageSquarePlus, Scissors, SendIcon, Trash2 } from "lucide-react";
import { KeyboardEvent, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

interface ChatInputProps {
  value: string;
  onChange: (value: string) => void;
  onSend: () => void;
  onNewChat?: () => void;
  onClearContext?: () => void;
  onClearChat?: () => void;
  disabled?: boolean;
  isTyping?: boolean;
  placeholder?: string;
  className?: string;
  showQuickActions?: boolean;
  quickActions?: React.ReactNode;
}

export function ChatInput({
  value,
  onChange,
  onSend,
  onNewChat,
  onClearContext,
  onClearChat,
  disabled = false,
  isTyping = false,
  placeholder,
  className,
  showQuickActions = false,
  quickActions,
}: ChatInputProps) {
  const { t } = useTranslation();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [keyboardHeight, setKeyboardHeight] = useState(0);

  // Handle mobile keyboard visibility
  useEffect(() => {
    if (typeof window === "undefined" || !window.visualViewport) return;

    const handleResize = () => {
      const viewport = window.visualViewport;
      if (!viewport) return;

      const windowHeight = window.innerHeight;
      const keyboardVisible = viewport.height < windowHeight * 0.85;
      const newKeyboardHeight = keyboardVisible ? windowHeight - viewport.height : 0;

      setKeyboardHeight(newKeyboardHeight);
    };

    window.visualViewport.addEventListener("resize", handleResize);
    return () => window.visualViewport?.removeEventListener("resize", handleResize);
  }, []);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        onSend();
      }
    },
    [onSend],
  );

  const handleInput = useCallback((e: React.FormEvent<HTMLTextAreaElement>) => {
    const target = e.target as HTMLTextAreaElement;
    target.style.height = "auto";
    target.style.height = `${Math.min(target.scrollHeight, 120)}px`;
  }, []);

  // Reset height when value changes externally
  useEffect(() => {
    if (textareaRef.current && !value) {
      textareaRef.current.style.height = "auto";
    }
  }, [value]);

  const defaultPlaceholder = placeholder || t("ai.parrot.chat-default-placeholder");

  return (
    <div
      className={cn(
        "shrink-0 p-3 md:p-4 border-t border-border bg-background transition-all",
        className,
      )}
      style={{ paddingBottom: keyboardHeight > 0 ? `${keyboardHeight + 16}px` : "max(16px, env(safe-area-inset-bottom))" }}
    >
      <div className="max-w-3xl mx-auto">
        {/* Quick Actions */}
        {showQuickActions && quickActions}

        {/* Toolbar - 工具栏 */}
        {(onNewChat || onClearContext || onClearChat) && (
          <div className="flex items-center gap-1 mb-2">
            {onNewChat && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onNewChat}
                aria-label={t("ai.aichat.sidebar.new-chat")}
                className="group/btn h-7 w-7 hover:w-auto px-0 hover:px-2 text-xs text-primary hover:text-primary hover:bg-accent transition-all overflow-hidden"
                title="⌘N"
              >
                <MessageSquarePlus className="w-3.5 h-3.5 shrink-0" />
                <span className="hidden group-hover/btn:inline ml-1 whitespace-nowrap">
                  {t("ai.aichat.sidebar.new-chat")}
                  <kbd className="ml-1.5 px-1 py-0.5 text-[10px] bg-accent rounded">⌘N</kbd>
                </span>
              </Button>
            )}
            {onClearContext && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onClearContext}
                aria-label={t("ai.clear-context")}
                className="group/btn h-7 w-7 hover:w-auto px-0 hover:px-2 text-xs text-muted-foreground hover:text-foreground hover:bg-muted transition-all overflow-hidden"
                title="⌘K"
              >
                <Scissors className="w-3.5 h-3.5 shrink-0" />
                <span className="hidden group-hover/btn:inline ml-1 whitespace-nowrap">
                  {t("ai.clear-context")}
                  <kbd className="ml-1.5 px-1 py-0.5 text-[10px] bg-muted rounded">⌘K</kbd>
                </span>
              </Button>
            )}
            {onClearChat && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onClearChat}
                aria-label={t("ai.clear-chat")}
                className="group/btn h-7 w-7 hover:w-auto px-0 hover:px-2 text-xs text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-all overflow-hidden"
                title="⌘L"
              >
                <Trash2 className="w-3.5 h-3.5 shrink-0" />
                <span className="hidden group-hover/btn:inline ml-1 whitespace-nowrap">
                  {t("ai.clear-chat")}
                  <kbd className="ml-1.5 px-1 py-0.5 text-[10px] bg-destructive/20 rounded">⌘L</kbd>
                </span>
              </Button>
            )}
          </div>
        )}

        {/* Input Box */}
        <div
          className={cn(
            "flex items-end gap-2 md:gap-3 p-2.5 md:p-3 rounded-xl border transition-all",
            "focus-within:ring-2 focus-within:ring-offset-2 shadow-sm",
            "bg-card border-border focus-within:ring-ring",
          )}
        >
          <Textarea
            ref={textareaRef}
            value={value}
            onChange={(e) => {
              onChange(e.target.value);
              handleInput(e);
            }}
            onKeyDown={handleKeyDown}
            placeholder={defaultPlaceholder}
            disabled={disabled || isTyping}
            className="flex-1 min-h-[44px] max-h-[120px] bg-transparent border-0 outline-none resize-none text-foreground placeholder:text-muted-foreground text-sm leading-relaxed font-sans"
            rows={1}
          />
          <Button
            size="icon"
            className={cn(
              "shrink-0 h-11 min-w-[44px] rounded-xl transition-all",
              "hover:scale-105 active:scale-95",
              value.trim() && !isTyping ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground",
              "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100",
            )}
            onClick={onSend}
            disabled={!value.trim() || isTyping || disabled}
            aria-label="Send message"
          >
            <SendIcon className="w-5 h-5" />
          </Button>
        </div>

        {/* Hint Text - Desktop only */}
        <p className="text-xs text-muted-foreground mt-1.5 text-center hidden md:block">{t("ai.aichat.input-hint")}</p>
      </div>
    </div>
  );
}
