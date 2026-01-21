package profile

import (
	"os"
	"testing"
)

// TestAIProfileDefaults 测试 AI 配置的默认值
func TestAIProfileDefaults(t *testing.T) {
	// 清除环境变量
	clearAIEnvVars()

	profile := &Profile{}
	profile.FromEnv()

	tests := []struct {
		name     string
		field    string
		expected string
		actual   string
	}{
		{"AIEnabled should be false by default", "AIEnabled", "false", boolToString(profile.AIEnabled)},
		{"AIEmbeddingProvider default", "AIEmbeddingProvider", "siliconflow", profile.AIEmbeddingProvider},
		{"AILLMProvider default", "AILLMProvider", "deepseek", profile.AILLMProvider},
		{"AISiliconFlowBaseURL default", "AISiliconFlowBaseURL", "https://api.siliconflow.cn/v1", profile.AISiliconFlowBaseURL},
		{"AIDeepSeekBaseURL default", "AIDeepSeekBaseURL", "https://api.deepseek.com", profile.AIDeepSeekBaseURL},
		{"AIOpenAIBaseURL default", "AIOpenAIBaseURL", "https://api.openai.com/v1", profile.AIOpenAIBaseURL},
		{"AIOllamaBaseURL default", "AIOllamaBaseURL", "http://localhost:11434", profile.AIOllamaBaseURL},
		{"AIEmbeddingModel default", "AIEmbeddingModel", "BAAI/bge-m3", profile.AIEmbeddingModel},
		{"AIRerankModel default", "AIRerankModel", "BAAI/bge-reranker-v2-m3", profile.AIRerankModel},
		{"AILLMModel default", "AILLMModel", "deepseek-chat", profile.AILLMModel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, tt.actual)
			}
		})
	}
}

// TestAIProfileFromEnv 测试从环境变量读取 AI 配置
func TestAIProfileFromEnv(t *testing.T) {
	// 清除环境变量
	clearAIEnvVars()

	tests := []struct {
		name     string
		envVar   string
		envValue string
		field    func(*Profile) string
		expected string
	}{
		{
			name:     "MEMOS_AI_ENABLED=true",
			envVar:   "MEMOS_AI_ENABLED",
			envValue: "true",
			field:    func(p *Profile) string { return boolToString(p.AIEnabled) },
			expected: "true",
		},
		{
			name:     "MEMOS_AI_EMBEDDING_PROVIDER",
			envVar:   "MEMOS_AI_EMBEDDING_PROVIDER",
			envValue: "openai",
			field:    func(p *Profile) string { return p.AIEmbeddingProvider },
			expected: "openai",
		},
		{
			name:     "MEMOS_AI_LLM_PROVIDER",
			envVar:   "MEMOS_AI_LLM_PROVIDER",
			envValue: "ollama",
			field:    func(p *Profile) string { return p.AILLMProvider },
			expected: "ollama",
		},
		{
			name:     "MEMOS_AI_SILICONFLOW_API_KEY",
			envVar:   "MEMOS_AI_SILICONFLOW_API_KEY",
			envValue: "test-key-123",
			field:    func(p *Profile) string { return p.AISiliconFlowAPIKey },
			expected: "test-key-123",
		},
		{
			name:     "MEMOS_AI_DEEPSEEK_API_KEY",
			envVar:   "MEMOS_AI_DEEPSEEK_API_KEY",
			envValue: "deepseek-key",
			field:    func(p *Profile) string { return p.AIDeepSeekAPIKey },
			expected: "deepseek-key",
		},
		{
			name:     "MEMOS_AI_OPENAI_API_KEY",
			envVar:   "MEMOS_AI_OPENAI_API_KEY",
			envValue: "openai-key",
			field:    func(p *Profile) string { return p.AIOpenAIAPIKey },
			expected: "openai-key",
		},
		{
			name:     "MEMOS_AI_OPENAI_BASE_URL",
			envVar:   "MEMOS_AI_OPENAI_BASE_URL",
			envValue: "https://custom.openai.proxy/v1",
			field:    func(p *Profile) string { return p.AIOpenAIBaseURL },
			expected: "https://custom.openai.proxy/v1",
		},
		{
			name:     "MEMOS_AI_EMBEDDING_MODEL",
			envVar:   "MEMOS_AI_EMBEDDING_MODEL",
			envValue: "custom-embedding-model",
			field:    func(p *Profile) string { return p.AIEmbeddingModel },
			expected: "custom-embedding-model",
		},
		{
			name:     "MEMOS_AI_RERANK_MODEL",
			envVar:   "MEMOS_AI_RERANK_MODEL",
			envValue: "custom-rerank-model",
			field:    func(p *Profile) string { return p.AIRerankModel },
			expected: "custom-rerank-model",
		},
		{
			name:     "MEMOS_AI_LLM_MODEL",
			envVar:   "MEMOS_AI_LLM_MODEL",
			envValue: "gpt-4",
			field:    func(p *Profile) string { return p.AILLMModel },
			expected: "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAIEnvVars()
			os.Setenv(tt.envVar, tt.envValue)

			profile := &Profile{}
			profile.FromEnv()

			actual := tt.field(profile)
			if actual != tt.expected {
				t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, actual)
			}
		})
	}
}

// TestIsAIEnabled 测试 IsAIEnabled 逻辑
func TestIsAIEnabled(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*Profile)
		expectedResult bool
	}{
		{
			name: "AIEnabled=false should return false",
			setup: func(p *Profile) {
				p.AIEnabled = false
			},
			expectedResult: false,
		},
		{
			name: "AIEnabled=true but no API key should return false",
			setup: func(p *Profile) {
				p.AIEnabled = true
				p.AISiliconFlowAPIKey = ""
				p.AIOpenAIAPIKey = ""
				p.AIOllamaBaseURL = ""
			},
			expectedResult: false,
		},
		{
			name: "AIEnabled=true with SiliconFlow API key should return true",
			setup: func(p *Profile) {
				p.AIEnabled = true
				p.AISiliconFlowAPIKey = "test-key"
				p.AIOpenAIAPIKey = ""
				p.AIOllamaBaseURL = ""
			},
			expectedResult: true,
		},
		{
			name: "AIEnabled=true with OpenAI API key should return true",
			setup: func(p *Profile) {
				p.AIEnabled = true
				p.AISiliconFlowAPIKey = ""
				p.AIOpenAIAPIKey = "test-key"
				p.AIOllamaBaseURL = ""
			},
			expectedResult: true,
		},
		{
			name: "AIEnabled=true with Ollama base URL should return true",
			setup: func(p *Profile) {
				p.AIEnabled = true
				p.AISiliconFlowAPIKey = ""
				p.AIOpenAIAPIKey = ""
				p.AIOllamaBaseURL = "http://localhost:11434"
			},
			expectedResult: true,
		},
		{
			name: "AIEnabled=false with API keys should return false",
			setup: func(p *Profile) {
				p.AIEnabled = false
				p.AISiliconFlowAPIKey = "test-key"
				p.AIOpenAIAPIKey = "test-key"
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &Profile{}
			tt.setup(profile)
			result := profile.IsAIEnabled()
			if result != tt.expectedResult {
				t.Errorf("IsAIEnabled(): expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

// Helper functions

func clearAIEnvVars() {
	aiEnvVars := []string{
		"MEMOS_AI_ENABLED",
		"MEMOS_AI_EMBEDDING_PROVIDER",
		"MEMOS_AI_LLM_PROVIDER",
		"MEMOS_AI_SILICONFLOW_API_KEY",
		"MEMOS_AI_SILICONFLOW_BASE_URL",
		"MEMOS_AI_DEEPSEEK_API_KEY",
		"MEMOS_AI_DEEPSEEK_BASE_URL",
		"MEMOS_AI_OPENAI_API_KEY",
		"MEMOS_AI_OPENAI_BASE_URL",
		"MEMOS_AI_OLLAMA_BASE_URL",
		"MEMOS_AI_EMBEDDING_MODEL",
		"MEMOS_AI_RERANK_MODEL",
		"MEMOS_AI_LLM_MODEL",
	}
	for _, envVar := range aiEnvVars {
		os.Unsetenv(envVar)
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
