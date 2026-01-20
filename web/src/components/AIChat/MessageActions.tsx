import { CopyIcon, RefreshCwIcon, TrashIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";

interface MessageActionsProps {
  onCopy: () => void;
  onRegenerate: () => void;
  onDelete: () => void;
}

const MessageActions = ({ onCopy, onRegenerate, onDelete }: MessageActionsProps) => {
  const { t } = useTranslation();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="h-6 w-6 opacity-0 group-hover:opacity-50 hover:opacity-100 transition-opacity">
          <svg className="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="1" />
            <circle cx="12" cy="5" r="1" />
            <circle cx="12" cy="19" r="1" />
          </svg>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={4}>
        <DropdownMenuItem onClick={onCopy}>
          <CopyIcon className="w-4 h-4 mr-2" />
          {t("common.copy")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onRegenerate}>
          <RefreshCwIcon className="w-4 h-4 mr-2" />
          {t("ai.regenerate")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onDelete} className="text-destructive focus:text-destructive">
          <TrashIcon className="w-4 h-4 mr-2" />
          {t("common.delete")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

export default MessageActions;
