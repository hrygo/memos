import { cn } from "@/lib/utils";
import { PARROT_THEMES, ParrotAgentType } from "@/types/parrot";

interface SkeletonLoaderProps {
  variant?: "message" | "avatar" | "card" | "input";
  parrotId?: ParrotAgentType;
  className?: string;
}

/**
 * AI Native Skeleton Loading Component
 * Provides intelligent loading states with subtle animations
 */
export function SkeletonLoader({ variant = "message", parrotId, className }: SkeletonLoaderProps) {
  const theme = parrotId ? PARROT_THEMES[parrotId] || PARROT_THEMES.DEFAULT : PARROT_THEMES.DEFAULT;

  if (variant === "avatar") {
    return <div className={cn("w-9 h-9 md:w-10 md:h-10 rounded-full animate-pulse", theme.iconBg, className)} />;
  }

  if (variant === "message") {
    return (
      <div className={cn("flex gap-3 md:gap-4", className)}>
        {/* Avatar skeleton */}
        <div className={cn("w-9 h-9 md:w-10 md:h-10 rounded-full animate-pulse shrink-0", theme.iconBg)} />
        {/* Message skeleton */}
        <div className="flex-1 space-y-2">
          <div className={cn("h-4 rounded-lg animate-pulse w-3/4", theme.bubbleBg, "opacity-60")} />
          <div className={cn("h-4 rounded-lg animate-pulse w-1/2", theme.bubbleBg, "opacity-40")} />
        </div>
      </div>
    );
  }

  if (variant === "card") {
    return (
      <div className={cn("rounded-xl border p-4 space-y-3", theme.cardBg, theme.cardBorder, "animate-pulse", className)}>
        <div className="flex items-center gap-3">
          <div className={cn("w-10 h-10 rounded-xl", theme.iconBg, "opacity-60")} />
          <div className="flex-1 space-y-2">
            <div className={cn("h-4 rounded w-24", "bg-zinc-200 dark:bg-zinc-700", "opacity-60")} />
            <div className={cn("h-3 rounded w-16", "bg-zinc-100 dark:bg-zinc-800", "opacity-40")} />
          </div>
        </div>
        <div className={cn("h-12 rounded-lg", "bg-zinc-100 dark:bg-zinc-800", "opacity-40")} />
      </div>
    );
  }

  if (variant === "input") {
    return <div className={cn("h-12 rounded-xl border animate-pulse", theme.inputBg, theme.inputBorder, className)} />;
  }

  return null;
}

/**
 * Streaming Skeleton - Shows AI is thinking and generating response
 */
interface StreamingSkeletonProps {
  parrotId?: ParrotAgentType;
  message?: boolean;
}

export function StreamingSkeleton({ parrotId, message = true }: StreamingSkeletonProps) {
  const theme = parrotId ? PARROT_THEMES[parrotId] || PARROT_THEMES.DEFAULT : PARROT_THEMES.DEFAULT;

  return (
    <div className="flex gap-3 md:gap-4">
      <div className={cn("w-9 h-9 md:w-10 md:h-10 rounded-full flex items-center justify-center shrink-0", theme.iconBg)}>
        <span className="text-lg md:text-xl animate-pulse">✨</span>
      </div>
      <div className={cn("px-4 py-3 rounded-2xl border shadow-sm max-w-[80%]", theme.bubbleBg, theme.bubbleBorder)}>
        {message && (
          <div className="space-y-2">
            <div className={cn("h-3 rounded w-full animate-pulse", theme.inputBg, "opacity-60")} style={{ animationDelay: "0ms" }} />
            <div className={cn("h-3 rounded w-5/6 animate-pulse", theme.inputBg, "opacity-40")} style={{ animationDelay: "150ms" }} />
            <div className={cn("h-3 rounded w-4/6 animate-pulse", theme.inputBg, "opacity-30")} style={{ animationDelay: "300ms" }} />
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * InitialLoadingSkeleton - Full page loading skeleton for HubView
 */
export function InitialLoadingSkeleton() {
  return (
    <div className="w-full h-full flex flex-col bg-[#F8F5F0] dark:bg-zinc-900">
      {/* Header skeleton */}
      <div className="px-4 md:px-8 py-4 border-b border-zinc-200/50 dark:border-zinc-800">
        <div className="h-6 w-32 bg-zinc-200 dark:bg-zinc-700 rounded animate-pulse" />
      </div>

      {/* Cards grid skeleton */}
      <div className="flex-1 overflow-auto p-3 md:p-6">
        <div className="max-w-4xl mx-auto">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div
                key={i}
                className="h-32 rounded-xl border bg-white dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 animate-pulse"
                style={{ animationDelay: `${i * 100}ms` }}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * ConversationItemSkeleton - Skeleton for conversation list item
 */
interface ConversationItemSkeletonProps {
  parrotId?: ParrotAgentType;
}

export function ConversationItemSkeleton({ parrotId }: ConversationItemSkeletonProps) {
  const theme = parrotId ? PARROT_THEMES[parrotId] || PARROT_THEMES.DEFAULT : PARROT_THEMES.DEFAULT;

  return (
    <div className={cn("flex items-center gap-3 p-3 rounded-xl border animate-pulse", theme.cardBg, theme.cardBorder)}>
      <div className={cn("w-8 h-8 rounded-full shrink-0", theme.iconBg)} />
      <div className="flex-1 min-w-0 space-y-2">
        <div className="h-4 w-3/4 rounded bg-zinc-200 dark:bg-zinc-700" />
        <div className="h-3 w-1/2 rounded bg-zinc-100 dark:bg-zinc-800" />
      </div>
    </div>
  );
}

export default SkeletonLoader;
