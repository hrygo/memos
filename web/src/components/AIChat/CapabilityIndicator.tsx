import { Sparkles, Clock, Calendar, Lightbulb, Wand2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { CapabilityType, CapabilityStatus, getCapabilitySound } from "@/types/capability";

interface CapabilityIndicatorProps {
  capability: CapabilityType;
  status: CapabilityStatus;
  onCapabilityChange?: (capability: CapabilityType) => void;
  className?: string;
  compact?: boolean;
}

// 能力图标映射
const CAPABILITY_ICONS: Record<CapabilityType, React.ReactNode> = {
  [CapabilityType.MEMO]: "🦜",
  [CapabilityType.SCHEDULE]: "⏰",
  [CapabilityType.AMAZING]: "🌟",
  [CapabilityType.CREATIVE]: "💡",
  [CapabilityType.AUTO]: "🤖",
};

// 能力 Lucide 图标映射（用于动画）
// const CAPABILITY_LUCIDE_ICONS: Record<CapabilityType, React.ReactNode> = {
//   [CapabilityType.MEMO]: <Sparkles className="w-4 h-4" />,
//   [CapabilityType.SCHEDULE]: <Calendar className="w-4 h-4" />,
//   [CapabilityType.AMAZING]: <Clock className="w-4 h-4" />,
//   [CapabilityType.CREATIVE]: <Lightbulb className="w-4 h-4" />,
//   [CapabilityType.AUTO]: <Wand2 className="w-4 h-4" />,
// };

// 能力主题色映射
const CAPABILITY_COLORS: Record<CapabilityType, { bg: string; text: string; border: string }> = {
  [CapabilityType.MEMO]: {
    bg: "bg-slate-100 dark:bg-slate-800/50",
    text: "text-slate-700 dark:text-slate-300",
    border: "border-slate-200 dark:border-slate-700",
  },
  [CapabilityType.SCHEDULE]: {
    bg: "bg-cyan-100 dark:bg-cyan-900/30",
    text: "text-cyan-700 dark:text-cyan-300",
    border: "border-cyan-200 dark:border-cyan-700",
  },
  [CapabilityType.AMAZING]: {
    bg: "bg-emerald-100 dark:bg-emerald-900/30",
    text: "text-emerald-700 dark:text-emerald-300",
    border: "border-emerald-200 dark:border-emerald-700",
  },
  [CapabilityType.CREATIVE]: {
    bg: "bg-lime-100 dark:bg-lime-900/30",
    text: "text-lime-700 dark:text-lime-300",
    border: "border-lime-200 dark:border-lime-700",
  },
  [CapabilityType.AUTO]: {
    bg: "bg-indigo-100 dark:bg-indigo-900/30",
    text: "text-indigo-700 dark:text-indigo-300",
    border: "border-indigo-200 dark:border-indigo-700",
  },
};

export function CapabilityIndicator({
  capability,
  status,
  onCapabilityChange,
  className,
  compact = false,
}: CapabilityIndicatorProps) {
  const { t } = useTranslation();
  const colors = CAPABILITY_COLORS[capability];
  const icon = CAPABILITY_ICONS[capability];

  // 获取能力显示名称
  const getCapabilityName = (cap: CapabilityType): string => {
    return t(`ai.capability.${cap.toLowerCase()}.name`) || cap;
  };

  // 获取拟声词
  const getSoundEffect = (): string => {
    switch (status) {
      case "thinking":
        return getCapabilitySound(capability, "thinking");
      case "processing":
        return getCapabilitySound(capability, "searching");
      default:
        return "";
    }
  };

  const soundEffect = getSoundEffect();

  if (compact) {
    return (
      <div
        className={cn(
          "flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium transition-all duration-300",
          colors.bg,
          colors.text,
          colors.border,
          "border",
          status === "thinking" && "animate-pulse",
          className,
        )}
      >
        <span className="text-sm">{icon}</span>
        <span>{getCapabilityName(capability)}</span>
        {soundEffect && <span className="text-[10px] opacity-60">{soundEffect}</span>}
      </div>
    );
  }

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-300",
        colors.bg,
        colors.text,
        colors.border,
        "border",
        status === "thinking" && "animate-pulse",
        status === "processing" && "animate-subtle-bounce",
        className,
      )}
    >
      {/* 能力图标 */}
      <span className="text-base" role="img" aria-label={getCapabilityName(capability)}>
        {icon}
      </span>

      {/* 能力名称 */}
      <span>{getCapabilityName(capability)}</span>

      {/* 状态指示 */}
      {status !== "idle" && (
        <span className="flex items-center gap-1 text-xs opacity-70">
          {status === "thinking" && (
            <>
              <span className="w-1.5 h-1.5 rounded-full bg-current animate-ping" />
              <span>{soundEffect || "思考中..."}</span>
            </>
          )}
          {status === "processing" && (
            <>
              <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" />
              <span>{soundEffect || "处理中..."}</span>
            </>
          )}
        </span>
      )}

      {/* 切换按钮（如果提供） */}
      {onCapabilityChange && (
        <button
          onClick={() => onCapabilityChange(CapabilityType.AUTO)}
          className="ml-1 p-0.5 rounded hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
          aria-label="切换能力"
        >
          <Wand2 className="w-3 h-3 opacity-50" />
        </button>
      )}
    </div>
  );
}

/**
 * 能力面板组件 - 显示所有可用能力
 */
interface CapabilityPanelProps {
  currentCapability: CapabilityType;
  status: CapabilityStatus;
  onCapabilityChange: (capability: CapabilityType) => void;
  className?: string;
}

export function CapabilityPanel({
  currentCapability,
  status,
  onCapabilityChange,
  className,
}: CapabilityPanelProps) {
  const capabilities: Array<{ type: CapabilityType; icon: string; label: string; labelAlt: string }> = [
    { type: CapabilityType.MEMO, icon: "🦜", label: "笔记", labelAlt: "Memo" },
    { type: CapabilityType.SCHEDULE, icon: "⏰", label: "日程", labelAlt: "Schedule" },
    { type: CapabilityType.AMAZING, icon: "🌟", label: "综合", labelAlt: "Amazing" },
    { type: CapabilityType.CREATIVE, icon: "💡", label: "创意", labelAlt: "Creative" },
    { type: CapabilityType.AUTO, icon: "🤖", label: "自动", labelAlt: "Auto" },
  ];

  return (
    <div className={cn("flex flex-wrap gap-2", className)}>
      {capabilities.map((cap) => {
        const isActive = cap.type === currentCapability;
        const colors = CAPABILITY_COLORS[cap.type];

        return (
          <button
            key={cap.type}
            onClick={() => onCapabilityChange(cap.type)}
            className={cn(
              "flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm font-medium transition-all duration-200",
              "border",
              colors.bg,
              colors.text,
              colors.border,
              isActive
                ? "ring-2 ring-offset-1 ring-zinc-900 dark:ring-zinc-100 scale-105 shadow-md"
                : "opacity-60 hover:opacity-100 hover:scale-102",
              status === "thinking" && isActive && "animate-pulse",
            )}
            aria-label={cap.label}
            aria-pressed={isActive}
          >
            <span className="text-base">{cap.icon}</span>
            <span>{cap.label}</span>
            {isActive && status !== "idle" && (
              <span className="w-1 h-1 rounded-full bg-current animate-ping" />
            )}
          </button>
        );
      })}
    </div>
  );
}
