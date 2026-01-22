import { MoreHorizontal, Pencil, Pin, PinOff, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { ConversationSummary } from "@/types/aichat";
import { PARROT_AGENTS, PARROT_ICONS, PARROT_THEMES } from "@/types/parrot";

interface ConversationItemProps {
  conversation: ConversationSummary;
  isActive: boolean;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onRename: (id: string, newTitle: string) => void;
  onTogglePin: (id: string) => void;
  className?: string;
}

export function ConversationItem({ conversation, isActive, onSelect, onDelete, onRename, onTogglePin, className }: ConversationItemProps) {
  const parrot = PARROT_AGENTS[conversation.parrotId];
  const parrotIcon = PARROT_ICONS[conversation.parrotId] || parrot?.icon || "🤖";
  const parrotTheme = PARROT_THEMES[conversation.parrotId] || PARROT_THEMES.DEFAULT;

  return (
    <div
      className={cn(
        "group relative rounded-lg transition-all",
        isActive ? "bg-zinc-100 dark:bg-zinc-800" : "hover:bg-zinc-50 dark:hover:bg-zinc-800/50",
        className,
      )}
    >
      <button
        onClick={() => onSelect(conversation.id)}
        className="w-full text-left p-3"
        aria-label={`Select conversation: ${conversation.title}`}
      >
        <div className="flex items-start gap-3">
          {/* Parrot Icon */}
          <div className={cn("w-9 h-9 rounded-lg flex items-center justify-center text-lg shrink-0", !parrotIcon.startsWith("/") && parrotTheme.iconBg)}>
            {parrotIcon.startsWith("/") ? (
              <img src={parrotIcon} alt={parrot?.displayName || ""} className="w-8 h-8 object-contain" />
            ) : (
              parrotIcon
            )}
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="font-medium text-sm text-zinc-900 dark:text-zinc-100 truncate">{conversation.title}</h3>
              {conversation.pinned && <Pin className="w-3 h-3 text-blue-500 shrink-0" />}
            </div>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
              {conversation.messageCount} messages · {formatTime(conversation.updatedAt)}
            </p>
          </div>
        </div>
      </button>

      {/* Action Menu */}
      <div className="absolute right-2 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity">
        <ActionMenu
          conversationId={conversation.id}
          conversationTitle={conversation.title}
          isPinned={conversation.pinned}
          onDelete={onDelete}
          onRename={onRename}
          onTogglePin={onTogglePin}
        />
      </div>
    </div>
  );
}

interface ActionMenuProps {
  conversationId: string;
  conversationTitle: string;
  isPinned: boolean;
  onDelete: (id: string) => void;
  onRename: (id: string, newTitle: string) => void;
  onTogglePin: (id: string) => void;
}

function ActionMenu({ conversationId, conversationTitle, isPinned, onDelete, onRename, onTogglePin }: ActionMenuProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(conversationTitle);

  useEffect(() => {
    setEditValue(conversationTitle);
  }, [conversationTitle]);

  const handleRename = () => {
    if (editValue.trim() && editValue !== conversationTitle) {
      onRename(conversationId, editValue.trim());
    }
    setIsEditing(false);
  };

  if (isEditing) {
    return (
      <div className="flex items-center gap-1 bg-white dark:bg-zinc-800 rounded-lg shadow-lg border border-zinc-200 dark:border-zinc-700 p-1">
        <input
          type="text"
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") handleRename();
            if (e.key === "Escape") setIsEditing(false);
          }}
          onBlur={handleRename}
          className="w-32 px-2 py-1 text-xs bg-transparent border-0 outline-none text-zinc-900 dark:text-zinc-100"
          autoFocus
          aria-label="Rename conversation"
        />
      </div>
    );
  }

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="p-1.5 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-700 transition-colors"
        aria-label={t("ai.more-options")}
      >
        <MoreHorizontal className="w-4 h-4 text-zinc-500" />
      </button>

      {isOpen && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setIsOpen(false)} />
          <div className="absolute right-0 top-full mt-1 z-20 bg-white dark:bg-zinc-800 rounded-lg shadow-lg border border-zinc-200 dark:border-zinc-700 py-1 min-w-[140px]">
            <button
              onClick={() => {
                onTogglePin(conversationId);
                setIsOpen(false);
              }}
              className="flex items-center gap-2 w-full px-3 py-2 text-xs text-left hover:bg-zinc-100 dark:hover:bg-zinc-700"
              aria-label={isPinned ? "Unpin conversation" : "Pin conversation"}
            >
              {isPinned ? (
                <>
                  <PinOff className="w-3.5 h-3.5" />
                  Unpin
                </>
              ) : (
                <>
                  <Pin className="w-3.5 h-3.5" />
                  Pin
                </>
              )}
            </button>
            <button
              onClick={() => {
                setIsEditing(true);
                setIsOpen(false);
              }}
              className="flex items-center gap-2 w-full px-3 py-2 text-xs text-left hover:bg-zinc-100 dark:hover:bg-zinc-700"
              aria-label="Rename conversation"
            >
              <Pencil className="w-3.5 h-3.5" />
              Rename
            </button>
            <div className="h-px bg-zinc-200 dark:bg-zinc-700 my-1" />
            <button
              onClick={() => {
                onDelete(conversationId);
                setIsOpen(false);
              }}
              className="flex items-center gap-2 w-full px-3 py-2 text-xs text-left text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
              aria-label="Delete conversation"
            >
              <Trash2 className="w-3.5 h-3.5" />
              Delete
            </button>
          </div>
        </>
      )}
    </div>
  );
}

function formatTime(timestamp: number): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
}
