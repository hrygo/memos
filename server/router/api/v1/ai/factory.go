package ai

import (
	"context"
	"fmt"

	agentpkg "github.com/usememos/memos/plugin/ai/agent"
	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/server/retrieval"
	"github.com/usememos/memos/server/service/schedule"
	"github.com/usememos/memos/store"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
)

// AgentType represents the type of agent to create.
type AgentType string

const (
	AgentTypeDefault  AgentType = "DEFAULT"
	AgentTypeMemo     AgentType = "MEMO"
	AgentTypeSchedule AgentType = "SCHEDULE"
	AgentTypeAmazing  AgentType = "AMAZING"
	AgentTypeCreative AgentType = "CREATIVE"
)

// String returns the string representation of the agent type.
func (t AgentType) String() string {
	return string(t)
}

// AgentTypeFromProto converts proto AgentType to internal AgentType.
func AgentTypeFromProto(protoType v1pb.AgentType) AgentType {
	switch protoType {
	case v1pb.AgentType_AGENT_TYPE_MEMO:
		return AgentTypeMemo
	case v1pb.AgentType_AGENT_TYPE_SCHEDULE:
		return AgentTypeSchedule
	case v1pb.AgentType_AGENT_TYPE_AMAZING:
		return AgentTypeAmazing
	case v1pb.AgentType_AGENT_TYPE_CREATIVE:
		return AgentTypeCreative
	default:
		return AgentTypeDefault
	}
}

// ToProto converts internal AgentType to proto AgentType.
func (t AgentType) ToProto() v1pb.AgentType {
	switch t {
	case AgentTypeMemo:
		return v1pb.AgentType_AGENT_TYPE_MEMO
	case AgentTypeSchedule:
		return v1pb.AgentType_AGENT_TYPE_SCHEDULE
	case AgentTypeAmazing:
		return v1pb.AgentType_AGENT_TYPE_AMAZING
	case AgentTypeCreative:
		return v1pb.AgentType_AGENT_TYPE_CREATIVE
	default:
		return v1pb.AgentType_AGENT_TYPE_DEFAULT
	}
}

// CreateConfig contains configuration for creating an agent.
type CreateConfig struct {
	Type     AgentType
	UserID   int32
	Timezone string
}

// AgentFactory creates parrot agents based on type.
type AgentFactory struct {
	llm        ai.LLMService
	retriever  *retrieval.AdaptiveRetriever
	store      *store.Store
}

// NewAgentFactory creates a new agent factory.
func NewAgentFactory(
	llm ai.LLMService,
	retriever *retrieval.AdaptiveRetriever,
	st *store.Store,
) *AgentFactory {
	return &AgentFactory{
		llm:       llm,
		retriever: retriever,
		store:     st,
	}
}

// Create creates an agent based on the configuration.
func (f *AgentFactory) Create(ctx context.Context, cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	if f.llm == nil {
		return nil, fmt.Errorf("llm service is required")
	}

	switch cfg.Type {
	case AgentTypeDefault:
		return f.createDefaultParrot(cfg)
	case AgentTypeMemo:
		return f.createMemoParrot(cfg)
	case AgentTypeSchedule:
		return f.createScheduleParrot(ctx, cfg)
	case AgentTypeAmazing:
		return f.createAmazingParrot(ctx, cfg)
	case AgentTypeCreative:
		return f.createCreativeParrot(cfg)
	default:
		return nil, fmt.Errorf("unknown agent type: %s", cfg.Type)
	}
}

// createMemoParrot creates a memo parrot agent.
func (f *AgentFactory) createMemoParrot(cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	if f.retriever == nil {
		return nil, fmt.Errorf("retriever is required for memo parrot")
	}

	agent, err := agentpkg.NewMemoParrot(
		f.retriever,
		f.llm,
		cfg.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create memo parrot: %w", err)
	}

	return agent, nil
}

// createScheduleParrot creates a schedule parrot agent.
func (f *AgentFactory) createScheduleParrot(_ context.Context, cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	if f.store == nil {
		return nil, fmt.Errorf("store is required for schedule parrot")
	}

	// Normalize timezone: use provided timezone or default
	timezone := NormalizeTimezone(cfg.Timezone)

	// Create schedule service
	scheduleSvc := schedule.NewService(f.store)

	// Create scheduler agent
	schedulerAgent, err := agentpkg.NewSchedulerAgent(
		f.llm,
		scheduleSvc,
		cfg.UserID,
		timezone,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler agent: %w", err)
	}

	// Wrap in schedule parrot
	parrot, err := agentpkg.NewScheduleParrot(schedulerAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule parrot: %w", err)
	}

	return parrot, nil
}

// createAmazingParrot creates an amazing parrot agent.
func (f *AgentFactory) createAmazingParrot(_ context.Context, cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	if f.retriever == nil {
		return nil, fmt.Errorf("retriever is required for amazing parrot")
	}
	if f.store == nil {
		return nil, fmt.Errorf("store is required for amazing parrot")
	}

	// Create schedule service
	scheduleSvc := schedule.NewService(f.store)

	agent, err := agentpkg.NewAmazingParrot(
		f.llm,
		f.retriever,
		scheduleSvc,
		cfg.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create amazing parrot: %w", err)
	}

	return agent, nil
}

// createCreativeParrot creates a creative parrot agent.
func (f *AgentFactory) createCreativeParrot(cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	agent, err := agentpkg.NewCreativeParrot(
		f.llm,
		cfg.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create creative parrot: %w", err)
	}

	return agent, nil
}

// createDefaultParrot creates a default parrot agent (羽飞/Navi).
func (f *AgentFactory) createDefaultParrot(cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	agent, err := agentpkg.NewDefaultParrot(
		f.llm,
		cfg.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create default parrot: %w", err)
	}

	return agent, nil
}

// IsDefaultType returns true if the agent type is DEFAULT (direct LLM).
func IsDefaultType(agentType v1pb.AgentType) bool {
	return agentType == v1pb.AgentType_AGENT_TYPE_DEFAULT
}
