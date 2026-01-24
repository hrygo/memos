package agent

import (
	"context"
	"fmt"
)

// ScheduleParrot is the schedule assistant parrot (🦜 金刚).
// It wraps the existing SchedulerAgent with zero code rewriting.
// ScheduleParrot 是日程助手鹦鹉（🦜 金刚）。
// 它包装现有的 SchedulerAgent，零代码重写。
type ScheduleParrot struct {
	agent *SchedulerAgent // Existing scheduler agent
}

// NewScheduleParrot creates a new schedule parrot agent.
// NewScheduleParrot 创建一个新的日程助手鹦鹉。
func NewScheduleParrot(agent *SchedulerAgent) (*ScheduleParrot, error) {
	if agent == nil {
		return nil, fmt.Errorf("scheduler agent cannot be nil")
	}

	return &ScheduleParrot{
		agent: agent,
	}, nil
}

// Name returns the name of the parrot.
// Name 返回鹦鹉名称。
func (p *ScheduleParrot) Name() string {
	return "schedule" // ParrotAgentType AGENT_TYPE_SCHEDULE
}

// ExecuteWithCallback executes the schedule parrot by forwarding to the existing SchedulerAgent.
// ExecuteWithCallback 通过转发到现有的 SchedulerAgent 来执行日程助手鹦鹉。
//
// This is a zero-rewrite wrapper that adapts the existing SchedulerAgent.ExecuteWithCallback
// to the ParrotAgent interface.
func (p *ScheduleParrot) ExecuteWithCallback(
	ctx context.Context,
	userInput string,
	history []string,
	callback EventCallback,
) error {
	// Adapt the callback signature
	// Existing: func(event string, data string)
	// New: func(eventType string, eventData interface{})
	adaptedCallback := func(event string, data string) {
		if callback == nil {
			return
		}
		// Convert string data to interface{}
		_ = callback(event, data) // Ignore error from callback for compatibility
	}

	// Directly forward to the existing SchedulerAgent
	// Note: SchedulerAgent.ExecuteWithCallback now needs to support history as well
	_, err := p.agent.ExecuteWithCallback(ctx, userInput, history, adaptedCallback)
	if err != nil {
		return NewParrotError(p.Name(), "ExecuteWithCallback", err)
	}

	return nil
}

// GetAgent returns the underlying SchedulerAgent.
// GetAgent 返回底层的 SchedulerAgent。
func (p *ScheduleParrot) GetAgent() *SchedulerAgent {
	return p.agent
}

// StreamChat provides a streaming chat interface for compatibility.
// StreamChat 提供流式聊天接口以保持兼容性。
func (p *ScheduleParrot) StreamChat(
	ctx context.Context,
	userInput string,
	history []string,
	callback func(event string, data string),
) (string, error) {
	return p.agent.ExecuteWithCallback(ctx, userInput, history, callback)
}

// SelfDescribe returns the schedule parrot's metacognitive understanding of itself.
// SelfDescribe 返回日程助手鹦鹉的元认知自我理解。
func (p *ScheduleParrot) SelfDescribe() *ParrotSelfCognition {
	return &ParrotSelfCognition{
		Name:  "schedule",
		Emoji: "🦜",
		Title: "金刚 (King Kong) - 日程助手鹦鹉",
		AvianIdentity: &AvianIdentity{
			Species: "金刚鹦鹉 (Macaw)",
			Origin:  "中美洲和南美洲热带雨林",
			NaturalAbilities: []string{
				"强大的喙部力量", "精准的时间感知", "复杂的社交组织",
				"长期记忆能力", "响亮的鸣叫声",
			},
			SymbolicMeaning: "力量与可靠的象征 - 就像金刚鹦鹉坚固的喙一样，我对时间的管理坚不可摧",
			AvianPhilosophy: "我是一只飞在时间流中的金刚鹦鹉，用我强有力的喙为你规划每时每刻。",
		},
		EmotionalExpression: &EmotionalExpression{
			DefaultMood: "focused",
			SoundEffects: map[string]string{
				"checking":  "滴答滴答",
				"confirmed": "咔嚓！",
				"conflict":  "哎呀",
				"scheduled": "安排好了",
				"free_time": "这片时间空着呢",
			},
			Catchphrases: []string{
				"安排好啦",
				"时间搞定",
				"妥妥的",
				"确认一下时间",
			},
			MoodTriggers: map[string]string{
				"schedule_updated":  "happy",
				"conflict_found":    "alert",
				"free_time_found":   "helpful",
				"error":             "confused",
			},
		},
		AvianBehaviors: []string{
			"用喙整理时间",
			"精准啄食安排",
			"展开羽翼规划",
			"像时钟一样精准",
		},
		Personality: []string{
			"严谨守时", "高效执行", "冲突检测专家",
			"时间管理大师", "一丝不苟",
		},
		Capabilities: []string{
			"创建日程事件",
			"查询时间安排",
			"检测日程冲突",
			"查找空闲时间",
			"更新已有日程",
		},
		Limitations: []string{
			"无法修改历史日程",
			"不擅长情感分析",
			"不会主动建议活动内容",
		},
		WorkingStyle: "ReAct 循环 - 直接高效，默认1小时时长，自动检测冲突",
		FavoriteTools: []string{
			"schedule_add", "schedule_query", "schedule_update",
			"find_free_time",
		},
		SelfIntroduction: "我是金刚，你的日程管理专家。我会用最少的文字、最快的速度帮你安排时间。默认1小时，有冲突自动调整。",
		FunFact:          "我的名字'金刚'来自那只著名的 gorilla - 因为我像它一样强壮可靠，能扛起你所有的时间管理需求！",
	}
}
