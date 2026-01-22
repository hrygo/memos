package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

// RedisCacheInterface defines the interface for Redis L2 cache.
// This is an optional interface - the system works fine with just memory cache.
// Redis is OPTIONAL and only needed for:
//   - Multi-instance deployments
//   - Cross-process cache sharing
//   - Persistent cache across restarts
//
// To enable Redis support:
//   1. Add redis dependency: go get github.com/redis/go-redis/v9
//   2. Uncomment the implementation below
//   3. Build with redis tag: go build -tags redis
type RedisCacheInterface interface {
	Set(ctx context.Context, key string, value any)
	SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration)
	Get(ctx context.Context, key string) (any, bool)
	Delete(ctx context.Context, key string)
	Clear(ctx context.Context)
	Close() error
}

// RedisCacheConfig holds the Redis connection configuration.
type RedisCacheConfig struct {
	Addr         string
	Password     string
	DB           int
	KeyPrefix    string
	DefaultTTL   time.Duration
	PoolSize     int
	MinIdleConns int
}

// DefaultRedisConfig returns the default Redis configuration.
func DefaultRedisConfig() *RedisCacheConfig {
	return &RedisCacheConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		KeyPrefix:    "memos:",
		DefaultTTL:   30 * time.Minute,
		PoolSize:     10,
		MinIdleConns: 2,
	}
}

// RedisConfigFromEnv creates Redis config from environment variables.
// Environment variables:
//   - MEMOS_CACHE_REDIS_ADDR: Redis address (default: localhost:6379)
//   - MEMOS_CACHE_REDIS_PASSWORD: Redis password (default: "")
//   - MEMOS_CACHE_REDIS_DB: Redis DB number (default: 0)
//   - MEMOS_CACHE_REDIS_PREFIX: Key prefix (default: "memos:")
func RedisConfigFromEnv() *RedisCacheConfig {
	config := DefaultRedisConfig()

	if addr := os.Getenv("MEMOS_CACHE_REDIS_ADDR"); addr != "" {
		config.Addr = addr
	}
	if password := os.Getenv("MEMOS_CACHE_REDIS_PASSWORD"); password != "" {
		config.Password = password
	}
	if prefix := os.Getenv("MEMOS_CACHE_REDIS_PREFIX"); prefix != "" {
		config.KeyPrefix = prefix
	}

	return config
}

// IsRedisEnabled checks if Redis caching should be enabled based on environment.
// Returns true if MEMOS_CACHE_REDIS_ADDR is set.
func IsRedisEnabled() bool {
	return os.Getenv("MEMOS_CACHE_REDIS_ADDR") != ""
}

// RedisCacheStats represents Redis cache statistics.
type RedisCacheStats struct {
	KeyCount int64   `json:"key_count"`
	Memory   int64   `json:"memory_bytes"`
	HitRate  float64 `json:"hit_rate"`
	Keyspace string  `json:"keyspace"`
}

// GenerateCacheKey generates a cache key from components.
func GenerateCacheKey(components ...string) string {
	return GenerateCacheKeyWithHash(components...)
}

// GenerateCacheKeyWithHash generates a cache key with hash for uniqueness.
func GenerateCacheKeyWithHash(components ...string) string {
	key := ""
	for i, c := range components {
		if i > 0 {
			key += ":"
		}
		key += c
	}
	// Add hash for uniqueness
	return fmt.Sprintf("%s:%s", key, KeyHash(key))
}

// KeyHash generates a SHA256 hash of the key for obfuscation.
func KeyHash(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])[:16]
}

// NilRedisCache is a no-op implementation of RedisCacheInterface.
// This allows the tiered cache to work without Redis.
type NilRedisCache struct{}

// NewNilRedisCache creates a no-op Redis cache.
func NewNilRedisCache() *NilRedisCache {
	return &NilRedisCache{}
}

func (n *NilRedisCache) Set(ctx context.Context, key string, value any) {}

func (n *NilRedisCache) SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) {}

func (n *NilRedisCache) Get(ctx context.Context, key string) (any, bool) {
	return nil, false
}

func (n *NilRedisCache) Delete(ctx context.Context, key string) {}

func (n *NilRedisCache) Clear(ctx context.Context) {}

func (n *NilRedisCache) Close() error {
	return nil
}

/*
// Redis implementation (requires github.com/redis/go-redis/v9)
// Uncomment and build with -tags redis to enable

// +build redis

package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"log/slog"
)

// RedisCache is a Redis-based cache implementation for L2 caching.
type RedisCache struct {
	client    *redis.Client
	keyPrefix string
	defaultTTL time.Duration
}

// NewRedisCache creates a new Redis cache.
func NewRedisCache(config *RedisCacheConfig) (*RedisCache, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, errors.Wrap(err, "failed to connect to Redis")
	}

	slog.Info("Redis cache connected", "addr", config.Addr)

	return &RedisCache{
		client:    client,
		keyPrefix: config.KeyPrefix,
		defaultTTL: config.DefaultTTL,
	}, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value any) {
	r.SetWithTTL(ctx, key, value, r.defaultTTL)
}

func (r *RedisCache) SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		slog.Warn("failed to marshal cache value", "key", key, "error", err)
		return
	}

	fullKey := r.fullKey(key)
	if err := r.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		slog.Warn("failed to set cache value", "key", key, "error", err)
	}
}

func (r *RedisCache) Get(ctx context.Context, key string) (any, bool) {
	fullKey := r.fullKey(key)
	data, err := r.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err != redis.Nil {
			slog.Warn("failed to get cache value", "key", key, "error", err)
		}
		return nil, false
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		var result interface{}
		if err2 := json.Unmarshal(data, &result); err2 != nil {
			slog.Warn("failed to unmarshal cache value", "key", key, "error", err)
			return nil, false
		}
		value = result
	}

	return value, true
}

func (r *RedisCache) Delete(ctx context.Context, key string) {
	fullKey := r.fullKey(key)
	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		slog.Warn("failed to delete cache value", "key", key, "error", err)
	}
}

func (r *RedisCache) Clear(ctx context.Context) {
	pattern := r.keyPrefix + "*"
	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 100 {
			r.client.Del(ctx, keys...)
			keys = keys[:0]
		}
	}
	if len(keys) > 0 {
		r.client.Del(ctx, keys...)
	}
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func (r *RedisCache) fullKey(key string) string {
	return r.keyPrefix + key
}
*/
