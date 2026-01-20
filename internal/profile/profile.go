package profile

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// Profile is the configuration to start main server.
type Profile struct {
	// Mode can be "prod" or "dev" or "demo"
	Mode string
	// Addr is the binding address for server
	Addr string
	// Port is the binding port for server
	Port int
	// UNIXSock is the IPC binding path. Overrides Addr and Port
	UNIXSock string
	// Data is the data directory
	Data string
	// DSN points to where memos stores its own data
	DSN string
	// Driver is the database driver
	// sqlite, mysql
	Driver string
	// Version is the current version of server
	Version string
	// InstanceURL is the url of your memos instance.
	InstanceURL string

	// AI Configuration
	AIEnabled            bool   // MEMOS_AI_ENABLED
	AIEmbeddingProvider  string // MEMOS_AI_EMBEDDING_PROVIDER (default: siliconflow)
	AILLMProvider        string // MEMOS_AI_LLM_PROVIDER (default: deepseek)
	AISiliconFlowAPIKey  string // MEMOS_AI_SILICONFLOW_API_KEY
	AISiliconFlowBaseURL string // MEMOS_AI_SILICONFLOW_BASE_URL (default: https://api.siliconflow.cn/v1)
	AIDeepSeekAPIKey     string // MEMOS_AI_DEEPSEEK_API_KEY
	AIDeepSeekBaseURL    string // MEMOS_AI_DEEPSEEK_BASE_URL (default: https://api.deepseek.com)
	AIOpenAIAPIKey       string // MEMOS_AI_OPENAI_API_KEY
	AIOpenAIBaseURL      string // MEMOS_AI_OPENAI_BASE_URL (default: https://api.openai.com/v1)
	AIOllamaBaseURL      string // MEMOS_AI_OLLAMA_BASE_URL (default: http://localhost:11434)
	AIEmbeddingModel     string // MEMOS_AI_EMBEDDING_MODEL (default: BAAI/bge-m3)
	AIRerankModel        string // MEMOS_AI_RERANK_MODEL (default: BAAI/bge-reranker-v2-m3)
	AILLMModel           string // MEMOS_AI_LLM_MODEL (default: deepseek-chat)
}

func (p *Profile) IsDev() bool {
	return p.Mode != "prod"
}

// IsAIEnabled returns true if AI is enabled and at least one API key or base URL is configured.
func (p *Profile) IsAIEnabled() bool {
	return p.AIEnabled && (p.AISiliconFlowAPIKey != "" || p.AIOpenAIAPIKey != "" || p.AIOllamaBaseURL != "" || p.AIDeepSeekAPIKey != "")
}

// getEnvOrDefault returns the environment variable value or the default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// FromEnv loads configuration from environment variables.
func (p *Profile) FromEnv() {
	p.AIEnabled = os.Getenv("MEMOS_AI_ENABLED") == "true"
	p.AIEmbeddingProvider = getEnvOrDefault("MEMOS_AI_EMBEDDING_PROVIDER", "siliconflow")
	p.AILLMProvider = getEnvOrDefault("MEMOS_AI_LLM_PROVIDER", "deepseek")
	p.AISiliconFlowAPIKey = os.Getenv("MEMOS_AI_SILICONFLOW_API_KEY")
	p.AISiliconFlowBaseURL = getEnvOrDefault("MEMOS_AI_SILICONFLOW_BASE_URL", "https://api.siliconflow.cn/v1")
	p.AIDeepSeekAPIKey = os.Getenv("MEMOS_AI_DEEPSEEK_API_KEY")
	p.AIDeepSeekBaseURL = getEnvOrDefault("MEMOS_AI_DEEPSEEK_BASE_URL", "https://api.deepseek.com")
	p.AIOpenAIAPIKey = os.Getenv("MEMOS_AI_OPENAI_API_KEY")
	p.AIOpenAIBaseURL = getEnvOrDefault("MEMOS_AI_OPENAI_BASE_URL", "https://api.openai.com/v1")
	p.AIOllamaBaseURL = getEnvOrDefault("MEMOS_AI_OLLAMA_BASE_URL", "http://localhost:11434")
	p.AIEmbeddingModel = getEnvOrDefault("MEMOS_AI_EMBEDDING_MODEL", "BAAI/bge-m3")
	p.AIRerankModel = getEnvOrDefault("MEMOS_AI_RERANK_MODEL", "BAAI/bge-reranker-v2-m3")
	p.AILLMModel = getEnvOrDefault("MEMOS_AI_LLM_MODEL", "deepseek-chat")
}

func checkDataDir(dataDir string) (string, error) {
	// Convert to absolute path if relative path is supplied.
	if !filepath.IsAbs(dataDir) {
		relativeDir := filepath.Join(filepath.Dir(os.Args[0]), dataDir)
		absDir, err := filepath.Abs(relativeDir)
		if err != nil {
			return "", err
		}
		dataDir = absDir
	}

	// Trim trailing \ or / in case user supplies
	dataDir = strings.TrimRight(dataDir, "\\/")
	if _, err := os.Stat(dataDir); err != nil {
		return "", errors.Wrapf(err, "unable to access data folder %s", dataDir)
	}
	return dataDir, nil
}

func (p *Profile) Validate() error {
	if p.Mode != "demo" && p.Mode != "dev" && p.Mode != "prod" {
		p.Mode = "demo"
	}

	if p.Mode == "prod" && p.Data == "" {
		if runtime.GOOS == "windows" {
			p.Data = filepath.Join(os.Getenv("ProgramData"), "memos")
			if _, err := os.Stat(p.Data); os.IsNotExist(err) {
				if err := os.MkdirAll(p.Data, 0770); err != nil {
					slog.Error("failed to create data directory", slog.String("data", p.Data), slog.String("error", err.Error()))
					return err
				}
			}
		} else {
			p.Data = "/var/opt/memos"
		}
	}

	dataDir, err := checkDataDir(p.Data)
	if err != nil {
		slog.Error("failed to check dsn", slog.String("data", dataDir), slog.String("error", err.Error()))
		return err
	}

	p.Data = dataDir
	if p.Driver == "sqlite" && p.DSN == "" {
		dbFile := fmt.Sprintf("memos_%s.db", p.Mode)
		p.DSN = filepath.Join(dataDir, dbFile)
	}

	return nil
}
