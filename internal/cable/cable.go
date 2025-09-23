package cable

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Message represents a pub/sub message
type Message struct {
	ID        string
	Channel   string
	Data      interface{}
	Metadata  map[string]string
	CreatedAt time.Time
}

// Subscription represents a channel subscription
type Subscription struct {
	ID       string
	Channel  string
	Handler  MessageHandler
	cancelFn context.CancelFunc
	messages chan *Message
}

// MessageHandler processes received messages
type MessageHandler func(ctx context.Context, msg *Message) error

// SolidCable implements a database-backed pub/sub system similar to Rails' Solid Cable
type SolidCable struct {
	db            *sql.DB
	subscriptions map[string]*Subscription
	mu            sync.RWMutex
	channels      map[string]map[string]*Subscription // channel -> subscriptionID -> subscription
	channelsMu    sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	pollInterval  time.Duration
	lastMessageID int64
}

// NewSolidCable creates a new database-backed pub/sub system
func NewSolidCable(dbPath string) (*SolidCable, error) {
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

	sc := &SolidCable{
		db:            db,
		subscriptions: make(map[string]*Subscription),
		channels:      make(map[string]map[string]*Subscription),
		ctx:           ctx,
		cancel:        cancel,
		pollInterval:  100 * time.Millisecond, // Fast polling for real-time
	}

	// Create messages table
	if err := sc.createTables(); err != nil {
		db.Close()
		return nil, err
	}

	// Get the latest message ID
	sc.getLatestMessageID()

	// Start message poller
	sc.wg.Add(1)
	go sc.messagePoller()

	// Start cleanup worker
	sc.wg.Add(1)
	go sc.cleanupWorker()

	return sc, nil
}

// createTables creates the necessary database tables
func (sc *SolidCable) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel TEXT NOT NULL,
		data TEXT NOT NULL,
		metadata TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel, id);
	CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);
	`

	_, err := sc.db.Exec(schema)
	return err
}

// getLatestMessageID retrieves the latest message ID from the database
func (sc *SolidCable) getLatestMessageID() {
	var id sql.NullInt64
	err := sc.db.QueryRow("SELECT MAX(id) FROM messages").Scan(&id)
	if err == nil && id.Valid {
		sc.lastMessageID = id.Int64
	}
}

// Publish sends a message to a channel
func (sc *SolidCable) Publish(ctx context.Context, channel string, data interface{}) error {
	return sc.PublishWithMetadata(ctx, channel, data, nil)
}

// PublishWithMetadata sends a message with metadata to a channel
func (sc *SolidCable) PublishWithMetadata(ctx context.Context, channel string, data interface{}, metadata map[string]string) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	var metadataJSON []byte
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO messages (channel, data, metadata)
		VALUES (?, ?, ?)
	`

	_, err = sc.db.Exec(query, channel, string(dataJSON), string(metadataJSON))
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published message to channel '%s'", channel)
	return nil
}

// Subscribe creates a subscription to a channel
func (sc *SolidCable) Subscribe(ctx context.Context, channel string, handler MessageHandler) (*Subscription, error) {
	subCtx, cancel := context.WithCancel(ctx)

	sub := &Subscription{
		ID:       fmt.Sprintf("%s-%d", channel, time.Now().UnixNano()),
		Channel:  channel,
		Handler:  handler,
		cancelFn: cancel,
		messages: make(chan *Message, 100),
	}

	// Register subscription
	sc.mu.Lock()
	sc.subscriptions[sub.ID] = sub
	sc.mu.Unlock()

	// Add to channel map
	sc.channelsMu.Lock()
	if sc.channels[channel] == nil {
		sc.channels[channel] = make(map[string]*Subscription)
	}
	sc.channels[channel][sub.ID] = sub
	sc.channelsMu.Unlock()

	// Start message processor for this subscription
	sc.wg.Add(1)
	go sc.processSubscription(subCtx, sub)

	log.Printf("Created subscription '%s' for channel '%s'", sub.ID, channel)
	return sub, nil
}

// SubscribePattern creates a subscription to channels matching a pattern
func (sc *SolidCable) SubscribePattern(ctx context.Context, pattern string, handler MessageHandler) (*Subscription, error) {
	// For simplicity, we'll treat patterns as exact channel matches with * wildcard
	// In a real implementation, you'd want proper pattern matching
	return sc.Subscribe(ctx, pattern, handler)
}

// Unsubscribe removes a subscription
func (sc *SolidCable) Unsubscribe(sub *Subscription) error {
	if sub == nil {
		return errors.New("subscription is nil")
	}

	// Cancel subscription context
	sub.cancelFn()

	// Remove from subscriptions map
	sc.mu.Lock()
	delete(sc.subscriptions, sub.ID)
	sc.mu.Unlock()

	// Remove from channels map
	sc.channelsMu.Lock()
	if channelSubs, exists := sc.channels[sub.Channel]; exists {
		delete(channelSubs, sub.ID)
		if len(channelSubs) == 0 {
			delete(sc.channels, sub.Channel)
		}
	}
	sc.channelsMu.Unlock()

	// Close message channel
	close(sub.messages)

	log.Printf("Unsubscribed '%s' from channel '%s'", sub.ID, sub.Channel)
	return nil
}

// processSubscription handles messages for a subscription
func (sc *SolidCable) processSubscription(ctx context.Context, sub *Subscription) {
	defer sc.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sub.messages:
			if !ok {
				return
			}

			// Process message with handler
			if err := sub.Handler(ctx, msg); err != nil {
				log.Printf("Error processing message in subscription '%s': %v", sub.ID, err)
			}
		}
	}
}

// messagePoller polls for new messages and distributes them to subscribers
func (sc *SolidCable) messagePoller() {
	defer sc.wg.Done()
	ticker := time.NewTicker(sc.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sc.ctx.Done():
			return
		case <-ticker.C:
			sc.pollMessages()
		}
	}
}

// pollMessages fetches new messages and distributes them
func (sc *SolidCable) pollMessages() {
	query := `
		SELECT id, channel, data, metadata, created_at
		FROM messages
		WHERE id > ?
		ORDER BY id
		LIMIT 100
	`

	rows, err := sc.db.Query(query, sc.lastMessageID)
	if err != nil {
		log.Printf("Failed to poll messages: %v", err)
		return
	}
	defer rows.Close()

	var messages []*Message
	var maxID int64

	for rows.Next() {
		var id int64
		var channel, dataStr string
		var metadataStr sql.NullString
		var createdAt time.Time

		if err := rows.Scan(&id, &channel, &dataStr, &metadataStr, &createdAt); err != nil {
			log.Printf("Failed to scan message: %v", err)
			continue
		}

		// Unmarshal data
		var data interface{}
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			log.Printf("Failed to unmarshal message data: %v", err)
			continue
		}

		// Unmarshal metadata
		var metadata map[string]string
		if metadataStr.Valid && metadataStr.String != "" {
			if err := json.Unmarshal([]byte(metadataStr.String), &metadata); err != nil {
				log.Printf("Failed to unmarshal message metadata: %v", err)
			}
		}

		msg := &Message{
			ID:        fmt.Sprintf("%d", id),
			Channel:   channel,
			Data:      data,
			Metadata:  metadata,
			CreatedAt: createdAt,
		}

		messages = append(messages, msg)
		if id > maxID {
			maxID = id
		}
	}

	// Update last message ID
	if maxID > 0 {
		sc.lastMessageID = maxID
	}

	// Distribute messages to subscribers
	sc.distributeMessages(messages)
}

// distributeMessages sends messages to appropriate subscribers
func (sc *SolidCable) distributeMessages(messages []*Message) {
	sc.channelsMu.RLock()
	defer sc.channelsMu.RUnlock()

	for _, msg := range messages {
		// Send to exact channel subscribers
		if subs, exists := sc.channels[msg.Channel]; exists {
			for _, sub := range subs {
				select {
				case sub.messages <- msg:
				default:
					// Channel full, log and skip
					log.Printf("Subscription '%s' message buffer full, skipping message", sub.ID)
				}
			}
		}

		// Handle pattern subscriptions (simplified - just check for "*" wildcard)
		if subs, exists := sc.channels["*"]; exists {
			for _, sub := range subs {
				select {
				case sub.messages <- msg:
				default:
					log.Printf("Subscription '%s' message buffer full, skipping message", sub.ID)
				}
			}
		}
	}
}

// cleanupWorker periodically removes old messages
func (sc *SolidCable) cleanupWorker() {
	defer sc.wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sc.ctx.Done():
			return
		case <-ticker.C:
			sc.cleanupOldMessages()
		}
	}
}

// cleanupOldMessages removes messages older than 24 hours
func (sc *SolidCable) cleanupOldMessages() {
	query := `
		DELETE FROM messages
		WHERE created_at < ?
	`

	cutoff := time.Now().Add(-24 * time.Hour)
	result, err := sc.db.Exec(query, cutoff)
	if err != nil {
		log.Printf("Failed to cleanup old messages: %v", err)
		return
	}

	if rows, _ := result.RowsAffected(); rows > 0 {
		log.Printf("Cleaned up %d old messages", rows)
	}
}

// GetStats returns pub/sub statistics
func (sc *SolidCable) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Subscription stats
	sc.mu.RLock()
	stats["total_subscriptions"] = len(sc.subscriptions)
	sc.mu.RUnlock()

	// Channel stats
	sc.channelsMu.RLock()
	channelStats := make(map[string]int)
	for channel, subs := range sc.channels {
		channelStats[channel] = len(subs)
	}
	stats["channels"] = channelStats
	stats["total_channels"] = len(sc.channels)
	sc.channelsMu.RUnlock()

	// Message stats
	var messageCount int64
	err := sc.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&messageCount)
	if err != nil {
		return nil, err
	}
	stats["total_messages"] = messageCount
	stats["last_message_id"] = sc.lastMessageID

	// Database size
	var dbSize int64
	err = sc.db.QueryRow("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&dbSize)
	if err == nil {
		stats["db_size_bytes"] = dbSize
		stats["db_size_mb"] = float64(dbSize) / (1024 * 1024)
	}

	return stats, nil
}

// Close gracefully shuts down the pub/sub system
func (sc *SolidCable) Close() error {
	log.Println("Closing Solid Cable...")

	// Cancel all subscriptions
	sc.mu.Lock()
	for _, sub := range sc.subscriptions {
		sub.cancelFn()
	}
	sc.mu.Unlock()

	// Cancel main context
	sc.cancel()

	// Wait for workers to finish
	sc.wg.Wait()

	return sc.db.Close()
}

// Broadcast sends a message to all subscribers
func (sc *SolidCable) Broadcast(ctx context.Context, data interface{}) error {
	return sc.Publish(ctx, "*", data)
}

// ChannelExists checks if a channel has any subscribers
func (sc *SolidCable) ChannelExists(channel string) bool {
	sc.channelsMu.RLock()
	defer sc.channelsMu.RUnlock()

	subs, exists := sc.channels[channel]
	return exists && len(subs) > 0
}

// ListChannels returns a list of active channels
func (sc *SolidCable) ListChannels() []string {
	sc.channelsMu.RLock()
	defer sc.channelsMu.RUnlock()

	channels := make([]string, 0, len(sc.channels))
	for channel := range sc.channels {
		channels = append(channels, channel)
	}
	return channels
}

// GetSubscriptionCount returns the number of subscriptions for a channel
func (sc *SolidCable) GetSubscriptionCount(channel string) int {
	sc.channelsMu.RLock()
	defer sc.channelsMu.RUnlock()

	if subs, exists := sc.channels[channel]; exists {
		return len(subs)
	}
	return 0
}
