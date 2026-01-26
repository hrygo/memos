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

// emptyMetadata is the default empty JSON object for message metadata.
const emptyMetadata = "{}"

// MaxMessageLimit is the maximum number of messages to return in a single request.
const MaxMessageLimit = 100

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

	// Collect pinned conversation IDs for batch message counting
	var pinnedIDs []int32
	for _, c := range conversations {
		if c.Pinned {
			pinnedIDs = append(pinnedIDs, c.ID)
		}
	}

	// Batch count messages for all pinned conversations at once (avoid N+1)
	messageCounts := make(map[int32]int32)
	if len(pinnedIDs) > 0 {
		allMessages, err := s.Store.ListAIMessages(ctx, &store.FindAIMessage{
			// Get messages for all pinned conversations
		})
		if err == nil {
			// Count non-SEPARATOR messages per conversation
			for _, m := range allMessages {
				if m.Type != store.AIMessageTypeSeparator {
					messageCounts[m.ConversationID]++
				}
			}
		}
	}

	response := &v1pb.ListAIConversationsResponse{
		Conversations: make([]*v1pb.AIConversation, 0, len(conversations)),
	}
	for _, c := range conversations {
		// Only return pinned (fixed) conversations to hide legacy/temporary conversations
		if c.Pinned {
			pbConv := convertAIConversationFromStore(c)
			pbConv.MessageCount = messageCounts[c.ID]
			response.Conversations = append(response.Conversations, pbConv)
		}
	}

	return response, nil
}

// ensureFixedConversations ensures all 5 fixed conversations exist for the user.
// This is called on ListAIConversations to guarantee users always see all available parrots.
// Uses batch query to avoid N+1 problem.
func (s *AIService) ensureFixedConversations(ctx context.Context, userID int32) {
	// Calculate all fixed conversation IDs first (3 core capabilities)
	agentTypes := []struct {
		agent v1pb.AgentType
		ai    aichat.AgentType
	}{
		{v1pb.AgentType_AGENT_TYPE_MEMO, aichat.AgentTypeMemo},
		{v1pb.AgentType_AGENT_TYPE_SCHEDULE, aichat.AgentTypeSchedule},
		{v1pb.AgentType_AGENT_TYPE_AMAZING, aichat.AgentTypeAmazing},
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

	// Count non-SEPARATOR messages for MessageCount
	messageCount := 0
	for _, m := range messages {
		pbConversation.Messages = append(pbConversation.Messages, convertAIMessageFromStore(m))
		if m.Type != store.AIMessageTypeSeparator {
			messageCount++
		}
	}
	pbConversation.MessageCount = int32(messageCount)

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

func (s *AIService) AddContextSeparator(ctx context.Context, req *v1pb.AddContextSeparatorRequest) (*emptypb.Empty, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Verify conversation ownership
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.ConversationId,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	// Prevent duplicate SEPARATOR: check if the last message is already a SEPARATOR
	messages, err := s.Store.ListAIMessages(ctx, &store.FindAIMessage{
		ConversationID: &req.ConversationId,
	})
	if err == nil && len(messages) > 0 {
		// Messages are ordered by created_ts ASC, so last element is the newest
		lastMessage := messages[len(messages)-1]
		if lastMessage.Type == store.AIMessageTypeSeparator {
			// Last message is already a SEPARATOR, silently succeed (idempotent)
			return &emptypb.Empty{}, nil
		}
	}

	// Create SEPARATOR message using the conversation service
	_, err = s.Store.CreateAIMessage(ctx, &store.AIMessage{
		UID:            shortuuid.New(),
		ConversationID: req.ConversationId,
		Type:           store.AIMessageTypeSeparator,
		Role:           store.AIMessageRoleSystem,
		Content:        "---", // Content marker for separator
		Metadata:       emptyMetadata,
		CreatedTs:      time.Now().Unix(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create separator message: %v", err)
	}

	// Update conversation timestamp
	now := time.Now().Unix()
	_, err = s.Store.UpdateAIConversation(ctx, &store.UpdateAIConversation{
		ID:        req.ConversationId,
		UpdatedTs: &now,
	})

	return &emptypb.Empty{}, nil
}

func convertAIConversationFromStore(c *store.AIConversation) *v1pb.AIConversation {
	// Convert ParrotID string to AgentType enum
	// Handle both short format ("MEMO") and long format ("AGENT_TYPE_MEMO")
	// DEFAULT and CREATIVE are deprecated - map to AMAZING
	var parrotId int32

	// Try direct lookup first (long format like "AGENT_TYPE_MEMO")
	if val, ok := v1pb.AgentType_value[c.ParrotID]; ok {
		parrotId = val
	} else {
		// Try short format lookup ("MEMO" → "AGENT_TYPE_MEMO")
		// Legacy: DEFAULT/CREATIVE → AMAZING
		shortToLong := map[string]v1pb.AgentType{
			"MEMO":     v1pb.AgentType_AGENT_TYPE_MEMO,
			"SCHEDULE": v1pb.AgentType_AGENT_TYPE_SCHEDULE,
			"AMAZING":  v1pb.AgentType_AGENT_TYPE_AMAZING,
			"DEFAULT":  v1pb.AgentType_AGENT_TYPE_AMAZING, // Legacy alias
			"CREATIVE": v1pb.AgentType_AGENT_TYPE_AMAZING, // Legacy alias
		}
		if val, ok := shortToLong[c.ParrotID]; ok {
			parrotId = int32(val)
		} else {
			// Unknown value, log warning and fallback to AMAZING
			slog.Default().Warn("Unknown ParrotID in conversation, falling back to AMAZING",
				"conversation_id", c.ID,
				"parrot_id", c.ParrotID,
			)
			parrotId = int32(v1pb.AgentType_AGENT_TYPE_AMAZING)
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

// ListMessages returns messages for a conversation with incremental sync support.
// - First load (lastMessageUid empty): returns latest MaxMessageLimit MSG (SEP included)
// - Incremental load (lastMessageUid provided): returns messages after that UID, max MaxMessageLimit MSG
// - SUMMARY messages are filtered out (never returned to frontend)
func (s *AIService) ListMessages(ctx context.Context, req *v1pb.ListMessagesRequest) (*v1pb.ListMessagesResponse, error) {
	// Parameter validation
	if req.ConversationId == 0 {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	limit := req.Limit
	if limit <= 0 || limit > MaxMessageLimit {
		limit = MaxMessageLimit // Default and max limit
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// Verify conversation ownership
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.ConversationId,
		CreatorID: &user.ID,
	})
	if err != nil || len(conversations) == 0 {
		return nil, status.Error(codes.NotFound, "conversation not found")
	}

	// Load all messages from database
	allMessages, err := s.Store.ListAIMessages(ctx, &store.FindAIMessage{
		ConversationID: &req.ConversationId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load messages")
	}

	// Filter out SUMMARY messages (SUMMARY is never returned to frontend)
	var visibleMessages []*store.AIMessage
	for _, msg := range allMessages {
		if msg.Type != store.AIMessageTypeSummary {
			visibleMessages = append(visibleMessages, msg)
		}
	}

	// Calculate total MSG count (for total_count)
	var totalMsgCount int32
	for _, msg := range visibleMessages {
		if msg.Type == store.AIMessageTypeMessage {
			totalMsgCount++
		}
	}

	// Determine starting position based on request type
	var startIndex int
	if req.LastMessageUid == "" {
		// First load: from end, count back MaxMessageLimit MSG to find start position
		msgCount := 0
		for i := len(visibleMessages) - 1; i >= 0; i-- {
			if visibleMessages[i].Type == store.AIMessageTypeMessage {
				msgCount++
				if msgCount > MaxMessageLimit {
					startIndex = i + 1
					break
				}
			}
		}
	} else {
		// Incremental load: find position after lastMessageUid
		found := false
		for i, msg := range visibleMessages {
			if msg.UID == req.LastMessageUid {
				startIndex = i + 1
				found = true
				break
			}
		}
		if !found {
			// UID not found - tell frontend to refresh
			return &v1pb.ListMessagesResponse{
				Messages:         []*v1pb.AIMessage{},
				HasMore:          false,
				TotalCount:       totalMsgCount,
				LatestMessageUid: getLatestMessageUID(visibleMessages),
				SyncRequired:     true,
			}, nil
		}
	}

	// Collect messages from startIndex, max MaxMessageLimit MSG (SEP included)
	var result []*store.AIMessage
	msgCount := 0
	for i := startIndex; i < len(visibleMessages) && msgCount < MaxMessageLimit; i++ {
		msg := visibleMessages[i]
		result = append(result, msg)
		if msg.Type == store.AIMessageTypeMessage {
			msgCount++
		}
		// SEPARATOR is included but not counted
	}

	// Convert to protobuf format
	var messages []*v1pb.AIMessage
	for _, msg := range result {
		messages = append(messages, convertAIMessageFromStore(msg))
	}

	return &v1pb.ListMessagesResponse{
		Messages:         messages,
		HasMore:          startIndex > 0, // More messages available before start index
		TotalCount:       totalMsgCount,
		LatestMessageUid: getLatestMessageUID(visibleMessages),
		SyncRequired:     false,
	}, nil
}

// getLatestMessageUID returns the UID of the latest message.
func getLatestMessageUID(messages []*store.AIMessage) string {
	if len(messages) == 0 {
		return ""
	}
	return messages[len(messages)-1].UID
}
