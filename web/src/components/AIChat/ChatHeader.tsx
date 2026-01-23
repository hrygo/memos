import { ChevronLeft, Eraser, MoreHorizontal, SparklesIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import type { ParrotAgentI18n } from "@/hooks/useParrots";
import { cn } from "@/lib/utils";
import { PARROT_ICONS, PARROT_THEMES } from "@/types/parrot";

interface ChatHeaderProps {
  parrot: ParrotAgentI18n;
  isThinking?: boolean;
  onBack: () => void;
  onClearContext?: () => void;
  onClearChat?: () => void;
  className?: string;
}

export function ChatHeader({ parrot, isThinking = false, onBack, onClearContext, onClearChat, className }: ChatHeaderProps) {
  const { t } = useTranslation();
  const theme = PARROT_THEMES[parrot.id] || PARROT_THEMES.DEFAULT;
  const icon = getParrotIcon(parrot.id);

  return (
    <header className={cn("flex items-center justify-between px-4 py-3 border-b transition-colors", theme.bubbleBorder, className)}>
      {/* Left Section */}
      <div className="flex items-center gap-3">
        <button
          onClick={onBack}
          className="p-2 -ml-2 text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100 transition-all active:scale-90 rounded-full hover:bg-zinc-100 dark:hover:bg-zinc-800 cursor-pointer"
          aria-label="Go back to hub"
        >
          <ChevronLeft className="w-5 h-5" />
        </button>

        <div className={cn("flex items-center gap-3 px-3 py-2 rounded-xl", theme.iconBg)}>
          {icon.startsWith("/") ? (
            <img src={icon} alt={parrot.displayName} className="w-6 h-6 object-contain" />
          ) : (
            <span className="text-xl">{icon}</span>
          )}
          <div>
            <div className="flex items-baseline gap-2">
              <h2 className="font-semibold text-zinc-900 dark:text-zinc-100 text-sm">{parrot.displayName}</h2>
              <span className="text-xs text-zinc-400 dark:text-zinc-500">{parrot.displayNameAlt}</span>
            </div>
            <p className={cn("text-xs font-medium", theme.iconText)}>{parrot.description}</p>
          </div>
        </div>
      </div>

      {/* Right Section */}
      <div className="flex items-center gap-3">
        {isThinking && (
          <div className="flex items-center gap-2 text-sm text-zinc-500 dark:text-zinc-400">
            <SparklesIcon className="w-4 h-4 animate-pulse text-yellow-500" />
            <span>{t("ai.thinking")}</span>
          </div>
        )}

        {/* More Options */}
        {(onClearContext || onClearChat) && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="p-2 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-all active:scale-90 cursor-pointer" aria-label="More options">
                <MoreHorizontal className="w-5 h-5 text-zinc-500" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              {onClearContext && (
                <DropdownMenuItem onClick={onClearContext} className="cursor-pointer">
                  <Eraser className="w-4 h-4 mr-2" />
                  <div>
                    <div className="font-medium">{t("ai.clear-context")}</div>
                    <div className="text-xs text-muted-foreground">{t("ai.clear-context-desc")}</div>
                  </div>
                </DropdownMenuItem>
              )}
              {onClearChat && (
                <DropdownMenuItem onClick={onClearChat} className="text-destructive focus:text-destructive cursor-pointer">
                  <Eraser className="w-4 h-4 mr-2" />
                  <div>
                    <div className="font-medium">{t("ai.clear-chat")}</div>
                    <div className="text-xs text-muted-foreground">{t("ai.clear-chat-desc")}</div>
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

function getParrotIcon(parrotId: string): string {
  return PARROT_ICONS[parrotId] || "🤖";
}
