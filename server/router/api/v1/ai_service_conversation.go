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
func (s *AIService) ensureFixedConversations(ctx context.Context, userID int32) {
	agentTypes := []v1pb.AgentType{
		v1pb.AgentType_AGENT_TYPE_DEFAULT,
		v1pb.AgentType_AGENT_TYPE_MEMO,
		v1pb.AgentType_AGENT_TYPE_SCHEDULE,
		v1pb.AgentType_AGENT_TYPE_AMAZING,
		v1pb.AgentType_AGENT_TYPE_CREATIVE,
	}

	for _, agentType := range agentTypes {
		fixedID := aichat.CalculateFixedConversationID(userID, aichat.AgentType(agentType))

		// Check if conversation exists
		existing, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
			ID:        &fixedID,
			CreatorID: &userID,
		})
		if err == nil && len(existing) > 0 {
			// Conversation exists, update timestamp
			now := time.Now().Unix()
			s.Store.UpdateAIConversation(ctx, &store.UpdateAIConversation{
				ID:        fixedID,
				UpdatedTs: &now,
			})
			continue
		}

		// Create new fixed conversation
		title := aichat.GetFixedConversationTitle(aichat.AgentType(agentType))
		_, _ = s.Store.CreateAIConversation(ctx, &store.AIConversation{
			ID:        fixedID,
			UID:       shortuuid.New(),
			CreatorID: userID,
			Title:     title,
			ParrotID:  agentType.String(),
			Pinned:    true,
			CreatedTs: time.Now().Unix(),
			UpdatedTs: time.Now().Unix(),
			RowStatus: store.Normal,
		})
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
	// Convert ParrotID string to AgentType enum with default fallback
	parrotId := v1pb.AgentType_value[c.ParrotID]
	if parrotId == 0 && c.ParrotID != "" && c.ParrotID != "AGENT_TYPE_UNSPECIFIED" {
		// Unknown value, log warning and fallback to DEFAULT
		slog.Default().Warn("Unknown ParrotID in conversation, falling back to DEFAULT",
			"conversation_id", c.ID,
			"parrot_id", c.ParrotID,
		)
		parrotId = int32(v1pb.AgentType_AGENT_TYPE_DEFAULT)
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
