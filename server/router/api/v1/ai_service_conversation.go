package v1

import (
	"context"
	"log/slog"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	aichat "github.com/usememos/memos/server/router/api/v1/ai"
	"github.com/usememos/memos/store"
)

func (s *AIService) ListAIConversations(ctx context.Context, _ *v1pb.ListAIConversationsRequest) (*v1pb.ListAIConversationsResponse, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Ensure all fixed conversations exist for the user
	s.ensureFixedConversations(ctx, user.ID)

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list conversations: %v", err)
	}

	response := &v1pb.ListAIConversationsResponse{
		Conversations: make([]*v1pb.AIConversation, 0, len(conversations)),
	}
	for _, c := range conversations {
		response.Conversations = append(response.Conversations, convertAIConversationFromStore(c))
	}

	return response, nil
}

// ensureFixedConversations ensures all 5 fixed conversations exist for the user.
// This is called on ListAIConversations to guarantee users always see all available parrots.
// Uses batch query to avoid N+1 problem.
func (s *AIService) ensureFixedConversations(ctx context.Context, userID int32) {
	// Calculate all fixed conversation IDs first
	agentTypes := []struct {
		agent v1pb.AgentType
		ai    aichat.AgentType
	}{
		{v1pb.AgentType_AGENT_TYPE_DEFAULT, aichat.AgentTypeDefault},
		{v1pb.AgentType_AGENT_TYPE_MEMO, aichat.AgentTypeMemo},
		{v1pb.AgentType_AGENT_TYPE_SCHEDULE, aichat.AgentTypeSchedule},
		{v1pb.AgentType_AGENT_TYPE_AMAZING, aichat.AgentTypeAmazing},
		{v1pb.AgentType_AGENT_TYPE_CREATIVE, aichat.AgentTypeCreative},
	}

	fixedIDs := make([]int32, len(agentTypes))
	idToAgentType := make(map[int32]struct {
		agent v1pb.AgentType
		ai    aichat.AgentType
	})

	for i, at := range agentTypes {
		id := aichat.CalculateFixedConversationID(userID, at.ai)
		fixedIDs[i] = id
		idToAgentType[id] = at
	}

	// Batch query: get all existing conversations in one call
	// Using ID IN clause would be ideal, but current API doesn't support it
	// Fall back to querying by user and filter in-memory
	existing, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		CreatorID: &userID,
	})
	if err != nil {
		slog.Default().Warn("Failed to list conversations for fixed check", "user_id", userID, "error", err)
	}

	// Build a set of existing fixed conversation IDs
	existingFixedIDs := make(map[int32]bool, len(existing))
	for _, c := range existing {
		if _, ok := idToAgentType[c.ID]; ok {
			existingFixedIDs[c.ID] = true
		}
	}

	// Create missing fixed conversations only
	now := time.Now().Unix()
	for _, id := range fixedIDs {
		if existingFixedIDs[id] {
			continue // Already exists
		}

		at := idToAgentType[id]
		title := aichat.GetFixedConversationTitle(at.ai)

		conv, err := s.Store.CreateAIConversation(ctx, &store.AIConversation{
			ID:        id,
			UID:       shortuuid.New(),
			CreatorID: userID,
			Title:     title,
			ParrotID:  at.agent.String(),
			Pinned:    true,
			CreatedTs: now,
			UpdatedTs: now,
			RowStatus: store.Normal,
		})
		if err != nil {
			slog.Default().Warn("Failed to create fixed conversation",
				"id", id,
				"agent_type", at.agent.String(),
				"user_id", userID,
				"error", err,
			)
		} else {
			slog.Default().Debug("Created fixed conversation",
				"id", conv.ID,
				"title", conv.Title,
			)
		}
	}
}

func (s *AIService) GetAIConversation(ctx context.Context, req *v1pb.GetAIConversationRequest) (*v1pb.AIConversation, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.Id,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	conversation := conversations[0]
	messages, err := s.Store.ListAIMessages(ctx, &store.FindAIMessage{
		ConversationID: &conversation.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list messages: %v", err)
	}

	pbConversation := convertAIConversationFromStore(conversation)
	pbConversation.Messages = make([]*v1pb.AIMessage, 0, len(messages))
	for _, m := range messages {
		pbConversation.Messages = append(pbConversation.Messages, convertAIMessageFromStore(m))
	}

	return pbConversation, nil
}

func (s *AIService) CreateAIConversation(ctx context.Context, req *v1pb.CreateAIConversationRequest) (*v1pb.AIConversation, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	now := time.Now().Unix()
	conversation, err := s.Store.CreateAIConversation(ctx, &store.AIConversation{
		UID:       shortuuid.New(),
		CreatorID: user.ID,
		Title:     req.Title,
		ParrotID:  req.ParrotId.String(),
		Pinned:    false,
		CreatedTs: now,
		UpdatedTs: now,
		RowStatus: store.Normal,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create conversation: %v", err)
	}

	return convertAIConversationFromStore(conversation), nil
}

func (s *AIService) UpdateAIConversation(ctx context.Context, req *v1pb.UpdateAIConversationRequest) (*v1pb.AIConversation, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Check ownership
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.Id,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	update := &store.UpdateAIConversation{
		ID:        req.Id,
		UpdatedTs: func() *int64 { t := time.Now().Unix(); return &t }(),
	}
	if req.Title != nil {
		update.Title = req.Title
	}
	if req.Pinned != nil {
		update.Pinned = req.Pinned
	}

	updated, err := s.Store.UpdateAIConversation(ctx, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update conversation: %v", err)
	}

	return convertAIConversationFromStore(updated), nil
}

func (s *AIService) DeleteAIConversation(ctx context.Context, req *v1pb.DeleteAIConversationRequest) (*emptypb.Empty, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Check ownership
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.Id,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	// Prevent deletion of fixed (pinned) conversations
	conversation := conversations[0]
	if conversation.Pinned {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot delete fixed conversation")
	}

	if err := s.Store.DeleteAIConversation(ctx, &store.DeleteAIConversation{ID: req.Id}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete conversation: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func convertAIConversationFromStore(c *store.AIConversation) *v1pb.AIConversation {
	// Convert ParrotID string to AgentType enum
	// Handle both short format ("MEMO") and long format ("AGENT_TYPE_MEMO")
	var parrotId int32

	// Try direct lookup first (long format like "AGENT_TYPE_MEMO")
	if val, ok := v1pb.AgentType_value[c.ParrotID]; ok {
		parrotId = val
	} else {
		// Try short format lookup ("MEMO" → "AGENT_TYPE_MEMO")
		shortToLong := map[string]v1pb.AgentType{
			"DEFAULT":  v1pb.AgentType_AGENT_TYPE_DEFAULT,
			"MEMO":     v1pb.AgentType_AGENT_TYPE_MEMO,
			"SCHEDULE":  v1pb.AgentType_AGENT_TYPE_SCHEDULE,
			"AMAZING":   v1pb.AgentType_AGENT_TYPE_AMAZING,
			"CREATIVE":  v1pb.AgentType_AGENT_TYPE_CREATIVE,
		}
		if val, ok := shortToLong[c.ParrotID]; ok {
			parrotId = int32(val)
		} else {
			// Unknown value, log warning and fallback to DEFAULT
			slog.Default().Warn("Unknown ParrotID in conversation, falling back to DEFAULT",
				"conversation_id", c.ID,
				"parrot_id", c.ParrotID,
			)
			parrotId = int32(v1pb.AgentType_AGENT_TYPE_DEFAULT)
		}
	}

	return &v1pb.AIConversation{
		Id:        c.ID,
		Uid:       c.UID,
		CreatorId: c.CreatorID,
		Title:     c.Title,
		ParrotId:  v1pb.AgentType(parrotId),
		Pinned:    c.Pinned,
		CreatedTs: c.CreatedTs,
		UpdatedTs: c.UpdatedTs,
	}
}

func convertAIMessageFromStore(m *store.AIMessage) *v1pb.AIMessage {
	return &v1pb.AIMessage{
		Id:             m.ID,
		Uid:            m.UID,
		ConversationId: m.ConversationID,
		Type:           string(m.Type),
		Role:           string(m.Role),
		Content:        m.Content,
		Metadata:       m.Metadata,
		CreatedTs:      m.CreatedTs,
	}
}
