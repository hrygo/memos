import { Check, ChevronDown, ChevronUp, Copy, Scissors } from "lucide-react";
import { ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import MessageActions from "@/components/AIChat/MessageActions";
import TypingCursor from "@/components/AIChat/TypingCursor";
import { CodeBlock } from "@/components/MemoContent/CodeBlock";
import { cn } from "@/lib/utils";
import { ChatItem, ConversationMessage } from "@/types/aichat";
import { PARROT_ICONS, PARROT_THEMES, ParrotAgentType } from "@/types/parrot";

interface ChatMessagesProps {
  items: ChatItem[];
  isTyping?: boolean;
  currentParrotId?: ParrotAgentType;
  onCopyMessage?: (content: string) => void;
  onRegenerate?: () => void;
  onDeleteMessage?: (index: number) => void;
  children?: ReactNode;
  className?: string;
  amazingInsightCard?: ReactNode;
}

const SCROLL_THRESHOLD = 100;

export function ChatMessages({
  items,
  isTyping = false,
  currentParrotId,
  onCopyMessage,
  onRegenerate,
  onDeleteMessage,
  children,
  className,
  amazingInsightCard,
}: ChatMessagesProps) {
  const { t } = useTranslation();
  const scrollRef = useRef<HTMLDivElement>(null);
  const endRef = useRef<HTMLDivElement>(null);
  const [isUserScrolling, setIsUserScrolling] = useState(false);

  const handleScroll = useCallback(() => {
    if (scrollRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
      const distanceToBottom = scrollHeight - scrollTop - clientHeight;
      setIsUserScrolling(distanceToBottom > SCROLL_THRESHOLD);
    }
  }, []);

  // Smooth scroll to bottom when new messages arrive, but only if user isn't scrolling
  useEffect(() => {
    if (!isUserScrolling && endRef.current) {
      endRef.current.scrollIntoView({ behavior: "smooth", block: "end" });
    }
  }, [items, isTyping, isUserScrolling]);

  // Reset user scrolling state when typing starts
  useEffect(() => {
    if (isTyping && scrollRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
      const distanceToBottom = scrollHeight - scrollTop - clientHeight;
      if (distanceToBottom <= SCROLL_THRESHOLD) {
        setIsUserScrolling(false);
      }
    }
  }, [isTyping]);

  const theme = currentParrotId ? PARROT_THEMES[currentParrotId] || PARROT_THEMES.DEFAULT : PARROT_THEMES.DEFAULT;
  const currentIcon = currentParrotId ? PARROT_ICONS[currentParrotId] || PARROT_ICONS.DEFAULT : PARROT_ICONS.DEFAULT;
  console.log("items", items);
  return (
    <div ref={scrollRef} onScroll={handleScroll} className={cn("flex-1 overflow-y-auto px-3 md:px-6 py-4", className)}>
      {children}

      {items.length > 0 && (
        <div className="max-w-3xl mx-auto space-y-4" ref={endRef}>
          {items.map((item, index) => {
            // Context separator - optimized visual design
            if ("type" in item && item.type === "context-separator") {
              return (
                <div key={`separator-${index}`} className="flex items-center justify-center gap-3 py-3 my-2 animate-in fade-in slide-in-from-top-2 duration-300">
                  <div className="flex-1 h-px bg-gradient-to-r from-transparent via-zinc-300 dark:via-zinc-700 to-transparent" />
                  <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 shadow-sm">
                    <Scissors className="w-3.5 h-3.5 text-zinc-500 dark:text-zinc-400 rotate-[-45deg]" />
                    <span className="text-xs text-zinc-600 dark:text-zinc-400 font-medium whitespace-nowrap">{t("ai.context-cleared")}</span>
                  </div>
                  <div className="flex-1 h-px bg-gradient-to-r from-transparent via-zinc-300 dark:via-zinc-700 to-transparent" />
                </div>
              );
            }

            const msg = item as ConversationMessage;
            const isLastMessage = index === items.length - 1;
            const isNew = Date.now() - msg.timestamp < 1000; // Animation for recent messages

            return (
              <MessageBubble
                key={msg.id}
                message={msg}
                theme={theme}
                icon={msg.role === "user" ? undefined : currentIcon}
                isLastAssistant={msg.role === "assistant" && isLastMessage}
                isNew={isNew}
                onCopy={() => onCopyMessage?.(msg.content)}
                onRegenerate={onRegenerate}
                onDelete={() => onDeleteMessage?.(index)}
              >
                {msg.role === "assistant" && isTyping && isLastMessage && !msg.error && (
                  <TypingCursor active={true} parrotId={currentParrotId} variant="dots" />
                )}
              </MessageBubble>
            );
          })}

          {/* Amazing Insight Card - rendered in message flow with exact same alignment as assistant messages */}
          {amazingInsightCard && !isTyping && items.length > 0 && (
            <div className="flex gap-3 md:gap-4 animate-in fade-in slide-in-from-bottom-2 duration-300">
              {/* Spacer for avatar alignment */}
              <div className="w-9 h-9 md:w-10 md:h-10 shrink-0 invisible" />
              <div className="flex-1 min-w-0">
                <div className="max-w-[85%] md:max-w-[80%]">
                  {amazingInsightCard}
                </div>
              </div>
            </div>
          )}

          {/* Typing indicator - AI Native design */}
          {isTyping &&
            (() => {
              const lastItem = items[items.length - 1];
              if (!lastItem) return true;
              if ("type" in lastItem && lastItem.type === "context-separator") return true;
              return "role" in lastItem && lastItem.role !== "assistant";
            })() && (
              <div className="flex gap-3 md:gap-4 animate-in fade-in slide-in-from-bottom-2 duration-300">
                <div className="w-9 h-9 md:w-10 md:h-10 rounded-full flex items-center justify-center shadow-sm">
                  {currentIcon.startsWith("/") ? (
                    <img src={currentIcon} alt="" className="w-8 h-8 md:w-9 md:h-9 object-contain" />
                  ) : (
                    <span className="text-lg md:text-xl">{currentIcon}</span>
                  )}
                </div>
                <div className={cn("px-4 py-3 rounded-2xl border shadow-sm", theme.bubbleBg, theme.bubbleBorder)}>
                  <TypingCursor active={true} parrotId={currentParrotId} variant="dots" />
                </div>
              </div>
            )}
          {/* Scroll anchor */}
          <div ref={endRef} className="h-1" />
        </div>
      )}
    </div>
  );
}

interface MessageBubbleProps {
  message: ConversationMessage;
  theme: (typeof PARROT_THEMES)[keyof typeof PARROT_THEMES];
  icon?: string;
  isLastAssistant?: boolean;
  isNew?: boolean;
  onCopy?: () => void;
  onRegenerate?: () => void;
  onDelete?: () => void;
  children?: ReactNode;
}

const MAX_MESSAGE_HEIGHT = 200;

function MessageBubble({
  message,
  theme,
  icon,
  isLastAssistant = false,
  isNew = false,
  onCopy,
  onRegenerate,
  onDelete,
  children,
}: MessageBubbleProps) {
  const { role, content, error } = message;
  const contentRef = useRef<HTMLDivElement>(null);
  const [isFolded, setIsFolded] = useState(true);
  const [shouldShowFold, setShouldShowFold] = useState(false);
  const [copied, setCopied] = useState(false);
  const { t } = useTranslation();

  // Detect height for auto-folding
  useEffect(() => {
    if (contentRef.current) {
      const height = contentRef.current.scrollHeight;
      if (height > MAX_MESSAGE_HEIGHT) {
        setShouldShowFold(true);
      } else {
        setShouldShowFold(false);
      }
    }
  }, [content, children]);

  const toggleFold = useCallback(() => {
    setIsFolded((prev) => !prev);
  }, []);

  const handleCopy = useCallback(() => {
    if (onCopy) {
      onCopy();
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [onCopy]);

  return (
    <div
      className={cn(
        "flex gap-3 md:gap-4 group/row",
        role === "user" ? "flex-row-reverse" : "flex-row",
        isNew && "animate-in fade-in slide-in-from-bottom-3 duration-300",
      )}
    >
      {/* Avatar */}
      <div className="w-9 h-9 md:w-10 md:h-10 rounded-full flex items-center justify-center shrink-0 shadow-sm overflow-hidden">
        {role === "user" ? (
          <img src="/images/parrots/icons/user_avatar.webp" alt="User" className="w-full h-full object-cover" />
        ) : icon?.startsWith("/") ? (
          <img src={icon} alt="" className="w-8 h-8 md:w-9 md:h-9 object-contain" />
        ) : (
          <span className="text-lg md:text-xl">{icon || "🤖"}</span>
        )}
      </div>

      {/* Message content area */}
      <div className="flex-1 min-w-0 flex flex-col gap-1">
        {/* Assistant Actions Header */}
        {role === "assistant" && isLastAssistant && onRegenerate && onDelete && (
          <div className="flex items-center gap-2 mb-0.5 opacity-0 group-row:opacity-100 transition-opacity">
            <MessageActions onCopy={handleCopy} onRegenerate={onRegenerate} onDelete={onDelete} />
          </div>
        )}

        <div className={cn("flex items-start gap-2", role === "user" ? "flex-row-reverse" : "flex-row")}>
          {error ? (
            <div className="max-w-[85%] md:max-w-[80%] p-3 rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 shadow-sm">
              <p className="text-sm text-red-700 dark:text-red-300">{content}</p>
            </div>
          ) : (
            <div
              className={cn(
                "relative rounded-2xl shadow-sm transition-all duration-300 group/bubble min-w-0 max-w-[85%] md:max-w-[80%]",
                role === "user" ? theme.bubbleUser : cn(theme.bubbleBg, theme.bubbleBorder, theme.text),
                shouldShowFold && isFolded ? "overflow-hidden" : "max-h-none",
              )}
              style={shouldShowFold && isFolded ? { maxHeight: `${MAX_MESSAGE_HEIGHT}px` } : {}}
            >
              {/* Floating Copy Button - Internal Top Right */}
              {!error && (
                <div className="absolute top-2 right-2 z-30">
                  <button
                    onClick={handleCopy}
                    className={cn(
                      "p-1.5 rounded-lg border shadow-sm transition-all active:scale-90",
                      role === "user"
                        ? "bg-white/10 border-white/20 text-white/80 hover:bg-white/30"
                        : "bg-zinc-50 dark:bg-zinc-800/50 border-zinc-200 dark:border-zinc-700 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 backdrop-blur-sm",
                      copied && (role === "user" ? "bg-white/40 border-white/40" : "bg-green-50 dark:bg-green-900/20 border-green-200 text-green-600")
                    )}
                  >
                    {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                </div>
              )}

              {/* Content and Markdown */}
              <div ref={contentRef} className="pl-4 pr-10 py-2.5">
                {role === "assistant" ? (
                  <div className="prose prose-sm dark:prose-invert max-w-none break-words text-sm font-normal font-sans">
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm, remarkBreaks]}
                      components={{
                        a: ({ node, ...props }) => (
                          <a {...props} className="text-blue-500 hover:underline" target="_blank" rel="noopener noreferrer" />
                        ),
                        p: ({ node, ...props }) => <p {...props} className="mb-1 last:mb-0 text-sm leading-relaxed" />,
                        pre: ({ node, ...props }) => <CodeBlock {...props} />,
                        code: ({ className, children, ...props }: any) =>
                          props.inline ? (
                            <code className={cn("px-1.5 py-0.5 rounded-md bg-zinc-100 dark:bg-zinc-800 text-xs", className)} {...props}>
                              {children}
                            </code>
                          ) : (
                            <code className={className} {...props}>
                              {children}
                            </code>
                          ),
                      }}
                    >
                      {content || "..."}
                    </ReactMarkdown>
                    {children}
                  </div>
                ) : (
                  <div className="whitespace-pre-wrap break-words text-sm font-sans">{content}</div>
                )}
              </div>

              {/* Fold Mask and Button */}
              {shouldShowFold && (
                <>
                  {isFolded && (
                    <div className="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-white/95 via-white/40 to-transparent dark:from-zinc-800/95 dark:via-zinc-800/40 pointer-events-none" />
                  )}
                  <div className={cn("flex justify-center p-1.5", isFolded ? "absolute bottom-0 inset-x-0 z-10" : "relative")}>
                    <button
                      onClick={toggleFold}
                      className="flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold uppercase bg-white dark:bg-zinc-700 border border-zinc-200 dark:border-zinc-600 shadow-sm hover:bg-zinc-50 dark:hover:bg-zinc-600 text-zinc-500"
                    >
                      {isFolded ? (
                        <><ChevronDown className="w-3 h-3" />{t("common.expand") || "Expand"}</>
                      ) : (
                        <><ChevronUp className="w-3 h-3" />{t("common.collapse") || "Collapse"}</>
                      )}
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

