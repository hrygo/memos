import * as React from "react";
import { cn } from "@/lib/utils";

const Textarea = React.forwardRef<HTMLTextAreaElement, React.ComponentProps<"textarea">>(({ className, ...props }, ref) => {
  return (
    <textarea
      ref={ref}
      data-slot="textarea"
      className={cn(
        "border-border placeholder:text-muted-foreground flex field-sizing-content min-h-16 w-full rounded-md border bg-transparent px-3 py-2 text-base shadow-sm transition-all disabled:cursor-not-allowed disabled:opacity-50 md:text-sm",
        "hover:border-muted-foreground/50",
        "focus-visible:border-primary focus-visible:ring-4 focus-visible:ring-primary/10 focus-visible:outline-none",
        className,
      )}
      {...props}
    />
  );
});
Textarea.displayName = "Textarea";

export { Textarea };
