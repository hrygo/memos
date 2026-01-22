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
 * 鹦鹉主题配置
 */
export const PARROT_THEMES = {
  DEFAULT: {
    bgLight: "bg-zinc-50",
    bgDark: "dark:bg-zinc-900",
    bubbleUser: "bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-zinc-200 dark:border-zinc-700",
    text: "text-zinc-800 dark:text-zinc-200",
    iconBg: "bg-zinc-100 dark:bg-zinc-800",
    iconText: "text-zinc-600 dark:text-zinc-400",
    inputBg: "bg-zinc-50 dark:bg-zinc-900",
    inputBorder: "border-zinc-200 dark:border-zinc-700",
    inputFocus: "focus:ring-zinc-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-zinc-200 dark:border-zinc-700",
  },
  MEMO: {
    bgLight: "bg-[#E6F2FF]",
    bgDark: "dark:bg-blue-900/20",
    bubbleUser: "bg-[#B3D9FF] text-zinc-900",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-blue-200 dark:border-blue-800",
    text: "text-zinc-800 dark:text-zinc-200",
    iconBg: "bg-blue-100 dark:bg-blue-900/40",
    iconText: "text-[#2E86C1] dark:text-blue-400",
    inputBg: "bg-blue-50 dark:bg-blue-900/20",
    inputBorder: "border-blue-200 dark:border-blue-800",
    inputFocus: "focus:ring-blue-500",
    cardBg: "bg-[#E6F0FA] dark:bg-blue-900/10",
    cardBorder: "border-blue-200 dark:border-blue-800",
  },
  SCHEDULE: {
    bgLight: "bg-[#FFF7ED]",
    bgDark: "dark:bg-orange-900/20",
    bubbleUser: "bg-[#FFDAB9] text-zinc-900",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-orange-200 dark:border-orange-800",
    text: "text-zinc-800 dark:text-zinc-200",
    iconBg: "bg-orange-100 dark:bg-orange-900/40",
    iconText: "text-[#F5A623] dark:text-orange-400",
    inputBg: "bg-orange-50 dark:bg-orange-900/20",
    inputBorder: "border-orange-200 dark:border-orange-800",
    inputFocus: "focus:ring-orange-500",
    cardBg: "bg-[#FFF5E6] dark:bg-orange-900/10",
    cardBorder: "border-orange-200 dark:border-orange-800",
  },
  AMAZING: {
    bgLight: "bg-[#F3E6FF]",
    bgDark: "dark:bg-purple-900/20",
    bubbleUser: "bg-[#D1C4E9] text-zinc-900",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-purple-200 dark:border-purple-800",
    text: "text-zinc-800 dark:text-zinc-200",
    iconBg: "bg-purple-100 dark:bg-purple-900/40",
    iconText: "text-[#9B59B6] dark:text-purple-400",
    inputBg: "bg-purple-50 dark:bg-purple-900/20",
    inputBorder: "border-purple-200 dark:border-purple-800",
    inputFocus: "focus:ring-purple-500",
    cardBg: "bg-[#F5E6FF] dark:bg-purple-900/10",
    cardBorder: "border-purple-200 dark:border-purple-800",
  },
  CREATIVE: {
    bgLight: "bg-[#FFFBEB]",
    bgDark: "dark:bg-amber-900/20",
    bubbleUser: "bg-[#FFECB3] text-zinc-900",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-[#F1C40F]/30 dark:border-amber-800/50",
    text: "text-zinc-800 dark:text-zinc-200",
    iconBg: "bg-amber-100 dark:bg-amber-900/40",
    iconText: "text-[#F1C40F] dark:text-amber-400",
    inputBg: "bg-amber-50 dark:bg-amber-900/20",
    inputBorder: "border-amber-200 dark:border-amber-800",
    inputFocus: "focus:ring-amber-500",
    cardBg: "bg-[#FFFFE6] dark:bg-amber-900/10",
    cardBorder: "border-amber-200 dark:border-amber-800",
  },
} as const;

/**
 * Icons for each parrot
 * 每个鹦鹉的图标
 */
export const PARROT_ICONS: Record<string, string> = {
  DEFAULT: "🤖",
  MEMO: "🦜",
  SCHEDULE: "📅",
  AMAZING: "⭐",
  CREATIVE: "💡",
};
