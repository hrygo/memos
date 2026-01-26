import { Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { ConversationSummary } from "@/types/aichat";
import { PARROT_AGENTS, PARROT_ICONS, PARROT_THEMES } from "@/types/parrot";

interface ConversationItemProps {
  conversation: ConversationSummary;
  isActive: boolean;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  className?: string;
  isLoaded?: boolean; // Whether this conversation has been loaded with messages
}

export function ConversationItem({ conversation, isActive, onSelect, onDelete, className, isLoaded = false }: ConversationItemProps) {
  const { t } = useTranslation();
  const parrot = PARROT_AGENTS[conversation.parrotId];
  const parrotIcon = PARROT_ICONS[conversation.parrotId] || parrot?.icon || "🤖";
  const parrotTheme = PARROT_THEMES[conversation.parrotId] || PARROT_THEMES.AMAZING;

  // Display message count: show "..." if not loaded yet, 0 if truly empty
  const displayMessageCount = isLoaded ? conversation.messageCount : "...";

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
        className="w-full text-left px-3 py-2.5 pr-12"
        aria-label={`Select conversation: ${conversation.title}`}
      >
        <div className="flex items-start gap-3">
          {/* Parrot Icon */}
          <div
            className={cn(
              "w-10 h-10 rounded-xl flex items-center justify-center text-lg shrink-0",
              !parrotIcon.startsWith("/") && parrotTheme.iconBg,
            )}
          >
            {parrotIcon.startsWith("/") ? (
              <img src={parrotIcon} alt={parrot?.displayName || ""} className="w-6 h-6 object-contain" />
            ) : (
              parrotIcon
            )}
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <h3 className="font-medium text-sm text-zinc-900 dark:text-zinc-100 truncate">{conversation.title}</h3>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
              {displayMessageCount === "..."
                ? t("ai.aichat.sidebar.message-count", { count: 0 })
                : t("ai.aichat.sidebar.message-count", { count: displayMessageCount })}{" "}
              · {formatTime(conversation.updatedAt, t)}
            </p>
          </div>
        </div>
      </button>

      {/* Delete Button - Show on hover */}
      <div className="absolute right-2 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity">
        <DeleteButton conversationId={conversation.id} onDelete={onDelete} />
      </div>
    </div>
  );
}

interface DeleteButtonProps {
  conversationId: string;
  onDelete: (id: string) => void;
}

function DeleteButton({ conversationId, onDelete }: DeleteButtonProps) {
  const { t } = useTranslation();

  return (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onDelete(conversationId);
      }}
      className={cn(
        "flex items-center justify-center",
        "w-8 h-8 rounded-lg",
        "text-zinc-400 dark:text-zinc-500",
        "hover:text-red-500 dark:hover:text-red-400",
        "hover:bg-red-50 dark:hover:bg-red-950/50",
        "transition-all duration-200",
      )}
      aria-label={t("common.delete")}
      title={t("common.delete")}
    >
      <Trash2 className="w-4 h-4" />
    </button>
  );
}

function formatTime(timestamp: number, t: (key: string, options?: Record<string, unknown>) => string): string {
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
