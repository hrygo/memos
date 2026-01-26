import { MessageSquare } from "lucide-react";
import { memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { CapabilityType } from "@/types/capability";

interface PartnerGreetingProps {
  userName?: string;
  recentMemoCount?: number;
  upcomingScheduleCount?: number;
  conversationCount?: number;
  onSendMessage?: (message: string) => void;
  className?: string;
}

/**
 * 获取时间相关的问候语
 */
function getTimeBasedGreeting(): { greeting: string; timeOfDay: string; emoji: string } {
  const hour = new Date().getHours();

  if (hour >= 5 && hour < 12) {
    return {
      greeting: "ai.parrot.partner.greeting-morning",
      timeOfDay: "morning",
      emoji: "🌅",
    };
  }
  if (hour >= 12 && hour < 18) {
    return {
      greeting: "ai.parrot.partner.greeting-afternoon",
      timeOfDay: "afternoon",
      emoji: "☀️",
    };
  }
  if (hour >= 18 && hour < 22) {
    return {
      greeting: "ai.parrot.partner.greeting-evening",
      timeOfDay: "evening",
      emoji: "🌆",
    };
  }
  return {
    greeting: "ai.parrot.partner.greeting-night",
    timeOfDay: "night",
    emoji: "🌙",
  };
}

/**
 * Partner Greeting - 统一入口设计
 *
 * UX/UI 设计原则：
 * - 示例提问代替能力选择，降低认知负担
 * - 用户无需理解系统内部能力边界
 * - 点击示例直接发送消息，智能路由自动处理
 */
export const PartnerGreeting = memo(function PartnerGreeting({ onSendMessage, className }: PartnerGreetingProps) {
  const { t } = useTranslation();
  const { greeting, timeOfDay } = useMemo(() => getTimeBasedGreeting(), []);

  const greetingText = t(greeting);

  // 时间相关提示（国际化）
  const timeHint = useMemo(() => {
    const hintKeys: Record<string, string> = {
      morning: "ai.parrot.partner.hint-morning",
      afternoon: "ai.parrot.partner.hint-afternoon",
      evening: "ai.parrot.partner.hint-evening",
      night: "ai.parrot.partner.hint-night",
    };
    return t(hintKeys[timeOfDay]);
  }, [timeOfDay, t]);

  // 示例提问 - 用户意图导向，而非能力导向
  const suggestedPrompts = useMemo(
    () => [
      { icon: "📝", promptKey: "ai.parrot.partner.prompt-memo", prompt: t("ai.parrot.partner.prompt-memo") },
      { icon: "📅", promptKey: "ai.parrot.partner.prompt-schedule", prompt: t("ai.parrot.partner.prompt-schedule") },
      { icon: "📊", promptKey: "ai.parrot.partner.prompt-summary", prompt: t("ai.parrot.partner.prompt-summary") },
      { icon: "✨", promptKey: "ai.parrot.partner.prompt-creative", prompt: t("ai.parrot.partner.prompt-creative") },
    ],
    [t],
  );

  return (
    <div className={cn("flex flex-col items-center justify-center h-full w-full px-6 py-8", className)}>
      {/* 主图标 */}
      <div className="mb-6">
        <div className="w-16 h-16 rounded-2xl bg-primary flex items-center justify-center text-3xl shadow-sm">🦜</div>
      </div>

      {/* 问候语区域 */}
      <div className="text-center mb-8">
        <h2 className="text-xl font-semibold text-foreground mb-2">{greetingText}</h2>
        <p className="text-sm text-muted-foreground">{timeHint}</p>
      </div>

      {/* 示例提问 - 点击直接发送 */}
      <div className="grid grid-cols-2 gap-3 w-full mb-8">
        {suggestedPrompts.map((item) => (
          <button
            key={item.promptKey}
            onClick={() => onSendMessage?.(item.prompt)}
            className={cn(
              "flex flex-row items-center gap-3 p-3 rounded-xl",
              "bg-card",
              "border border-border",
              "hover:border-primary/50",
              "hover:bg-accent",
              "transition-all duration-200",
              "active:scale-95",
              "min-h-[56px]",
            )}
            title={item.prompt}
          >
            <span className="text-2xl shrink-0">{item.icon}</span>
            <span className="text-sm font-medium text-foreground text-left leading-tight line-clamp-2">{item.prompt}</span>
          </button>
        ))}
      </div>

      {/* 底部提示 */}
      <p className="text-xs text-muted-foreground flex items-center gap-1.5">
        <MessageSquare className="w-3.5 h-3.5" />
        {t("ai.parrot.partner.input-hint")}
      </p>
    </div>
  );
});

/**
 * 简化版伙伴问候 - 用于对话列表中展示
 */
interface MiniPartnerGreetingProps {
  message?: string;
  capability?: CapabilityType;
  className?: string;
}

export const MiniPartnerGreeting = memo(function MiniPartnerGreeting({ message, capability, className }: MiniPartnerGreetingProps) {
  const { t } = useTranslation();
  const { greeting } = useMemo(() => getTimeBasedGreeting(), []);
  const greetingText = t(greeting);

  const capabilityEmojis: Record<CapabilityType, string> = {
    [CapabilityType.MEMO]: "🦜",
    [CapabilityType.SCHEDULE]: "⏰",
    [CapabilityType.AMAZING]: "🌟",
    [CapabilityType.AUTO]: "🤖",
  };

  return (
    <div className={cn("flex items-start gap-3 p-4", className)}>
      <div className="w-9 h-9 md:w-10 md:h-10 rounded-xl bg-primary flex items-center justify-center text-lg shrink-0 shadow-sm">
        {capability ? capabilityEmojis[capability] : "🦜"}
      </div>
      <div className="flex-1 min-w-0">
        <p className="font-medium text-foreground mb-1">{greetingText}</p>
        <p className="text-xs text-muted-foreground line-clamp-2">{message || t("ai.parrot.partner.default-hint")}</p>
      </div>
    </div>
  );
});
