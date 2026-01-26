import { Wand2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { CapabilityStatus, CapabilityType } from "@/types/capability";

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
  [CapabilityType.AUTO]: "🤖",
};

// 能力主题色映射 - 简化
const CAPABILITY_COLORS: Record<CapabilityType, { bg: string; text: string; border: string }> = {
  [CapabilityType.MEMO]: {
    bg: "bg-slate-50 dark:bg-slate-900/30",
    text: "text-slate-700 dark:text-slate-300",
    border: "border-slate-200 dark:border-slate-700",
  },
  [CapabilityType.SCHEDULE]: {
    bg: "bg-cyan-50 dark:bg-cyan-900/20",
    text: "text-cyan-700 dark:text-cyan-300",
    border: "border-cyan-200 dark:border-cyan-800",
  },
  [CapabilityType.AMAZING]: {
    bg: "bg-emerald-50 dark:bg-emerald-900/20",
    text: "text-emerald-700 dark:text-emerald-300",
    border: "border-emerald-200 dark:border-emerald-800",
  },
  [CapabilityType.AUTO]: {
    bg: "bg-indigo-50 dark:bg-indigo-900/20",
    text: "text-indigo-700 dark:text-indigo-300",
    border: "border-indigo-200 dark:border-indigo-800",
  },
};

/**
 * CapabilityIndicator - 精简能力指示器
 *
 * UX/UI 改进：
 * - 简化状态显示
 * - 统一颜色主题
 * - 优化动画效果
 */
export function CapabilityIndicator({ capability, status, onCapabilityChange, className, compact = false }: CapabilityIndicatorProps) {
  const { t } = useTranslation();
  const colors = CAPABILITY_COLORS[capability];
  const icon = CAPABILITY_ICONS[capability];

  // 获取能力显示名称
  const getCapabilityName = (cap: CapabilityType): string => {
    return t(`ai.capability.${cap.toLowerCase()}.name`) || cap;
  };

  if (compact) {
    return (
      <div
        className={cn(
          "inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium",
          colors.bg,
          colors.text,
          colors.border,
          "border transition-all duration-200",
          status === "thinking" && "animate-pulse",
          className,
        )}
      >
        <span className="text-sm">{icon}</span>
        <span>{getCapabilityName(capability)}</span>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "inline-flex items-center gap-2 px-2.5 py-1 rounded-lg text-sm font-medium",
        colors.bg,
        colors.text,
        colors.border,
        "border transition-all duration-200",
        status === "thinking" && "animate-pulse",
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
        <span className="flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-current animate-ping" />
        </span>
      )}

      {/* 切换按钮（如果提供） */}
      {onCapabilityChange && (
        <button
          onClick={() => onCapabilityChange(CapabilityType.AUTO)}
          className="ml-0.5 p-0.5 rounded hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
          aria-label="切换能力"
        >
          <Wand2 className="w-3 h-3 opacity-40" />
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

export function CapabilityPanel({ currentCapability, status, onCapabilityChange, className }: CapabilityPanelProps) {
  const capabilities: Array<{ type: CapabilityType; icon: string; label: string }> = [
    { type: CapabilityType.MEMO, icon: "🦜", label: "笔记" },
    { type: CapabilityType.SCHEDULE, icon: "⏰", label: "日程" },
    { type: CapabilityType.AMAZING, icon: "🌟", label: "综合" },
    { type: CapabilityType.AUTO, icon: "🤖", label: "自动" },
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
              colors.bg,
              colors.text,
              colors.border,
              "border",
              isActive ? "ring-1 ring-zinc-400 dark:ring-zinc-600 scale-105 shadow-sm" : "opacity-60 hover:opacity-100 hover:scale-102",
              status === "thinking" && isActive && "animate-pulse",
            )}
            aria-label={cap.label}
            aria-pressed={isActive}
          >
            <span className="text-base">{cap.icon}</span>
            <span>{cap.label}</span>
          </button>
        );
      })}
    </div>
  );
}
