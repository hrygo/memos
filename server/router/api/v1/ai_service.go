package v1

import (
	"context"
	"fmt"

	"github.com/usememos/memos/plugin/ai"
	agentpkg "github.com/usememos/memos/plugin/ai/agent"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/server/finops"
	"github.com/usememos/memos/server/middleware"
	"github.com/usememos/memos/server/queryengine"
	"github.com/usememos/memos/server/retrieval"
	"github.com/usememos/memos/store"
)

// Global AI rate limiter
var globalAILimiter = middleware.NewRateLimiter()

// AIService provides AI-powered features for memo management.
type AIService struct {
	v1pb.UnimplementedAIServiceServer

	Store *store.Store

	EmbeddingService ai.EmbeddingService
	RerankerService  ai.RerankerService
	LLMService       ai.LLMService

	// 优化组件（Phase 1）
	QueryRouter       *queryengine.QueryRouter
	AdaptiveRetriever *retrieval.AdaptiveRetriever
	CostMonitor       *finops.CostMonitor

	// 鹦鹉系统（Milestone 1）
	ParrotRouter *agentpkg.ParrotRouter
}

// IsEnabled returns whether AI features are enabled.
func (s *AIService) IsEnabled() bool {
	return s.EmbeddingService != nil
}

// getCurrentUser gets the authenticated user from context.
func getCurrentUser(ctx context.Context, st *store.Store) (*store.User, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("user not found in context")
	}
	user, err := st.GetUser(ctx, &store.FindUser{
		ID: &userID,
	})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user %d not found", userID)
	}
	return user, nil
}
