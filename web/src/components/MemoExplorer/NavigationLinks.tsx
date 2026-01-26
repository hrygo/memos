import { CalendarDays } from "lucide-react";
import { useLocation, useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import { Routes } from "@/router";
import { type Translations, useTranslate } from "@/utils/i18n";

const NavigationLinks = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const t = useTranslate();

  const links = [
    {
      id: "schedule",
      icon: CalendarDays,
      label: "schedule.title" as Translations,
      path: Routes.SCHEDULE,
      color: "text-blue-600",
      bgColor: "bg-blue-50 dark:bg-blue-900/20",
    },
  ];

  return (
    <div className="w-full flex flex-col justify-start items-start mt-4 px-1">
      <div className="w-full flex flex-row justify-between items-center mb-2 px-1">
        <span className="text-sm leading-6 text-muted-foreground select-none">{t("common.add" as Translations) || "Navigation"}</span>
      </div>
      <div className="w-full flex flex-col gap-1">
        {links.map((link) => {
          const Icon = link.icon;
          const isActive = location.pathname === link.path;
          return (
            <button
              key={link.id}
              onClick={() => navigate(link.path)}
              className={cn(
                "w-full text-sm rounded-md leading-7 px-2 flex flex-row justify-start items-center",
                "select-none gap-2 transition-colors cursor-pointer",
                "text-muted-foreground hover:text-foreground",
                isActive && "bg-muted font-medium text-foreground",
                !isActive && link.bgColor,
              )}
            >
              <Icon className={cn("w-4 h-auto shrink-0", link.color)} />
              <span className="truncate">{t(link.label)}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
};

export default NavigationLinks;
