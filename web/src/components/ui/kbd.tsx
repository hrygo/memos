import * as React from "react";
import { cn } from "@/lib/utils";

const Kbd = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "inline-flex items-center justify-center rounded-md border border-border bg-muted px-2 py-1 text-xs font-medium text-foreground shadow-sm",
      className,
    )}
    {...props}
  />
));
Kbd.displayName = "Kbd";

export { Kbd };
