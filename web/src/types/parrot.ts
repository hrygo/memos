import { AgentType } from "@/types/proto/api/v1/ai_service_pb";

/**
 * Parrot agent types enumeration
 * 鹦鹉代理类型枚举
 */
export enum ParrotAgentType {
  DEFAULT = "DEFAULT",
  MEMO = "MEMO", // 🦜 灰灰 - Memo Parrot
  SCHEDULE = "SCHEDULE", // 🦜 金刚 - Schedule Parrot
  AMAZING = "AMAZING", // 🦜 惊奇 - Amazing Parrot (Milestone 2)
  CREATIVE = "CREATIVE", // 🦜 灵灵 - Creative Parrot (Milestone 4)
}

/**
 * Default pinned agents in the sidebar
 * 侧边栏默认固定的鹦鹉代理
 */
export const PINNED_PARROT_AGENTS = [
  ParrotAgentType.DEFAULT,
  ParrotAgentType.MEMO,
  ParrotAgentType.SCHEDULE,
  ParrotAgentType.AMAZING,
  ParrotAgentType.CREATIVE,
];

/**
 * Emotional state of a parrot
 * 鹦鹉的情感状态
 */
export type EmotionalState = "focused" | "curious" | "excited" | "thoughtful" | "confused" | "happy" | "delighted" | "helpful" | "alert";

/**
 * Parrot cognition configuration from backend
 * 鹦鹉认知配置（来自后端）
 */
export interface ParrotCognition {
  emotional_expression?: {
    default_mood: EmotionalState;
    sound_effects: Record<string, string>;
    catchphrases: string[];
    mood_triggers?: Record<string, EmotionalState>;
  };
  avian_behaviors?: string[];
}

/**
 * Event to emotional state mapping for frontend inference
 * 前端推断的事件到情感状态映射
 */
export const EVENT_TO_MOOD: Record<string, EmotionalState> = {
  thinking: "focused",
  tool_use: "curious",
  memo_query_result: "excited",
  schedule_query_result: "happy",
  schedule_updated: "happy",
  error: "confused",
};

/**
 * Sound effects for each parrot by context
 * 每只鹦鹉的拟声词（按上下文）
 */
export const PARROT_SOUND_EFFECTS: Record<ParrotAgentType, Record<string, string>> = {
  [ParrotAgentType.MEMO]: {
    thinking: "嘎...",
    searching: "扑棱扑棱",
    found: "嗯嗯~",
    no_result: "咕...",
    done: "扑棱！",
  },
  [ParrotAgentType.SCHEDULE]: {
    checking: "滴答滴答",
    confirmed: "咔嚓！",
    conflict: "哎呀",
    scheduled: "安排好了",
    free_time: "这片时间空着呢",
  },
  [ParrotAgentType.CREATIVE]: {
    thinking: "啾...",
    idea: "灵感来了~",
    brainstorm: "咻咻~",
    done: "噗~搞定",
    excited: "啾啾！",
  },
  [ParrotAgentType.AMAZING]: {
    searching: "咻...",
    insight: "哇哦~",
    done: "噢！综合完成",
    analyzing: "看看这个...",
    multi_task: "同时搜索中",
  },
  [ParrotAgentType.DEFAULT]: {
    thinking: "嗯...让我想想",
    insight: "咻~有了",
    done: "✓",
    analyzing: "看看这个...",
  },
};

/**
 * Catchphrases for each parrot
 * 每只鹦鹉的口头禅
 */
export const PARROT_CATCHPHRASES: Record<ParrotAgentType, string[]> = {
  [ParrotAgentType.MEMO]: ["让我想想...", "笔记里说...", "在记忆里找找..."],
  [ParrotAgentType.SCHEDULE]: ["安排好啦", "时间搞定", "妥妥的"],
  [ParrotAgentType.CREATIVE]: ["灵感来了~", "想想还有", "有意思！"],
  [ParrotAgentType.AMAZING]: ["看看这个...", "综合来看", "发现规律了"],
  [ParrotAgentType.DEFAULT]: ["看看这个...", "综合来看", "发现规律了"],
};

/**
 * Avian behaviors for each parrot
 * 每只鹦鹉的鸟类行为描述
 */
export const PARROT_BEHAVIORS: Record<ParrotAgentType, string[]> = {
  [ParrotAgentType.MEMO]: ["用翅膀翻找笔记", "在记忆森林中飞翔", "用喙精准啄取信息"],
  [ParrotAgentType.SCHEDULE]: ["用喙整理时间", "精准啄食安排", "展开羽翼规划"],
  [ParrotAgentType.CREATIVE]: ["羽毛变色", "思维跳跃", "自由飞翔想象"],
  [ParrotAgentType.AMAZING]: ["在数据树丛中穿梭", "多维飞行", "综合视野"],
  [ParrotAgentType.DEFAULT]: ["展开羽翼导航", "翱翔在信息天空", "用锐利的目光洞察"],
};

/**
 * Convert AgentType enum from proto to ParrotAgentType
 * 将 proto 的 AgentType 枚举转换为 ParrotAgentType
 */
export function protoToParrotAgentType(agentType: AgentType): ParrotAgentType {
  switch (agentType) {
    case AgentType.MEMO:
      return ParrotAgentType.MEMO;
    case AgentType.SCHEDULE:
      return ParrotAgentType.SCHEDULE;
    case AgentType.AMAZING:
      return ParrotAgentType.AMAZING;
    case AgentType.CREATIVE:
      return ParrotAgentType.CREATIVE;
    case AgentType.DEFAULT:
    default:
      return ParrotAgentType.DEFAULT;
  }
}

/**
 * Convert ParrotAgentType to proto AgentType
 * 将 ParrotAgentType 转换为 proto AgentType
 */
export function parrotToProtoAgentType(agentType: ParrotAgentType): AgentType {
  switch (agentType) {
    case ParrotAgentType.MEMO:
      return AgentType.MEMO;
    case ParrotAgentType.SCHEDULE:
      return AgentType.SCHEDULE;
    case ParrotAgentType.AMAZING:
      return AgentType.AMAZING;
    case ParrotAgentType.CREATIVE:
      return AgentType.CREATIVE;
    case ParrotAgentType.DEFAULT:
    default:
      return AgentType.DEFAULT;
  }
}

/**
 * Parrot agent metadata
 * 鹦鹉代理元数据
 * Note: displayName, description, and examplePrompts should be localized via useParrots hook
 */
export interface ParrotAgent {
  id: ParrotAgentType;
  name: string;
  icon: string;
  displayName: string; // Default English, should be overridden by i18n
  description: string; // Default English, should be overridden by i18n
  color: string;
  available: boolean; // Whether this parrot is available in current milestone
  examplePrompts?: string[]; // Default English prompts, should be overridden by i18n
  backgroundImage?: string; // Background image for the agent card
}

/**
 * All parrot agents configuration (English defaults)
 * 所有鹦鹉代理配置（英文默认值）
 * Localized versions are provided by useParrots hook
 */
export const PARROT_AGENTS: Record<ParrotAgentType, ParrotAgent> = {
  [ParrotAgentType.DEFAULT]: {
    id: ParrotAgentType.DEFAULT,
    name: "default",
    icon: "/images/parrots/icons/navi_icon.webp",
    displayName: "Navi",
    description: "Universal Navigator directly connected to top-tier LLMs, providing boundless creative inspiration",
    color: "indigo",
    available: true,
    examplePrompts: ["Help me build a logical framework", "Draft a formal communication email"],
    backgroundImage: "/images/parrots/navi_bg.webp",
  },
  [ParrotAgentType.MEMO]: {
    id: ParrotAgentType.MEMO,
    name: "memo",
    icon: "/images/parrots/icons/memo_icon.webp",
    displayName: "Memo",
    description: "Note assistant for searching, summarizing, and managing memos",
    color: "blue",
    available: true,
    examplePrompts: ["Search for programming notes", "Summarize recent work memos", "Find project management notes"],
    backgroundImage: "/images/parrots/memo_parrot_bg.webp",
  },
  [ParrotAgentType.SCHEDULE]: {
    id: ParrotAgentType.SCHEDULE,
    name: "schedule",
    icon: "/images/parrots/icons/schedule_icon.webp",
    displayName: "Schedule",
    description: "Schedule assistant for creating, querying, and managing schedules",
    color: "orange",
    available: true,
    examplePrompts: ["What's on my schedule today", "Am I free tomorrow afternoon", "Create a meeting reminder for next week"],
    backgroundImage: "/images/parrots/schedule_bg.webp",
  },
  [ParrotAgentType.AMAZING]: {
    id: ParrotAgentType.AMAZING,
    name: "amazing",
    icon: "/images/parrots/icons/amazing_icon.webp",
    displayName: "Amazing",
    description: "Comprehensive assistant combining memo and schedule features",
    color: "purple",
    available: true,
    examplePrompts: ["Summarize today's memos and schedule", "Help me plan next week's work", "Search recent project-related content"],
    backgroundImage: "/images/parrots/amazing_bg.webp",
  },
  [ParrotAgentType.CREATIVE]: {
    id: ParrotAgentType.CREATIVE,
    name: "creative",
    icon: "/images/parrots/icons/creative_icon.webp",
    displayName: "Creative",
    description: "Creative writing assistant for brainstorming and content creation",
    color: "pink",
    available: true,
    examplePrompts: ["Brainstorm product promotion ideas", "Write a project progress email", "Improve this text expression"],
    backgroundImage: "/images/parrots/creative_bg.webp",
  },
};

/**
 * Get available parrot agents for current milestone
 * 获取当前里程碑可用的鹦鹉代理
 */
export function getAvailableParrots(): ParrotAgent[] {
  return Object.values(PARROT_AGENTS).filter((agent) => agent.available);
}

/**
 * Get parrot agent by type
 * 根据类型获取鹦鹉代理
 */
export function getParrotAgent(type: ParrotAgentType): ParrotAgent {
  return PARROT_AGENTS[type] || PARROT_AGENTS[ParrotAgentType.DEFAULT];
}

/**
 * Memo query result data
 * 笔记查询结果数据
 */
export interface MemoQueryResultData {
  memos: MemoSummary[];
  query: string;
  count: number;
}

/**
 * Memo summary
 * 笔记摘要
 */
export interface MemoSummary {
  uid: string;
  content: string;
  score: number;
}

/**
 * Schedule query result data
 * 日程查询结果数据
 */
export interface ScheduleQueryResultData {
  schedules: ScheduleSummary[];
  query: string;
  count: number;
  timeRangeDescription: string;
  queryType: string; // e.g., "upcoming", "range", "filter"
}

/**
 * Schedule summary
 * 日程摘要
 */
export interface ScheduleSummary {
  uid: string;
  title: string;
  startTimestamp: number;
  endTimestamp: number;
  allDay: boolean;
  location?: string;
  status: string;
}

/**
 * Parrot chat callbacks
 * 鹦鹉聊天回调函数
 */
export interface ParrotChatCallbacks {
  onContent?: (content: string) => void;
  onMemoQueryResult?: (result: MemoQueryResultData) => void;
  onScheduleQueryResult?: (result: ScheduleQueryResultData) => void;
  onThinking?: (message: string) => void;
  onToolUse?: (toolName: string) => void;
  onToolResult?: (result: string) => void;
  onDone?: () => void;
  onError?: (error: Error) => void;
}

/**
 * Parrot chat parameters
 * 鹦鹉聊天参数
 */
export interface ParrotChatParams {
  agentType: ParrotAgentType;
  message: string;
  conversationId?: number; // Backend will build history from this ID
  history?: string[]; // Deprecated: Kept for backward compatibility
  userTimezone?: string;
}

/**
 * Parrot event types
 * 鹦鹉事件类型
 */
export enum ParrotEventType {
  THINKING = "thinking",
  TOOL_USE = "tool_use",
  TOOL_RESULT = "tool_result",
  ANSWER = "answer",
  ERROR = "error",
  MEMO_QUERY_RESULT = "memo_query_result",
  SCHEDULE_QUERY_RESULT = "schedule_query_result",
  SCHEDULE_UPDATED = "schedule_updated",
}

/**
 * Parrot theme configuration
 * 鹦鹉主题配置 - 信息清晰优先 design
 *
 * 设计原则:
 * - 信息清晰优先于视觉效果
 * - 高对比度确保可读性
 * - 简洁干净的视觉
 * - 每个鹦鹉独立且协调的色系
 */
export const PARROT_THEMES = {
  DEFAULT: {
    // 默认助手 - 领航员 (Navi) - 靛青色 (Indigo)
    bubbleUser: "bg-indigo-600 dark:bg-indigo-500 text-white",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-indigo-200 dark:border-indigo-700",
    text: "text-slate-800 dark:text-indigo-50",
    textSecondary: "text-slate-600 dark:text-indigo-200",
    iconBg: "bg-indigo-50 dark:bg-indigo-900/30",
    iconText: "text-indigo-700 dark:text-indigo-300",
    inputBg: "bg-indigo-50/50 dark:bg-indigo-950/20",
    inputBorder: "border-indigo-200 dark:border-indigo-700",
    inputFocus: "focus:ring-indigo-500 focus:border-indigo-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-indigo-200 dark:border-indigo-700",
    accent: "bg-indigo-500",
    accentText: "text-white",
  },
  // 灰灰 - 非洲灰鹦鹉 (African Grey Parrot)
  // DNA: 银灰羽毛 + 红色点缀 (subtle)
  MEMO: {
    bubbleUser: "bg-slate-800 dark:bg-slate-300 text-white dark:text-slate-800",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-slate-200 dark:border-slate-700",
    text: "text-slate-800 dark:text-slate-100",
    textSecondary: "text-slate-600 dark:text-slate-400",
    iconBg: "bg-slate-100 dark:bg-slate-700",
    iconText: "text-slate-700 dark:text-slate-300",
    inputBg: "bg-slate-50 dark:bg-slate-900",
    inputBorder: "border-slate-200 dark:border-slate-700",
    inputFocus: "focus:ring-slate-500 focus:border-slate-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-slate-200 dark:border-slate-700",
    accent: "bg-red-500",
    accentText: "text-white",
  },
  // 金刚 - 蓝黄金刚鹦鹉 (Blue-and-yellow Macaw)
  // DNA: 蓝黄 (simplified, high contrast)
  SCHEDULE: {
    bubbleUser: "bg-cyan-600 dark:bg-cyan-500 text-white",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-cyan-200 dark:border-cyan-700",
    text: "text-slate-800 dark:text-cyan-50",
    textSecondary: "text-slate-600 dark:text-cyan-200",
    iconBg: "bg-cyan-100 dark:bg-cyan-900",
    iconText: "text-cyan-700 dark:text-cyan-300",
    inputBg: "bg-cyan-50 dark:bg-cyan-950",
    inputBorder: "border-cyan-200 dark:border-cyan-700",
    inputFocus: "focus:ring-cyan-500 focus:border-cyan-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-cyan-200 dark:border-cyan-700",
    accent: "bg-cyan-500",
    accentText: "text-white",
  },
  // 惊奇 - 亚马逊鹦鹉 (Amazon Parrot)
  // DNA: 绿色 (simplified, high contrast)
  AMAZING: {
    bubbleUser: "bg-emerald-600 dark:bg-emerald-500 text-white",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-emerald-200 dark:border-emerald-700",
    text: "text-slate-800 dark:text-emerald-50",
    textSecondary: "text-slate-600 dark:text-emerald-200",
    iconBg: "bg-emerald-100 dark:bg-emerald-900",
    iconText: "text-emerald-700 dark:text-emerald-300",
    inputBg: "bg-emerald-50 dark:bg-emerald-950",
    inputBorder: "border-emerald-200 dark:border-emerald-700",
    inputFocus: "focus:ring-emerald-500 focus:border-emerald-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-emerald-200 dark:border-emerald-700",
    accent: "bg-emerald-500",
    accentText: "text-white",
  },
  // 灵灵 - 虎皮鹦鹉 (Budgerigar)
  // DNA: 绿色 (simplified, high contrast)
  CREATIVE: {
    bubbleUser: "bg-lime-600 dark:bg-lime-500 text-white",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-lime-200 dark:border-lime-700",
    text: "text-slate-800 dark:text-lime-50",
    textSecondary: "text-slate-600 dark:text-lime-200",
    iconBg: "bg-lime-100 dark:bg-lime-900",
    iconText: "text-lime-700 dark:text-lime-300",
    inputBg: "bg-lime-50 dark:bg-lime-950",
    inputBorder: "border-lime-200 dark:border-lime-700",
    inputFocus: "focus:ring-lime-500 focus:border-lime-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-lime-200 dark:border-lime-700",
    accent: "bg-lime-500",
    accentText: "text-white",
  },
} as const;

/**
 * Icons for each parrot
 * 每个鹦鹉的图标
 */
export const PARROT_ICONS: Record<string, string> = {
  DEFAULT: "/images/parrots/icons/navi_icon.webp",
  MEMO: "/images/parrots/icons/memo_icon.webp",
  SCHEDULE: "/images/parrots/icons/schedule_icon.webp",
  AMAZING: "/images/parrots/icons/amazing_icon.webp",
  CREATIVE: "/images/parrots/icons/creative_icon.webp",
};
