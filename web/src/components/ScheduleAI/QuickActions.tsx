import { ArrowRight, Zap } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import type { QuickActionsProps, UIQuickActionData } from "./types";

export function QuickActions({ data, onAction, onDismiss }: QuickActionsProps) {
  const t = useTranslate();

  // Get translations (keys must exist in i18n files)
  const titleText = t("quick.actions.title") || data.title;

  return (
    <div
      className={cn(
        "rounded-xl border p-4 transition-all duration-200",
        "animate-in fade-in slide-in-from-top-2",
        "bg-muted/50 border-border",
      )}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
            <Zap className="w-4 h-4 text-primary" />
          </div>
          <h4 className="font-semibold text-foreground">{titleText}</h4>
        </div>
        {onDismiss && (
          <button
            type="button"
            onClick={onDismiss}
            aria-label="Dismiss"
            className="text-muted-foreground hover:text-foreground transition-colors"
          >
            ×
          </button>
        )}
      </div>

      {data.description && (
        <p className="text-sm text-muted-foreground mb-3">{data.description}</p>
      )}

      <div className="flex gap-2 overflow-x-auto pb-2 -mx-1 px-1 scrollbar-thin scrollbar-thumb-muted-foreground/20 scrollbar-track-transparent">
        {data.actions.map((action) => (
          <QuickActionButton
            key={action.id}
            action={action}
            onClick={() => onAction(action)}
          />
        ))}
      </div>
    </div>
  );
}

interface QuickActionButtonProps {
  action: UIQuickActionData;
  onClick: () => void;
}

function QuickActionButton({ action, onClick }: QuickActionButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex-shrink-0 min-w-[7rem] flex flex-col items-start gap-2 p-3 rounded-lg",
        "bg-background hover:bg-primary/10",
        "border border-border hover:border-primary/30",
        "transition-all duration-200",
        "text-left group",
      )}
    >
      <div className="flex items-center gap-2 w-full">
        {action.icon && (
          <span className="text-lg" role="img" aria-label={action.label}>
            {action.icon}
          </span>
        )}
        <span className="font-medium text-sm text-foreground flex-1">{action.label}</span>
        <ArrowRight className="w-4 h-4 text-muted-foreground group-hover:text-primary transition-colors" />
      </div>
      {action.description && (
        <p className="text-xs text-muted-foreground line-clamp-2">{action.description}</p>
      )}
    </button>
  );
}
