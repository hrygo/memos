import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { CapabilityType, capabilityToParrotAgent, IntentRecognitionResult, parrotAgentToCapability } from "@/types/capability";
import { ParrotAgentType } from "@/types/parrot";

/**
 * 意图识别关键词映射
 * 前端智能路由规则：根据用户输入关键词推断意图
 */
const INTENT_KEYWORDS: Record<CapabilityType, string[]> = {
  [CapabilityType.MEMO]: [
    "笔记",
    "memo",
    "记录",
    "搜索",
    "查找",
    "回忆",
    "总结",
    "有没有",
    "写过",
    "提到",
    "关于",
    "note",
    "search",
    "find",
    "recall",
  ],
  [CapabilityType.SCHEDULE]: [
    "日程",
    "schedule",
    "安排",
    "时间",
    "几点",
    "今天",
    "明天",
    "下午",
    "上午",
    "有空",
    "会议",
    "提醒",
    "创建",
    "添加",
    "plan",
    "calendar",
    "meeting",
    "reminder",
  ],
  [CapabilityType.AMAZING]: [
    "总结",
    "综合",
    "本周",
    "最近",
    "分析",
    "整体",
    "全部",
    "overview",
    "summary",
    "analyze",
    "本周工作",
    "今日总结",
    "周报",
  ],
  [CapabilityType.CREATIVE]: ["创意", "想法", "头脑风暴", "brainstorm", "写", "润色", "优化", "建议", "灵感", "draft", "creative", "idea"],
  [CapabilityType.AUTO]: [],
};

/**
 * 计算意图置信度
 * @param input 用户输入
 * @param capability 能力类型
 */
function calculateConfidence(input: string, capability: CapabilityType): number {
  if (capability === CapabilityType.AUTO) return 0.5;

  const keywords = INTENT_KEYWORDS[capability] || [];
  const lowerInput = input.toLowerCase();

  // 精确匹配加分
  let score = 0;
  for (const keyword of keywords) {
    if (lowerInput.includes(keyword.toLowerCase())) {
      score += 1;
    }
  }

  // 归一化到 0-1
  return Math.min(score / 3, 1);
}

/**
 * 智能意图识别
 * @param input 用户输入
 * @param currentCapability 当前能力（用于上下文）
 */
export function recognizeIntent(input: string, currentCapability: CapabilityType = CapabilityType.AUTO): IntentRecognitionResult {
  const lowerInput = input.trim().toLowerCase();

  // 空输入返回 AUTO
  if (!lowerInput) {
    return {
      capability: CapabilityType.AUTO,
      confidence: 0,
    };
  }

  // 计算每个能力的置信度
  const scores: Array<{ capability: CapabilityType; confidence: number }> = [
    {
      capability: CapabilityType.MEMO,
      confidence: calculateConfidence(input, CapabilityType.MEMO),
    },
    {
      capability: CapabilityType.SCHEDULE,
      confidence: calculateConfidence(input, CapabilityType.SCHEDULE),
    },
    {
      capability: CapabilityType.AMAZING,
      confidence: calculateConfidence(input, CapabilityType.AMAZING),
    },
    {
      capability: CapabilityType.CREATIVE,
      confidence: calculateConfidence(input, CapabilityType.CREATIVE),
    },
  ];

  // 找出最高分的能力
  const bestMatch = scores.reduce((best, current) => (current.confidence > best.confidence ? current : best));

  // 如果最高分太低，返回 AUTO
  if (bestMatch.confidence < 0.3) {
    return {
      capability: currentCapability !== CapabilityType.AUTO ? currentCapability : CapabilityType.AUTO,
      confidence: 0.3,
      reasoning: "意图不明确，使用当前能力或默认",
    };
  }

  // 特殊规则：如果同时涉及笔记和日程，使用 AMAZING
  const hasMemoKeyword = INTENT_KEYWORDS[CapabilityType.MEMO].some((k) => lowerInput.includes(k.toLowerCase()));
  const hasScheduleKeyword = INTENT_KEYWORDS[CapabilityType.SCHEDULE].some((k) => lowerInput.includes(k.toLowerCase()));

  if (hasMemoKeyword && hasScheduleKeyword) {
    return {
      capability: CapabilityType.AMAZING,
      confidence: 0.9,
      reasoning: "同时涉及笔记和日程，使用综合能力",
    };
  }

  return {
    capability: bestMatch.capability,
    confidence: bestMatch.confidence,
    reasoning: `识别到 "${bestMatch.capability}" 相关关键词`,
  };
}

/**
 * 能力路由 Hook
 * 提供智能路由和能力管理功能
 */
export function useCapabilityRouter() {
  const { t } = useTranslation();

  // 所有可用能力列表
  const availableCapabilities = useMemo(() => Object.values(CapabilityType).filter((c) => c !== CapabilityType.AUTO), []);

  /**
   * 根据用户输入路由到合适的能力
   */
  const route = (input: string, currentCapability?: CapabilityType): IntentRecognitionResult => {
    return recognizeIntent(input, currentCapability);
  };

  /**
   * 获取能力显示信息
   */
  const getCapabilityInfo = (capability: CapabilityType) => {
    switch (capability) {
      case CapabilityType.MEMO:
        return {
          name: t("ai.capability.memo.name") || "笔记",
          nameAlt: "Memo",
          description: t("ai.capability.memo.description") || "搜索与问答",
          icon: "🦜",
        };
      case CapabilityType.SCHEDULE:
        return {
          name: t("ai.capability.schedule.name") || "日程",
          nameAlt: "Schedule",
          description: t("ai.capability.schedule.description") || "规划与管理",
          icon: "⏰",
        };
      case CapabilityType.AMAZING:
        return {
          name: t("ai.capability.amazing.name") || "综合",
          nameAlt: "Amazing",
          description: t("ai.capability.amazing.description") || "笔记 + 日程",
          icon: "🌟",
        };
      case CapabilityType.CREATIVE:
        return {
          name: t("ai.capability.creative.name") || "创意",
          nameAlt: "Creative",
          description: t("ai.capability.creative.description") || "头脑风暴",
          icon: "💡",
        };
      case CapabilityType.AUTO:
      default:
        return {
          name: t("ai.capability.auto.name") || "自动",
          nameAlt: "Auto",
          description: t("ai.capability.auto.description") || "智能识别",
          icon: "🤖",
        };
    }
  };

  /**
   * 将能力转换为后台 Agent 类型
   */
  const toParrotAgent = (capability: CapabilityType): ParrotAgentType => {
    return capabilityToParrotAgent(capability);
  };

  /**
   * 将 Agent 类型转换为能力
   */
  const fromParrotAgent = (agentType: ParrotAgentType): CapabilityType => {
    return parrotAgentToCapability(agentType);
  };

  return {
    availableCapabilities,
    route,
    getCapabilityInfo,
    toParrotAgent,
    fromParrotAgent,
  };
}
