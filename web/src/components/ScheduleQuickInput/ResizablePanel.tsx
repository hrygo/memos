import { GripVertical, X } from "lucide-react";
import type { RefObject } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

/**
 * Focus trap hook for modal/panel dialogs
 * Ensures keyboard navigation stays within the panel when open
 */
function useFocusTrap(isActive: boolean, containerRef: RefObject<HTMLElement>) {
  const previousActiveElement = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!isActive) return;

    // Store the previously focused element
    previousActiveElement.current = document.activeElement as HTMLElement;

    const container = containerRef.current;
    if (!container) return;

    // Find all focusable elements within the container
    const focusableElements = container.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
    const firstFocusable = focusableElements[0] as HTMLElement;
    const lastFocusable = focusableElements[focusableElements.length - 1] as HTMLElement;

    // Focus the first element
    firstFocusable?.focus();

    // Handle Tab key to trap focus
    const handleTabKey = (e: KeyboardEvent) => {
      if (e.key !== "Tab") return;

      if (e.shiftKey) {
        // Shift + Tab
        if (document.activeElement === firstFocusable) {
          e.preventDefault();
          lastFocusable?.focus();
        }
      } else {
        // Tab
        if (document.activeElement === lastFocusable) {
          e.preventDefault();
          firstFocusable?.focus();
        }
      }
    };

    document.addEventListener("keydown", handleTabKey);

    return () => {
      document.removeEventListener("keydown", handleTabKey);
      // Restore focus to the previous element when panel closes
      previousActiveElement.current?.focus();
    };
  }, [isActive, containerRef]);
}

type PanelPosition = "bottom" | "right";

interface ResizablePanelProps {
  /** Whether panel is open */
  open: boolean;
  /** Called when open state changes */
  onOpenChange: (open: boolean) => void;
  /** Panel content */
  children: React.ReactNode;
  /** Panel position (default: "right") */
  position?: PanelPosition;
  /** Initial height/width percentage (default: 30) */
  initialSize?: number;
  /** Minimum size percentage (default: 20) */
  minSize?: number;
  /** Maximum size percentage (default: 85) */
  maxSize?: number;
  /** Optional className */
  className?: string;
  /** Optional header content */
  header?: React.ReactNode;
  /** Whether to show close button (default: true) */
  showCloseButton?: boolean;
  /** Container element ref for bottom panel height calculation */
  containerRef?: React.RefObject<HTMLElement>;
}

export function ResizablePanel({
  open,
  onOpenChange,
  children,
  position = "right",
  initialSize = 30,
  minSize = 20,
  maxSize = 85,
  className,
  header,
  showCloseButton = true,
  containerRef,
}: ResizablePanelProps) {
  const t = useTranslate();
  const [sizePercent, setSizePercent] = useState(initialSize);
  const [isResizing, setIsResizing] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const resizeHandleRef = useRef<HTMLDivElement>(null);

  const isRight = position === "right";
  const cursorClass = isRight ? "ew-resize" : "ns-resize";
  const gripRotation = isRight ? "rotate-90" : "";

  // Reset size when opening
  useEffect(() => {
    if (open) {
      setSizePercent(initialSize);
    }
  }, [open, initialSize]);

  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      let newSizePx: number;

      if (isRight) {
        // For right panel, measure from right edge
        newSizePx = window.innerWidth - e.clientX;
        const newSizePercent = (newSizePx / window.innerWidth) * 100;
        const clampedPercent = Math.max(minSize, Math.min(maxSize, newSizePercent));
        setSizePercent(clampedPercent);
      } else {
        // For bottom panel, measure from bottom edge
        // Get container height if available, otherwise use viewport
        const containerHeight = containerRef?.current?.offsetHeight || window.innerHeight;
        const containerTop = containerRef?.current?.getBoundingClientRect().top || 0;
        newSizePx = window.innerHeight - e.clientY - containerTop;
        const newSizePercent = (newSizePx / containerHeight) * 100;
        const clampedPercent = Math.max(minSize, Math.min(maxSize, newSizePercent));
        setSizePercent(clampedPercent);
      }
    };

    const handleMouseUp = () => {
      setIsResizing(false);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);

    // Prevent text selection during resize
    document.body.style.userSelect = "none";
    document.body.style.cursor = cursorClass;

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.userSelect = "";
      document.body.style.cursor = "";
    };
  }, [isResizing, minSize, maxSize, isRight, cursorClass, containerRef]);

  const handleResizeStart = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  };

  const handleTouchStart = (e: React.TouchEvent) => {
    e.preventDefault();
    const touch = e.touches[0];
    const mouseEvent = new MouseEvent("mousedown", {
      clientX: touch.clientX,
      clientY: touch.clientY,
    });
    handleResizeStart(mouseEvent as any);
  };

  // Handle Escape key to close panel
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onOpenChange(false);
      }
    },
    [onOpenChange],
  );

  // Focus trap
  useFocusTrap(open, panelRef);

  // Add global Escape key listener
  useEffect(() => {
    if (!open) return;

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, handleKeyDown]);

  if (!open) return null;

  const panelStyle = isRight ? { width: `${sizePercent}%`, height: "100%" } : { height: `${sizePercent}%`, width: "100%" };

  const panelClass = isRight
    ? "absolute top-0 bottom-0 right-0 border-l border-border/50 rounded-l-2xl"
    : "absolute bottom-0 left-0 right-0 border-t border-border/50 rounded-t-2xl";

  const handleClass = isRight
    ? "flex-row items-center justify-center py-3 w-3 cursor-ew-resize hover:bg-muted/50"
    : "flex items-center justify-center py-3 cursor-ns-resize hover:bg-muted/50";

  const gripContainerClass = isRight ? "flex flex-col items-center gap-3" : "flex items-center gap-3";

  const indicatorClass = isRight ? "w-1 h-24 rounded-full bg-muted-foreground/20" : "h-1 w-24 rounded-full bg-muted-foreground/20";

  // For bottom panel, use absolute positioning within container
  if (!isRight) {
    return (
      <div className="absolute inset-0 z-50 pointer-events-none">
        {/* Backdrop */}
        <div className="absolute inset-0 bg-black/20 pointer-events-auto cursor-pointer" onClick={() => onOpenChange(false)} aria-hidden="true" />

        {/* Resizable Panel */}
        <div
          ref={panelRef}
          role="dialog"
          aria-modal="true"
          className={cn(panelClass, "bg-background shadow-2xl pointer-events-auto", "flex flex-col", className)}
          style={panelStyle}
        >
          {/* Resize Handle at top */}
          <div
            ref={resizeHandleRef}
            className={cn("absolute left-0 right-0 top-0 select-none touch-none transition-colors", handleClass)}
            onMouseDown={handleResizeStart}
            onTouchStart={handleTouchStart}
            role="separator"
            aria-orientation="horizontal"
            aria-label={t("schedule.drag-resize-panel") as string}
          >
            <div className={gripContainerClass}>
              <GripVertical className={cn("h-4 w-4 text-muted-foreground", gripRotation)} />
              <div className={indicatorClass} />
            </div>
          </div>

          {/* Close Button */}
          {showCloseButton && (
            <button
              type="button"
              onClick={() => onOpenChange(false)}
              aria-label={t("schedule.close-panel") as string}
              className="absolute right-4 top-1/2 -translate-y-1/2 p-1.5 rounded-full hover:bg-muted transition-colors focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
            >
              <X className="h-4 w-4 text-muted-foreground" />
            </button>
          )}

          {/* Header (optional) */}
          {header && <div className="border-b border-border/50 px-4 pb-3">{header}</div>}

          {/* Content */}
          <div className="flex-1 overflow-hidden pt-2 px-4">{children}</div>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 pointer-events-none">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/20 pointer-events-auto cursor-pointer" onClick={() => onOpenChange(false)} aria-hidden="true" />

      {/* Resizable Panel */}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        className={cn(panelClass, "bg-background shadow-2xl pointer-events-auto", "flex flex-col", className)}
        style={panelStyle}
      >
        {/* Resize Handle */}
        <div
          ref={resizeHandleRef}
          className={cn(
            "absolute select-none touch-none transition-colors",
            isRight ? "left-0 top-0 bottom-0" : "left-0 right-0 top-0",
            handleClass,
          )}
          onMouseDown={handleResizeStart}
          onTouchStart={handleTouchStart}
          role="separator"
          aria-orientation={isRight ? "vertical" : "horizontal"}
          aria-label={t("schedule.drag-resize-panel") as string}
        >
          <div className={gripContainerClass}>
            <GripVertical className={cn("h-4 w-4 text-muted-foreground", gripRotation)} />
            <div className={indicatorClass} />
          </div>
        </div>

        {/* Close Button */}
        {showCloseButton && (
          <button
            type="button"
            onClick={() => onOpenChange(false)}
            aria-label={t("schedule.close-panel") as string}
            className={cn(
              "absolute p-1.5 rounded-full hover:bg-muted transition-colors focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2",
              isRight ? "top-4 right-4" : "right-4 top-1/2 -translate-y-1/2",
            )}
          >
            <X className="h-4 w-4 text-muted-foreground" />
          </button>
        )}

        {/* Header (optional) */}
        {header && <div className={cn("border-b border-border/50", isRight ? "px-4 pt-4 pb-3" : "px-4 pb-3")}>{header}</div>}

        {/* Content - no scrollbar */}
        <div className={cn("flex-1 overflow-hidden", isRight ? "pt-4 px-4" : "")}>{children}</div>
      </div>
    </div>
  );
}

/**
 * Compact version - minimal resize handle
 */
interface CompactResizablePanelProps extends Omit<ResizablePanelProps, "header"> {
  title?: string;
}

export function CompactResizablePanel({ title, children, ...props }: CompactResizablePanelProps) {
  const t = useTranslate();

  return (
    <ResizablePanel position="right" initialSize={30} minSize={20} maxSize={60} {...props}>
      {/* Minimal header with just resize handle */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-border/50">
        <div
          className="flex items-center gap-2 cursor-ew-resize py-1"
          onMouseDown={(e) => {
            const panel = e.currentTarget.closest("[data-resizable-panel]");
            const handle = panel?.querySelector("[data-resize-handle]") as HTMLElement;
            handle?.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
          }}
        >
          <GripVertical className="h-4 w-4 text-muted-foreground rotate-90" data-resize-handle />
          {title && <span className="text-sm font-medium">{title}</span>}
        </div>
        <button
          type="button"
          onClick={() => props.onOpenChange(false)}
          aria-label={t("schedule.close-panel") as string}
          className="p-1 rounded-full hover:bg-muted transition-colors focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
        >
          <X className="h-4 w-4 text-muted-foreground" />
        </button>
      </div>

      {/* Content */}
      <div className="overflow-hidden h-full p-4">{children}</div>
    </ResizablePanel>
  );
}
