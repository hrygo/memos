import { cn } from "@/lib/utils";

/**
 * Unified card style constants for consistent card styling across the application.
 * Replaces scattered card classes with semantic, maintainable constants.
 */

/**
 * Base card variants for different visual styles
 */
export const CARD_VARIANTS = {
  /** Default card with border and background */
  default: "bg-card border border-border rounded-lg",
  /** Elevated card with shadow */
  elevated: "bg-card border border-border rounded-lg shadow-md",
  /** Flat card without border */
  flat: "bg-card rounded-lg",
  /** Interactive card with hover effects */
  interactive: "bg-card border border-border rounded-lg hover:bg-muted/50 transition-colors",
} as const;

/**
 * Padding size variants for cards
 */
export const CARD_SIZES = {
  sm: "px-3 py-2",
  md: "px-4 py-3",
  lg: "px-6 py-4",
} as const;

/**
 * Base classes for memo cards
 * Combines variant, size, and layout classes
 */
export const MEMO_CARD = cn(
  CARD_VARIANTS.default,
  CARD_SIZES.md,
  "relative group flex flex-col justify-start items-start w-full mb-2 gap-2 text-card-foreground transition-colors"
);

/**
 * Base classes for memo editor
 * Similar to MEMO_CARD but with different padding and without bottom margin
 */
export const MEMO_EDITOR_CARD = cn(
  CARD_VARIANTS.default,
  "px-4 pt-3 pb-1",
  "group relative w-full flex flex-col justify-between items-start gap-2"
);

/**
 * Base classes for memo card skeleton (loading state)
 */
export const MEMO_CARD_SKELETON = cn(
  CARD_VARIANTS.default,
  CARD_SIZES.md,
  "relative flex flex-col justify-start items-start w-full mb-2 gap-2 animate-pulse"
);

/**
 * Base classes for related memo cards (smaller, clickable)
 */
export const RELATED_MEMO_CARD = cn(
  CARD_VARIANTS.interactive,
  "p-3 group"
);
