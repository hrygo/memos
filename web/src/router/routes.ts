export const ROUTES = {
  ROOT: "/",
  HOME: "/home",
  ATTACHMENTS: "/attachments",
  INBOX: "/inbox",
  ARCHIVED: "/archived",
  SETTING: "/setting",
  EXPLORE: "/explore",
  AUTH: "/auth",
  CHAT: "/chat",
  SCHEDULE: "/schedule",
  REVIEW: "/review",
} as const;

export type RouteKey = keyof typeof ROUTES;
export type RoutePath = (typeof ROUTES)[RouteKey];
