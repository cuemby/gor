package gor

import (
	"context"
	"time"
)

// Cache defines the caching interface.
// Inspired by Rails 8's Solid Cache - disk-based caching with database fallback.
type Cache interface {
	// Basic cache operations
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Bulk operations
	GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error)
	SetMulti(ctx context.Context, items map[string]CacheItem) error
	DeleteMulti(ctx context.Context, keys []string) error

	// Pattern operations
	DeletePattern(ctx context.Context, pattern string) error
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Atomic operations
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	Decrement(ctx context.Context, key string, delta int64) (int64, error)

	// Advanced operations
	GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error)
	Touch(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Cache management
	Clear(ctx context.Context) error
	Stats(ctx context.Context) (CacheStats, error)
	Size(ctx context.Context) (int64, error)

	// Namespace support
	Namespace(prefix string) Cache

	// Tags for group invalidation
	Tagged(tags ...string) TaggedCache
}

// TaggedCache provides cache operations with tag-based invalidation.
type TaggedCache interface {
	Cache
	InvalidateTag(ctx context.Context, tag string) error
	InvalidateTags(ctx context.Context, tags []string) error
}

// CacheItem represents an item stored in the cache.
type CacheItem struct {
	Key       string        `json:"key"`
	Value     interface{}   `json:"value"`
	TTL       time.Duration `json:"ttl"`
	Tags      []string      `json:"tags,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	ExpiresAt *time.Time    `json:"expires_at,omitempty"`
}

// CacheStats provides statistics about cache performance.
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	Keys        int64   `json:"keys"`
	Size        int64   `json:"size"`
	Memory      int64   `json:"memory"`
	Disk        int64   `json:"disk"`
	Evictions   int64   `json:"evictions"`
	Connections int     `json:"connections"`
}

// CacheAdapter defines the interface for different cache backends.
type CacheAdapter interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Clear(ctx context.Context) error
	Close() error
}

// Multi-tier cache configuration
type CacheConfig struct {
	// Memory tier (L1)
	Memory MemoryCacheConfig `yaml:"memory"`

	// Disk tier (L2)
	Disk DiskCacheConfig `yaml:"disk"`

	// Database tier (L3)
	Database DatabaseCacheConfig `yaml:"database"`

	// Global settings
	DefaultTTL     time.Duration `yaml:"default_ttl"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
	MaxKeyLength   int           `yaml:"max_key_length"`
	Compression    bool          `yaml:"compression"`
	Encryption     bool          `yaml:"encryption"`
}

type MemoryCacheConfig struct {
	Enabled    bool          `yaml:"enabled"`
	MaxSize    int64         `yaml:"max_size"`    // in bytes
	MaxKeys    int           `yaml:"max_keys"`
	TTL        time.Duration `yaml:"ttl"`
	EvictPolicy string       `yaml:"evict_policy"` // LRU, LFU, FIFO
}

type DiskCacheConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Directory string        `yaml:"directory"`
	MaxSize   int64         `yaml:"max_size"`    // in bytes
	TTL       time.Duration `yaml:"ttl"`
	FileMode  uint32        `yaml:"file_mode"`
	DirMode   uint32        `yaml:"dir_mode"`
}

type DatabaseCacheConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Table     string        `yaml:"table"`
	TTL       time.Duration `yaml:"ttl"`
	BatchSize int           `yaml:"batch_size"`
}

// Serialization interface for cache values
type CacheSerializer interface {
	Serialize(value interface{}) ([]byte, error)
	Deserialize(data []byte, dest interface{}) error
}

// Cache middleware for request-response caching
type CacheMiddleware interface {
	HTTPCache(next HandlerFunc) HandlerFunc
	QueryCache(next func() interface{}) interface{}
	ViewCache(template string, data interface{}) (string, error)
}

// Fragment caching for partial template caching
type FragmentCache interface {
	Fragment(ctx context.Context, key string, ttl time.Duration, fn func() (string, error)) (string, error)
	InvalidateFragment(ctx context.Context, key string) error
}

// Cache warming interface
type CacheWarmer interface {
	Warm(ctx context.Context, keys []string) error
	WarmPattern(ctx context.Context, pattern string) error
	Schedule(ctx context.Context, keys []string, schedule string) error
}

// Cache events for monitoring
type CacheEvent struct {
	Type      CacheEventType `json:"type"`
	Key       string         `json:"key"`
	Tier      string         `json:"tier"`      // memory, disk, database
	Size      int64          `json:"size"`
	TTL       time.Duration  `json:"ttl"`
	Duration  time.Duration  `json:"duration"`
	Error     string         `json:"error,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

type CacheEventType string

const (
	CacheHit        CacheEventType = "hit"
	CacheMiss       CacheEventType = "miss"
	CacheSet        CacheEventType = "set"
	CacheDelete     CacheEventType = "delete"
	CacheEvict      CacheEventType = "evict"
	CacheExpire     CacheEventType = "expire"
	CachePromote    CacheEventType = "promote"    // move from lower to higher tier
	CacheDemote     CacheEventType = "demote"     // move from higher to lower tier
)

// Cache strategies
type CacheStrategy interface {
	ShouldCache(ctx context.Context, key string, value interface{}) bool
	TTL(ctx context.Context, key string, value interface{}) time.Duration
	Priority(ctx context.Context, key string, value interface{}) int
}

// Common cache strategies
type WriteThrough struct{}
type WriteBack struct{}
type WriteAround struct{}

func (w WriteThrough) ShouldCache(ctx context.Context, key string, value interface{}) bool {
	return true // Always cache on write
}

func (w WriteThrough) TTL(ctx context.Context, key string, value interface{}) time.Duration {
	return 1 * time.Hour // Default TTL
}

func (w WriteThrough) Priority(ctx context.Context, key string, value interface{}) int {
	return 1 // Normal priority
}

// Cache key builders for consistent key generation
type KeyBuilder interface {
	Build(components ...string) string
	BuildWithTags(tags []string, components ...string) string
}

// Default key builder implementation
type DefaultKeyBuilder struct {
	Prefix    string
	Separator string
}

func (kb DefaultKeyBuilder) Build(components ...string) string {
	// Implementation would join components with separator and add prefix
	return ""
}

func (kb DefaultKeyBuilder) BuildWithTags(tags []string, components ...string) string {
	// Implementation would include tags in key structure
	return ""
}

// Cache monitoring and alerting
type CacheMonitor interface {
	OnEvent(ctx context.Context, event CacheEvent) error
	HealthCheck(ctx context.Context) error
	Alert(ctx context.Context, level AlertLevel, message string) error
}

type AlertLevel string

const (
	AlertInfo    AlertLevel = "info"
	AlertWarning AlertLevel = "warning"
	AlertError   AlertLevel = "error"
	AlertCritical AlertLevel = "critical"
)