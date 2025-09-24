package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// MockChannelHandler for testing
type MockChannelHandler struct {
	OnSubscribeCalled   bool
	OnUnsubscribeCalled bool
	OnMessageCalled     bool
	mu                  sync.Mutex
}

func (m *MockChannelHandler) OnSubscribe(client *Client, channel string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OnSubscribeCalled = true
}

func (m *MockChannelHandler) OnUnsubscribe(client *Client, channel string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OnUnsubscribeCalled = true
}

func (m *MockChannelHandler) OnMessage(client *Client, channel string, data map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OnMessageCalled = true
}

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	if hub.clients == nil {
		t.Error("Clients map not initialized")
	}

	if hub.broadcast == nil {
		t.Error("Broadcast channel not initialized")
	}

	if hub.register == nil {
		t.Error("Register channel not initialized")
	}

	if hub.unregister == nil {
		t.Error("Unregister channel not initialized")
	}

	if hub.channels == nil {
		t.Error("Channels map not initialized")
	}

	if hub.handlers == nil {
		t.Error("Handlers map not initialized")
	}
}

func TestHub_Run(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Give the hub time to start
	time.Sleep(10 * time.Millisecond)

	// Create a mock client
	client := &Client{
		hub:           hub,
		send:          make(chan []byte, 256),
		id:            "test-client-1",
		subscriptions: make(map[string]bool),
	}

	// Test registration
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.mu.RLock()
	if !hub.clients[client] {
		t.Error("Client was not registered")
	}
	hub.mu.RUnlock()

	// Test unregistration
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	hub.mu.RLock()
	if hub.clients[client] {
		t.Error("Client was not unregistered")
	}
	hub.mu.RUnlock()
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create multiple clients
	clients := make([]*Client, 3)
	for i := range clients {
		clients[i] = &Client{
			hub:           hub,
			send:          make(chan []byte, 256),
			id:            generateClientID(),
			subscriptions: make(map[string]bool),
		}
		hub.register <- clients[i]
	}

	// Give time for registration
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	testMessage := []byte("test broadcast")
	hub.broadcast <- testMessage

	// Check all clients received the message
	for _, client := range clients {
		select {
		case msg := <-client.send:
			if string(msg) != string(testMessage) {
				t.Errorf("Expected message %s, got %s", testMessage, msg)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Client did not receive broadcast message")
		}
	}
}

func TestHub_Subscribe(t *testing.T) {
	hub := NewHub()

	client := &Client{
		hub:           hub,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	channel := "test-channel"
	hub.Subscribe(client, channel)

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Check hub's channel map
	if !hub.channels[channel][client] {
		t.Error("Client not added to channel")
	}

	// Check client's subscriptions
	if !client.subscriptions[channel] {
		t.Error("Channel not added to client subscriptions")
	}
}

func TestHub_Subscribe_WithHandler(t *testing.T) {
	hub := NewHub()
	handler := &MockChannelHandler{}

	channel := "test-channel"
	hub.RegisterHandler(channel, handler)

	client := &Client{
		hub:           hub,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	hub.Subscribe(client, channel)

	if !handler.OnSubscribeCalled {
		t.Error("OnSubscribe handler not called")
	}
}

func TestHub_Unsubscribe(t *testing.T) {
	hub := NewHub()

	client := &Client{
		hub:           hub,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	channel := "test-channel"

	// Subscribe first
	hub.Subscribe(client, channel)

	// Then unsubscribe
	hub.Unsubscribe(client, channel)

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Check hub's channel map
	if hub.channels[channel][client] {
		t.Error("Client not removed from channel")
	}

	// Check client's subscriptions
	if client.subscriptions[channel] {
		t.Error("Channel not removed from client subscriptions")
	}
}

func TestHub_Unsubscribe_WithHandler(t *testing.T) {
	hub := NewHub()
	handler := &MockChannelHandler{}

	channel := "test-channel"
	hub.RegisterHandler(channel, handler)

	client := &Client{
		hub:           hub,
		send:          make(chan []byte, 256),
		id:            "test-client",
		subscriptions: make(map[string]bool),
	}

	hub.Subscribe(client, channel)
	hub.Unsubscribe(client, channel)

	if !handler.OnUnsubscribeCalled {
		t.Error("OnUnsubscribe handler not called")
	}
}

func TestHub_BroadcastToChannel(t *testing.T) {
	hub := NewHub()

	// Create clients
	client1 := &Client{
		hub:           hub,
		send:          make(chan []byte, 256),
		id:            "client-1",
		subscriptions: make(map[string]bool),
	}
	client2 := &Client{
		hub:           hub,
		send:          make(chan []byte, 256),
		id:            "client-2",
		subscriptions: make(map[string]bool),
	}
	client3 := &Client{
		hub:           hub,
		send:          make(chan []byte, 256),
		id:            "client-3",
		subscriptions: make(map[string]bool),
	}

	// Subscribe clients to different channels
	hub.Subscribe(client1, "channel-1")
	hub.Subscribe(client2, "channel-1")
	hub.Subscribe(client3, "channel-2")

	// Broadcast to channel-1
	testMessage := "test message"
	err := hub.BroadcastToChannel("channel-1", testMessage)
	if err != nil {
		t.Errorf("BroadcastToChannel error: %v", err)
	}

	// Check client1 and client2 received message
	for _, client := range []*Client{client1, client2} {
		select {
		case msg := <-client.send:
			var data map[string]interface{}
			if err := json.Unmarshal(msg, &data); err != nil {
				t.Errorf("Failed to unmarshal message: %v", err)
			}
			if data["channel"] != "channel-1" {
				t.Errorf("Expected channel 'channel-1', got %v", data["channel"])
			}
			if data["message"] != testMessage {
				t.Errorf("Expected message %s, got %v", testMessage, data["message"])
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Client did not receive channel broadcast")
		}
	}

	// Check client3 did NOT receive message
	select {
	case <-client3.send:
		t.Error("Client3 should not have received message")
	case <-time.After(50 * time.Millisecond):
		// Expected behavior
	}
}

func TestHub_SendToClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:           hub,
		send:          make(chan []byte, 256),
		id:            "target-client",
		subscriptions: make(map[string]bool),
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Send message to specific client
	testMessage := map[string]string{"msg": "direct message"}
	err := hub.SendToClient("target-client", testMessage)
	if err != nil {
		t.Errorf("SendToClient error: %v", err)
	}

	// Check client received message
	select {
	case msg := <-client.send:
		var data map[string]string
		if err := json.Unmarshal(msg, &data); err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
		}
		if data["msg"] != "direct message" {
			t.Errorf("Expected 'direct message', got %s", data["msg"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Client did not receive direct message")
	}
}

func TestHub_SendToClient_NotFound(t *testing.T) {
	hub := NewHub()

	err := hub.SendToClient("nonexistent-client", "message")
	if err == nil {
		t.Error("Expected error for nonexistent client")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention client not found: %v", err)
	}
}

func TestHub_GetConnectedClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Initially no clients
	if count := hub.GetConnectedClients(); count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}

	// Add clients
	for i := 0; i < 5; i++ {
		client := &Client{
			hub:           hub,
			send:          make(chan []byte, 256),
			id:            generateClientID(),
			subscriptions: make(map[string]bool),
		}
		hub.register <- client
	}

	time.Sleep(50 * time.Millisecond)

	if count := hub.GetConnectedClients(); count != 5 {
		t.Errorf("Expected 5 clients, got %d", count)
	}
}

func TestHub_GetChannelClients(t *testing.T) {
	hub := NewHub()

	// Initially no clients in channel
	if count := hub.GetChannelClients("test-channel"); count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}

	// Add clients to channel
	for i := 0; i < 3; i++ {
		client := &Client{
			hub:           hub,
			send:          make(chan []byte, 256),
			id:            generateClientID(),
			subscriptions: make(map[string]bool),
		}
		hub.Subscribe(client, "test-channel")
	}

	if count := hub.GetChannelClients("test-channel"); count != 3 {
		t.Errorf("Expected 3 clients in channel, got %d", count)
	}
}

func TestHub_RegisterHandler(t *testing.T) {
	hub := NewHub()
	handler := &MockChannelHandler{}

	hub.RegisterHandler("test-channel", handler)

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	if hub.handlers["test-channel"] != handler {
		t.Error("Handler not registered correctly")
	}
}

func TestHub_ServeWS(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(w, r)
	}))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect as WebSocket client
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Give time for client to register
	time.Sleep(50 * time.Millisecond)

	// Verify client was registered
	if count := hub.GetConnectedClients(); count != 1 {
		t.Errorf("Expected 1 connected client, got %d", count)
	}
}

func TestGenerateClientID(t *testing.T) {
	id1 := generateClientID()
	id2 := generateClientID()

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	if !strings.HasPrefix(id1, "client_") {
		t.Error("ID should start with 'client_'")
	}

	if !strings.HasPrefix(id2, "client_") {
		t.Error("ID should start with 'client_'")
	}
}

// Benchmark tests
func BenchmarkHub_BroadcastToChannel(b *testing.B) {
	hub := NewHub()

	// Create clients
	for i := 0; i < 100; i++ {
		client := &Client{
			hub:           hub,
			send:          make(chan []byte, 256),
			id:            generateClientID(),
			subscriptions: make(map[string]bool),
		}
		hub.Subscribe(client, "bench-channel")
	}

	message := "benchmark message"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hub.BroadcastToChannel("bench-channel", message)
	}
}

func BenchmarkHub_Subscribe(b *testing.B) {
	hub := NewHub()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := &Client{
			hub:           hub,
			send:          make(chan []byte, 256),
			id:            generateClientID(),
			subscriptions: make(map[string]bool),
		}
		hub.Subscribe(client, "bench-channel")
	}
}

func BenchmarkHub_GetConnectedClients(b *testing.B) {
	hub := NewHub()

	// Add some clients
	for i := 0; i < 100; i++ {
		hub.mu.Lock()
		hub.clients[&Client{id: generateClientID()}] = true
		hub.mu.Unlock()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hub.GetConnectedClients()
	}
}
