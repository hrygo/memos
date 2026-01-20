import { SparklesIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import SuggestedPrompts from "./SuggestedPrompts";

interface EmptyStateProps {
  onSuggestedPrompt: (query: string) => void;
}

const EmptyState = ({ onSuggestedPrompt }: EmptyStateProps) => {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center justify-center h-full py-12 px-4 animate-in fade-in-0 duration-500">
      <div className="relative mb-6">
        <div className="absolute inset-0 bg-primary/20 blur-3xl rounded-full" />
        <div className="relative w-20 h-20 rounded-2xl bg-gradient-to-br from-primary to-blue-600 flex items-center justify-center shadow-lg">
          <SparklesIcon className="w-10 h-10 text-white" />
        </div>
      </div>

      <h2 className="text-xl font-semibold mb-2 text-center">{t("ai.welcome-title")}</h2>
      <p className="text-muted-foreground text-center max-w-md mb-8">{t("ai.welcome-description")}</p>

      <SuggestedPrompts onSelect={onSuggestedPrompt} />

      <div className="flex items-center gap-4 mt-8 text-xs text-muted-foreground">
        <span className="flex items-center gap-1">
          <kbd className="px-1.5 py-0.5 rounded bg-muted border text-[10px]">Enter</kbd>
          {t("ai.send-shortcut")}
        </span>
        <span className="flex items-center gap-1">
          <kbd className="px-1.5 py-0.5 rounded bg-muted border text-[10px]">Shift + Enter</kbd>
          {t("ai.newline-shortcut")}
        </span>
      </div>
    </div>
  );
};

export default EmptyState;
