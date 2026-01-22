import { SendIcon, EraserIcon, MoreHorizontalIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { KeyboardEvent, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { PARROT_THEMES } from "@/types/parrot";
import { cn } from "@/lib/utils";
import { ParrotAgentType } from "@/types/parrot";

interface ChatInputProps {
  value: string;
  onChange: (value: string) => void;
  onSend: () => void;
  onClearChat?: () => void;
  onClearContext?: () => void;
  disabled?: boolean;
  isTyping?: boolean;
  currentParrotId?: ParrotAgentType;
  placeholder?: string;
  className?: string;
  showQuickActions?: boolean;
  quickActions?: React.ReactNode;
}

export function ChatInput({
  value,
  onChange,
  onSend,
  onClearChat,
  onClearContext,
  disabled = false,
  isTyping = false,
  currentParrotId,
  placeholder,
  className,
  showQuickActions = false,
  quickActions,
}: ChatInputProps) {
  const { t } = useTranslation();
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const theme = currentParrotId
    ? PARROT_THEMES[currentParrotId] || PARROT_THEMES.DEFAULT
    : PARROT_THEMES.DEFAULT;

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      onSend();
    }
  };

  const handleInput = (e: React.FormEvent<HTMLTextAreaElement>) => {
    const target = e.target as HTMLTextAreaElement;
    target.style.height = "auto";
    target.style.height = `${Math.min(target.scrollHeight, 120)}px`;
  };

  // Reset height when value changes externally
  useEffect(() => {
    if (textareaRef.current && !value) {
      textareaRef.current.style.height = "auto";
    }
  }, [value]);

  const defaultPlaceholder = placeholder || t("ai.parrot.chat-default-placeholder");

  return (
    <div className={cn("shrink-0 p-3 md:p-4 border-t border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-950", className)}>
      <div className="max-w-3xl mx-auto">
        {/* Quick Actions */}
        {showQuickActions && quickActions}

        {/* Clear Chat Button - Desktop */}
        {(onClearChat || onClearContext) && (
          <div className="hidden md:block absolute -top-10 right-4">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
                >
                  <EraserIcon className="w-3.5 h-3.5 mr-1" />
                  {t("ai.clear")}
                  <MoreHorizontalIcon className="w-3 h-3 ml-1" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                {onClearContext && (
                  <DropdownMenuItem onClick={onClearContext} className="cursor-pointer">
                    <EraserIcon className="w-4 h-4 mr-2" />
                    <div>
                      <div className="font-medium">{t("ai.clear-context")}</div>
                      <div className="text-xs text-muted-foreground">{t("ai.clear-context-desc")}</div>
                    </div>
                  </DropdownMenuItem>
                )}
                {onClearChat && (
                  <DropdownMenuItem
                    onClick={onClearChat}
                    className="text-destructive focus:text-destructive cursor-pointer"
                  >
                    <EraserIcon className="w-4 h-4 mr-2" />
                    <div>
                      <div className="font-medium">{t("ai.clear-chat")}</div>
                      <div className="text-xs text-muted-foreground">{t("ai.clear-chat-desc")}</div>
                    </div>
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}

        {/* Input Box */}
        <div
          className={cn(
            "flex items-end gap-2 md:gap-3 p-2.5 md:p-3 rounded-xl border transition-all",
            "focus-within:ring-2 focus-within:ring-offset-2",
            theme.inputBg,
            theme.inputBorder,
            theme.inputFocus
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
            className="flex-1 min-h-[40px] md:min-h-[44px] max-h-[120px] bg-transparent border-0 outline-none resize-none text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 text-sm md:text-base"
            rows={1}
          />
          <Button
            size="icon"
            className={cn(
              "shrink-0 h-9 w-9 md:h-10 md:w-10 rounded-lg transition-all",
              "hover:scale-105 active:scale-95",
              value.trim() && !isTyping
                ? cn(theme.iconBg, theme.iconText)
                : "bg-zinc-100 dark:bg-zinc-800 text-zinc-400 dark:text-zinc-600",
              "disabled:opacity-50 disabled:cursor-not-allowed"
            )}
            onClick={onSend}
            disabled={!value.trim() || isTyping || disabled}
          >
            <SendIcon className="w-4 h-4" />
          </Button>
        </div>

        {/* Hint Text */}
        <p className="text-xs text-zinc-400 dark:text-zinc-600 mt-1.5 text-center hidden md:block">
          Press Enter to send, Shift + Enter for new line
        </p>
      </div>
    </div>
  );
}
