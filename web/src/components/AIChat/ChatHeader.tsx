import { ChevronLeft, Eraser, MoreHorizontal, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { CapabilityType } from "@/types/capability";

interface ChatHeaderProps {
  isThinking?: boolean;
  onBack: () => void;
  onClearContext?: () => void;
  onClearChat?: () => void;
  className?: string;
  currentCapability?: CapabilityType;
  capabilityStatus?: string;
}

/**
 * Chat Header - 精简统一设计
 *
 * UX/UI 改进：
 * - 移除冗余的助手信息显示（已在侧边栏显示）
 * - 简化能力徽章样式
 * - 优化间距和视觉层次
 * - 统一与侧边栏的设计语言
 */
// 统一的助手形象配置
const ASSISTANT_ICON = "🦜";

// 能力信息配置 - 简化样式
const CAPABILITY_INFO: Record<CapabilityType, { icon: string; name: string; bg: string; text: string; border: string }> = {
  [CapabilityType.AUTO]: {
    icon: "🤖",
    name: "自动",
    bg: "bg-indigo-50 dark:bg-indigo-900/20",
    text: "text-indigo-700 dark:text-indigo-300",
    border: "border-indigo-200 dark:border-indigo-800",
  },
  [CapabilityType.MEMO]: {
    icon: "🦜",
    name: "笔记",
    bg: "bg-slate-100 dark:bg-slate-800/50",
    text: "text-slate-700 dark:text-slate-300",
    border: "border-slate-200 dark:border-slate-700",
  },
  [CapabilityType.SCHEDULE]: {
    icon: "⏰",
    name: "日程",
    bg: "bg-cyan-50 dark:bg-cyan-900/20",
    text: "text-cyan-700 dark:text-cyan-300",
    border: "border-cyan-200 dark:border-cyan-800",
  },
  [CapabilityType.AMAZING]: {
    icon: "🌟",
    name: "综合",
    bg: "bg-emerald-50 dark:bg-emerald-900/20",
    text: "text-emerald-700 dark:text-emerald-300",
    border: "border-emerald-200 dark:border-emerald-800",
  },
  [CapabilityType.CREATIVE]: {
    icon: "💡",
    name: "创意",
    bg: "bg-amber-50 dark:bg-amber-900/20",
    text: "text-amber-700 dark:text-amber-300",
    border: "border-amber-200 dark:border-amber-800",
  },
};

export function ChatHeader({
  isThinking = false,
  onBack,
  onClearContext,
  onClearChat,
  className,
  currentCapability = CapabilityType.AUTO,
  capabilityStatus = "idle",
}: ChatHeaderProps) {
  const { t } = useTranslation();
  const capInfo = CAPABILITY_INFO[currentCapability];
  const assistantName = t("ai.assistant-name") || "AI 助手";

  return (
    <header className={cn(
      "flex items-center justify-between px-4 h-14 shrink-0",
      "border-b border-zinc-200/80 dark:border-zinc-800/80",
      "bg-white/80 dark:bg-zinc-900/80 backdrop-blur-sm",
      className
    )}>
      {/* Left Section */}
      <div className="flex items-center gap-3">
        <button
          onClick={onBack}
          className="p-2 -ml-2 text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100 transition-all active:scale-95 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-800"
          aria-label="Go back"
        >
          <ChevronLeft className="w-5 h-5" />
        </button>

        {/* 简化的助手标题 */}
        <div className="flex items-center gap-2.5">
          <span className="text-xl">{ASSISTANT_ICON}</span>
          <h1 className="font-semibold text-zinc-900 dark:text-zinc-100">{assistantName}</h1>

          {/* 能力徽章 - 精简样式 */}
          <span className={cn(
            "flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium border",
            capInfo.bg, capInfo.text, capInfo.border
          )}>
            <span>{capInfo.icon}</span>
            <span>{capInfo.name}</span>
            {capabilityStatus === "thinking" && (
              <span className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" />
            )}
          </span>
        </div>
      </div>

      {/* Right Section */}
      <div className="flex items-center gap-2">
        {isThinking && (
          <div className="hidden sm:flex items-center gap-1.5 text-sm text-zinc-500 dark:text-zinc-400 mr-1">
            <Sparkles className="w-4 h-4 animate-pulse text-amber-500" />
            <span>{t("ai.thinking")}</span>
          </div>
        )}

        {/* More Options */}
        {(onClearContext || onClearChat) && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                className="p-2 rounded-lg text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-all active:scale-95"
                aria-label="More options"
              >
                <MoreHorizontal className="w-5 h-5" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-52">
              {onClearContext && (
                <DropdownMenuItem onClick={onClearContext} className="cursor-pointer">
                  <Eraser className="w-4 h-4 mr-2 text-zinc-500" />
                  <div>
                    <div className="font-medium">{t("ai.clear-context")}</div>
                    <div className="text-xs text-zinc-500">{t("ai.clear-context-desc")}</div>
                  </div>
                </DropdownMenuItem>
              )}
              {onClearChat && (
                <DropdownMenuItem onClick={onClearChat} className="text-red-600 dark:text-red-400 focus:text-red-600 dark:focus:text-red-400 cursor-pointer">
                  <Eraser className="w-4 h-4 mr-2" />
                  <div>
                    <div className="font-medium">{t("ai.clear-chat")}</div>
                    <div className="text-xs text-zinc-500">{t("ai.clear-chat-desc")}</div>
                  </div>
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </header>
  );
}
