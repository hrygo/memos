import { Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { CapabilityStatus, CapabilityType } from "@/types/capability";

interface ChatHeaderProps {
  isThinking?: boolean;
  className?: string;
  currentCapability?: CapabilityType;
  capabilityStatus?: CapabilityStatus;
}

/**
 * Chat Header - 简洁状态显示
 *
 * UX/UI 设计原则：
 * - 仅展示助手信息和状态
 * - 工具按钮移至输入框工具栏
 * - 简洁清晰的视觉层次
 */


/**
 * 根据当前能力和状态获取动作描述
 */
function getActionDescription(capability: CapabilityType, status: CapabilityStatus, t: (key: string) => string): string | null {
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
        "border-b border-border/80",
        "bg-background/80 backdrop-blur-sm",
        className,
      )}
    >
      {/* Left Section */}
      <div className="flex items-center gap-2.5">
        <div className="w-9 h-9 flex items-center justify-center">
          <img src="/logo.webp" alt={assistantName} className="h-9 w-auto object-contain" />
        </div>
        <div className="flex flex-col">
          <h1 className="font-semibold text-foreground text-sm leading-tight">{assistantName}</h1>
          {/* 动作描述 - 替代能力徽章 */}
          {actionDescription ? (
            <span className="text-xs text-primary flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" />
              {actionDescription}
            </span>
          ) : (
            <span className="text-xs text-muted-foreground">{t("ai.ready")}</span>
          )}
        </div>
      </div>

      {/* Right Section - Status indicator */}
      {isThinking && (
        <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <Sparkles className="w-4 h-4 animate-pulse text-primary" />
        </div>
      )}
    </header>
  );
}
