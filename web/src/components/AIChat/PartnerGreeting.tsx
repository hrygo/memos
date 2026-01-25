import { memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Sparkles, Clock, MessageSquare, Sun, Moon } from "lucide-react";
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
function getTimeBasedGreeting(): { icon: React.ReactNode; greeting: string; timeOfDay: string } {
  const hour = new Date().getHours();
  const t = (key: string) => key; // 简化版，实际使用i18n

  if (hour >= 5 && hour < 12) {
    return {
      icon: <Sun className="w-5 h-5 text-amber-500" />,
      greeting: t("ai.partner.greeting-morning") || "早上好",
      timeOfDay: "morning",
    };
  }
  if (hour >= 12 && hour < 18) {
    return {
      icon: <Sun className="w-5 h-5 text-orange-500" />,
      greeting: t("ai.partner.greeting-afternoon") || "下午好",
      timeOfDay: "afternoon",
    };
  }
  if (hour >= 18 && hour < 22) {
    return {
      icon: <Moon className="w-5 h-5 text-indigo-500" />,
      greeting: t("ai.partner.greeting-evening") || "晚上好",
      timeOfDay: "evening",
    };
  }
  return {
    icon: <Moon className="w-5 h-5 text-slate-500" />,
    greeting: t("ai.partner.greeting-night") || "夜深了",
    timeOfDay: "night",
  };
}

/**
 * 伙伴型问候组件
 * 提供个性化的、有温度的问候，展示用户数据概览
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
  const { icon, greeting, timeOfDay } = useMemo(() => getTimeBasedGreeting(), [t]);

  // 生成个性化问候消息
  const personalizedMessage = useMemo(() => {
    const messages: string[] = [];

    // 时间相关
    if (timeOfDay === "morning") {
      messages.push("今天是个创造的好天气 ☀️");
    } else if (timeOfDay === "afternoon") {
      messages.push("下午茶时间，来聊聊？");
    } else if (timeOfDay === "evening") {
      messages.push("辛苦了一天，放松一下 🌙");
    } else {
      messages.push("夜深了，注意休息");
    }

    // 数据相关
    const dataHints: string[] = [];
    if (recentMemoCount > 0) {
      dataHints.push(`你最近记录了 ${recentMemoCount} 条笔记`);
    }
    if (upcomingScheduleCount > 0) {
      dataHints.push(`今天还有 ${upcomingScheduleCount} 个日程`);
    }
    if (conversationCount > 3) {
      dataHints.push("我们聊了很多次了");
    }

    return {
      greeting,
      hint: messages[0] || "今天想聊点什么？",
      dataHint: dataHints.length > 0 ? dataHints.join("，") + "..." : null,
    };
  }, [timeOfDay, recentMemoCount, upcomingScheduleCount, conversationCount, greeting]);

  // 快捷操作配置
  const quickActions = useMemo(
    () => [
      {
        key: "memo" as const,
        icon: "🦜",
        label: t("ai.partner.quick-memo") || "查看笔记",
        description: t("ai.partner.quick-memo-desc") || "搜索最近的记录",
        color: "bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border-slate-200 dark:border-slate-700",
      },
      {
        key: "schedule" as const,
        icon: "⏰",
        label: t("ai.partner.quick-schedule") || "查看日程",
        description: t("ai.partner.quick-schedule-desc") || "今天的安排",
        color: "bg-cyan-100 dark:bg-cyan-900/30 text-cyan-700 dark:text-cyan-300 border-cyan-200 dark:border-cyan-700",
      },
      {
        key: "summary" as const,
        icon: "🌟",
        label: t("ai.partner.quick-summary") || "今日总结",
        description: t("ai.partner.quick-summary-desc") || "笔记 + 日程",
        color: "bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300 border-emerald-200 dark:border-emerald-700",
      },
      {
        key: "chat" as const,
        icon: "💬",
        label: t("ai.partner.quick-chat") || "随便聊聊",
        description: t("ai.partner.quick-chat-desc") || "自由对话",
        color: "bg-indigo-100 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300 border-indigo-200 dark:border-indigo-700",
      },
    ],
    [t],
  );

  return (
    <div className={cn("flex flex-col items-center justify-center h-full px-6 py-8", className)}>
      {/* 主图标和问候 */}
      <div className="relative mb-6">
        {/* 背景装饰 */}
        <div className="absolute inset-0 bg-gradient-to-br from-indigo-100 to-purple-100 dark:from-indigo-900/30 dark:to-purple-900/30 rounded-full blur-2xl opacity-60" />

        {/* 主图标 */}
        <div className="relative w-20 h-20 md:w-24 md:h-24 rounded-2xl bg-gradient-to-br from-indigo-500 to-purple-600 dark:from-indigo-600 dark:to-purple-700 flex items-center justify-center text-4xl shadow-lg">
          🦜
        </div>

        {/* 状态指示 */}
        <div className="absolute -bottom-1 -right-1 w-8 h-8 bg-green-500 rounded-full border-4 border-white dark:border-zinc-900 flex items-center justify-center">
          <Sparkles className="w-3 h-3 text-white" />
        </div>
      </div>

      {/* 问候语 */}
      <h2 className="text-xl md:text-2xl font-bold text-zinc-900 dark:text-zinc-100 mb-2">
        {personalizedMessage.greeting}！{userName ? ` ${userName}` : ""}
      </h2>

      {/* 个性化提示 */}
      <p className="text-sm md:text-base text-zinc-600 dark:text-zinc-400 mb-1 text-center max-w-md">
        {personalizedMessage.hint}
      </p>

      {/* 数据感知提示 */}
      {personalizedMessage.dataHint && (
        <p className="text-xs text-zinc-500 dark:text-zinc-500 mb-6 flex items-center gap-1.5">
          <Clock className="w-3 h-3" />
          {personalizedMessage.dataHint}
        </p>
      )}

      {/* 快捷操作 */}
      <div className="grid grid-cols-2 gap-3 w-full max-w-lg">
        {quickActions.map((action) => (
          <button
            key={action.key}
            onClick={() => onQuickAction?.(action.key)}
            className={cn(
              "flex flex-col items-start p-4 rounded-xl border-2 transition-all duration-200",
              "hover:scale-102 hover:shadow-md active:scale-98",
              action.color,
            )}
          >
            <span className="text-2xl mb-2">{action.icon}</span>
            <span className="font-semibold text-sm">{action.label}</span>
            <span className="text-xs opacity-70 mt-0.5">{action.description}</span>
          </button>
        ))}
      </div>

      {/* 底部提示 */}
      <p className="mt-8 text-xs text-zinc-400 dark:text-zinc-600 flex items-center gap-1.5">
        <MessageSquare className="w-3 h-3" />
        {t("ai.partner.input-hint") || "直接输入消息，我会自动理解你的意图"}
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
  const { greeting } = useMemo(() => getTimeBasedGreeting(), [t]);

  const capabilityEmojis: Record<CapabilityType, string> = {
    [CapabilityType.MEMO]: "🦜",
    [CapabilityType.SCHEDULE]: "⏰",
    [CapabilityType.AMAZING]: "🌟",
    [CapabilityType.CREATIVE]: "💡",
    [CapabilityType.AUTO]: "🤖",
  };

  return (
    <div className={cn("flex items-start gap-3 p-4", className)}>
      <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-xl shrink-0">
        {capability ? capabilityEmojis[capability] : "🦜"}
      </div>
      <div className="flex-1">
        <p className="font-medium text-zinc-900 dark:text-zinc-100 mb-1">
          {greeting}！{message || "今天想聊点什么？"}
        </p>
        <p className="text-xs text-zinc-500 dark:text-zinc-500">
          我可以帮你搜索笔记、管理日程，或者一起头脑风暴 💡
        </p>
      </div>
    </div>
  );
});
