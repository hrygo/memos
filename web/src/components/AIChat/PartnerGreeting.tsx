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
  onQuickAction?: (action: "memo" | "schedule" | "summary" | "chat") => void;
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
 * Partner Greeting - 精简优化的欢迎界面
 *
 * UX/UI 设计原则：
 * - 清晰的视觉层次：问候语 > 快捷操作 > 提示文本
 * - 统一的间距系统：基于 4px 的倍数
 * - 简洁的交互：明确的点击反馈
 */
export const PartnerGreeting = memo(function PartnerGreeting({
  onQuickAction,
  className,
}: PartnerGreetingProps) {
  const { t } = useTranslation();
  const { greeting, timeOfDay } = useMemo(() => getTimeBasedGreeting(), []);

  const greetingText = t(greeting);

  // 时间相关提示
  const timeHint = useMemo(() => {
    const hints = {
      morning: "新的一天，有什么计划？",
      afternoon: "下午茶时间，来聊聊？",
      evening: "辛苦了一天，放松一下",
      night: "夜深了，注意休息",
    };
    return hints[timeOfDay as keyof typeof hints];
  }, [timeOfDay]);

  // 快捷操作配置
  const quickActions = useMemo(
    () => [
      { key: "memo" as const, icon: "🦜", labelKey: "ai.parrot.partner.quick-memo" },
      { key: "schedule" as const, icon: "⏰", labelKey: "ai.parrot.partner.quick-schedule" },
      { key: "summary" as const, icon: "🌟", labelKey: "ai.parrot.partner.quick-summary" },
      { key: "chat" as const, icon: "💬", labelKey: "ai.parrot.partner.quick-chat" },
    ],
    [],
  );

  return (
    <div className={cn("flex flex-col items-center justify-center h-full w-full px-6 py-8", className)}>
      {/* 主图标 - 简化设计 */}
      <div className="mb-6">
        <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-emerald-500 to-green-600 flex items-center justify-center text-3xl shadow-sm">
          🦜
        </div>
      </div>

      {/* 问候语区域 - 主要内容 */}
      <div className="text-center mb-8">
        <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100 mb-2">
          {greetingText}
        </h2>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">{timeHint}</p>
      </div>

      {/* 快捷操作 - 统一样式 */}
      <div className="grid grid-cols-2 gap-3 w-full mb-8">
        {quickActions.map((action) => (
          <button
            key={action.key}
            onClick={() => onQuickAction?.(action.key)}
            className={cn(
              "flex flex-row items-center gap-3 p-3 rounded-xl",
              "bg-white dark:bg-zinc-800",
              "border border-zinc-200 dark:border-zinc-700",
              "hover:border-emerald-300 dark:hover:border-emerald-700",
              "hover:bg-emerald-50 dark:hover:bg-emerald-900/20",
              "transition-all duration-200",
              "active:scale-95",
              "min-h-[56px]",
            )}
            title={t(action.labelKey)}
          >
            <span className="text-2xl shrink-0">{action.icon}</span>
            <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300 text-left leading-tight">
              {t(action.labelKey)}
            </span>
          </button>
        ))}
      </div>

      {/* 底部提示 - 次要信息 */}
      <p className="text-xs text-zinc-400 dark:text-zinc-500 flex items-center gap-1.5">
        <MessageSquare className="w-3.5 h-3.5" />
        {t("ai.parrot.partner.input-hint") || "直接输入消息，我会自动理解你的意图"}
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

export const MiniPartnerGreeting = memo(function MiniPartnerGreeting({
  message,
  capability,
  className,
}: MiniPartnerGreetingProps) {
  const { t } = useTranslation();
  const { greeting } = useMemo(() => getTimeBasedGreeting(), []);
  const greetingText = t(greeting);

  const capabilityEmojis: Record<CapabilityType, string> = {
    [CapabilityType.MEMO]: "🦜",
    [CapabilityType.SCHEDULE]: "⏰",
    [CapabilityType.AMAZING]: "🌟",
    [CapabilityType.CREATIVE]: "💡",
    [CapabilityType.AUTO]: "🤖",
  };

  return (
    <div className={cn("flex items-start gap-3 p-4", className)}>
      <div className="w-9 h-9 md:w-10 md:h-10 rounded-xl bg-gradient-to-br from-emerald-500 to-green-600 flex items-center justify-center text-lg shrink-0 shadow-sm">
        {capability ? capabilityEmojis[capability] : "🦜"}
      </div>
      <div className="flex-1 min-w-0">
        <p className="font-medium text-zinc-900 dark:text-zinc-100 mb-1">
          {greetingText}！{message || "今天想聊点什么？"}
        </p>
        <p className="text-xs text-zinc-500 dark:text-zinc-500 line-clamp-2">
          我可以帮你搜索笔记、管理日程，或者一起头脑风暴 💡
        </p>
      </div>
    </div>
  );
});
