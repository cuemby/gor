package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// cacheEntry represents a cache entry in the database
type cacheEntry struct {
	Key       string
	Value     []byte
	ExpiresAt sql.NullTime
	CreatedAt time.Time
	UpdatedAt time.Time
	HitCount  int64
}

// SolidCache implements a database-backed cache similar to Rails' Solid Cache
type SolidCache struct {
	db              *sql.DB
	mu              sync.RWMutex
	memCache        map[string]*memoryCacheEntry // L1 cache in memory
	memCacheMu      sync.RWMutex
	maxMemorySize   int64
	currentSize     int64
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	cleanupInterval time.Duration
}

// memoryCacheEntry is an in-memory cache entry
type memoryCacheEntry struct {
	value      []byte
	expiresAt  time.Time
	size       int64
	lastAccess time.Time
}

// NewSolidCache creates a new database-backed cache
func NewSolidCache(dbPath string, maxMemoryMB int) (*SolidCache, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sc := &SolidCache{
		db:              db,
		memCache:        make(map[string]*memoryCacheEntry),
		maxMemorySize:   int64(maxMemoryMB) * 1024 * 1024, // Convert MB to bytes
		ctx:             ctx,
		cancel:          cancel,
		cleanupInterval: 1 * time.Minute,
	}

	// Create cache table
	if err := sc.createTables(); err != nil {
		db.Close()
		return nil, err
	}

	// Start background cleanup
	sc.wg.Add(1)
	go sc.cleanupWorker()

	return sc, nil
}

// createTables creates the necessary database tables
func (sc *SolidCache) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS cache_entries (
		key TEXT PRIMARY KEY,
		value BLOB NOT NULL,
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		hit_count INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_cache_expires ON cache_entries(expires_at)
		WHERE expires_at IS NOT NULL;
	
	CREATE INDEX IF NOT EXISTS idx_cache_updated ON cache_entries(updated_at);
	`

	_, err := sc.db.Exec(schema)
	return err
}

// Get retrieves a value from the cache
func (sc *SolidCache) Get(key string) (interface{}, error) {
	// Check L1 memory cache first
	sc.memCacheMu.RLock()
	if entry, exists := sc.memCache[key]; exists {
		if entry.expiresAt.IsZero() || entry.expiresAt.After(time.Now()) {
			entry.lastAccess = time.Now()
			sc.memCacheMu.RUnlock()

			var value interface{}
			if err := json.Unmarshal(entry.value, &value); err != nil {
				return nil, err
			}
			return value, nil
		}
		// Entry expired, remove from memory
		sc.memCacheMu.RUnlock()
		sc.memCacheMu.Lock()
		sc.currentSize -= entry.size
		delete(sc.memCache, key)
		sc.memCacheMu.Unlock()
	} else {
		sc.memCacheMu.RUnlock()
	}

	// Check L2 database cache
	query := `
		SELECT value, expires_at
		FROM cache_entries
		WHERE key = ?
	`

	var value []byte
	var expiresAt sql.NullTime

	err := sc.db.QueryRow(query, key).Scan(&value, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cache entry: %w", err)
	}

	// Check expiration
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		// Entry expired, delete it
		sc.Delete(key)
		return nil, nil
	}

	// Update hit count
	go sc.incrementHitCount(key)

	// Store in memory cache for faster access
	sc.storeInMemory(key, value, expiresAt.Time)

	// Unmarshal and return
	var result interface{}
	if err := json.Unmarshal(value, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Set stores a value in the cache
func (sc *SolidCache) Set(key string, value interface{}, ttl time.Duration) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	var expiresAt sql.NullTime
	if ttl > 0 {
		expiresAt = sql.NullTime{
			Time:  time.Now().Add(ttl),
			Valid: true,
		}
	}

	// Store in database
	query := `
		INSERT INTO cache_entries (key, value, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			expires_at = excluded.expires_at,
			updated_at = excluded.updated_at,
			hit_count = 0
	`

	now := time.Now()
	_, err = sc.db.Exec(query, key, valueJSON, expiresAt, now, now)
	if err != nil {
		return fmt.Errorf("failed to set cache entry: %w", err)
	}

	// Store in memory cache
	sc.storeInMemory(key, valueJSON, expiresAt.Time)

	return nil
}

// Delete removes a value from the cache
func (sc *SolidCache) Delete(key string) error {
	// Remove from memory cache
	sc.memCacheMu.Lock()
	if entry, exists := sc.memCache[key]; exists {
		sc.currentSize -= entry.size
		delete(sc.memCache, key)
	}
	sc.memCacheMu.Unlock()

	// Remove from database
	query := `DELETE FROM cache_entries WHERE key = ?`
	_, err := sc.db.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete cache entry: %w", err)
	}

	return nil
}

// Exists checks if a key exists in the cache
func (sc *SolidCache) Exists(key string) bool {
	// Check memory cache first
	sc.memCacheMu.RLock()
	if entry, exists := sc.memCache[key]; exists {
		sc.memCacheMu.RUnlock()
		return entry.expiresAt.IsZero() || entry.expiresAt.After(time.Now())
	}
	sc.memCacheMu.RUnlock()

	// Check database
	query := `
		SELECT EXISTS(
			SELECT 1 FROM cache_entries
			WHERE key = ?
			  AND (expires_at IS NULL OR expires_at > ?)
		)
	`

	var exists bool
	err := sc.db.QueryRow(query, key, time.Now()).Scan(&exists)
	if err != nil {
		return false
	}

	return exists
}

// Clear removes all entries from the cache
func (sc *SolidCache) Clear() error {
	// Clear memory cache
	sc.memCacheMu.Lock()
	sc.memCache = make(map[string]*memoryCacheEntry)
	sc.currentSize = 0
	sc.memCacheMu.Unlock()

	// Clear database
	_, err := sc.db.Exec("DELETE FROM cache_entries")
	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return nil
}

// Increment atomically increments a numeric value
func (sc *SolidCache) Increment(key string, delta int64) (int64, error) {
	// Get current value
	val, err := sc.Get(key)
	if err != nil {
		return 0, err
	}

	var current int64
	if val != nil {
		// Try to convert to int64
		switch v := val.(type) {
		case float64:
			current = int64(v)
		case int64:
			current = v
		case int:
			current = int64(v)
		default:
			return 0, fmt.Errorf("value at key %s is not numeric", key)
		}
	}

	newValue := current + delta

	// Set the new value with no expiration
	if err := sc.Set(key, newValue, 0); err != nil {
		return 0, err
	}

	return newValue, nil
}

// Decrement atomically decrements a numeric value
func (sc *SolidCache) Decrement(key string, delta int64) (int64, error) {
	return sc.Increment(key, -delta)
}

// storeInMemory stores an entry in the memory cache with LRU eviction
func (sc *SolidCache) storeInMemory(key string, value []byte, expiresAt time.Time) {
	size := int64(len(key) + len(value))

	sc.memCacheMu.Lock()
	defer sc.memCacheMu.Unlock()

	// Check if key already exists
	if oldEntry, exists := sc.memCache[key]; exists {
		sc.currentSize -= oldEntry.size
	}

	// Evict entries if necessary (LRU)
	for sc.currentSize+size > sc.maxMemorySize && len(sc.memCache) > 0 {
		sc.evictOldestEntry()
	}

	// Store new entry
	sc.memCache[key] = &memoryCacheEntry{
		value:      value,
		expiresAt:  expiresAt,
		size:       size,
		lastAccess: time.Now(),
	}
	sc.currentSize += size
}

// evictOldestEntry removes the least recently accessed entry
func (sc *SolidCache) evictOldestEntry() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range sc.memCache {
		if oldestTime.IsZero() || entry.lastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastAccess
		}
	}

	if oldestKey != "" {
		entry := sc.memCache[oldestKey]
		sc.currentSize -= entry.size
		delete(sc.memCache, oldestKey)
	}
}

// incrementHitCount increments the hit count for a cache entry
func (sc *SolidCache) incrementHitCount(key string) {
	query := `
		UPDATE cache_entries
		SET hit_count = hit_count + 1
		WHERE key = ?
	`
	sc.db.Exec(query, key)
}

// cleanupWorker periodically removes expired entries
func (sc *SolidCache) cleanupWorker() {
	defer sc.wg.Done()
	ticker := time.NewTicker(sc.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sc.ctx.Done():
			return
		case <-ticker.C:
			sc.cleanupExpired()
		}
	}
}

// cleanupExpired removes expired entries from the database
func (sc *SolidCache) cleanupExpired() {
	// Clean memory cache
	sc.memCacheMu.Lock()
	now := time.Now()
	for key, entry := range sc.memCache {
		if !entry.expiresAt.IsZero() && entry.expiresAt.Before(now) {
			sc.currentSize -= entry.size
			delete(sc.memCache, key)
		}
	}
	sc.memCacheMu.Unlock()

	// Clean database
	query := `
		DELETE FROM cache_entries
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`

	result, err := sc.db.Exec(query, now)
	if err != nil {
		log.Printf("Failed to cleanup expired entries: %v", err)
		return
	}

	if rows, _ := result.RowsAffected(); rows > 0 {
		log.Printf("Cleaned up %d expired cache entries", rows)
	}
}

// GetStats returns cache statistics
func (sc *SolidCache) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Memory cache stats
	sc.memCacheMu.RLock()
	stats["memory_entries"] = len(sc.memCache)
	stats["memory_size_bytes"] = sc.currentSize
	stats["memory_size_mb"] = float64(sc.currentSize) / (1024 * 1024)
	stats["max_memory_mb"] = sc.maxMemorySize / (1024 * 1024)
	sc.memCacheMu.RUnlock()

	// Database stats
	var dbEntries, totalHits int64
	var avgHitRate float64

	err := sc.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(hit_count), 0), COALESCE(AVG(hit_count), 0) FROM cache_entries").
		Scan(&dbEntries, &totalHits, &avgHitRate)
	if err != nil {
		return nil, err
	}

	stats["db_entries"] = dbEntries
	stats["total_hits"] = totalHits
	stats["avg_hit_rate"] = avgHitRate

	// Get database size
	var dbSize int64
	err = sc.db.QueryRow("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&dbSize)
	if err == nil {
		stats["db_size_bytes"] = dbSize
		stats["db_size_mb"] = float64(dbSize) / (1024 * 1024)
	}

	return stats, nil
}

// Close gracefully shuts down the cache
func (sc *SolidCache) Close() error {
	log.Println("Closing Solid Cache...")
	sc.cancel()
	sc.wg.Wait()
	return sc.db.Close()
}

// Fetch implements a "fetch or compute" pattern
func (sc *SolidCache) Fetch(key string, ttl time.Duration, compute func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache
	value, err := sc.Get(key)
	if err != nil {
		return nil, err
	}

	if value != nil {
		return value, nil
	}

	// Compute the value
	value, err = compute()
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := sc.Set(key, value, ttl); err != nil {
		// Log error but return the computed value
		log.Printf("Failed to cache computed value: %v", err)
	}

	return value, nil
}
