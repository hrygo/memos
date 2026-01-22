package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"

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

	Store   *store.Store
	LLM     ai.LLMService
	Profile *profile.Profile
}

// NewScheduleAgentService creates a new schedule agent service.
func NewScheduleAgentService(store *store.Store, llm ai.LLMService, profile *profile.Profile) *ScheduleAgentService {
	return &ScheduleAgentService{
		Store:   store,
		LLM:     llm,
		Profile: profile,
	}
}

// Chat handles non-streaming schedule agent chat requests.
func (s *ScheduleAgentService) Chat(ctx context.Context, req *v1pb.ScheduleAgentChatRequest) (*v1pb.ScheduleAgentChatResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Get user timezone
	userTimezone := req.UserTimezone
	if userTimezone == "" {
		userTimezone = DefaultTimezone
	}

	// Create schedule service
	scheduleSvc := schedule.NewService(s.Store)

	// Create scheduler agent
	schedulerAgent, err := agent.NewSchedulerAgent(s.LLM, scheduleSvc, userID, userTimezone)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create scheduler agent: %v", err)
	}

	// Execute agent (non-streaming)
	response, err := schedulerAgent.Execute(ctx, req.Message)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "agent execution failed: %v", err)
	}

	return &v1pb.ScheduleAgentChatResponse{
		Response: response,
	}, nil
}

// ChatStream handles streaming schedule agent chat requests.
func (s *ScheduleAgentService) ChatStream(req *v1pb.ScheduleAgentChatRequest, stream v1pb.ScheduleAgentService_ChatStreamServer) error {
	ctx := stream.Context()

	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Get user timezone
	userTimezone := req.UserTimezone
	if userTimezone == "" {
		userTimezone = DefaultTimezone
	}

	// Create schedule service
	scheduleSvc := schedule.NewService(s.Store)

	// Create scheduler agent
	schedulerAgent, err := agent.NewSchedulerAgent(s.LLM, scheduleSvc, userID, userTimezone)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create scheduler agent: %v", err)
	}

	// Define callback for streaming events
	eventCallback := func(eventType string, eventData string) {
		// Send event as JSON
		eventJSON, err := json.Marshal(map[string]string{
			"type": eventType,
			"data": eventData,
		})
		if err != nil {
			slog.Error("failed to marshal event", "error", err, "type", eventType)
			return
		}

		if err := stream.Send(&v1pb.ScheduleAgentStreamResponse{
			Event: string(eventJSON),
		}); err != nil {
			slog.Error("failed to send event", "error", err, "type", eventType)
			return
		}

		// Send schedule_updated event if needed (only if first send succeeded)
		if eventType == "tool_result" && containsSuccessMessage(eventData) {
			updateJSON, err := json.Marshal(map[string]string{
				"type": "schedule_updated",
				"data": "{}",
			})
			if err != nil {
				slog.Error("failed to marshal schedule_updated event", "error", err)
				return
			}
			if err := stream.Send(&v1pb.ScheduleAgentStreamResponse{
				Event: string(updateJSON),
			}); err != nil {
				slog.Error("failed to send schedule_updated event", "error", err)
			}
		}
	}

	// Execute agent with callback
	response, err := schedulerAgent.ExecuteWithCallback(ctx, req.Message, nil, eventCallback)
	if err != nil {
		errorJSON, jsonErr := json.Marshal(map[string]string{
			"type": "error",
			"data": fmt.Sprintf("Agent execution failed: %v", err),
		})
		if jsonErr != nil {
			slog.Error("failed to marshal error event", "error", jsonErr)
		} else if sendErr := stream.Send(&v1pb.ScheduleAgentStreamResponse{
			Event: string(errorJSON),
		}); sendErr != nil {
			slog.Error("failed to send error event", "error", sendErr)
		}
		return status.Errorf(codes.Internal, "agent execution failed: %v", err)
	}

	// Send final response
	finalJSON, err := json.Marshal(map[string]string{
		"type": "answer",
		"data": response,
	})
	if err != nil {
		slog.Error("failed to marshal final response", "error", err)
		// Continue anyway, we can still send the response
		finalJSON = []byte(`{"type":"answer","data":"(error formatting response)"}`)
	}

	if err := stream.Send(&v1pb.ScheduleAgentStreamResponse{
		Event:   string(finalJSON),
		Content: response,
		Done:    true,
	}); err != nil {
		slog.Error("failed to send final response", "error", err)
		return status.Errorf(codes.Internal, "failed to send response: %v", err)
	}

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
