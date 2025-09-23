package websocket

import (
	"log"
	"time"
)

// ChannelHandler defines the interface for handling channel events
type ChannelHandler interface {
	OnSubscribe(client *Client, channel string)
	OnUnsubscribe(client *Client, channel string)
	OnMessage(client *Client, channel string, data map[string]interface{})
}

// BaseChannel provides a default implementation of ChannelHandler
type BaseChannel struct{}

func (bc *BaseChannel) OnSubscribe(client *Client, channel string) {
	log.Printf("Client %s subscribed to %s", client.GetID(), channel)
}

func (bc *BaseChannel) OnUnsubscribe(client *Client, channel string) {
	log.Printf("Client %s unsubscribed from %s", client.GetID(), channel)
}

func (bc *BaseChannel) OnMessage(client *Client, channel string, data map[string]interface{}) {
	log.Printf("Message from %s on %s: %v", client.GetID(), channel, data)
}

// ChatChannel handles chat room functionality
type ChatChannel struct {
	BaseChannel
	hub *Hub
}

func NewChatChannel(hub *Hub) *ChatChannel {
	return &ChatChannel{hub: hub}
}

func (cc *ChatChannel) OnSubscribe(client *Client, channel string) {
	// Notify other users that someone joined
	cc.hub.BroadcastToChannel(channel, map[string]interface{}{
		"type":     "user_joined",
		"user":     client.GetID(),
		"userName": client.userName,
		"time":     time.Now().Format(time.RFC3339),
	})
}

func (cc *ChatChannel) OnUnsubscribe(client *Client, channel string) {
	// Notify other users that someone left
	cc.hub.BroadcastToChannel(channel, map[string]interface{}{
		"type":     "user_left",
		"user":     client.GetID(),
		"userName": client.userName,
		"time":     time.Now().Format(time.RFC3339),
	})
}

func (cc *ChatChannel) OnMessage(client *Client, channel string, data map[string]interface{}) {
	// Broadcast chat message to all users in the channel
	cc.hub.BroadcastToChannel(channel, map[string]interface{}{
		"type":     "chat_message",
		"user":     client.GetID(),
		"userName": client.userName,
		"message":  data["message"],
		"time":     time.Now().Format(time.RFC3339),
	})
}

// PresenceChannel tracks user presence
type PresenceChannel struct {
	BaseChannel
	hub      *Hub
	presence map[string]map[string]interface{} // channel -> user data
}

func NewPresenceChannel(hub *Hub) *PresenceChannel {
	return &PresenceChannel{
		hub:      hub,
		presence: make(map[string]map[string]interface{}),
	}
}

func (pc *PresenceChannel) OnSubscribe(client *Client, channel string) {
	if pc.presence[channel] == nil {
		pc.presence[channel] = make(map[string]interface{})
	}

	userData := map[string]interface{}{
		"id":       client.GetID(),
		"userName": client.userName,
		"joinedAt": time.Now(),
	}

	pc.presence[channel][client.GetID()] = userData

	// Send current presence list to the new client
	client.sendMessage("presence_list", map[string]interface{}{
		"channel": channel,
		"users":   pc.presence[channel],
	})

	// Notify others of new presence
	pc.hub.BroadcastToChannel(channel, map[string]interface{}{
		"type": "presence_join",
		"user": userData,
	})
}

func (pc *PresenceChannel) OnUnsubscribe(client *Client, channel string) {
	if pc.presence[channel] != nil {
		delete(pc.presence[channel], client.GetID())

		// Notify others of presence leave
		pc.hub.BroadcastToChannel(channel, map[string]interface{}{
			"type":   "presence_leave",
			"userID": client.GetID(),
		})

		if len(pc.presence[channel]) == 0 {
			delete(pc.presence, channel)
		}
	}
}

func (pc *PresenceChannel) OnMessage(client *Client, channel string, data map[string]interface{}) {
	// Handle presence-specific messages
	if data["action"] == "get_presence" {
		client.sendMessage("presence_list", map[string]interface{}{
			"channel": channel,
			"users":   pc.presence[channel],
		})
	}
}

// NotificationChannel handles real-time notifications
type NotificationChannel struct {
	BaseChannel
	hub *Hub
}

func NewNotificationChannel(hub *Hub) *NotificationChannel {
	return &NotificationChannel{hub: hub}
}

func (nc *NotificationChannel) OnMessage(client *Client, channel string, data map[string]interface{}) {
	// Send notification to specific user
	if targetUser, ok := data["targetUser"].(string); ok {
		nc.hub.SendToClient(targetUser, map[string]interface{}{
			"type":         "notification",
			"from":         client.GetID(),
			"notification": data["notification"],
			"time":         time.Now().Format(time.RFC3339),
		})
	} else {
		// Broadcast notification to all in channel
		nc.hub.BroadcastToChannel(channel, map[string]interface{}{
			"type":         "notification",
			"notification": data["notification"],
			"time":         time.Now().Format(time.RFC3339),
		})
	}
}

// LiveUpdateChannel for real-time data updates
type LiveUpdateChannel struct {
	BaseChannel
	hub *Hub
}

func NewLiveUpdateChannel(hub *Hub) *LiveUpdateChannel {
	return &LiveUpdateChannel{hub: hub}
}

func (lc *LiveUpdateChannel) OnMessage(client *Client, channel string, data map[string]interface{}) {
	// Broadcast live update to all subscribers
	lc.hub.BroadcastToChannel(channel, map[string]interface{}{
		"type":   "update",
		"entity": data["entity"],
		"action": data["action"], // created, updated, deleted
		"data":   data["data"],
		"time":   time.Now().Format(time.RFC3339),
	})
}
