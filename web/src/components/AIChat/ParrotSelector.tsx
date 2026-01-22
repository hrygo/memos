import { useEffect, useCallback, useState } from "react";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";
import { ParrotAgent, getAvailableParrots } from "@/types/parrot";

interface ParrotSelectorProps {
  onSelect: (parrot: ParrotAgent) => void;
  onClose: () => void;
  position?: { x: number; y: number };
}

export function ParrotSelector({ onSelect, onClose, position }: ParrotSelectorProps) {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [availableParrots] = useState<ParrotAgent[]>(getAvailableParrots());

  // Handle keyboard navigation
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setSelectedIndex((prev) => (prev + 1) % availableParrots.length);
        break;
      case "ArrowUp":
        e.preventDefault();
        setSelectedIndex((prev) => (prev - 1 + availableParrots.length) % availableParrots.length);
        break;
      case "Enter":
        e.preventDefault();
        onSelect(availableParrots[selectedIndex]);
        onClose();
        break;
      case "Escape":
        e.preventDefault();
        onClose();
        break;
    }
  }, [availableParrots, selectedIndex, onSelect, onClose]);

  // Handle click outside to close
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest(".parrot-selector")) {
        onClose();
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [handleKeyDown, onClose]);

  const handleParrotClick = (parrot: ParrotAgent) => {
    onSelect(parrot);
    onClose();
  };

  return (
    <div
      className={cn(
        "parrot-selector fixed z-50 bg-white dark:bg-zinc-900 rounded-lg shadow-xl border border-zinc-200 dark:border-zinc-700 w-80 max-h-96 overflow-auto",
        "animate-in fade-in-0 zoom-in-95 duration-200"
      )}
      style={{
        left: position?.x ?? 0,
        top: position?.y ?? 0,
      }}
    >
      <div className="p-2">
        <div className="text-xs font-medium text-zinc-500 dark:text-zinc-400 px-3 py-2">
          选择鹦鹉助手
        </div>
        <div className="space-y-1">
          {availableParrots.map((parrot, idx) => (
            <button
              key={parrot.id}
              onClick={() => handleParrotClick(parrot)}
              onMouseEnter={() => setSelectedIndex(idx)}
              className={cn(
                "w-full flex items-center justify-between px-3 py-3 rounded-md transition-colors",
                "text-left hover:bg-zinc-100 dark:hover:bg-zinc-800",
                selectedIndex === idx && "bg-blue-50 dark:bg-blue-900/20"
              )}
            >
              <div className="flex items-center space-x-3">
                <span className="text-2xl" role="img" aria-label={parrot.displayName}>
                  {parrot.icon}
                </span>
                <div>
                  <div className="font-medium text-zinc-900 dark:text-zinc-100 flex items-center">
                    {parrot.displayName}
                    {parrot.available && (
                      <span className="ml-2 text-xs px-2 py-0.5 rounded-full bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400">
                        可用
                      </span>
                    )}
                  </div>
                  <div className="text-sm text-zinc-500 dark:text-zinc-400 mt-0.5">
                    {parrot.description}
                  </div>
                </div>
              </div>
              {selectedIndex === idx && (
                <Check className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              )}
            </button>
          ))}
        </div>
      </div>
      <div className="border-t border-zinc-200 dark:border-zinc-700 px-3 py-2 bg-zinc-50 dark:bg-zinc-800/50">
        <div className="text-xs text-zinc-500 dark:text-zinc-400 flex items-center">
          <span className="mr-2">键盘快捷键:</span>
          <kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-600 text-xs">
            ↑↓
          </kbd>
          <span className="mx-1">选择</span>
          <kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-600 text-xs">
            Enter
          </kbd>
          <span className="mx-1">确认</span>
          <kbd className="px-1.5 py-0.5 rounded bg-white dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-600 text-xs">
            Esc
          </kbd>
          <span className="mx-1">关闭</span>
        </div>
      </div>
    </div>
  );
}
