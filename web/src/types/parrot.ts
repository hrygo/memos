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
  },
  [ParrotAgentType.MEMO]: {
    id: ParrotAgentType.MEMO,
    name: "memo",
    icon: "🦜",
    displayName: "灰灰",
    description: "笔记助手，专注于检索、总结和管理笔记",
    color: "blue",
    available: true, // Milestone 1
  },
  [ParrotAgentType.SCHEDULE]: {
    id: ParrotAgentType.SCHEDULE,
    name: "schedule",
    icon: "🦜",
    displayName: "金刚",
    description: "日程助手，帮助创建、查询和管理日程",
    color: "purple",
    available: true, // Milestone 1
  },
  [ParrotAgentType.AMAZING]: {
    id: ParrotAgentType.AMAZING,
    name: "amazing",
    icon: "🦜",
    displayName: "惊奇",
    description: "综合助手，结合笔记和日程功能（Milestone 2）",
    color: "orange",
    available: false, // Milestone 2
  },
  [ParrotAgentType.CREATIVE]: {
    id: ParrotAgentType.CREATIVE,
    name: "creative",
    icon: "🦜",
    displayName: "灵灵",
    description: "创意助手，提供创意写作和头脑风暴（Milestone 4）",
    color: "pink",
    available: false, // Milestone 4
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
