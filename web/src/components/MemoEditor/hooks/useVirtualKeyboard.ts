import { useEffect, useState } from "react";

/**
 * Hook to detect and track virtual keyboard height on mobile devices.
 * Uses VisualViewport API to detect when the keyboard is shown/hidden.
 *
 * @returns The current keyboard height in pixels (0 when keyboard is hidden)
 */
export const useVirtualKeyboard = () => {
  const [keyboardHeight, setKeyboardHeight] = useState(0);

  useEffect(() => {
    if (typeof window === "undefined" || !window.visualViewport) return;

    const handleResize = () => {
      const viewport = window.visualViewport;
      if (!viewport) return;

      const windowHeight = window.innerHeight;
      // Keyboard is considered visible if viewport height is less than 85% of window height
      const keyboardVisible = viewport.height < windowHeight * 0.85;
      const newKeyboardHeight = keyboardVisible ? windowHeight - viewport.height : 0;

      setKeyboardHeight(newKeyboardHeight);
    };

    window.visualViewport.addEventListener("resize", handleResize);
    // Initial check in case keyboard is already open
    handleResize();

    return () => window.visualViewport?.removeEventListener("resize", handleResize);
  }, []);

  return keyboardHeight;
};
