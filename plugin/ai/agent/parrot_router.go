package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/store"
)

// ParrotRouter routes user requests to the appropriate parrot agent.
// ParrotRouter 将用户请求路由到相应的鹦鹉代理。
type ParrotRouter struct {
	agents map[string]ParrotAgent
	llm    ai.LLMService
	store  *store.Store
	mu     sync.RWMutex
}

// NewParrotRouter creates a new parrot router.
// NewParrotRouter 创建一个新的鹦鹉路由器。
func NewParrotRouter(
	llm ai.LLMService,
	store *store.Store,
) (*ParrotRouter, error) {
	if llm == nil {
		return nil, fmt.Errorf("llm service cannot be nil")
	}
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	router := &ParrotRouter{
		agents: make(map[string]ParrotAgent),
		llm:    llm,
		store:  store,
	}

	return router, nil
}

// Register registers a parrot agent with the router.
// Register 向路由器注册鹦鹉代理。
func (r *ParrotRouter) Register(agentType string, agent ParrotAgent) error {
	if agentType == "" {
		return fmt.Errorf("agent type cannot be empty")
	}
	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agentType]; exists {
		return fmt.Errorf("agent %s already registered", agentType)
	}

	r.agents[agentType] = agent
	return nil
}

// Get retrieves a registered agent by type.
// Get 按类型检索已注册的代理。
func (r *ParrotRouter) Get(agentType string) (ParrotAgent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentType]
	return agent, exists
}

// Route routes a request to the appropriate parrot agent.
// Route 将请求路由到相应的鹦鹉代理。
func (r *ParrotRouter) Route(
	ctx context.Context,
	agentType string,
	userInput string,
	history []string,
	stream ParrotStream,
) error {
	// Validate input
	if userInput == "" {
		return fmt.Errorf("user input cannot be empty")
	}

	// Get agent
	agent, exists := r.Get(agentType)
	if !exists {
		return fmt.Errorf("unknown agent type: %s", agentType)
	}

	// Create callback wrapper
	callback := func(eventType string, eventData interface{}) error {
		return stream.Send(eventType, eventData)
	}

	// Execute agent
	if err := agent.ExecuteWithCallback(ctx, userInput, history, callback); err != nil {
		return NewParrotError(agent.Name(), "ExecuteWithCallback", err)
	}

	return nil
}

// ListAgents returns all registered agent types.
// ListAgents 返回所有已注册的代理类型。
func (r *ParrotRouter) ListAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]string, 0, len(r.agents))
	for agentType := range r.agents {
		agents = append(agents, agentType)
	}
	return agents
}

// Count returns the number of registered agents.
// Count 返回已注册代理的数量。
func (r *ParrotRouter) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// AgentInfo represents information about a registered agent.
// AgentInfo 表示已注册代理的信息。
type AgentInfo struct {
	Type string
	Name string
}

// ListAgentInfo returns information about all registered agents.
// ListAgentInfo 返回所有已注册代理的信息。
func (r *ParrotRouter) ListAgentInfo() []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]AgentInfo, 0, len(r.agents))
	for _, agent := range r.agents {
		infos = append(infos, AgentInfo{
			Type: agent.Name(), // Use agent name as type
			Name: agent.Name(),
		})
	}
	return infos
}

// RouteDecision represents a routing decision.
// RouteDecision 表示路由决策。
type RouteDecision struct {
	AgentType  string
	Confidence float64
	Reasoning  string
}

// AutoRoute automatically determines the best agent for the given input.
// AutoRoute 自动确定给定输入的最佳代理。
//
// This is a simple heuristic-based router. In the future, this can be enhanced
// with ML-based routing or more sophisticated intent detection.
func (r *ParrotRouter) AutoRoute(ctx context.Context, userInput string) (*RouteDecision, error) {
	// Simple keyword-based routing
	// TODO: Enhance with ML-based intent detection

	// Check for schedule-related keywords
	scheduleKeywords := []string{
		"日程", "安排", "会议", "提醒", "时间表",
		"schedule", "meeting", "reminder", "calendar",
	}

	// Check for memo-related keywords
	memoKeywords := []string{
		"笔记", "记录", "搜索", "查找",
		"memo", "note", "search", "find",
	}

	// Count keyword matches
	scheduleScore := countKeywords(userInput, scheduleKeywords)
	memoScore := countKeywords(userInput, memoKeywords)

	// Make decision
	if scheduleScore > memoScore {
		return &RouteDecision{
			AgentType:  "schedule",
			Confidence: min(0.9, 0.6+float64(scheduleScore)*0.1),
			Reasoning:  "Detected schedule-related keywords",
		}, nil
	}

	if memoScore > 0 {
		return &RouteDecision{
			AgentType:  "memo",
			Confidence: min(0.9, 0.6+float64(memoScore)*0.1),
			Reasoning:  "Detected memo-related keywords",
		}, nil
	}

	// Default to memo agent for general queries
	return &RouteDecision{
		AgentType:  "memo",
		Confidence: 0.5,
		Reasoning:  "Default agent for general queries",
	}, nil
}

// countKeywords counts how many keywords appear in the input.
func countKeywords(input string, keywords []string) int {
	count := 0
	inputLower := strings.ToLower(input)
	for _, keyword := range keywords {
		if strings.Contains(inputLower, strings.ToLower(keyword)) {
			count++
		}
	}
	return count
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
