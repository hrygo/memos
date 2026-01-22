import { Coffee, Dumbbell, Phone, UtensilsCrossed, Video } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import type { ScheduleTemplate } from "./types";

/**
 * Hook to detect clicks outside a component
 */
function useClickOutside(ref: React.RefObject<HTMLElement>, callback: () => void, enabled: boolean = true) {
  useEffect(() => {
    if (!enabled) return;

    const handleClick = (event: MouseEvent | TouchEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        callback();
      }
    };

    document.addEventListener("mousedown", handleClick);
    document.addEventListener("touchstart", handleClick);

    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("touchstart", handleClick);
    };
  }, [ref, callback, enabled]);
}

interface QuickTemplatesProps {
  /** Called when a template is selected */
  onSelect?: (template: ScheduleTemplate) => void;
  /** Optional className */
  className?: string;
  /** List of templates to display (defaults to common templates) */
  templates?: ScheduleTemplate[];
}

export const DEFAULT_TEMPLATES: ScheduleTemplate[] = [
  {
    id: "meeting-15",
    title: "15分钟会议",
    i18nKey: "schedule.quick-input.template-meeting-15",
    promptI18nKey: "schedule.quick-input.prompt-meeting-15",
    icon: "users",
    duration: 15,
    defaultTitle: "快速会议",
    color: "bg-blue-500",
  },
  {
    id: "meeting-30",
    title: "30分钟会议",
    i18nKey: "schedule.quick-input.template-meeting-30",
    promptI18nKey: "schedule.quick-input.prompt-meeting-30",
    icon: "users",
    duration: 30,
    defaultTitle: "会议",
    color: "bg-blue-500",
  },
  {
    id: "call",
    title: "电话",
    i18nKey: "schedule.quick-input.template-call",
    promptI18nKey: "schedule.quick-input.prompt-call",
    icon: "phone",
    duration: 30,
    defaultTitle: "电话会议",
    color: "bg-green-500",
  },
  {
    id: "video-call",
    title: "视频会议",
    i18nKey: "schedule.quick-input.template-video",
    promptI18nKey: "schedule.quick-input.prompt-video",
    icon: "video",
    duration: 45,
    defaultTitle: "视频会议",
    color: "bg-purple-500",
  },
  {
    id: "lunch",
    title: "午餐",
    i18nKey: "schedule.quick-input.template-lunch",
    promptI18nKey: "schedule.quick-input.prompt-lunch",
    icon: "lunch",
    duration: 60,
    defaultTitle: "午餐",
    color: "bg-orange-500",
  },
  {
    id: "coffee",
    title: "咖啡",
    i18nKey: "schedule.quick-input.template-coffee",
    promptI18nKey: "schedule.quick-input.prompt-coffee",
    icon: "coffee",
    duration: 30,
    defaultTitle: "咖啡聊天",
    color: "bg-amber-600",
  },
  {
    id: "workout",
    title: "运动",
    i18nKey: "schedule.quick-input.template-workout",
    promptI18nKey: "schedule.quick-input.prompt-workout",
    icon: "dumbbell",
    duration: 60,
    defaultTitle: "锻炼",
    color: "bg-red-500",
  },
  {
    id: "focus-60",
    title: "专注时间",
    i18nKey: "schedule.quick-input.template-focus",
    promptI18nKey: "schedule.quick-input.prompt-focus",
    icon: "focus",
    duration: 60,
    defaultTitle: "专注时间",
    color: "bg-indigo-500",
  },
];

const ICON_MAP: Record<string, React.ElementType> = {
  users: ({ className }: { className?: string }) => (
    <svg
      className={className}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
      <path d="M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
  ),
  phone: Phone,
  video: Video,
  lunch: UtensilsCrossed,
  coffee: Coffee,
  dumbbell: Dumbbell,
  focus: ({ className }: { className?: string }) => (
    <svg
      className={className}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx="12" cy="12" r="10" />
      <circle cx="12" cy="12" r="6" />
      <circle cx="12" cy="12" r="2" />
    </svg>
  ),
};

export function QuickTemplates({ onSelect, className, templates = DEFAULT_TEMPLATES }: QuickTemplatesProps) {
  const t = useTranslate();

  return (
    <div className={cn("flex flex-wrap gap-1.5", className)}>
      {templates.map((template) => {
        const IconComponent = ICON_MAP[template.icon] || ICON_MAP.users;
        const displayTitle = template.i18nKey ? (t(template.i18nKey) || template.title) : template.title;

        return (
          <button
            key={template.id}
            onClick={() => onSelect?.(template)}
            className={cn(
              "flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs font-medium",
              "bg-muted/50 hover:bg-muted transition-colors",
              "border border-transparent hover:border-border/50",
              "text-foreground/80 hover:text-foreground",
            )}
            title={displayTitle}
          >
            <IconComponent className="h-3.5 w-3.5 shrink-0" />
            <span>{displayTitle}</span>
          </button>
        );
      })}
    </div>
  );
}

interface QuickTemplateDropdownProps {
  /** Called when a template is selected */
  onSelect?: (template: ScheduleTemplate) => void;
  /** Optional className */
  className?: string;
  /** Whether dropdown is open */
  open?: boolean;
  /** Toggle dropdown open state */
  onToggle?: () => void;
  /** Whether the dropdown is disabled */
  disabled?: boolean;
}

/**
 * Dropdown version of quick templates
 */
export function QuickTemplateDropdown({ onSelect, className, open, onToggle, disabled = false }: QuickTemplateDropdownProps) {
  const t = useTranslate();
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [dropdownStyle, setDropdownStyle] = useState<React.CSSProperties>({});

  // Calculate dropdown position when opening
  useEffect(() => {
    if (!open || !triggerRef.current) return;

    const triggerRect = triggerRef.current.getBoundingClientRect();
    const dropdownWidth = 208; // w-52 = 13rem = 208px

    setDropdownStyle({
      position: "fixed",
      left: `${triggerRect.left}px`,
      bottom: `${window.innerHeight - triggerRect.top + 8}px`, // Position above with 8px gap
      width: `${dropdownWidth}px`,
    });
  }, [open]);

  // Handle Escape key to close dropdown
  useEffect(() => {
    if (!open || !onToggle) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onToggle();
        triggerRef.current?.focus();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, onToggle]);

  useClickOutside(
    dropdownRef,
    () => {
      if (open) {
        onToggle?.();
      }
    },
    open,
  );

  const dropdownContent = open ? (
    <div
      ref={dropdownRef}
      style={dropdownStyle}
      className="p-1.5 bg-popover rounded-lg border shadow-lg z-[9999] max-h-[60vh] overflow-y-auto"
      role="menu"
      aria-label={t("schedule.quick-input.quick-create")}
    >
      <div className="text-[10px] text-muted-foreground mb-1.5 px-1">{t("schedule.quick-input.quick-create")}</div>
      <div className="flex flex-col gap-0.5" role="presentation">
        {DEFAULT_TEMPLATES.map((template) => {
          const IconComponent = ICON_MAP[template.icon] || ICON_MAP.users;
          const displayTitle = template.i18nKey ? (t(template.i18nKey) || template.title) : template.title;

          return (
            <button
              key={template.id}
              type="button"
              role="menuitem"
              tabIndex={0}
              onClick={() => {
                onSelect?.(template);
                onToggle?.(); // Close dropdown
              }}
              className={cn(
                "flex items-center gap-2 px-2 py-2 rounded-md text-xs min-h-[44px]",
                "hover:bg-muted transition-colors",
                "text-left w-full focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-1",
              )}
              aria-label={`${displayTitle}，${template.duration}分钟`}
            >
              <div
                className={cn("h-5 w-5 rounded flex items-center justify-center shrink-0", template.color, "text-white")}
                aria-hidden="true"
              >
                <IconComponent className="h-3 w-3" />
              </div>
              <span className="font-medium truncate flex-1">{displayTitle}</span>
              <span className="text-[10px] text-muted-foreground shrink-0">
                {template.duration}
                {t("schedule.quick-input.minutes-abbr") as string}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  ) : null;

  return (
    <div className={cn("relative", className)}>
      <button
        ref={triggerRef}
        type="button"
        onClick={onToggle}
        disabled={disabled}
        aria-expanded={open}
        aria-haspopup="menu"
        aria-label={t("schedule.quick-input.templates") as string}
        className={cn(
          "flex items-center justify-center text-xs font-medium transition-all duration-200 focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-1",
          // Golden ratio: width ≈ height * 1.618, using 32x52 for compact square
          "h-9 w-[52px] rounded-lg",
          disabled
            ? "text-muted-foreground/40 cursor-not-allowed opacity-50"
            : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
        )}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="h-4 w-4"
          aria-hidden="true"
        >
          <rect x="3" y="3" width="7" height="7" />
          <rect x="14" y="3" width="7" height="7" />
          <rect x="14" y="14" width="7" height="7" />
          <rect x="3" y="14" width="7" height="7" />
        </svg>
      </button>

      {dropdownContent && createPortal(dropdownContent, document.body)}
    </div>
  );
}
