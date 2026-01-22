import { ChevronLeft, MoreHorizontal, SparklesIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ParrotAgent, PARROT_THEMES } from "@/types/parrot";
import { cn } from "@/lib/utils";

interface ChatHeaderProps {
  parrot: ParrotAgent;
  isThinking?: boolean;
  onBack: () => void;
  className?: string;
}

export function ChatHeader({ parrot, isThinking = false, onBack, className }: ChatHeaderProps) {
  const { t } = useTranslation();
  const theme = PARROT_THEMES[parrot.id] || PARROT_THEMES.DEFAULT;
  const icon = getParrotIcon(parrot.id);

  return (
    <header
      className={cn(
        "flex items-center justify-between px-4 py-3 border-b transition-colors",
        theme.bubbleBorder,
        className
      )}
    >
      {/* Left Section */}
      <div className="flex items-center gap-3">
        <button
          onClick={onBack}
          className="p-2 -ml-2 text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100 transition-colors"
          aria-label="Go back to hub"
        >
          <ChevronLeft className="w-5 h-5" />
        </button>

        <div className={cn("flex items-center gap-3 px-3 py-2 rounded-xl", theme.iconBg)}>
          <span className="text-xl">{icon}</span>
          <div>
            <h2 className="font-semibold text-zinc-900 dark:text-zinc-100 text-sm">{parrot.displayName}</h2>
            <p className={cn("text-xs font-medium", theme.iconText)}>{parrot.description}</p>
          </div>
        </div>
      </div>

      {/* Right Section */}
      <div className="flex items-center gap-3">
        {isThinking && (
          <div className="flex items-center gap-2 text-sm text-zinc-500 dark:text-zinc-400">
            <SparklesIcon className="w-4 h-4 animate-pulse" />
            <span>{t("ai.thinking")}</span>
          </div>
        )}

        {/* More Options */}
        <button
          className="p-2 rounded-md hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
          aria-label="More options"
        >
          <MoreHorizontal className="w-5 h-5 text-zinc-500" />
        </button>
      </div>
    </header>
  );
}

function getParrotIcon(parrotId: string): string {
  const icons: Record<string, string> = {
    DEFAULT: "🤖",
    MEMO: "🦜",
    SCHEDULE: "📅",
    AMAZING: "⭐",
    CREATIVE: "💡",
  };
  return icons[parrotId] || "🤖";
}
