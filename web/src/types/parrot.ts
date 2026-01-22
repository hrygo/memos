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
 */
export interface ParrotAgent {
  id: ParrotAgentType;
  name: string;
  icon: string;
  displayName: string;
  description: string;
  color: string;
  available: boolean; // Whether this parrot is available in current milestone
  examplePrompts?: string[]; // Suggested prompts for this parrot
  backgroundImage?: string; // Background image for the agent card
}

/**
 * All parrot agents configuration
 * 所有鹦鹉代理配置
 */
export const PARROT_AGENTS: Record<ParrotAgentType, ParrotAgent> = {
  [ParrotAgentType.DEFAULT]: {
    id: ParrotAgentType.DEFAULT,
    name: "default",
    icon: "🤖",
    displayName: "默认助手",
    description: "默认 AI 助手，使用 RAG 系统回答问题",
    color: "gray",
    available: true,
    examplePrompts: ["总结最近的笔记", "帮我搜索关于 Python 的内容", "今天有什么安排"],
  },
  [ParrotAgentType.MEMO]: {
    id: ParrotAgentType.MEMO,
    name: "memo",
    icon: "🦜",
    displayName: "灰灰",
    description: "笔记助手，专注于检索、总结和管理笔记",
    color: "blue",
    available: true,
    examplePrompts: ["搜索关于编程的笔记", "总结最近的工作备忘", "查找包含项目管理的笔记"],
    backgroundImage: "/images/parrots/memo_parrot_bg.webp",
  },
  [ParrotAgentType.SCHEDULE]: {
    id: ParrotAgentType.SCHEDULE,
    name: "schedule",
    icon: "📅",
    displayName: "金刚",
    description: "日程助手，帮助创建、查询和管理日程",
    color: "orange",
    available: true,
    examplePrompts: ["今天有什么安排", "明天下午有空吗", "帮我创建下周会议提醒"],
    backgroundImage: "/images/parrots/schedule_bg.webp",
  },
  [ParrotAgentType.AMAZING]: {
    id: ParrotAgentType.AMAZING,
    name: "amazing",
    icon: "⭐",
    displayName: "惊奇",
    description: "综合助手，结合笔记和日程功能",
    color: "purple",
    available: true,
    examplePrompts: ["总结今天的笔记和日程", "帮我规划下周工作", "查询最近的项目相关内容"],
    backgroundImage: "/images/parrots/amazing_bg.webp",
  },
  [ParrotAgentType.CREATIVE]: {
    id: ParrotAgentType.CREATIVE,
    name: "creative",
    icon: "💡",
    displayName: "灵灵",
    description: "创意助手，提供创意写作和头脑风暴",
    color: "pink",
    available: true,
    examplePrompts: ["帮我头脑风暴产品推广创意", "写一封项目进度汇报邮件", "优化这段文字的表达"],
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
  history?: string[];
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
 * 鹦鹉主题配置 - 信息清晰优先设计
 *
 * 设计原则:
 * - 信息清晰优先于视觉效果
 * - 高对比度确保可读性
 * - 简洁干净的视觉
 * - 每个鹦鹉独立且协调的色系
 */
export const PARROT_THEMES = {
  DEFAULT: {
    // 默认助手 - 中性灰
    bubbleUser: "bg-zinc-900 dark:bg-zinc-200 text-white dark:text-zinc-900",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-zinc-200 dark:border-zinc-700",
    text: "text-zinc-900 dark:text-zinc-100",
    textSecondary: "text-zinc-600 dark:text-zinc-400",
    iconBg: "bg-zinc-100 dark:bg-zinc-700",
    iconText: "text-zinc-700 dark:text-zinc-300",
    inputBg: "bg-zinc-50 dark:bg-zinc-900",
    inputBorder: "border-zinc-200 dark:border-zinc-700",
    inputFocus: "focus:ring-zinc-500 focus:border-zinc-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-zinc-200 dark:border-zinc-700",
    accent: "bg-zinc-500",
    accentText: "text-white",
  },
  // 灰灰 - 非洲灰鹦鹉 (African Grey Parrot)
  // DNA: 银灰羽毛 + 红色点缀 (subtle)
  MEMO: {
    bubbleUser: "bg-slate-700 dark:bg-slate-300 text-white dark:text-slate-900",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-slate-200 dark:border-slate-700",
    text: "text-slate-900 dark:text-slate-100",
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
    text: "text-slate-900 dark:text-cyan-50",
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
    text: "text-slate-900 dark:text-emerald-50",
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
    text: "text-slate-900 dark:text-lime-50",
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
  DEFAULT: "🤖",
  MEMO: "/images/parrots/icons/memo_icon.webp",
  SCHEDULE: "/images/parrots/icons/schedule_icon.webp",
  AMAZING: "/images/parrots/icons/amazing_icon.webp",
  CREATIVE: "/images/parrots/icons/creative_icon.webp",
};
