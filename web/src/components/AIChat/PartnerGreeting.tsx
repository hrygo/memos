import { memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Sparkles, MessageSquare } from "lucide-react";
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
      greeting: "ai.partner.greeting-morning", // 早上好
      timeOfDay: "morning",
      emoji: "🌅",
    };
  }
  if (hour >= 12 && hour < 18) {
    return {
      greeting: "ai.partner.greeting-afternoon", // 下午好
      timeOfDay: "afternoon",
      emoji: "☀️",
    };
  }
  if (hour >= 18 && hour < 22) {
    return {
      greeting: "ai.partner.greeting-evening", // 晚上好
      timeOfDay: "evening",
      emoji: "🌆",
    };
  }
  return {
    greeting: "ai.partner.greeting-night", // 夜深了
    timeOfDay: "night",
    emoji: "🌙",
  };
}

/**
 * Partner Greeting - 精简优化的欢迎界面
 *
 * UX/UI 改进：
 * - 简化视觉元素，聚焦核心操作
 * - 统一卡片样式和间距
 * - 优化快捷操作的视觉层次
 * - 移除冗余的状态指示
 */
export const PartnerGreeting = memo(function PartnerGreeting({
  userName,
  recentMemoCount = 0,
  upcomingScheduleCount = 0,
  conversationCount = 0,
  onQuickAction,
  className,
}: PartnerGreetingProps) {
  const { t } = useTranslation();
  const { greeting, timeOfDay, emoji } = useMemo(() => getTimeBasedGreeting(), []);

  const greetingText = t(greeting);

  // 生成时间相关提示
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
      { key: "memo" as const, icon: "🦜", labelKey: "ai.partner.quick-memo" },
      { key: "schedule" as const, icon: "⏰", labelKey: "ai.partner.quick-schedule" },
      { key: "summary" as const, icon: "🌟", labelKey: "ai.partner.quick-summary" },
      { key: "chat" as const, icon: "💬", labelKey: "ai.partner.quick-chat" },
    ],
    [],
  );

  return (
    <div className={cn("flex flex-col items-center justify-center h-full w-full", className)}>
      <div className="w-full max-w-sm px-4 flex flex-col items-center">
        {/* 主图标区域 */}
        <div className="relative mb-5">
          <div className="relative w-14 h-14 rounded-2xl bg-gradient-to-br from-emerald-500 to-green-600 flex items-center justify-center text-3xl shadow-lg">
            🦜
          </div>
          <div className="absolute -bottom-0.5 -right-0.5 w-4 h-4 rounded-full bg-green-500 border-2 border-white dark:border-zinc-900 flex items-center justify-center">
            <Sparkles className="w-2.5 h-2.5 text-white" />
          </div>
        </div>

        {/* 问候语 */}
        <div className="text-center mb-5">
          <div className="flex items-center justify-center gap-1.5 mb-1">
            <span className="text-xl">{emoji}</span>
            <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
              {greetingText}
            </h2>
          </div>
          <p className="text-xs text-zinc-500 dark:text-zinc-400">{timeHint}</p>
        </div>

        {/* 快捷操作 - 简化为紧凑的行布局 */}
        <div className="grid grid-cols-4 gap-2 w-full mb-4">
          {quickActions.map((action) => (
            <button
              key={action.key}
              onClick={() => onQuickAction?.(action.key)}
              className={cn(
                "flex flex-col items-center gap-1 p-2.5 rounded-xl border",
                "bg-white dark:bg-zinc-800",
                "border-zinc-200 dark:border-zinc-700",
                "hover:border-zinc-300 dark:hover:border-zinc-600",
                "hover:bg-zinc-50 dark:hover:bg-zinc-700/50",
                "transition-all duration-150",
                "active:scale-95",
              )}
              title={t(action.labelKey)}
            >
              <span className="text-xl">{action.icon}</span>
              <span className="text-[10px] font-medium text-zinc-700 dark:text-zinc-300 leading-tight text-center">
                {t(action.labelKey)}
              </span>
            </button>
          ))}
        </div>

        {/* 底部提示 */}
        <p className="text-[10px] text-zinc-400 dark:text-zinc-600 flex items-center gap-1">
          <MessageSquare className="w-3 h-3" />
          {t("ai.partner.input-hint") || "直接输入消息，我会自动理解你的意图"}
        </p>
      </div>
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
