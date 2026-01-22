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
	_, err := p.agent.ExecuteWithCallback(ctx, userInput, adaptedCallback)
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
	callback func(event string, data string),
) (string, error) {
	return p.agent.ExecuteWithCallback(ctx, userInput, callback)
}
