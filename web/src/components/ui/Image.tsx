import * as React from "react";
import { cn } from "@/lib/utils";

export interface ImageProps extends React.ImgHTMLAttributes<HTMLImageElement> {
  src: string;
  alt: string;
  /**
   * Makes the image lazy-loaded (default: true)
   */
  lazy?: boolean;
  /**
   * Optional width hint for responsive images
   */
  widthHint?: number;
  /**
   * Optional height hint for responsive images
   */
  heightHint?: number;
}

/**
 * Optimized Image component with performance enhancements:
 * - Lazy loading by default (loading="lazy")
 * - Async decoding for non-blocking rendering
 * - Responsive sizing hints
 */
export const Image = React.forwardRef<HTMLImageElement, ImageProps>(
  ({ className, src, alt, lazy = true, widthHint, heightHint, ...props }, ref) => {
    const [isLoaded, setIsLoaded] = React.useState(false);
    const [_hasError, setHasError] = React.useState(false);

    return (
      <img
        ref={ref}
        src={src}
        alt={alt}
        loading={lazy ? "lazy" : "eager"}
        decoding="async"
        className={cn(
          "transition-opacity duration-200",
          isLoaded ? "opacity-100" : "opacity-0",
          className,
        )}
        onLoad={() => setIsLoaded(true)}
        onError={() => {
          setIsLoaded(true);
          setHasError(true);
        }}
        style={{
          width: widthHint ? `${widthHint}px` : undefined,
          height: heightHint ? `${heightHint}px` : undefined,
          ...props.style,
        }}
        {...props}
      />
    );
  },
);

Image.displayName = "Image";

export default Image;
