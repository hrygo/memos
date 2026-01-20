import { LightbulbIcon, MessageSquareIcon, BookOpenIcon, SparklesIcon } from "lucide-react";
import { useTranslation } from "react-i18next";

interface SuggestedPromptsProps {
  onSelect: (query: string) => void;
}

const suggestedPrompts = [
  { icon: LightbulbIcon, key: "prompt-summarize" },
  { icon: SparklesIcon, key: "prompt-ideas" },
  { icon: BookOpenIcon, key: "prompt-explain" },
  { icon: MessageSquareIcon, key: "prompt-discuss" },
];

const SuggestedPrompts = ({ onSelect }: SuggestedPromptsProps) => {
  const { t } = useTranslation();

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 w-full max-w-2xl">
      {suggestedPrompts.map((prompt) => (
        <button
          key={prompt.key}
          onClick={() => onSelect(t(`ai.${prompt.key}`))}
          className="flex items-center gap-3 p-3 text-left rounded-lg border border-border/50 bg-muted/30 hover:bg-muted/50 transition-all group"
        >
          <prompt.icon className="w-5 h-5 text-primary/70 group-hover:text-primary transition-colors shrink-0" />
          <span className="text-sm text-foreground/80">{t(`ai.${prompt.key}`)}</span>
        </button>
      ))}
    </div>
  );
};

export default SuggestedPrompts;
