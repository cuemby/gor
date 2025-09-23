package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// TODO: Implement proper origin checking for production
		return true
	},
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Channel subscriptions
	channels map[string]map[*Client]bool

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Channel handlers
	handlers map[string]ChannelHandler
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		channels:   make(map[string]map[*Client]bool),
		handlers:   make(map[string]ChannelHandler),
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: %s", client.id)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// Remove from all channels
				for channel := range h.channels {
					delete(h.channels[channel], client)
				}
			}
			h.mu.Unlock()
			log.Printf("Client unregistered: %s", client.id)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close it
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ServeWS handles websocket requests from clients
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:           h,
		conn:          conn,
		send:          make(chan []byte, 256),
		id:            generateClientID(),
		subscriptions: make(map[string]bool),
	}

	client.hub.register <- client

	// Allow collection of memory referenced by the caller
	go client.writePump()
	go client.readPump()
}

// Subscribe adds a client to a channel
func (h *Hub) Subscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.channels[channel] == nil {
		h.channels[channel] = make(map[*Client]bool)
	}

	h.channels[channel][client] = true
	client.subscriptions[channel] = true

	// Call channel handler if exists
	if handler, ok := h.handlers[channel]; ok {
		handler.OnSubscribe(client, channel)
	}

	log.Printf("Client %s subscribed to channel %s", client.id, channel)
}

// Unsubscribe removes a client from a channel
func (h *Hub) Unsubscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.channels[channel]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channels, channel)
		}
	}

	delete(client.subscriptions, channel)

	// Call channel handler if exists
	if handler, ok := h.handlers[channel]; ok {
		handler.OnUnsubscribe(client, channel)
	}

	log.Printf("Client %s unsubscribed from channel %s", client.id, channel)
}

// BroadcastToChannel sends a message to all clients in a channel
func (h *Hub) BroadcastToChannel(channel string, message interface{}) error {
	data, err := json.Marshal(map[string]interface{}{
		"channel": channel,
		"message": message,
		"time":    time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	h.mu.RLock()
	clients := h.channels[channel]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- data:
		default:
			// Client's send channel is full
			log.Printf("Client %s send buffer full", client.id)
		}
	}

	return nil
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.id == clientID {
			select {
			case client.send <- data:
				return nil
			default:
				return fmt.Errorf("client %s send buffer full", clientID)
			}
		}
	}

	return fmt.Errorf("client %s not found", clientID)
}

// RegisterHandler registers a channel handler
func (h *Hub) RegisterHandler(channel string, handler ChannelHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[channel] = handler
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetChannelClients returns the number of clients in a channel
func (h *Hub) GetChannelClients(channel string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.channels[channel])
}

func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}
