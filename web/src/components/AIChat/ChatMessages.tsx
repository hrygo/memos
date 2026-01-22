import { UserIcon } from "lucide-react";
import { ReactNode, useRef, useEffect } from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "@/components/MemoContent/CodeBlock";
import MessageActions from "@/components/AIChat/MessageActions";
import TypingCursor from "@/components/AIChat/TypingCursor";
import { PARROT_THEMES, PARROT_ICONS } from "@/types/parrot";
import { cn } from "@/lib/utils";
import { ChatItem, ConversationMessage } from "@/types/aichat";
import { ParrotAgentType } from "@/types/parrot";

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
  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [items, isTyping]);

  const theme = currentParrotId
    ? PARROT_THEMES[currentParrotId] || PARROT_THEMES.DEFAULT
    : PARROT_THEMES.DEFAULT;
  const currentIcon = currentParrotId
    ? PARROT_ICONS[currentParrotId] || PARROT_ICONS.DEFAULT
    : PARROT_ICONS.DEFAULT;

  return (
    <div
      ref={scrollRef}
      className={cn("flex-1 overflow-y-auto px-3 md:px-6 py-4", className)}
    >
      {children}

      <div className="max-w-3xl mx-auto space-y-4">
        {items.map((item, index) => {
          // Context separator
          if ("type" in item && item.type === "context-separator") {
            return (
              <div key={`separator-${index}`} className="flex items-center gap-4 py-2">
                <div className="flex-1 h-px bg-border" />
                <span className="text-xs text-muted-foreground whitespace-nowrap">
                  Context cleared
                </span>
                <div className="flex-1 h-px bg-border" />
              </div>
            );
          }

          const msg = item as ConversationMessage;
          const isLastMessage = index === items.length - 1;

          return (
            <MessageBubble
              key={msg.id}
              message={msg}
              theme={theme}
              icon={msg.role === "user" ? undefined : currentIcon}
              isLastAssistant={msg.role === "assistant" && isLastMessage}
              onCopy={() => onCopyMessage?.(msg.content)}
              onRegenerate={onRegenerate}
              onDelete={() => onDeleteMessage?.(index)}
            >
              {msg.role === "assistant" && isTyping && isLastMessage && !msg.error && (
                <TypingCursor active={true} />
              )}
            </MessageBubble>
          );
        })}

        {/* Typing indicator */}
        {isTyping && (() => {
          const lastItem = items[items.length - 1];
          if (!lastItem) return true;
          if ("type" in lastItem && lastItem.type === "context-separator") return true;
          return "role" in lastItem && lastItem.role !== "assistant";
        })() && (
          <div className="flex gap-3 md:gap-4">
            <div className={cn("w-8 h-8 md:w-9 md:h-9 rounded-full flex items-center justify-center", theme.iconBg)}>
              <span className="text-base md:text-lg">{currentIcon}</span>
            </div>
            <div className={cn("px-3 md:px-4 py-2 md:py-3 rounded-2xl border-2", theme.bubbleBg, theme.bubbleBorder)}>
              <div className="flex gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-zinc-400 animate-bounce [animation-delay:-0.3s]" />
                <span className="w-1.5 h-1.5 rounded-full bg-zinc-400 animate-bounce [animation-delay:-0.15s]" />
                <span className="w-1.5 h-1.5 rounded-full bg-zinc-400 animate-bounce" />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

interface MessageBubbleProps {
  message: ConversationMessage;
  theme: typeof PARROT_THEMES[keyof typeof PARROT_THEMES];
  icon?: string;
  isLastAssistant?: boolean;
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
        role === "user" ? "flex-row-reverse" : "flex-row"
      )}
    >
      {/* Avatar */}
      <div
        className={cn(
          "w-8 h-8 md:w-9 md:h-9 rounded-full flex items-center justify-center shrink-0",
          role === "user"
            ? theme.bubbleUser
            : theme.iconBg
        )}
      >
        {role === "user" ? (
          <UserIcon className="w-4 h-4" />
        ) : (
          <span className="text-base md:text-lg">{icon || "🤖"}</span>
        )}
      </div>

      {/* Message content */}
      <div className="flex-1 min-w-0">
        {role === "assistant" && isLastAssistant && onCopy && onRegenerate && onDelete && (
          <div className="flex items-start gap-2 mb-1">
            <MessageActions
              onCopy={onCopy}
              onRegenerate={onRegenerate}
              onDelete={onDelete}
            />
          </div>
        )}

        {error ? (
          <div className="p-3 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
            <p className="text-sm text-red-700 dark:text-red-300">{content}</p>
          </div>
        ) : (
          <div
            className={cn(
              "max-w-[85%] md:max-w-[80%] rounded-2xl px-3 md:px-4 py-2 md:py-3",
              role === "user"
                ? theme.bubbleUser
                : cn(theme.bubbleBg, theme.bubbleBorder, theme.text)
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
                    code: ({ node, className, children, ...props }: any) =>
                      props.inline ? (
                        <code className={cn("px-1 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-sm", className)} {...props}>
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
              <div className="whitespace-pre-wrap break-words text-sm">{content}</div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
