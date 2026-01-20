package queryengine

import (
	"fmt"
)

// Config RAG 查询引擎配置
// P2 改进：配置化，将硬编码提取为可配置项
type Config struct {
	// 时间范围配置
	TimeRange TimeRangeConfig `json:"timeRange" yaml:"timeRange"`

	// 查询限制配置
	QueryLimits QueryLimitsConfig `json:"queryLimits" yaml:"queryLimits"`

	// 检索配置
	Retrieval RetrievalConfig `json:"retrieval" yaml:"retrieval"`

	// 评分配置
	Scoring ScoringConfig `json:"scoring" yaml:"scoring"`
}

// TimeRangeConfig 时间范围配置
type TimeRangeConfig struct {
	// 最大允许的未来时间（天数）
	MaxFutureDays int `json:"maxFutureDays" yaml:"maxFutureDays"`
	// 最大时间范围（天数）
	MaxRangeDays int `json:"maxRangeDays" yaml:"maxRangeDays"`
	// 时区（使用 UTC）
	Timezone string `json:"timezone" yaml:"timezone"`
}

// QueryLimitsConfig 查询限制配置
type QueryLimitsConfig struct {
	// 最大查询长度（字符数）
	MaxQueryLength int `json:"maxQueryLength" yaml:"maxQueryLength"`
	// 最大结果数量
	MaxResults int `json:"maxResults" yaml:"maxResults"`
	// 最小分数阈值
	MinScore float32 `json:"minScore" yaml:"minScore"`
}

// RetrievalConfig 检索配置
type RetrievalConfig struct {
	// 向量检索限制
	VectorLimit int `json:"vectorLimit" yaml:"vectorLimit"`
	// 混合检索限制
	HybridLimit int `json:"hybridLimit" yaml:"hybridLimit"`
	// 扩展检索限制
	ExpandLimit int `json:"expandLimit" yaml:"expandLimit"`
	// 是否启用 Reranker
	EnableReranker bool `json:"enableReranker" yaml:"enableReranker"`
	// 最大文档长度（字符数）
	MaxDocLength int `json:"maxDocLength" yaml:"maxDocLength"`
}

// ScoringConfig 评分配置
type ScoringConfig struct {
	// BM25 权重范围
	BM25WeightMin float32 `json:"bm25WeightMin" yaml:"bm25WeightMin"`
	BM25WeightMax float32 `json:"bm25WeightMax" yaml:"bm25WeightMax"`
	// 语义权重
	SemanticWeight float32 `json:"semanticWeight" yaml:"semanticWeight"`
	// 高质量阈值
	HighQualityThreshold float32 `json:"highQualityThreshold" yaml:"highQualityThreshold"`
	// 中等质量阈值
	MediumQualityThreshold float32 `json:"mediumQualityThreshold" yaml:"mediumQualityThreshold"`
	// 分数差距阈值（用于判断是否需要重排）
	ScoreGapThreshold float32 `json:"scoreGapThreshold" yaml:"scoreGapThreshold"`
	// 最小重排结果数
	MinRerankResults int `json:"minRerankResults" yaml:"minRerankResults"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		TimeRange: TimeRangeConfig{
			MaxFutureDays: 30,
			MaxRangeDays: 90,
			Timezone:     "UTC",
		},
		QueryLimits: QueryLimitsConfig{
			MaxQueryLength: 1000,
			MaxResults:     20,
			MinScore:       0.5,
		},
		Retrieval: RetrievalConfig{
			VectorLimit:     5,
			HybridLimit:     20,
			ExpandLimit:     20,
			EnableReranker:  true,
			MaxDocLength:    5000,
		},
		Scoring: ScoringConfig{
			BM25WeightMin:          0.3,
			BM25WeightMax:          0.7,
			SemanticWeight:         0.5,
			HighQualityThreshold:   0.90,
			MediumQualityThreshold: 0.70,
			ScoreGapThreshold:      0.15,
			MinRerankResults:       5,
		},
	}
}

// ApplyConfig 应用配置到 QueryRouter
// P2 改进：支持运行时配置更新
func (r *QueryRouter) ApplyConfig(config *Config) {
	r.configMutex.Lock()
	defer r.configMutex.Unlock()
	r.config = config
}

// GetConfig 获取当前配置（线程安全）
func (r *QueryRouter) GetConfig() *Config {
	r.configMutex.RLock()
	defer r.configMutex.RUnlock()
	return r.config
}

// ValidateConfig 验证配置有效性
func ValidateConfig(config *Config) error {
	// 验证时间范围配置
	if config.TimeRange.MaxFutureDays < 0 || config.TimeRange.MaxFutureDays > 365 {
		return ErrInvalidConfig{Field: "TimeRange.MaxFutureDays", Value: config.TimeRange.MaxFutureDays}
	}
	if config.TimeRange.MaxRangeDays < 1 || config.TimeRange.MaxRangeDays > 365 {
		return ErrInvalidConfig{Field: "TimeRange.MaxRangeDays", Value: config.TimeRange.MaxRangeDays}
	}

	// 验证查询限制配置
	if config.QueryLimits.MaxQueryLength < 10 || config.QueryLimits.MaxQueryLength > 10000 {
		return ErrInvalidConfig{Field: "QueryLimits.MaxQueryLength", Value: config.QueryLimits.MaxQueryLength}
	}
	if config.QueryLimits.MaxResults < 1 || config.QueryLimits.MaxResults > 1000 {
		return ErrInvalidConfig{Field: "QueryLimits.MaxResults", Value: config.QueryLimits.MaxResults}
	}
	if config.QueryLimits.MinScore < 0 || config.QueryLimits.MinScore > 1 {
		return ErrInvalidConfig{Field: "QueryLimits.MinScore", Value: config.QueryLimits.MinScore}
	}

	// 验证检索配置
	if config.Retrieval.VectorLimit < 1 || config.Retrieval.VectorLimit > 1000 {
		return ErrInvalidConfig{Field: "Retrieval.VectorLimit", Value: config.Retrieval.VectorLimit}
	}
	if config.Retrieval.MaxDocLength < 100 || config.Retrieval.MaxDocLength > 100000 {
		return ErrInvalidConfig{Field: "Retrieval.MaxDocLength", Value: config.Retrieval.MaxDocLength}
	}

	// 验证评分配置
	if config.Scoring.HighQualityThreshold < 0 || config.Scoring.HighQualityThreshold > 1 {
		return ErrInvalidConfig{Field: "Scoring.HighQualityThreshold", Value: config.Scoring.HighQualityThreshold}
	}
	if config.Scoring.MediumQualityThreshold < 0 || config.Scoring.MediumQualityThreshold > 1 {
		return ErrInvalidConfig{Field: "Scoring.MediumQualityThreshold", Value: config.Scoring.MediumQualityThreshold}
	}

	return nil
}

// ErrInvalidConfig 配置无效错误
type ErrInvalidConfig struct {
	Field string
	Value interface{}
}

func (e ErrInvalidConfig) Error() string {
	return fmt.Sprintf("invalid config field '%s': %v", e.Field, e.Value)
}
