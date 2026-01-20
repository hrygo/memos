import { AlertCircleIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

interface ErrorMessageProps {
  error: string;
  onRetry: () => void;
}

const ErrorMessage = ({ error, onRetry }: ErrorMessageProps) => {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-3 p-3 rounded-lg bg-destructive/10 border border-destructive/20">
      <AlertCircleIcon className="w-5 h-5 text-destructive shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-sm text-destructive">{error}</p>
      </div>
      <Button size="sm" variant="outline" onClick={onRetry} className="shrink-0">
        {t("ai.retry")}
      </Button>
    </div>
  );
};

export default ErrorMessage;
