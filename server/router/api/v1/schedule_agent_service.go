package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/plugin/ai/agent"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/server/service/schedule"
	"github.com/usememos/memos/store"
)

const (
	// DefaultTimezone is the default timezone for schedule operations
	DefaultTimezone = "Asia/Shanghai"
)

// ScheduleAgentService is a dedicated service for schedule agent interactions.
type ScheduleAgentService struct {
	v1pb.UnimplementedScheduleAgentServiceServer

	Store        *store.Store
	LLM          ai.LLMService
	Profile      *profile.Profile
	ContextStore *agent.ContextStore // TODO: Persist to PostgreSQL for cross-restart context recovery
}

// NewScheduleAgentService creates a new schedule agent service.
func NewScheduleAgentService(store *store.Store, llm ai.LLMService, profile *profile.Profile) *ScheduleAgentService {
	return &ScheduleAgentService{
		Store:        store,
		LLM:          llm,
		Profile:      profile,
		// ContextStore uses in-memory storage; context is lost on service restart.
		// TODO: Migrate to PostgreSQL for persistent storage.
		ContextStore: agent.NewContextStore(),
	}
}

// Chat handles non-streaming schedule agent chat requests.
func (s *ScheduleAgentService) Chat(ctx context.Context, req *v1pb.ScheduleAgentChatRequest) (*v1pb.ScheduleAgentChatResponse, error) {
	start := time.Now()
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	logger := slog.With("method", "ScheduleAgentService.Chat", "user_id", userID)
	logger.Info("Received chat request", "message_len", len(req.Message))

	// Get user timezone
	userTimezone := req.UserTimezone
	if userTimezone == "" {
		userTimezone = DefaultTimezone
	}

	// Create schedule service
	scheduleSvc := schedule.NewService(s.Store)

	// Create scheduler agent
	schedulerAgent, err := agent.NewSchedulerAgentV2(s.LLM, scheduleSvc, userID, userTimezone)
	if err != nil {
		logger.Error("Failed to create scheduler agent", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create scheduler agent: %v", err)
	}

	// Execute agent (non-streaming)
	response, err := schedulerAgent.Execute(ctx, req.Message)
	if err != nil {
		logger.Error("Agent execution failed", "error", err, "duration", time.Since(start))
		return nil, status.Errorf(codes.Internal, "agent execution failed: %v", err)
	}

	logger.Info("Chat request completed successfully", "duration", time.Since(start), "response_len", len(response))

	return &v1pb.ScheduleAgentChatResponse{
		Response: response,
	}, nil
}

// ChatStream handles streaming schedule agent chat requests.
func (s *ScheduleAgentService) ChatStream(req *v1pb.ScheduleAgentChatRequest, stream v1pb.ScheduleAgentService_ChatStreamServer) error {
	start := time.Now()
	ctx := stream.Context()

	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	logger := slog.With("method", "ScheduleAgentService.ChatStream", "user_id", userID)
	logger.Info("Received chat stream request", "message_len", len(req.Message))

	// Get user timezone
	userTimezone := req.UserTimezone
	if userTimezone == "" {
		userTimezone = DefaultTimezone
	}

	// Create schedule service
	scheduleSvc := schedule.NewService(s.Store)

	// Create scheduler agent
	schedulerAgent, err := agent.NewSchedulerAgentV2(s.LLM, scheduleSvc, userID, userTimezone)
	if err != nil {
		logger.Error("Failed to create scheduler agent", "error", err)
		return status.Errorf(codes.Internal, "failed to create scheduler agent: %v", err)
	}

	// Context Management: Get or create conversation context
	// Session ID is derived from user ID; TODO: accept session ID from request for multi-session support
	sessionID := fmt.Sprintf("default-session-%d", userID)

	// Retrieve conversation context from service's ContextStore
	conversationCtx := s.ContextStore.GetOrCreate(sessionID, userID, userTimezone)

	// Define callback for streaming events
	eventCallback := func(eventType string, eventData string) {
		// Log significant events
		if eventType == "tool_use" || eventType == "error" {
			logger.Info("Agent event", "type", eventType, "data_len", len(eventData))
		} else {
			logger.Debug("Agent event", "type", eventType, "data_len", len(eventData))
		}

		// Record turn part if needed?
		// Real recording happens after execution.

		// Send event as JSON
		eventJSON, err := json.Marshal(map[string]string{
			"type": eventType,
			"data": eventData,
		})
		if err != nil {
			logger.Error("Failed to marshal event", "error", err, "type", eventType)
			return
		}

		// Enhanced Server-Side Logging
		// Provide visibility into Agent's thought process in the backend logs
		switch eventType {
		case "tool_use":
			logger.Info("🛠️ Agent Tool Start", "tool", eventData)
		case "tool_result":
			// Truncate result for logging
			preview := eventData
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			logger.Info("✅ Agent Tool Result", "result", preview)
		case "thought":
			// Log reasoning trace
			logger.Info("🧠 Agent Thought", "trace", eventData)
		case "thinking":
			// logger.Debug("🤔 Agent Thinking...") // Keep debug to reduce noise
		}

		if err := stream.Send(&v1pb.ScheduleAgentStreamResponse{
			Event: string(eventJSON),
		}); err != nil {
			logger.Error("Failed to send event", "error", err, "type", eventType)
			return
		}

		// Send schedule_updated event if needed (only if first send succeeded)
		if eventType == "tool_result" && containsSuccessMessage(eventData) {
			updateJSON, err := json.Marshal(map[string]string{
				"type": "schedule_updated",
				"data": "{}",
			})
			if err != nil {
				logger.Error("Failed to marshal schedule_updated event", "error", err)
				return
			}
			if err := stream.Send(&v1pb.ScheduleAgentStreamResponse{
				Event: string(updateJSON),
			}); err != nil {
				logger.Error("Failed to send schedule_updated event", "error", err)
			}
		}
	}

	// Execute agent with callback AND context
	response, err := schedulerAgent.ExecuteWithCallback(ctx, req.Message, conversationCtx, eventCallback)
	if err != nil {
		logger.Error("Agent execution failed", "error", err, "duration", time.Since(start))
		errorJSON, jsonErr := json.Marshal(map[string]string{
			"type": "error",
			"data": fmt.Sprintf("Agent execution failed: %v", err),
		})
		if jsonErr != nil {
			logger.Error("Failed to marshal error event", "error", jsonErr)
		} else if sendErr := stream.Send(&v1pb.ScheduleAgentStreamResponse{
			Event: string(errorJSON),
		}); sendErr != nil {
			logger.Error("Failed to send error event", "error", sendErr)
		}
		return status.Errorf(codes.Internal, "agent execution failed: %v", err)
	}

	// Update Context with the turn
	// Note: ToolCalls capture is not fully implemented in callback yet,
	// but we record the text turn at least.
	conversationCtx.AddTurn(req.Message, response, nil)

	// Send final response
	finalJSON, err := json.Marshal(map[string]string{
		"type": "answer",
		"data": response,
	})
	if err != nil {
		logger.Error("Failed to marshal final response", "error", err)
		// Continue anyway, we can still send the response
		finalJSON = []byte(`{"type":"answer","data":"(error formatting response)"}`)
	}

	if err := stream.Send(&v1pb.ScheduleAgentStreamResponse{
		Event:   string(finalJSON),
		Content: response,
		Done:    true,
	}); err != nil {
		logger.Error("Failed to send final response", "error", err)
		return status.Errorf(codes.Internal, "failed to send response: %v", err)
	}

	logger.Info("Chat stream completed successfully", "duration", time.Since(start), "response_len", len(response))

	return nil
}

// containsSuccessMessage checks if the tool result indicates a schedule was created successfully.
func containsSuccessMessage(result string) bool {
	// Check for success indicators in the result (all lowercase for efficiency)
	successKeywords := []string{
		"successfully created",
		"schedule created",
		"已成功创建",
		"日程已创建",
	}

	resultLower := strings.ToLower(result)
	for _, keyword := range successKeywords {
		if strings.Contains(resultLower, keyword) {
			return true
		}
	}

	return false
}
