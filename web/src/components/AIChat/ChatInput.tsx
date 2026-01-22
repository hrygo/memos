import { EraserIcon, MoreHorizontalIcon, SendIcon } from "lucide-react";
import { KeyboardEvent, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AgentMentionPopover } from "@/components/AIChat/AgentMentionPopover";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { PARROT_THEMES, ParrotAgentType } from "@/types/parrot";
import type { ParrotAgentI18n } from "@/hooks/useParrots";
import { useAvailableParrots } from "@/hooks/useParrots";

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
  onParrotChange?: (parrot: ParrotAgentI18n) => void;
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
  onParrotChange,
}: ChatInputProps) {
  const { t } = useTranslation();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [keyboardHeight, setKeyboardHeight] = useState(0);
  const availableParrots = useAvailableParrots();

  // Mention/Autocomplete state
  const [mentionOpen, setMentionOpen] = useState(false);
  const [mentionQuery, setMentionQuery] = useState("");
  const [mentionStartPos, setMentionStartPos] = useState<number | null>(null);

  const theme = currentParrotId ? PARROT_THEMES[currentParrotId] || PARROT_THEMES.DEFAULT : PARROT_THEMES.DEFAULT;

  // Check if should show mention popup based on value and cursor position
  const checkMentionTrigger = useCallback(() => {
    if (!textareaRef.current || !value) return false;

    const cursorPos = textareaRef.current.selectionStart;
    const charBeforeCursor = value[cursorPos - 1];
    const charBeforeThat = value[cursorPos - 2];

    // Trigger on @ when:
    // 1. @ is at the start of input, or
    // 2. @ is preceded by a space
    return charBeforeCursor === "@" && (cursorPos === 1 || charBeforeThat === " " || charBeforeThat === undefined);
  }, [value]);

  // Handle input change with mention check
  const handleChange = useCallback((newValue: string) => {
    onChange(newValue);

    // Check if @ was just typed
    const cursorPos = textareaRef.current?.selectionStart ?? newValue.length;
    const charBeforeCursor = newValue[cursorPos - 1];
    const charBeforeThat = newValue[cursorPos - 2];

    // Trigger on @ when:
    // 1. @ is at the start of input, or
    // 2. @ is preceded by a space
    const shouldTrigger = charBeforeCursor === "@" && (cursorPos === 1 || charBeforeThat === " " || charBeforeThat === undefined);

    if (shouldTrigger) {
      setMentionStartPos(cursorPos - 1);
      setMentionQuery("");
      setMentionOpen(true);
      return;
    }

    // Close mention if conditions not met
    if (mentionOpen && !checkMentionTrigger()) {
      setMentionOpen(false);
      setMentionQuery("");
      setMentionStartPos(null);
    }
  }, [onChange, mentionOpen, checkMentionTrigger]);

  // Handle cursor position changes (click, arrow keys)
  const handleCursorChange = useCallback(() => {
    if (mentionOpen && !checkMentionTrigger()) {
      setMentionOpen(false);
      setMentionQuery("");
      setMentionStartPos(null);
    }
  }, [mentionOpen, checkMentionTrigger]);

  // Listen for cursor changes
  useEffect(() => {
    if (!mentionOpen) return;

    const handleKeyUp = () => handleCursorChange();
    const handleClick = () => handleCursorChange();

    document.addEventListener("keyup", handleKeyUp);
    document.addEventListener("click", handleClick);
    return () => {
      document.removeEventListener("keyup", handleKeyUp);
      document.removeEventListener("click", handleClick);
    };
  }, [mentionOpen, value]);

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

  // Handle agent selection from mention popover
  const handleSelectAgent = useCallback((agent: ParrotAgentI18n) => {
    if (mentionStartPos === null) return;

    // Clear the @ from input and switch to the selected agent
    const beforeMention = value.slice(0, mentionStartPos);
    const afterMention = value.slice(mentionStartPos + 1); // +1 to skip the @
    const newValue = `${beforeMention}${afterMention}`.trim();

    onChange(newValue);
    setMentionOpen(false);
    setMentionQuery("");
    setMentionStartPos(null);

    // Trigger parrot change
    if (onParrotChange) {
      onParrotChange(agent);
    }

    // Focus textarea
    setTimeout(() => {
      textareaRef.current?.focus();
    }, 0);
  }, [mentionStartPos, value, onChange, onParrotChange]);

  // Close mention popover
  const handleCloseMention = useCallback(() => {
    setMentionOpen(false);
    setMentionQuery("");
    setMentionStartPos(null);
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent<HTMLTextAreaElement>) => {
    // Don't handle Enter if mention is open (let the popover handle it)
    if (mentionOpen && (e.key === "Enter" || e.key === "ArrowUp" || e.key === "ArrowDown" || e.key === "Escape")) {
      return;
    }
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      onSend();
    }
  }, [mentionOpen, onSend]);

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
        "shrink-0 p-3 md:p-4 border-t border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 transition-all",
        className,
      )}
      style={{ paddingBottom: keyboardHeight > 0 ? `${keyboardHeight + 16}px` : "max(16px, env(safe-area-inset-bottom))" }}
    >
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
                  <DropdownMenuItem onClick={onClearChat} className="text-destructive focus:text-destructive cursor-pointer">
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
            "focus-within:ring-2 focus-within:ring-offset-2 shadow-sm",
            theme.inputBg,
            theme.inputBorder,
            theme.inputFocus,
          )}
        >
          <Textarea
            ref={textareaRef}
            value={value}
            onChange={(e) => {
              handleChange(e.target.value);
              handleInput(e);
            }}
            onKeyDown={handleKeyDown}
            placeholder={defaultPlaceholder}
            disabled={disabled || isTyping}
            className="flex-1 min-h-[44px] max-h-[120px] bg-transparent border-0 outline-none resize-none text-zinc-900 dark:text-zinc-100 placeholder:text-zinc-400 text-base leading-relaxed"
            rows={1}
          />
          <Button
            size="icon"
            className={cn(
              "shrink-0 h-11 min-w-[44px] rounded-xl transition-all",
              "hover:scale-105 active:scale-95",
              value.trim() && !isTyping
                ? cn(theme.iconBg, theme.iconText)
                : "bg-zinc-100 dark:bg-zinc-800 text-zinc-400 dark:text-zinc-600",
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
        <p className="text-xs text-zinc-400 dark:text-zinc-600 mt-1.5 text-center hidden md:block">
          Press Enter to send, Shift + Enter for new line, @ to mention agent
        </p>
      </div>

      {/* Agent Mention Popover */}
      <AgentMentionPopover
        open={mentionOpen}
        onClose={handleCloseMention}
        onSelectAgent={handleSelectAgent}
        agents={availableParrots}
        filterText={mentionQuery}
        triggerRef={textareaRef}
      />
    </div>
  );
}
