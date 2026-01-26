import * as DialogPrimitive from "@radix-ui/react-dialog";
import { cva, type VariantProps } from "class-variance-authority";
import { XIcon } from "lucide-react";
import * as React from "react";
import { cn } from "@/lib/utils";

const Dialog = DialogPrimitive.Root;

const DialogTrigger = DialogPrimitive.Trigger;

const DialogPortal = DialogPrimitive.Portal;

const DialogClose = DialogPrimitive.Close;

const DialogOverlay = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Overlay>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Overlay>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Overlay
    ref={ref}
    data-slot="dialog-overlay"
    className={cn(
      "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 fixed inset-0 z-50 bg-black/50",
      className,
    )}
    {...props}
  />
));
DialogOverlay.displayName = DialogPrimitive.Overlay.displayName;

/**
 * Dialog 尺寸系统 - Tailwind CSS 4 兼容设计
 *
 * 定位策略：使用 flexbox 居中（避免 transform 兼容问题）
 *
 * 重要：Tailwind CSS 4 中 max-w-sm/md/lg 使用 --spacing-* 变量
 * 而不是传统的容器宽度，所以这里使用明确的 rem 值。
 *
 * 传统 Tailwind 3 宽度对照：
 * - sm: 24rem (384px)
 * - md: 28rem (448px)
 * - lg: 32rem (512px)
 * - xl: 36rem (576px)
 * - 2xl: 42rem (672px)
 */
const dialogContentVariants = cva(
  [
    // 外观
    "bg-background rounded-lg border shadow-lg",
    // 内部布局
    "flex flex-col p-6",
    // 宽度：使用百分比确保移动端有边距
    "w-full mx-4",
    // 高度限制
    "max-h-[85vh]",
    // 动画
    "data-[state=open]:animate-in data-[state=closed]:animate-out",
    "data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
    "data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
    "duration-200",
  ].join(" "),
  {
    variants: {
      size: {
        // 使用明确的 rem 值，避免 Tailwind 4 的 --spacing-* 问题
        sm: "max-w-[24rem]",
        md: "max-w-[28rem]",
        default: "max-w-[32rem]",
        lg: "max-w-[32rem]",
        xl: "max-w-[36rem]",
        "2xl": "max-w-[42rem]",
        full: "max-w-none mx-0",
      },
    },
    defaultVariants: {
      size: "default",
    },
  },
);

const DialogContent = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content> &
    VariantProps<typeof dialogContentVariants> & {
      showCloseButton?: boolean;
    }
>(({ className, children, showCloseButton = true, size, ...props }, ref) => (
  <DialogPortal>
    <DialogOverlay />
    {/* 居中容器：使用 flexbox 居中，避免 transform 兼容问题 */}
    <div className="fixed inset-0 z-50 flex items-center justify-center pointer-events-none">
      <DialogPrimitive.Content
        ref={ref}
        className={cn("pointer-events-auto relative", dialogContentVariants({ size }), className)}
        onCloseAutoFocus={(e) => {
          e.preventDefault();
          document.body.style.pointerEvents = "auto";
        }}
        {...props}
      >
        <div className="w-full overflow-y-auto overflow-x-hidden flex flex-col gap-4">{children}</div>
        {showCloseButton && (
          <DialogPrimitive.Close className="absolute top-4 right-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none data-[state=open]:bg-accent data-[state=open]:text-muted-foreground">
            <XIcon className="h-4 w-4" />
            <span className="sr-only">Close</span>
          </DialogPrimitive.Close>
        )}
      </DialogPrimitive.Content>
    </div>
  </DialogPortal>
));
DialogContent.displayName = DialogPrimitive.Content.displayName;

const DialogHeader = React.forwardRef<React.ElementRef<"div">, React.ComponentPropsWithoutRef<"div">>(({ className, ...props }, ref) => (
  <div ref={ref} className={cn("flex flex-col gap-2 text-center sm:text-left", className)} {...props} />
));
DialogHeader.displayName = "DialogHeader";

const DialogFooter = React.forwardRef<React.ElementRef<"div">, React.ComponentPropsWithoutRef<"div">>(({ className, ...props }, ref) => (
  <div ref={ref} className={cn("flex flex-col-reverse gap-2 sm:flex-row sm:justify-end", className)} {...props} />
));
DialogFooter.displayName = "DialogFooter";

const DialogTitle = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Title>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Title>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Title ref={ref} className={cn("text-lg font-semibold leading-none tracking-tight", className)} {...props} />
));
DialogTitle.displayName = DialogPrimitive.Title.displayName;

const DialogDescription = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Description>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Description>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Description ref={ref} className={cn("text-sm text-muted-foreground", className)} {...props} />
));
DialogDescription.displayName = DialogPrimitive.Description.displayName;

export {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
};
