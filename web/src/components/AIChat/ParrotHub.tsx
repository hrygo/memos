import { MessageSquarePlus, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { CapabilityStatus, CapabilityType } from "@/types/capability";
import { PARROT_THEMES, ParrotAgentType } from "@/types/parrot";

/**
 * 能力卡片配置
 */
interface CapabilityCard {
  id: CapabilityType;
  parrotId: ParrotAgentType;
  icon: string;
  iconAlt: string;
  nameKey: string;
  nameAltKey: string;
  descriptionKey: string;
  theme: (typeof PARROT_THEMES)[keyof typeof PARROT_THEMES];
  nameAlt: string;
}

const CAPABILITY_CARDS: CapabilityCard[] = [
  {
    id: CapabilityType.MEMO,
    parrotId: ParrotAgentType.MEMO,
    icon: "🦜",
    iconAlt: "/images/parrots/icons/memo_icon.webp",
    nameKey: "ai.capability.memo.name",
    nameAltKey: "ai.capability.memo.nameAlt",
    descriptionKey: "ai.capability.memo.description",
    theme: PARROT_THEMES.MEMO,
    nameAlt: "Memo",
  },
  {
    id: CapabilityType.SCHEDULE,
    parrotId: ParrotAgentType.SCHEDULE,
    icon: "⏰",
    iconAlt: "/images/parrots/icons/schedule_icon.webp",
    nameKey: "ai.capability.schedule.name",
    nameAltKey: "ai.capability.schedule.nameAlt",
    descriptionKey: "ai.capability.schedule.description",
    theme: PARROT_THEMES.SCHEDULE,
    nameAlt: "Schedule",
  },
  {
    id: CapabilityType.AMAZING,
    parrotId: ParrotAgentType.AMAZING,
    icon: "🌟",
    iconAlt: "/images/parrots/icons/amazing_icon.webp",
    nameKey: "ai.capability.amazing.name",
    nameAltKey: "ai.capability.amazing.nameAlt",
    descriptionKey: "ai.capability.amazing.description",
    theme: PARROT_THEMES.AMAZING,
    nameAlt: "Amazing",
  },
];

interface ParrotHubProps {
  currentCapability?: CapabilityType;
  capabilityStatus?: CapabilityStatus;
  onCapabilitySelect?: (capability: CapabilityType) => void;
  className?: string;
}

/**
 * 能力面板组件 (原 ParrotHub)
 *
 * 设计变化：
 * - 从"选择Agent入口"变为"能力指示器"
 * - 强调当前激活的能力，而非多选入口
 * - 保留鹦鹉形象，但重新定位为"能力卡片"
 */
export function ParrotHub({
  currentCapability = CapabilityType.AUTO,
  capabilityStatus = "idle",
  onCapabilitySelect,
  className,
}: ParrotHubProps) {
  const { t } = useTranslation();

  return (
    <div className={cn("w-full h-full overflow-y-auto bg-sidebar p-4 md:p-8", className)}>
      <div className="max-w-4xl mx-auto">
        {/* 头部标题 - 强调"能力"而非"选择" */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-2 mb-3">
            <Sparkles className="w-5 h-5 text-primary" />
            <h2 className="text-lg md:text-xl font-semibold text-foreground">{t("ai.capability.title") || "我的能力"}</h2>
            <Sparkles className="w-5 h-5 text-primary" />
          </div>
          <p className="text-sm text-muted-foreground">{t("ai.capability.subtitle") || "我可以帮你搜索笔记、管理日程、综合分析"}</p>
        </div>

        {/* 能力卡片网格 */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 md:gap-6">
          {CAPABILITY_CARDS.map((card) => {
            const isActive = card.id === currentCapability;
            const theme = card.theme;
            const icon = card.icon;

            return (
              <button
                key={card.id}
                onClick={() => onCapabilitySelect?.(card.id)}
                className={cn(
                  "flex flex-col text-left p-5 md:p-6 rounded-2xl border-2 transition-all duration-300 group relative overflow-hidden",
                  "bg-card",
                  isActive
                    ? theme.cardBorder + " ring-2 ring-offset-2 ring-foreground shadow-lg scale-[1.02]"
                    : "border-border hover:border-border hover:shadow-md",
                  "focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-foreground",
                )}
              >
                {/* 背景装饰 */}
                <div
                  className={cn(
                    "absolute top-0 right-0 w-32 h-32 rounded-full blur-3xl opacity-0 group-hover:opacity-10 transition-opacity duration-500",
                    theme.accent,
                  )}
                />

                {/* 活跃指示器 */}
                {isActive && (
                  <div className="absolute top-3 right-3 flex items-center gap-1.5 px-2 py-1 rounded-full bg-foreground text-background text-xs font-medium">
                    <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
                    {t("ai.capability.active") || "使用中"}
                  </div>
                )}

                {/* 图标 */}
                <div
                  className={cn(
                    "w-12 h-12 rounded-xl flex items-center justify-center text-2xl md:text-3xl mb-4 transition-transform group-hover:scale-110 duration-300",
                    theme.iconBg,
                  )}
                >
                  {icon}
                </div>

                {/* 名称 */}
                <h3 className={cn("text-base md:text-lg font-bold mb-1 transition-colors", theme.text)}>
                  {t(card.nameKey) || card.nameAlt}
                  <span className="text-xs font-medium text-muted-foreground ml-2">{t(card.nameAltKey)}</span>
                </h3>

                {/* 描述 */}
                <p className="text-sm text-muted-foreground leading-relaxed mb-4 flex-grow">{t(card.descriptionKey)}</p>

                {/* 底部提示 */}
                <div className={cn("flex items-center text-sm font-medium", theme.iconText)}>
                  {isActive ? (
                    <>
                      <span>{t("ai.capability.in-use") || "正在使用"}</span>
                      <Sparkles className="w-4 h-4 ml-1.5 animate-pulse" />
                    </>
                  ) : (
                    <>
                      <span>{t("ai.capability.tap-to-activate") || "点击激活"}</span>
                      <MessageSquarePlus className="w-4 h-4 ml-1.5 transition-transform group-hover:translate-x-1" />
                    </>
                  )}
                </div>

                {/* 处理中状态动画 */}
                {isActive && capabilityStatus === "thinking" && (
                  <div className="absolute inset-0 bg-card/50 flex items-center justify-center">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                      <div className="w-5 h-5 border-2 border-muted-foreground border-t-transparent rounded-full animate-spin" />
                      <span>{t("ai.capability.thinking") || "思考中..."}</span>
                    </div>
                  </div>
                )}
              </button>
            );
          })}
        </div>

        {/* 底部提示 - 强调"自动路由" */}
        <div className="mt-8 p-4 rounded-xl bg-accent border border-border">
          <p className="text-sm text-center text-foreground flex items-center justify-center gap-2">
            <Sparkles className="w-4 h-4" />
            <span>{t("ai.capability.auto-hint") || "💡 提示：你也可以直接开始聊天，我会自动理解你的意图并调用相应能力"}</span>
          </p>
        </div>
      </div>
    </div>
  );
}

/**
 * 导出为 CapabilityPanel 别名（语义更清晰）
 */
export const CapabilityPanel = ParrotHub;

export default ParrotHub;
