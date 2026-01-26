import { Eraser, MoreHorizontal, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { CapabilityType } from "@/types/capability";

interface ChatHeaderProps {
  isThinking?: boolean;
  onClearContext?: () => void;
  onClearChat?: () => void;
  className?: string;
  currentCapability?: CapabilityType;
  capabilityStatus?: string;
}

/**
 * Chat Header - 统一入口设计
 *
 * UX/UI 设计原则：
 * - 移除能力徽章，不暴露系统内部能力边界
 * - 状态展示改为动作描述（"搜索笔记中..."而非"笔记能力"）
 * - 简洁清晰的视觉层次
 */
const ASSISTANT_ICON = "🦜";

/**
 * 根据当前能力和状态获取动作描述
 */
function getActionDescription(capability: CapabilityType, status: string, t: (key: string) => string): string | null {
  if (status === "idle") return null;

  if (status === "thinking") {
    return t("ai.thinking");
  }

  if (status === "processing") {
    switch (capability) {
      case CapabilityType.MEMO:
        return t("ai.parrot.status.searching-memos");
      case CapabilityType.SCHEDULE:
        return t("ai.parrot.status.querying-schedule");
      case CapabilityType.AMAZING:
        return t("ai.parrot.status.analyzing");
      default:
        return t("ai.processing");
    }
  }

  return null;
}

export function ChatHeader({
  isThinking = false,
  onClearContext,
  onClearChat,
  className,
  currentCapability = CapabilityType.AUTO,
  capabilityStatus = "idle",
}: ChatHeaderProps) {
  const { t } = useTranslation();
  const assistantName = t("ai.assistant-name");
  const actionDescription = getActionDescription(currentCapability, capabilityStatus, t);

  return (
    <header
      className={cn(
        "flex items-center justify-between px-4 h-14 shrink-0",
        "border-b border-zinc-200/80 dark:border-zinc-800/80",
        "bg-white/80 dark:bg-zinc-900/80 backdrop-blur-sm",
        className,
      )}
    >
      {/* Left Section */}
      <div className="flex items-center gap-2.5">
        <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-emerald-500 to-green-600 flex items-center justify-center text-lg shadow-sm">
          {ASSISTANT_ICON}
        </div>
        <div className="flex flex-col">
          <h1 className="font-semibold text-zinc-900 dark:text-zinc-100 text-sm leading-tight">{assistantName}</h1>
          {/* 动作描述 - 替代能力徽章 */}
          {actionDescription ? (
            <span className="text-xs text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" />
              {actionDescription}
            </span>
          ) : (
            <span className="text-xs text-zinc-400 dark:text-zinc-500">{t("ai.ready")}</span>
          )}
        </div>
      </div>

      {/* Right Section */}
      <div className="flex items-center gap-2">
        {isThinking && (
          <div className="hidden sm:flex items-center gap-1.5 text-sm text-zinc-500 dark:text-zinc-400 mr-1">
            <Sparkles className="w-4 h-4 animate-pulse text-amber-500" />
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
                <DropdownMenuItem
                  onClick={onClearChat}
                  className="text-red-600 dark:text-red-400 focus:text-red-600 dark:focus:text-red-400 cursor-pointer"
                >
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
