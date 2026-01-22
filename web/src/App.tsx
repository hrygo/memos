import { useEffect } from "react";
import { Outlet } from "react-router-dom";
import { useInstance } from "./contexts/InstanceContext";
import { MemoFilterProvider } from "./contexts/MemoFilterContext";
import useNavigateTo from "./hooks/useNavigateTo";
import { useUserLocale } from "./hooks/useUserLocale";
import { useUserTheme } from "./hooks/useUserTheme";
import { cleanupExpiredOAuthState } from "./utils/oauth";

// Security: Basic validation for custom scripts to prevent obvious XSS patterns
// Note: This is a defense-in-depth measure; instance owner is trusted.
const validateScript = (script: string): boolean => {
  // Block dangerous patterns
  const dangerousPatterns = [
    /<script/i,
    /javascript:/i,
    /on\w+\s*=/i, // Event handlers like onclick=
    /<iframe/i,
    /<object/i,
    /<embed/i,
    /document\.cookie/i,
    /document\.write/i,
    /eval\s*\(/i,
    /new\s+Function/i,
  ];
  return !dangerousPatterns.some((pattern) => pattern.test(script));
};

const App = () => {
  const navigateTo = useNavigateTo();
  const { profile: instanceProfile, generalSetting: instanceGeneralSetting } = useInstance();

  // Apply user preferences reactively
  useUserLocale();
  useUserTheme();

  // Clean up expired OAuth states on app initialization
  useEffect(() => {
    cleanupExpiredOAuthState();
  }, []);

  // Redirect to sign up page if no instance owner
  useEffect(() => {
    if (!instanceProfile.owner) {
      navigateTo("/auth/signup");
    }
  }, [instanceProfile.owner, navigateTo]);

  useEffect(() => {
    if (instanceGeneralSetting.additionalStyle) {
      // Security: Custom styles are from trusted instance owner.
      // For production, consider adding CSP nonce or sandbox restrictions.
      const styleEl = document.createElement("style");
      styleEl.innerHTML = instanceGeneralSetting.additionalStyle;
      styleEl.setAttribute("type", "text/css");
      document.body.insertAdjacentElement("beforeend", styleEl);
    }
  }, [instanceGeneralSetting.additionalStyle]);

  useEffect(() => {
    if (instanceGeneralSetting.additionalScript) {
      // Security: Validate custom scripts before injection
      if (!validateScript(instanceGeneralSetting.additionalScript)) {
        console.warn("[Security] Blocked potentially dangerous custom script");
        return;
      }
      const scriptEl = document.createElement("script");
      scriptEl.innerHTML = instanceGeneralSetting.additionalScript;
      document.head.appendChild(scriptEl);
    }
  }, [instanceGeneralSetting.additionalScript]);

  // Dynamic update metadata with customized profile
  useEffect(() => {
    if (!instanceGeneralSetting.customProfile) {
      return;
    }

    document.title = instanceGeneralSetting.customProfile.title;
    const link = document.querySelector("link[rel~='icon']") as HTMLLinkElement;
    link.href = instanceGeneralSetting.customProfile.logoUrl || "/logo.webp";
  }, [instanceGeneralSetting.customProfile]);

  return (
    <MemoFilterProvider>
      <Outlet />
    </MemoFilterProvider>
  );
};

export default App;
