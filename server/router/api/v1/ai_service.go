package v1

import (
	"context"
	"fmt"
	"sync"

	pluginai "github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/plugin/ai/memory"
	"github.com/usememos/memos/plugin/ai/router"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/server/middleware"
	"github.com/usememos/memos/server/retrieval"
	aichat "github.com/usememos/memos/server/router/api/v1/ai"
	"github.com/usememos/memos/store"
)

// Global AI rate limiter
var globalAILimiter = middleware.NewRateLimiter()

// Default history retention count for router memory service
const DefaultHistoryRetention = 10

// AIService provides AI-powered features for memo management.
type AIService struct {
	v1pb.UnimplementedAIServiceServer

	Store *store.Store

	EmbeddingService pluginai.EmbeddingService
	EmbeddingModel   string // embedding model name for duplicate detection
	RerankerService  pluginai.RerankerService
	LLMService       pluginai.LLMService

	// Adaptive retriever for RAG operations
	AdaptiveRetriever *retrieval.AdaptiveRetriever

	// Intent classifier configuration for chat routing
	IntentClassifierConfig *pluginai.IntentClassifierConfig

	// Router service for three-layer intent classification (lazily initialized)
	routerServiceOnce sync.Once
	routerService      *router.Service

	// Chat event bus and conversation service (lazily initialized)
	chatEventBusMu      sync.RWMutex
	chatEventBus        *aichat.EventBus
	conversationService *aichat.ConversationService

	// Context builder and summarizer (lazily initialized)
	contextBuilderMu         sync.RWMutex
	contextBuilder           *aichat.ContextBuilder
	conversationSummarizerMu sync.RWMutex
	conversationSummarizer   *aichat.ConversationSummarizer
}

// IsEnabled returns whether AI features are enabled.
// For basic features (embedding, search), only EmbeddingService is required.
// For Agent features (Memo, Schedule, etc.), both EmbeddingService and LLMService are required.
func (s *AIService) IsEnabled() bool {
	return s.EmbeddingService != nil
}

// IsLLMEnabled returns whether LLM features are enabled (required for Agents).
func (s *AIService) IsLLMEnabled() bool {
	return s.LLMService != nil
}

// getRouterService returns the router service, initializing it on first use.
// Returns nil if Store is not available, which is safe as callers check for nil.
func (s *AIService) getRouterService() *router.Service {
	s.routerServiceOnce.Do(func() {
		if s.Store == nil {
			// Store not available, routerService remains nil
			return
		}

		// Create memory service for router
		memService := memory.NewService(s.Store, DefaultHistoryRetention)

		// Create LLM client wrapper for router
		var llmClient router.LLMClient
		if s.LLMService != nil {
			llmClient = &routerLLMClient{llm: s.LLMService}
		}

		s.routerService = router.NewService(router.Config{
			MemoryService: memService,
			LLMClient:     llmClient,
		})
	})

	return s.routerService
}

// routerLLMClient adapts LLMService to router.LLMClient interface.
type routerLLMClient struct {
	llm pluginai.LLMService
}

func (c *routerLLMClient) Complete(ctx context.Context, prompt string, config router.ModelConfig) (string, error) {
	// Convert router request to LLM chat
	messages := []pluginai.Message{
		{Role: "system", Content: "You are an intent classifier. Respond only with the intent type."},
		{Role: "user", Content: prompt},
	}
	// Apply model configuration for the LLM call
	// Note: Currently the LLM service uses global configuration, but config.MaxTokens
	// and config.Temperature are available here for future per-request configuration.
	return c.llm.Chat(ctx, messages)
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
