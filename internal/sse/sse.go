package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Event represents an SSE event
type Event struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type,omitempty"`
	Data    interface{} `json:"data"`
	Retry   int         `json:"retry,omitempty"`
	Channel string      `json:"channel,omitempty"`
}

// Client represents an SSE client
type Client struct {
	ID            string
	EventChannel  chan *Event
	CloseChannel  chan bool
	Subscriptions map[string]bool
	mu            sync.RWMutex
}

// Server manages SSE connections
type Server struct {
	clients    map[string]*Client
	channels   map[string]map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Event
	mu         sync.RWMutex
}

// NewServer creates a new SSE server
func NewServer() *Server {
	return &Server{
		clients:    make(map[string]*Client),
		channels:   make(map[string]map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Event),
	}
}

// Run starts the SSE server event loop
func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client.ID] = client
			s.mu.Unlock()
			log.Printf("SSE client registered: %s", client.ID)

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client.ID]; ok {
				delete(s.clients, client.ID)
				close(client.EventChannel)
				close(client.CloseChannel)

				// Remove from all channels
				for channel := range client.Subscriptions {
					if channelClients, exists := s.channels[channel]; exists {
						delete(channelClients, client.ID)
						if len(channelClients) == 0 {
							delete(s.channels, channel)
						}
					}
				}
			}
			s.mu.Unlock()
			log.Printf("SSE client unregistered: %s", client.ID)

		case event := <-s.broadcast:
			if event.Channel != "" {
				s.BroadcastToChannel(event.Channel, event)
			} else {
				s.BroadcastToAll(event)
			}
		}
	}
}

// ServeHTTP handles SSE connections
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create new client
	client := &Client{
		ID:            generateClientID(),
		EventChannel:  make(chan *Event, 10),
		CloseChannel:  make(chan bool),
		Subscriptions: make(map[string]bool),
	}

	// Register client
	s.register <- client

	// Remove client on disconnect
	defer func() {
		s.unregister <- client
	}()

	// Send initial connection event
	client.EventChannel <- &Event{
		Type: "connected",
		Data: map[string]string{"message": "Connected to SSE server"},
	}

	// Check for channel subscriptions in query params
	if channels := r.URL.Query().Get("channels"); channels != "" {
		for _, channel := range splitChannels(channels) {
			s.Subscribe(client, channel)
		}
	}

	// Flush the response
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send events to client
	for {
		select {
		case event, ok := <-client.EventChannel:
			if !ok {
				return
			}

			// Format SSE message
			if event.ID != "" {
				fmt.Fprintf(w, "id: %s\n", event.ID)
			}
			if event.Type != "" {
				fmt.Fprintf(w, "event: %s\n", event.Type)
			}
			if event.Retry > 0 {
				fmt.Fprintf(w, "retry: %d\n", event.Retry)
			}

			// Marshal data to JSON
			data, err := json.Marshal(event.Data)
			if err != nil {
				log.Printf("Error marshaling event data: %v", err)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-client.CloseChannel:
			return

		case <-r.Context().Done():
			return
		}
	}
}

// Subscribe adds a client to a channel
func (s *Server) Subscribe(client *Client, channel string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.channels[channel] == nil {
		s.channels[channel] = make(map[string]*Client)
	}

	s.channels[channel][client.ID] = client
	client.Subscriptions[channel] = true

	// Send subscription confirmation
	client.EventChannel <- &Event{
		Type: "subscribed",
		Data: map[string]string{"channel": channel},
	}

	log.Printf("Client %s subscribed to channel %s", client.ID, channel)
}

// Unsubscribe removes a client from a channel
func (s *Server) Unsubscribe(client *Client, channel string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if channelClients, exists := s.channels[channel]; exists {
		delete(channelClients, client.ID)
		if len(channelClients) == 0 {
			delete(s.channels, channel)
		}
	}

	delete(client.Subscriptions, channel)

	// Send unsubscription confirmation
	client.EventChannel <- &Event{
		Type: "unsubscribed",
		Data: map[string]string{"channel": channel},
	}

	log.Printf("Client %s unsubscribed from channel %s", client.ID, channel)
}

// BroadcastToAll sends an event to all connected clients
func (s *Server) BroadcastToAll(event *Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		select {
		case client.EventChannel <- event:
		default:
			log.Printf("Client %s event channel full", client.ID)
		}
	}
}

// BroadcastToChannel sends an event to all clients in a channel
func (s *Server) BroadcastToChannel(channel string, event *Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if channelClients, exists := s.channels[channel]; exists {
		for _, client := range channelClients {
			select {
			case client.EventChannel <- event:
			default:
				log.Printf("Client %s event channel full", client.ID)
			}
		}
	}
}

// SendToClient sends an event to a specific client
func (s *Server) SendToClient(clientID string, event *Event) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if client, exists := s.clients[clientID]; exists {
		select {
		case client.EventChannel <- event:
			return nil
		default:
			return fmt.Errorf("client %s event channel full", clientID)
		}
	}

	return fmt.Errorf("client %s not found", clientID)
}

// GetConnectedClients returns the number of connected clients
func (s *Server) GetConnectedClients() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// GetChannelClients returns the number of clients in a channel
func (s *Server) GetChannelClients(channel string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.channels[channel])
}

// SendNotification sends a notification event
func (s *Server) SendNotification(channel string, title, message string, level string) {
	event := &Event{
		Type:    "notification",
		Channel: channel,
		Data: map[string]interface{}{
			"title":   title,
			"message": message,
			"level":   level, // info, success, warning, error
			"time":    time.Now().Format(time.RFC3339),
		},
	}

	if channel != "" {
		s.BroadcastToChannel(channel, event)
	} else {
		s.BroadcastToAll(event)
	}
}

// SendDataUpdate sends a data update event
func (s *Server) SendDataUpdate(channel string, entity string, action string, data interface{}) {
	event := &Event{
		Type:    "data_update",
		Channel: channel,
		Data: map[string]interface{}{
			"entity": entity,
			"action": action, // created, updated, deleted
			"data":   data,
			"time":   time.Now().Format(time.RFC3339),
		},
	}

	if channel != "" {
		s.BroadcastToChannel(channel, event)
	} else {
		s.BroadcastToAll(event)
	}
}

// SendProgress sends a progress update event
func (s *Server) SendProgress(channel string, taskID string, progress int, message string) {
	event := &Event{
		Type:    "progress",
		Channel: channel,
		Data: map[string]interface{}{
			"taskID":   taskID,
			"progress": progress,
			"message":  message,
			"time":     time.Now().Format(time.RFC3339),
		},
	}

	if channel != "" {
		s.BroadcastToChannel(channel, event)
	} else {
		s.BroadcastToAll(event)
	}
}

// Utility functions

func generateClientID() string {
	return fmt.Sprintf("sse_%d", time.Now().UnixNano())
}

func splitChannels(channels string) []string {
	var result []string
	for _, channel := range strings.Split(channels, ",") {
		channel = strings.TrimSpace(channel)
		if channel != "" {
			result = append(result, channel)
		}
	}
	return result
}