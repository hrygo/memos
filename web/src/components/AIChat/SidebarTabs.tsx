import { History, FileText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { SidebarTab } from "@/types/aichat";

interface SidebarTabsProps {
  activeTab: SidebarTab;
  onTabChange: (tab: SidebarTab) => void;
  className?: string;
}

export function SidebarTabs({ activeTab, onTabChange, className }: SidebarTabsProps) {
  const { t } = useTranslation();

  return (
    <div className={cn("flex items-center gap-1 p-1 bg-zinc-100 dark:bg-zinc-800 rounded-lg", className)}>
      <TabButton
        active={activeTab === "history"}
        onClick={() => onTabChange("history")}
        icon={<History className="w-4 h-4" />}
        label={t("ai.aichat.sidebar.history")}
      />
      <TabButton
        active={activeTab === "memos"}
        onClick={() => onTabChange("memos")}
        icon={<FileText className="w-4 h-4" />}
        label={t("ai.aichat.sidebar.memos")}
      />
    </div>
  );
}

interface TabButtonProps {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}

function TabButton({ active, onClick, icon, label }: TabButtonProps) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex items-center gap-2 px-3 py-2 rounded-md text-sm font-medium transition-all",
        active
          ? "bg-white dark:bg-zinc-700 text-zinc-900 dark:text-zinc-100 shadow-sm"
          : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300"
      )}
    >
      {icon}
      <span>{label}</span>
    </button>
  );
}
