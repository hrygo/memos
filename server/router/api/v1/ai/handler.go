package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hrygo/divinesense/plugin/ai"
	agentpkg "github.com/hrygo/divinesense/plugin/ai/agent"
	"github.com/hrygo/divinesense/plugin/ai/router"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/internal/errors"
	"github.com/hrygo/divinesense/server/internal/observability"
)

// ChatStream represents the streaming response interface for AI chat.
type ChatStream interface {
	Send(*v1pb.ChatResponse) error
	Context() context.Context
}

// ParrotHandler handles all parrot agent requests (DEFAULT, MEMO, SCHEDULE, AMAZING, CREATIVE).
type ParrotHandler struct {
	factory    *AgentFactory
	llm        ai.LLMService
	chatRouter *agentpkg.ChatRouter
}

// NewParrotHandler creates a new parrot handler.
func NewParrotHandler(factory *AgentFactory, llm ai.LLMService) *ParrotHandler {
	return &ParrotHandler{
		factory: factory,
		llm:     llm,
	}
}

// SetChatRouter configures the intelligent chat router for auto-routing.
func (h *ParrotHandler) SetChatRouter(router *agentpkg.ChatRouter) {
	h.chatRouter = router
}

// Handle implements Handler interface for parrot agent requests.
func (h *ParrotHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	if h.llm == nil {
		return status.Error(codes.Unavailable, "LLM service is not available")
	}

	// Auto-route if AgentType is AUTO
	agentType := req.AgentType
	if agentType == AgentTypeAuto && h.chatRouter != nil {
		// Add user ID to context for history matching.
		// Note: req.UserID is already authenticated by the gRPC interceptor middleware.
		ctx = router.WithUserID(ctx, req.UserID)
		routeResult, err := h.chatRouter.Route(ctx, req.Message)
		if err != nil {
			slog.Warn("chat router failed, defaulting to amazing",
				"error", err,
				"message", req.Message[:min(len(req.Message), 30)])
			agentType = AgentTypeAmazing
		} else {
			// Map ChatRouteType to AgentType
			switch routeResult.Route {
			case agentpkg.RouteTypeMemo:
				agentType = AgentTypeMemo
			case agentpkg.RouteTypeSchedule:
				agentType = AgentTypeSchedule
			default:
				agentType = AgentTypeAmazing
			}
			slog.Info("chat auto-routed",
				"route", routeResult.Route,
				"method", routeResult.Method,
				"confidence", routeResult.Confidence)
		}
	} else if agentType == AgentTypeAuto {
		// No router configured, fallback to amazing
		agentType = AgentTypeAmazing
	}

	// Create logger for this request
	logger := observability.NewRequestContext(slog.Default(), agentType.String(), req.UserID)
	logger.Info("AI chat started (parrot agent)",
		slog.Int(observability.LogFieldMessageLen, len(req.Message)),
		slog.Int("history_count", len(req.History)),
	)

	// Create agent using factory
	agent, err := h.factory.Create(ctx, &CreateConfig{
		Type:     req.AgentType,
		UserID:   req.UserID,
		Timezone: req.Timezone,
	})
	if err != nil {
		logger.Error("Failed to create agent", err)
		return status.Error(codes.Internal, fmt.Sprintf("failed to create agent: %v", err))
	}

	logger.Debug("Agent created",
		slog.String("agent_name", agent.Name()),
	)

	// Execute agent with streaming
	if err := h.executeAgent(ctx, agent, req, stream, logger); err != nil {
		logger.Error("AI chat failed", err)
		return status.Error(codes.Internal, fmt.Sprintf("agent execution failed: %v", err))
	}

	logger.Info("AI chat completed",
		slog.Int64(observability.LogFieldDuration, logger.DurationMs()),
	)

	return nil
}

// executeAgent executes the agent and streams responses.
func (h *ParrotHandler) executeAgent(
	ctx context.Context,
	agent agentpkg.ParrotAgent,
	req *ChatRequest,
	stream ChatStream,
	logger *observability.RequestContext,
) error {
	// Track events for logging
	eventCount := make(map[string]int)
	var totalChunks int
	var streamMu sync.Mutex

	// Create stream adapter
	streamAdapter := agentpkg.NewParrotStreamAdapter(func(eventType string, eventData any) error {
		// Track events
		eventCount[eventType]++
		if eventType == "answer" || eventType == "content" {
			totalChunks++
		}

		// Log important events
		logger.Debug("Agent event",
			slog.String(observability.LogFieldEventType, eventType),
			slog.Int("event_count", eventCount[eventType]),
		)

		// Convert event data to string for streaming
		var dataStr string
		switch v := eventData.(type) {
		case string:
			dataStr = v
		case error:
			dataStr = v.Error()
		default:
			// Use fmt.Sprintf for other types
			dataStr = fmt.Sprintf("%v", v)
		}

		// Thread-safe send
		streamMu.Lock()
		defer streamMu.Unlock()

		return stream.Send(&v1pb.ChatResponse{
			EventType: eventType,
			EventData: dataStr,
		})
	})

	// Create callback wrapper
	callback := func(eventType string, eventData any) error {
		return streamAdapter.Send(eventType, eventData)
	}

	// Execute agent
	if err := agent.ExecuteWithCallback(ctx, req.Message, req.History, callback); err != nil {
		return err
	}

	// Send done marker
	streamMu.Lock()
	defer streamMu.Unlock()
	if err := stream.Send(&v1pb.ChatResponse{
		Done: true,
	}); err != nil {
		return err

	}

	logger.Debug("Agent execution completed",
		slog.Int("total_chunks", totalChunks),
		slog.Int("unique_events", len(eventCount)),
	)

	return nil
}

// RoutingHandler routes all agent requests through the parrot handler.
// All agent types (including DEFAULT) are now implemented as standard parrots.
type RoutingHandler struct {
	parrotHandler *ParrotHandler
}

// NewRoutingHandler creates a new routing handler.
func NewRoutingHandler(parrot *ParrotHandler) *RoutingHandler {
	return &RoutingHandler{
		parrotHandler: parrot,
	}
}

// Handle implements Handler interface by routing to the appropriate handler.
func (h *RoutingHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	// All agent types (including DEFAULT) now use parrot handler
	// DEFAULT parrot (羽飞/Navi) is implemented as a standard parrot with pure LLM mode
	return h.parrotHandler.Handle(ctx, req, stream)
}

// ToChatRequest converts a protobuf request to an internal ChatRequest.
func ToChatRequest(pbReq *v1pb.ChatRequest) *ChatRequest {
	return &ChatRequest{
		Message:            pbReq.Message,
		History:            pbReq.History,
		AgentType:          AgentTypeFromProto(pbReq.AgentType),
		Timezone:           pbReq.UserTimezone,
		ConversationID:     pbReq.ConversationId,
		IsTempConversation: pbReq.IsTempConversation,
	}
}

// HandleError converts an error to an appropriate gRPC status error.
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	// If it's already a gRPC status error, return as-is
	if _, ok := status.FromError(err); ok {
		return err
	}

	// If it's an AIError, convert it
	if aiErr, ok := err.(*errors.AIError); ok {
		return FromAIError(aiErr)
	}

	// Default to internal error
	return status.Error(codes.Internal, err.Error())
}

// NewChatRouter creates a new chat router for auto-routing based on intent classification.
// Optionally accepts a router.Service for enhanced three-layer routing.
func NewChatRouter(cfg *ai.IntentClassifierConfig, routerSvc *router.Service) *agentpkg.ChatRouter {
	return agentpkg.NewChatRouter(agentpkg.ChatRouterConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
	}, routerSvc)
}
