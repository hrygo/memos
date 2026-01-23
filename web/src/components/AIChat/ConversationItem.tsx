import { Scissors } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { ConversationSummary } from "@/types/aichat";
import { PARROT_AGENTS, PARROT_ICONS, PARROT_THEMES } from "@/types/parrot";

interface ConversationItemProps {
  conversation: ConversationSummary;
  isActive: boolean;
  onSelect: (id: string) => void;
  onResetContext: (id: string) => void;
  className?: string;
}

export function ConversationItem({ conversation, isActive, onSelect, onResetContext, className }: ConversationItemProps) {
  const { t } = useTranslation();
  const parrot = PARROT_AGENTS[conversation.parrotId];
  const parrotIcon = PARROT_ICONS[conversation.parrotId] || parrot?.icon || "🤖";
  const parrotTheme = PARROT_THEMES[conversation.parrotId] || PARROT_THEMES.DEFAULT;

  return (
    <div
      className={cn(
        "group relative rounded-lg transition-all",
        isActive ? "bg-zinc-100 dark:bg-zinc-800" : "hover:bg-zinc-50 dark:hover:bg-zinc-800/50",
        className,
      )}
    >
      <button
        onClick={() => onSelect(conversation.id)}
        className="w-full text-left p-3 pr-12"
        aria-label={`Select conversation: ${conversation.title}`}
      >
        <div className="flex items-start gap-3">
          {/* Parrot Icon */}
          <div className={cn("w-9 h-9 rounded-lg flex items-center justify-center text-lg shrink-0", !parrotIcon.startsWith("/") && parrotTheme.iconBg)}>
            {parrotIcon.startsWith("/") ? (
              <img src={parrotIcon} alt={parrot?.displayName || ""} className="w-8 h-8 object-contain" />
            ) : (
              parrotIcon
            )}
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <h3 className="font-medium text-sm text-zinc-900 dark:text-zinc-100 truncate">{conversation.title}</h3>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
              {t("ai.aichat.sidebar.message-count", { count: conversation.messageCount })} · {formatTime(conversation.updatedAt, t)}
            </p>
          </div>
        </div>
      </button>

      {/* Clear Context Button - Compact icon-only, expands on hover */}
      <div className="absolute right-2 top-1/2 -translate-y-1/2">
        <ClearContextButton
          conversationId={conversation.id}
          onResetContext={onResetContext}
        />
      </div>
    </div>
  );
}

interface ClearContextButtonProps {
  conversationId: string;
  onResetContext: (id: string) => void;
}

function ClearContextButton({ conversationId, onResetContext }: ClearContextButtonProps) {
  const { t } = useTranslation();

  return (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onResetContext(conversationId);
      }}
      className={cn(
        "flex items-center justify-center",
        "w-8 h-8 rounded-lg",
        "text-zinc-400 dark:text-zinc-500",
        "hover:text-zinc-600 dark:hover:text-zinc-300",
        "hover:bg-zinc-200 dark:hover:bg-zinc-700",
        "transition-all duration-200",
        "group/btn",
        // Expand on hover
        "hover:w-auto hover:px-2 hover:gap-1.5",
        "overflow-hidden",
      )}
      aria-label={t("ai.clear-context")}
      title={t("ai.clear-context-shortcut")}
    >
      <Scissors className="w-3.5 h-3.5 shrink-0 rotate-[-45deg]" />
      <span className="hidden text-xs font-medium whitespace-nowrap group-hover/btn:inline">
        {t("ai.clear")}
      </span>
    </button>
  );
}

function formatTime(timestamp: number, t: (key: string, options?: any) => string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return t("ai.aichat.sidebar.time-just-now");
  if (diffMins < 60) return t("ai.aichat.sidebar.time-minutes-ago", { count: diffMins });
  if (diffHours < 24) return t("ai.aichat.sidebar.time-hours-ago", { count: diffHours });
  if (diffDays < 7) return t("ai.aichat.sidebar.time-days-ago", { count: diffDays });

  return date.toLocaleDateString();
}
