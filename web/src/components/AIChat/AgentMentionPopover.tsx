import { CheckCircle2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ParrotAgentI18n } from "@/hooks/useParrots";
import { cn } from "@/lib/utils";
import { PARROT_ICONS, PARROT_THEMES } from "@/types/parrot";

interface AgentMentionPopoverProps {
  open: boolean;
  onClose: () => void;
  onSelectAgent: (agent: ParrotAgentI18n) => void;
  agents: ParrotAgentI18n[];
  filterText?: string;
  triggerRef: React.RefObject<HTMLTextAreaElement>;
}

export function AgentMentionPopover({ open, onClose, onSelectAgent, agents, filterText = "", triggerRef }: AgentMentionPopoverProps) {
  const { t } = useTranslation();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const popoverRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });

  // Filter agents based on filter text
  const filteredAgents = useMemo(() => {
    if (!filterText) return agents;
    const lowerFilter = filterText.toLowerCase();
    return agents.filter(
      (agent) =>
        agent.displayName.toLowerCase().includes(lowerFilter) ||
        agent.displayNameAlt.toLowerCase().includes(lowerFilter) ||
        agent.name.toLowerCase().includes(lowerFilter),
    );
  }, [agents, filterText]);

  // Reset selected index when filtered agents change
  useEffect(() => {
    setSelectedIndex(0);
  }, [filterText]);

  // Calculate position based on textarea
  useEffect(() => {
    if (!open || !triggerRef.current) return;

    const textarea = triggerRef.current;
    const rect = textarea.getBoundingClientRect();

    // Position above the textarea (fixed positioning, so no scroll offset needed)
    setPosition({
      top: rect.top,
      left: rect.left,
    });
  }, [open, triggerRef]);

  // Handle keyboard navigation
  useEffect(() => {
    if (!open) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setSelectedIndex((prev) => (prev + 1) % filteredAgents.length);
          break;
        case "ArrowUp":
          e.preventDefault();
          setSelectedIndex((prev) => (prev - 1 + filteredAgents.length) % filteredAgents.length);
          break;
        case "Enter":
          e.preventDefault();
          if (filteredAgents[selectedIndex]) {
            onSelectAgent(filteredAgents[selectedIndex]);
          }
          break;
        case "Escape":
          e.preventDefault();
          onClose();
          break;
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, filteredAgents, selectedIndex, onSelectAgent, onClose]);

  // Handle click outside
  useEffect(() => {
    if (!open) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [open, onClose]);

  // Scroll selected item into view
  useEffect(() => {
    if (popoverRef.current && open) {
      const selectedElement = popoverRef.current.querySelector(`[data-index="${selectedIndex}"]`);
      if (selectedElement) {
        selectedElement.scrollIntoView({ block: "nearest" });
      }
    }
  }, [selectedIndex, open]);

  if (!open || filteredAgents.length === 0) {
    return null;
  }

  return (
    <div
      ref={popoverRef}
      className="fixed z-[100] w-[min(300px,calc(100vw-32px))] bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-700 shadow-lg p-1.5 max-h-64 overflow-y-auto -mt-1"
      style={{
        top: `${position.top}px`,
        left: `${position.left}px`,
        transform: "translateY(-100%)",
      }}
    >
      <div className="space-y-0.5">
        <div className="px-2 py-1 text-xs text-zinc-500 dark:text-zinc-400 border-b border-zinc-200 dark:border-zinc-700 mb-1">
          {t("ai.parrot.mention-hint")}
        </div>
        {filteredAgents.map((agent, index) => {
          const isSelected = index === selectedIndex;
          const theme = PARROT_THEMES[agent.id] || PARROT_THEMES.DEFAULT;
          const icon = PARROT_ICONS[agent.id] || agent.icon;

          return (
            <button
              key={agent.id}
              data-index={index}
              type="button"
              onClick={() => onSelectAgent(agent)}
              className={cn(
                "w-full flex items-center gap-2 px-2 py-2 rounded-md text-left transition-colors",
                "hover:bg-zinc-100 dark:hover:bg-zinc-800",
                "focus:outline-none focus:ring-2 focus:ring-zinc-400 focus:ring-inset",
                isSelected && "bg-zinc-100 dark:bg-zinc-800",
              )}
            >
              {/* Icon */}
              <div className="w-7 h-7 rounded-md flex items-center justify-center shrink-0">
                {icon.startsWith("/") ? (
                  <img src={icon} alt={agent.displayName} className="w-6 h-6 object-contain" />
                ) : (
                  <span className="text-sm">{icon}</span>
                )}
              </div>

              {/* Name */}
              <div className="flex-1 min-w-0">
                <div className="flex items-baseline gap-1.5">
                  <span className="text-sm font-medium text-zinc-900 dark:text-zinc-100 truncate">{agent.displayName}</span>
                  <span className="text-xs text-zinc-400 dark:text-zinc-500 truncate">{agent.displayNameAlt}</span>
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 truncate">{agent.description}</p>
              </div>

              {/* Selected indicator */}
              {isSelected && <CheckCircle2 className={cn("w-4 h-4 shrink-0", theme.iconText)} />}
            </button>
          );
        })}
      </div>
    </div>
  );
}
