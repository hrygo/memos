import { BotIcon, LoaderIcon, SendIcon, SparklesIcon, UserIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import MobileHeader from "@/components/MobileHeader";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useChatWithMemos } from "@/hooks/useAIQueries";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

interface Message {
  role: "user" | "assistant";
  content: string;
}

const AIChat = () => {
  const { t } = useTranslation();
  const md = useMediaQuery("md");
  const [input, setInput] = useState("");
  const [messages, setMessages] = useState<Message[]>([]);
  const [isTyping, setIsTyping] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const chatHook = useChatWithMemos();

  const scrollToBottom = () => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isTyping]);

  const handleSend = async () => {
    if (!input.trim() || isTyping) return;

    const userMessage = input.trim();
    setInput("");
    setMessages((prev) => [...prev, { role: "user", content: userMessage }]);
    setIsTyping(true);

    try {
      const history = messages.map((m) => m.content);
      let currentAssistantMessage = "";
      setMessages((prev) => [...prev, { role: "assistant", content: "" }]);

      await chatHook.stream(
        { message: userMessage, history },
        {
          onContent: (content) => {
            currentAssistantMessage += content;
            setMessages((prev) => {
              const newMessages = [...prev];
              newMessages[newMessages.length - 1].content = currentAssistantMessage;
              return newMessages;
            });
          },
          onDone: () => {
            setIsTyping(false);
          },
          onError: (err) => {
            console.error("Chat error:", err);
            setIsTyping(false);
            setMessages((prev) => [...prev, { role: "assistant", content: "⚠️ Error: " + err.message }]);
          },
        },
      );
    } catch (_error) {
      setIsTyping(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <section className="w-full h-[calc(100vh-4rem)] md:h-[calc(100vh-2rem)] flex flex-col relative">
      {!md && (
        <MobileHeader>
          <div className="flex flex-row justify-between items-center w-full">
            <div className="flex items-center gap-1 font-medium text-foreground">
              <SparklesIcon className="w-5 h-5 text-blue-500" />
              {t("common.ai-assistant")}
            </div>
          </div>
        </MobileHeader>
      )}

      {/* Messages Area - Flexible height */}
      <div className="flex-1 overflow-y-auto px-4 py-6 space-y-6" ref={scrollRef}>
        {messages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground opacity-50 pb-20">
            <BotIcon className="w-16 h-16 mb-4" />
            <p className="text-lg">{t("common.ai-greeting")}</p>
          </div>
        )}

        {messages.map((msg, index) => (
          <div key={index} className={cn("flex gap-4 max-w-3xl mx-auto", msg.role === "user" ? "flex-row-reverse" : "flex-row")}>
            <div
              className={cn(
                "w-8 h-8 rounded-full flex items-center justify-center shrink-0 mt-1 shadow-sm",
                msg.role === "user"
                  ? "bg-primary text-primary-foreground"
                  : "bg-blue-100 text-blue-600 dark:bg-blue-900 dark:text-blue-300",
              )}
            >
              {msg.role === "user" ? <UserIcon size={16} /> : <BotIcon size={16} />}
            </div>

            <div
              className={cn(
                "flex-1 min-w-0 rounded-2xl p-4 text-sm leading-relaxed shadow-sm",
                msg.role === "user"
                  ? "bg-primary text-primary-foreground rounded-tr-sm"
                  : "bg-white dark:bg-zinc-800 border border-border/50 rounded-tl-sm",
              )}
            >
              {msg.role === "assistant" ? (
                <div className="prose dark:prose-invert prose-sm max-w-none break-words">
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm, remarkBreaks]}
                    components={{
                      a: ({ node, ...props }) => (
                        <a {...props} className="text-blue-500 hover:underline" target="_blank" rel="noopener noreferrer" />
                      ),
                      p: ({ node, ...props }) => <p {...props} className="mb-2 last:mb-0" />,
                    }}
                  >
                    {msg.content || "..."}
                  </ReactMarkdown>
                </div>
              ) : (
                <div className="whitespace-pre-wrap break-words">{msg.content}</div>
              )}
            </div>
          </div>
        ))}

        {isTyping && messages[messages.length - 1]?.role !== "assistant" && (
          <div className="flex gap-4 max-w-3xl mx-auto">
            <div className="w-8 h-8 rounded-full bg-blue-100 text-blue-600 dark:bg-blue-900 dark:text-blue-300 flex items-center justify-center shrink-0 shadow-sm mt-1">
              <LoaderIcon className="w-4 h-4 animate-spin" />
            </div>
          </div>
        )}
      </div>

      {/* Input Area - Fixed at bottom */}
      <div className="shrink-0 p-4 border-t bg-background/80 backdrop-blur-md sticky bottom-0 z-10">
        <div className="max-w-3xl mx-auto relative flex items-end gap-2 p-2 bg-muted/50 rounded-xl border focus-within:ring-1 focus-within:ring-ring focus-within:bg-background transition-all">
          <Textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t("common.ai-placeholder")}
            className="min-h-[44px] max-h-[150px] w-full resize-none border-0 bg-transparent focus-visible:ring-0 px-3 py-2.5 shadow-none"
            rows={1}
            style={{ height: "auto" }}
            onInput={(e) => {
              const target = e.target as HTMLTextAreaElement;
              target.style.height = "auto";
              target.style.height = `${Math.min(target.scrollHeight, 150)}px`;
            }}
          />
          <Button
            size="icon"
            className="shrink-0 h-9 w-9 mb-1 mr-1 rounded-lg transition-all"
            onClick={handleSend}
            disabled={!input.trim() || isTyping}
          >
            <SendIcon className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </section>
  );
};

export default AIChat;
