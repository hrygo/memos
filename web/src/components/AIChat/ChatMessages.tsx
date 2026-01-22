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

  return (
    <div ref={scrollRef} onScroll={handleScroll} className={cn("flex-1 overflow-y-auto px-3 md:px-6 py-4", className)}>
      {children}

      <div className="max-w-3xl mx-auto space-y-4">
        {items.map((item, index) => {
          // Context separator
          if ("type" in item && item.type === "context-separator") {
            return (
              <div key={`separator-${index}`} className="flex items-center gap-4 py-4 animate-in fade-in duration-300">
                <div className="flex-1 h-px bg-zinc-300 dark:bg-zinc-700" />
                <span className="text-xs text-zinc-500 dark:text-zinc-500 whitespace-nowrap font-medium">{t("ai.context-cleared")}</span>
                <div className="flex-1 h-px bg-zinc-300 dark:bg-zinc-700" />
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

  return (
    <div
      className={cn(
        "flex gap-3 md:gap-4",
        role === "user" ? "flex-row-reverse" : "flex-row",
        isNew && "animate-in fade-in slide-in-from-bottom-3 duration-300",
      )}
    >
      {/* Avatar - Larger for better touch */}
      <div
        className={cn(
          "w-9 h-9 md:w-10 md:h-10 rounded-full flex items-center justify-center shrink-0 shadow-sm overflow-hidden",
          role === "user" ? "" : "",
        )}
      >
        {role === "user" ? (
          <img src="/images/parrots/icons/user_avatar.png" alt="User" className="w-full h-full object-cover" />
        ) : icon?.startsWith("/") ? (
          <img src={icon} alt="" className="w-8 h-8 md:w-9 md:h-9 object-contain" />
        ) : (
          <span className="text-lg md:text-xl">{icon || "🤖"}</span>
        )}
      </div>

      {/* Message content */}
      <div className="flex-1 min-w-0">
        {role === "assistant" && isLastAssistant && onCopy && onRegenerate && onDelete && (
          <div className="flex items-start gap-2 mb-1.5">
            <MessageActions onCopy={onCopy} onRegenerate={onRegenerate} onDelete={onDelete} />
          </div>
        )}

        {error ? (
          <div className="p-4 rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 shadow-sm">
            <p className="text-sm text-red-700 dark:text-red-300">{content}</p>
          </div>
        ) : (
          <div
            className={cn(
              "max-w-[85%] md:max-w-[80%] rounded-2xl px-4 py-3 shadow-sm",
              role === "user" ? theme.bubbleUser : cn(theme.bubbleBg, theme.bubbleBorder, theme.text),
            )}
          >
            {role === "assistant" ? (
              <div className="prose prose-sm dark:prose-invert max-w-none break-words">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm, remarkBreaks]}
                  components={{
                    a: ({ node, ...props }) => (
                      <a {...props} className="text-blue-500 hover:underline" target="_blank" rel="noopener noreferrer" />
                    ),
                    p: ({ node, ...props }) => <p {...props} className="mb-2 last:mb-0" />,
                    pre: ({ node, ...props }) => <CodeBlock {...props} />,
                    code: (
                      { className, children, ...props }: any, // biome-ignore lint/suspicious/noExplicitAny: react-markdown component types
                    ) =>
                      props.inline ? (
                        <code className={cn("px-1.5 py-0.5 rounded-md bg-zinc-100 dark:bg-zinc-800 text-sm", className)} {...props}>
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
              <div className="whitespace-pre-wrap break-words text-sm md:text-base">{content}</div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
