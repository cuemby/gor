package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client represents a websocket client connection
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	id   string

	// User information (if authenticated)
	userID   string
	userName string

	// Channel subscriptions
	subscriptions map[string]bool

	// Custom data
	data map[string]interface{}
}

// Message represents a websocket message
type Message struct {
	Type    string                 `json:"type"`
	Channel string                 `json:"channel,omitempty"`
	Action  string                 `json:"action,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		// Parse the message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			c.sendError("Invalid message format")
			continue
		}

		// Handle message based on type
		c.handleMessage(&msg)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case "subscribe":
		if msg.Channel != "" {
			c.hub.Subscribe(c, msg.Channel)
			c.sendSuccess("subscribe", map[string]interface{}{
				"channel": msg.Channel,
			})
		}

	case "unsubscribe":
		if msg.Channel != "" {
			c.hub.Unsubscribe(c, msg.Channel)
			c.sendSuccess("unsubscribe", map[string]interface{}{
				"channel": msg.Channel,
			})
		}

	case "message":
		if msg.Channel != "" {
			// Check if client is subscribed to the channel
			if !c.subscriptions[msg.Channel] {
				c.sendError("Not subscribed to channel")
				return
			}

			// Call channel handler if exists
			if handler, ok := c.hub.handlers[msg.Channel]; ok {
				handler.OnMessage(c, msg.Channel, msg.Data)
			} else {
				// Default: broadcast to channel
				c.hub.BroadcastToChannel(msg.Channel, msg.Data)
			}
		}

	case "ping":
		c.sendMessage("pong", nil)

	default:
		c.sendError("Unknown message type")
	}
}

// Send methods

func (c *Client) sendMessage(msgType string, data interface{}) {
	msg := map[string]interface{}{
		"type": msgType,
		"data": data,
		"time": time.Now().Unix(),
	}

	if encoded, err := json.Marshal(msg); err == nil {
		select {
		case c.send <- encoded:
		default:
			log.Printf("Client %s send buffer full", c.id)
		}
	}
}

func (c *Client) sendError(error string) {
	c.sendMessage("error", map[string]string{"error": error})
}

func (c *Client) sendSuccess(action string, data interface{}) {
	c.sendMessage("success", map[string]interface{}{
		"action": action,
		"data":   data,
	})
}

// SendToChannel sends a message to a specific channel
func (c *Client) SendToChannel(channel string, message interface{}) error {
	if !c.subscriptions[channel] {
		return fmt.Errorf("not subscribed to channel %s", channel)
	}

	msg := map[string]interface{}{
		"type":    "message",
		"channel": channel,
		"data":    message,
		"time":    time.Now().Unix(),
	}

	encoded, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.send <- encoded:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// GetID returns the client's ID
func (c *Client) GetID() string {
	return c.id
}

// SetUserInfo sets the user information for authenticated connections
func (c *Client) SetUserInfo(userID, userName string) {
	c.userID = userID
	c.userName = userName
}

// GetUserID returns the user ID if authenticated
func (c *Client) GetUserID() string {
	return c.userID
}

// SetData sets custom data for the client
func (c *Client) SetData(key string, value interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = value
}

// GetData gets custom data for the client
func (c *Client) GetData(key string) interface{} {
	if c.data == nil {
		return nil
	}
	return c.data[key]
}
