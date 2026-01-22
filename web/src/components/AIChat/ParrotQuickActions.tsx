import { cn } from "@/lib/utils";
import { ParrotAgent, getAvailableParrots } from "@/types/parrot";

interface ParrotQuickActionsProps {
  currentParrot: ParrotAgent | null;
  onParrotChange: (parrot: ParrotAgent) => void;
  disabled?: boolean;
}

export function ParrotQuickActions({ currentParrot, onParrotChange, disabled = false }: ParrotQuickActionsProps) {
  const availableParrots = getAvailableParrots();

  const handleParrotSelect = (parrot: ParrotAgent) => {
    if (!disabled) {
      onParrotChange(parrot);
    }
  };

  return (
    <div className="flex items-center space-x-2 w-full overflow-x-auto pb-2">
      {availableParrots.map((parrot) => {
        const isSelected = currentParrot?.id === parrot.id;
        const colorClass = getColorClass(parrot.color);

        return (
          <button
            key={parrot.id}
            onClick={() => handleParrotSelect(parrot)}
            disabled={disabled}
            className={cn(
              "flex-shrink-0 flex items-center space-x-2 px-4 py-3 rounded-lg border-2 transition-all",
              "hover:shadow-md disabled:opacity-50 disabled:cursor-not-allowed",
              isSelected
                ? `${colorClass} border-current shadow-sm`
                : "border-zinc-200 dark:border-zinc-700 hover:border-zinc-300 dark:hover:border-zinc-600 bg-white dark:bg-zinc-800"
            )}
          >
            <span className="text-2xl" role="img" aria-label={parrot.displayName}>
              {parrot.icon}
            </span>
            <div className="text-left">
              <div className={cn(
                "font-semibold text-sm",
                isSelected ? "text-current" : "text-zinc-900 dark:text-zinc-100"
              )}>
                {parrot.displayName}
              </div>
              <div className={cn(
                "text-xs",
                isSelected ? "text-current opacity-80" : "text-zinc-500 dark:text-zinc-400"
              )}>
                {parrot.name}
              </div>
            </div>
            {isSelected && (
              <div className="ml-auto">
                <div className="w-2 h-2 rounded-full bg-current" />
              </div>
            )}
          </button>
        );
      })}
    </div>
  );
}

function getColorClass(color: string): string {
  const colorMap: Record<string, string> = {
    blue: "bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 border-blue-300 dark:border-blue-700",
    purple: "bg-purple-50 dark:bg-purple-900/20 text-purple-700 dark:text-purple-300 border-purple-300 dark:border-purple-700",
    orange: "bg-orange-50 dark:bg-orange-900/20 text-orange-700 dark:text-orange-300 border-orange-300 dark:border-orange-700",
    pink: "bg-pink-50 dark:bg-pink-900/20 text-pink-700 dark:text-pink-300 border-pink-300 dark:border-pink-700",
    gray: "bg-gray-50 dark:bg-gray-900/20 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-700",
    green: "bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300 border-green-300 dark:border-green-700",
    red: "bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-300 border-red-300 dark:border-red-700",
    yellow: "bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-300 border-yellow-300 dark:border-yellow-700",
  };

  return colorMap[color] || colorMap.gray;
}
