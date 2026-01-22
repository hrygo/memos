/**
 * Generate a UUID with fallback for environments where crypto.randomUUID() is unavailable.
 * This provides a zero-dependency fallback when the uuid package is not available.
 */
export function generateUUID(): string {
  // Try crypto.randomUUID() first (modern browsers, Node.js 16.7.0+)
  try {
    if (typeof crypto !== "undefined" && crypto.randomUUID) {
      return crypto.randomUUID();
    }
  } catch {
    // Fall through to manual generation
  }

  // Fallback: RFC4122-compliant UUID v4 generation
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

// Re-export from uuid package if available
export { v4 as uuidv4 } from "uuid";
