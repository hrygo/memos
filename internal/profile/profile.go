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
	// DSN points to where divinesense stores its own data
	DSN string
	// Driver is the database driver (sqlite or postgres)
	Driver string
	// Version is the current version of server
	Version string
	// InstanceURL is the url of your divinesense instance.
	InstanceURL string

	// AI Configuration
	AIEnabled            bool   // DIVINESENSE_AI_ENABLED (legacy: MEMOS_AI_ENABLED)
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

	// Attachment Processing Configuration
	OCREnabled          bool   // MEMOS_OCR_ENABLED (default: false)
	TextExtractEnabled  bool   // MEMOS_TEXTEXTRACT_ENABLED (default: false)
	TesseractPath       string // MEMOS_OCR_TESSERACT_PATH (default: tesseract)
	TessdataPath        string // MEMOS_OCR_TESSDATA_PATH (default: "")
	OCRLanguages        string // MEMOS_OCR_LANGUAGES (default: chi_sim+eng)
	TikaServerURL       string // MEMOS_TEXTEXTRACT_TIKA_URL (default: http://localhost:9998)
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
// Supports both DIVINESENSE_* (new) and MEMOS_* (legacy) prefixes.
func (p *Profile) FromEnv() {
	// Helper to get env value with legacy fallback
	// Skips empty values to allow defaults to take effect
	getEnvWithFallback := func(newKey, legacyKey string) string {
		if val := os.Getenv(newKey); val != "" {
			return val
		}
		return os.Getenv(legacyKey)
	}

	// Helper to get env value with legacy fallback and default value
	getEnvWithDefault := func(newKey, legacyKey, defaultValue string) string {
		if val := os.Getenv(newKey); val != "" {
			return val
		}
		if val := os.Getenv(legacyKey); val != "" {
			return val
		}
		return defaultValue
	}

	// Helper to get bool env value with legacy fallback
	getBoolEnvWithFallback := func(newKey, legacyKey string) bool {
		return getEnvWithFallback(newKey, legacyKey) == "true"
	}

	p.AIEnabled = getBoolEnvWithFallback("DIVINESENSE_AI_ENABLED", "MEMOS_AI_ENABLED")
	p.AIEmbeddingProvider = getEnvWithDefault("DIVINESENSE_AI_EMBEDDING_PROVIDER", "MEMOS_AI_EMBEDDING_PROVIDER", "siliconflow")
	p.AILLMProvider = getEnvWithDefault("DIVINESENSE_AI_LLM_PROVIDER", "MEMOS_AI_LLM_PROVIDER", "deepseek")
	p.AISiliconFlowAPIKey = getEnvWithFallback("DIVINESENSE_AI_SILICONFLOW_API_KEY", os.Getenv("MEMOS_AI_SILICONFLOW_API_KEY"))
	p.AISiliconFlowBaseURL = getEnvWithDefault("DIVINESENSE_AI_SILICONFLOW_BASE_URL", "MEMOS_AI_SILICONFLOW_BASE_URL", "https://api.siliconflow.cn/v1")
	p.AIDeepSeekAPIKey = getEnvWithFallback("DIVINESENSE_AI_DEEPSEEK_API_KEY", os.Getenv("MEMOS_AI_DEEPSEEK_API_KEY"))
	p.AIDeepSeekBaseURL = getEnvWithDefault("DIVINESENSE_AI_DEEPSEEK_BASE_URL", "MEMOS_AI_DEEPSEEK_BASE_URL", "https://api.deepseek.com")
	p.AIOpenAIAPIKey = getEnvWithFallback("DIVINESENSE_AI_OPENAI_API_KEY", os.Getenv("MEMOS_AI_OPENAI_API_KEY"))
	p.AIOpenAIBaseURL = getEnvWithDefault("DIVINESENSE_AI_OPENAI_BASE_URL", "MEMOS_AI_OPENAI_BASE_URL", "https://api.openai.com/v1")
	p.AIOllamaBaseURL = getEnvWithDefault("DIVINESENSE_AI_OLLAMA_BASE_URL", "MEMOS_AI_OLLAMA_BASE_URL", "http://localhost:11434")
	p.AIEmbeddingModel = getEnvWithDefault("DIVINESENSE_AI_EMBEDDING_MODEL", "MEMOS_AI_EMBEDDING_MODEL", "BAAI/bge-m3")
	p.AIRerankModel = getEnvWithDefault("DIVINESENSE_AI_RERANK_MODEL", "MEMOS_AI_RERANK_MODEL", "BAAI/bge-reranker-v2-m3")
	p.AILLMModel = getEnvWithDefault("DIVINESENSE_AI_LLM_MODEL", "MEMOS_AI_LLM_MODEL", "deepseek-chat")

	// Attachment processing configuration
	p.OCREnabled = getBoolEnvWithFallback("DIVINESENSE_OCR_ENABLED", "MEMOS_OCR_ENABLED")
	p.TextExtractEnabled = getBoolEnvWithFallback("DIVINESENSE_TEXTEXTRACT_ENABLED", "MEMOS_TEXTEXTRACT_ENABLED")
	p.TesseractPath = getEnvWithFallback("DIVINESENSE_OCR_TESSERACT_PATH", getEnvOrDefault("MEMOS_OCR_TESSERACT_PATH", "tesseract"))
	p.TessdataPath = getEnvWithFallback("DIVINESENSE_OCR_TESSDATA_PATH", os.Getenv("MEMOS_OCR_TESSDATA_PATH"))
	p.OCRLanguages = getEnvWithFallback("DIVINESENSE_OCR_LANGUAGES", getEnvOrDefault("MEMOS_OCR_LANGUAGES", "chi_sim+eng"))
	p.TikaServerURL = getEnvWithFallback("DIVINESENSE_TEXTEXTRACT_TIKA_URL", getEnvOrDefault("MEMOS_TEXTEXTRACT_TIKA_URL", "http://localhost:9998"))
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
			p.Data = filepath.Join(os.Getenv("ProgramData"), "divinesense")
			if _, err := os.Stat(p.Data); os.IsNotExist(err) {
				if err := os.MkdirAll(p.Data, 0770); err != nil {
					slog.Error("failed to create data directory", slog.String("data", p.Data), slog.String("error", err.Error()))
					return err
				}
			}
		} else {
			p.Data = "/var/opt/divinesense"
		}
	}

	dataDir, err := checkDataDir(p.Data)
	if err != nil {
		slog.Error("failed to check dsn", slog.String("data", dataDir), slog.String("error", err.Error()))
		return err
	}

	p.Data = dataDir
	if p.Driver == "sqlite" && p.DSN == "" {
		dbFile := fmt.Sprintf("divinesense_%s.db", p.Mode)
		p.DSN = filepath.Join(dataDir, dbFile)
	}

	return nil
}
