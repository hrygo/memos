import { BotIcon } from "lucide-react";
import { useTranslation } from "react-i18next";

const ThinkingIndicator = () => {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-1.5 px-3 py-2 bg-muted/50 rounded-full">
      <BotIcon className="w-4 h-4 text-primary animate-pulse" />
      <div className="flex gap-1">
        <span className="w-1.5 h-1.5 bg-primary rounded-full animate-bounce [animation-delay:-0.3s]" />
        <span className="w-1.5 h-1.5 bg-primary rounded-full animate-bounce [animation-delay:-0.15s]" />
        <span className="w-1.5 h-1.5 bg-primary rounded-full animate-bounce" />
      </div>
      <span className="text-xs text-muted-foreground ml-1">{t("ai.thinking")}</span>
    </div>
  );
};

export default ThinkingIndicator;
