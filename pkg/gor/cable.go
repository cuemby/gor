// Copyright (c) 2025 Cuemby
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package gor

import (
	"context"
	"net/http"
	"time"
)

// Cable defines the real-time messaging interface.
// Inspired by Rails 8's Solid Cable - database-backed WebSocket pub/sub system.
type Cable interface {
	// Connection management
	HandleWebSocket(w http.ResponseWriter, r *http.Request) error
	HandleSSE(w http.ResponseWriter, r *http.Request) error

	// Channel management
	Subscribe(ctx context.Context, connectionID, channel string, params map[string]interface{}) error
	Unsubscribe(ctx context.Context, connectionID, channel string) error
	UnsubscribeAll(ctx context.Context, connectionID string) error

	// Message broadcasting
	Broadcast(ctx context.Context, channel string, message interface{}) error
	BroadcastTo(ctx context.Context, connectionIDs []string, message interface{}) error
	BroadcastToUser(ctx context.Context, userID string, message interface{}) error

	// Connection tracking
	ConnectionCount(ctx context.Context) (int, error)
	ChannelConnections(ctx context.Context, channel string) ([]string, error)
	UserConnections(ctx context.Context, userID string) ([]string, error)

	// Server management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Stats(ctx context.Context) (CableStats, error)
}

// Connection represents a WebSocket or SSE connection.
type Connection interface {
	// Connection identification
	ID() string
	UserID() string
	IPAddress() string
	UserAgent() string

	// Connection state
	IsConnected() bool
	ConnectedAt() time.Time
	LastPing() time.Time

	// Message handling
	Send(message interface{}) error
	SendJSON(data interface{}) error
	SendText(text string) error

	// Channel management
	Subscribe(channel string, params map[string]interface{}) error
	Unsubscribe(channel string) error
	Channels() []string

	// Connection control
	Close() error
	Ping() error

	// Context and metadata
	Context() context.Context
	Metadata() map[string]interface{}
	SetMetadata(key string, value interface{})
}

// Channel defines the interface for real-time channels.
type Channel interface {
	// Channel identification
	Name() string
	Pattern() string

	// Subscription lifecycle
	OnSubscribe(ctx context.Context, conn Connection, params map[string]interface{}) error
	OnUnsubscribe(ctx context.Context, conn Connection) error
	OnMessage(ctx context.Context, conn Connection, message interface{}) error

	// Authorization
	Authorize(ctx context.Context, conn Connection, params map[string]interface{}) bool

	// Message filtering
	Filter(ctx context.Context, conn Connection, message interface{}) bool

	// Channel-specific logic
	BeforeBroadcast(ctx context.Context, message interface{}) (interface{}, error)
	AfterBroadcast(ctx context.Context, message interface{}, connections []Connection) error
}

// BaseChannel provides default implementation for common channel functionality.
type BaseChannel struct {
	ChannelName    string
	ChannelPattern string
}

func (c *BaseChannel) Name() string    { return c.ChannelName }
func (c *BaseChannel) Pattern() string { return c.ChannelPattern }

func (c *BaseChannel) OnSubscribe(ctx context.Context, conn Connection, params map[string]interface{}) error {
	return nil
}

func (c *BaseChannel) OnUnsubscribe(ctx context.Context, conn Connection) error {
	return nil
}

func (c *BaseChannel) OnMessage(ctx context.Context, conn Connection, message interface{}) error {
	return nil
}

func (c *BaseChannel) Authorize(ctx context.Context, conn Connection, params map[string]interface{}) bool {
	return true // Allow by default
}

func (c *BaseChannel) Filter(ctx context.Context, conn Connection, message interface{}) bool {
	return true // Send to all by default
}

func (c *BaseChannel) BeforeBroadcast(ctx context.Context, message interface{}) (interface{}, error) {
	return message, nil // No transformation by default
}

func (c *BaseChannel) AfterBroadcast(ctx context.Context, message interface{}, connections []Connection) error {
	return nil
}

// Message represents a real-time message.
type Message struct {
	ID        string                 `json:"id"`
	Type      MessageType            `json:"type"`
	Channel   string                 `json:"channel"`
	Data      map[string]interface{} `json:"data"`
	UserID    string                 `json:"user_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	TTL       time.Duration          `json:"ttl,omitempty"`
}

type MessageType string

const (
	MessageBroadcast   MessageType = "broadcast"
	MessageSubscribe   MessageType = "subscribe"
	MessageUnsubscribe MessageType = "unsubscribe"
	MessagePing        MessageType = "ping"
	MessagePong        MessageType = "pong"
	MessageError       MessageType = "error"
	MessageConfirm     MessageType = "confirm"
	MessageReject      MessageType = "reject"
)

// CableStats provides statistics about cable performance.
type CableStats struct {
	Connections CableConnectionStats  `json:"connections"`
	Channels    CableChannelStats     `json:"channels"`
	Messages    CableMessageStats     `json:"messages"`
	Performance CablePerformanceStats `json:"performance"`
	Memory      CableMemoryStats      `json:"memory"`
}

type CableConnectionStats struct {
	Total         int `json:"total"`
	WebSocket     int `json:"websocket"`
	SSE           int `json:"sse"`
	Authenticated int `json:"authenticated"`
	Anonymous     int `json:"anonymous"`
}

type CableChannelStats struct {
	Total         int `json:"total"`
	Subscriptions int `json:"subscriptions"`
	Private       int `json:"private"`
	Public        int `json:"public"`
	Presence      int `json:"presence"`
}

type CableMessageStats struct {
	Sent      int64   `json:"sent"`
	Received  int64   `json:"received"`
	Broadcast int64   `json:"broadcast"`
	Failed    int64   `json:"failed"`
	Rate      float64 `json:"rate"` // messages per second
}

type CablePerformanceStats struct {
	AverageLatency    time.Duration `json:"average_latency"`
	MessageThroughput float64       `json:"message_throughput"`
	ConnectionTime    time.Duration `json:"connection_time"`
	BroadcastTime     time.Duration `json:"broadcast_time"`
}

type CableMemoryStats struct {
	Connections int64 `json:"connections"`
	Channels    int64 `json:"channels"`
	Messages    int64 `json:"messages"`
	Total       int64 `json:"total"`
}

// Channel types for common use cases
type PresenceChannel struct {
	BaseChannel
	members map[string]interface{}
}

func (c *PresenceChannel) OnSubscribe(ctx context.Context, conn Connection, params map[string]interface{}) error {
	// Add user to presence list
	c.members[conn.UserID()] = params
	// Broadcast presence update
	return nil
}

func (c *PresenceChannel) OnUnsubscribe(ctx context.Context, conn Connection) error {
	// Remove user from presence list
	delete(c.members, conn.UserID())
	// Broadcast presence update
	return nil
}

type PrivateChannel struct {
	BaseChannel
}

func (c *PrivateChannel) Authorize(ctx context.Context, conn Connection, params map[string]interface{}) bool {
	// Check if user has access to private channel
	return conn.UserID() != ""
}

type ChatChannel struct {
	BaseChannel
}

func (c *ChatChannel) OnMessage(ctx context.Context, conn Connection, message interface{}) error {
	// Process chat message, save to database, broadcast to channel
	return nil
}

// Cable adapter interface for different backends
type CableAdapter interface {
	// Connection management
	RegisterConnection(conn Connection) error
	UnregisterConnection(connectionID string) error
	GetConnection(connectionID string) (Connection, error)

	// Channel subscription
	Subscribe(connectionID, channel string, params map[string]interface{}) error
	Unsubscribe(connectionID, channel string) error

	// Message broadcasting
	Broadcast(channel string, message interface{}) error
	BroadcastToConnections(connectionIDs []string, message interface{}) error

	// Statistics
	ConnectionCount() int
	ChannelSubscriptions(channel string) []string
}

// Cable events for monitoring and hooks
type CableEvent struct {
	Type         CableEventType `json:"type"`
	ConnectionID string         `json:"connection_id"`
	UserID       string         `json:"user_id,omitempty"`
	Channel      string         `json:"channel,omitempty"`
	Message      interface{}    `json:"message,omitempty"`
	Error        string         `json:"error,omitempty"`
	Duration     time.Duration  `json:"duration,omitempty"`
	Timestamp    time.Time      `json:"timestamp"`
}

type CableEventType string

const (
	CableConnected       CableEventType = "connected"
	CableDisconnected    CableEventType = "disconnected"
	CableSubscribed      CableEventType = "subscribed"
	CableUnsubscribed    CableEventType = "unsubscribed"
	CableMessageSent     CableEventType = "message_sent"
	CableMessageReceived CableEventType = "message_received"
	CableBroadcast       CableEventType = "broadcast"
	CableError           CableEventType = "error"
)

// Authentication and authorization
type CableAuth interface {
	AuthenticateConnection(r *http.Request) (string, error) // Returns user ID
	AuthorizeChannel(userID, channel string, params map[string]interface{}) bool
}

// Rate limiting for connections and messages
type CableRateLimit interface {
	AllowConnection(ip string) bool
	AllowMessage(connectionID string) bool
	AllowBroadcast(channel string) bool
}

// Message persistence for delivery guarantees
type MessageStore interface {
	Store(ctx context.Context, message Message) error
	Retrieve(ctx context.Context, channel string, since time.Time) ([]Message, error)
	Delete(ctx context.Context, messageID string) error
	Cleanup(ctx context.Context, before time.Time) error
}

// Horizontal scaling support
type CableCluster interface {
	// Node management
	JoinCluster(nodeID string) error
	LeaveCluster(nodeID string) error
	Nodes() []string

	// Cross-node broadcasting
	BroadcastToCluster(message interface{}) error
	BroadcastToNode(nodeID string, message interface{}) error

	// Load balancing
	RouteConnection(r *http.Request) (string, error) // Returns node ID
	BalanceLoad() error
}
