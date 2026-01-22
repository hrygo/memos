import { cn } from "@/lib/utils";
import { ParrotAgent } from "@/types/parrot";
import { Activity, Clock } from "lucide-react";

interface ParrotStatusProps {
  parrot: ParrotAgent | null;
  thinking?: boolean;
  className?: string;
}

export function ParrotStatus({ parrot, thinking = false, className }: ParrotStatusProps) {
  if (!parrot) {
    return null;
  }

  const colorClass = getColorClass(parrot.color);

  return (
    <div className={cn(
      "flex items-center space-x-2 px-3 py-2 rounded-lg bg-zinc-50 dark:bg-zinc-800/50",
      className
    )}>
      {/* Parrot Icon and Name */}
      <div className={cn(
        "flex items-center space-x-2 px-3 py-1.5 rounded-md",
        colorClass
      )}>
        <span className="text-xl" role="img" aria-label={parrot.displayName}>
          {parrot.icon}
        </span>
        <div className="flex flex-col">
          <span className="font-semibold text-sm leading-tight">
            {parrot.displayName}
          </span>
          <span className="text-xs opacity-80">
            {parrot.name}
          </span>
        </div>
      </div>

      {/* Status Indicator */}
      {thinking && (
        <div className="flex items-center space-x-1.5 text-blue-600 dark:text-blue-400">
          <Activity className="w-4 h-4 animate-pulse" />
          <span className="text-xs font-medium">思考中...</span>
        </div>
      )}

      {/* Parrot Description */}
      {!thinking && (
        <div className="text-xs text-zinc-500 dark:text-zinc-400 max-w-xs truncate">
          {parrot.description}
        </div>
      )}
    </div>
  );
}

interface ParrotStatusCompactProps {
  parrot: ParrotAgent | null;
  className?: string;
}

export function ParrotStatusCompact({ parrot, className }: ParrotStatusCompactProps) {
  if (!parrot) {
    return null;
  }

  const colorClass = getColorClass(parrot.color);

  return (
    <div className={cn(
      "inline-flex items-center space-x-1.5 px-2 py-1 rounded-md text-xs font-medium",
      colorClass,
      className
    )}>
      <span className="text-sm">{parrot.icon}</span>
      <span>{parrot.displayName}</span>
    </div>
  );
}

interface ParrotThinkingIndicatorProps {
  message?: string;
  className?: string;
}

export function ParrotThinkingIndicator({ message = "思考中...", className }: ParrotThinkingIndicatorProps) {
  return (
    <div className={cn(
      "flex items-center space-x-2 text-zinc-500 dark:text-zinc-400 text-sm",
      className
    )}>
      <Clock className="w-4 h-4 animate-spin" />
      <span>{message}</span>
    </div>
  );
}

function getColorClass(color: string): string {
  const colorMap: Record<string, string> = {
    blue: "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300",
    purple: "bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300",
    orange: "bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300",
    pink: "bg-pink-100 dark:bg-pink-900/30 text-pink-700 dark:text-pink-300",
    gray: "bg-gray-100 dark:bg-gray-900/30 text-gray-700 dark:text-gray-300",
    green: "bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300",
    red: "bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300",
    yellow: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300",
  };

  return colorMap[color] || colorMap.gray;
}
