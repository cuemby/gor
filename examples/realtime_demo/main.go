package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/cuemby/gor/internal/sse"
	"github.com/cuemby/gor/internal/websocket"
)

var wsHub *websocket.Hub
var sseServer *sse.Server

func main() {
	fmt.Println("\n🌐 Gor Framework - Real-time Features Demo")
	fmt.Println("===========================================")
	fmt.Println("Visit http://localhost:8084")
	fmt.Println("Press Ctrl+C to stop")

	// Initialize WebSocket hub
	wsHub = websocket.NewHub()
	go wsHub.Run()

	// Register WebSocket channels
	wsHub.RegisterHandler("chat", websocket.NewChatChannel(wsHub))
	wsHub.RegisterHandler("presence", websocket.NewPresenceChannel(wsHub))
	wsHub.RegisterHandler("notifications", websocket.NewNotificationChannel(wsHub))
	wsHub.RegisterHandler("updates", websocket.NewLiveUpdateChannel(wsHub))

	// Initialize SSE server
	sseServer = sse.NewServer()
	go sseServer.Run()

	// HTTP routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/ws", wsHub.ServeWS)
	http.HandleFunc("/sse", sseServer.ServeHTTP)
	http.HandleFunc("/api/broadcast", handleBroadcast)
	http.HandleFunc("/api/notify", handleNotify)
	http.HandleFunc("/api/update", handleUpdate)

	// Simulate some background activity
	go simulateActivity()

	// Start server with proper timeouts for security
	server := &http.Server{
		Addr:         ":8084",
		Handler:      http.DefaultServeMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Real-time server starting on :8084")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("home").Parse(homeHTML))
	tmpl.Execute(w, nil)
}

func handleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	message := r.FormValue("message")
	channel := r.FormValue("channel")

	// Broadcast via WebSocket
	wsHub.BroadcastToChannel(channel, map[string]interface{}{
		"type":    "broadcast",
		"message": message,
		"time":    time.Now().Format(time.RFC3339),
	})

	// Broadcast via SSE
	sseServer.BroadcastToChannel(channel, &sse.Event{
		Type: "broadcast",
		Data: map[string]interface{}{
			"message": message,
			"time":    time.Now().Format(time.RFC3339),
		},
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Broadcasted to channel: %s", channel)
}

func handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	title := r.FormValue("title")
	message := r.FormValue("message")
	level := r.FormValue("level")

	// Send notification via SSE
	sseServer.SendNotification("", title, message, level)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Notification sent")
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entity := r.FormValue("entity")
	action := r.FormValue("action")

	// Send data update via SSE
	sseServer.SendDataUpdate("updates", entity, action, map[string]interface{}{
		"id":        time.Now().Unix(),
		"timestamp": time.Now().Format(time.RFC3339),
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Update sent for %s", entity)
}

func simulateActivity() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	counter := 0
	for range ticker.C {
		counter++

		// Send periodic updates via SSE
		sseServer.SendProgress("", "background-task", (counter*10)%100,
			fmt.Sprintf("Processing step %d", counter))

		// Send periodic notifications
		if counter%3 == 0 {
			sseServer.SendNotification("", "System Update",
				fmt.Sprintf("Background task completed iteration %d", counter), "info")
		}

		// Broadcast to WebSocket
		if counter%2 == 0 {
			wsHub.BroadcastToChannel("updates", map[string]interface{}{
				"type":      "auto_update",
				"counter":   counter,
				"timestamp": time.Now().Format(time.RFC3339),
			})
		}
	}
}

const homeHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Gor Real-time Demo</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            background: white;
            border-radius: 10px;
            padding: 30px;
            margin-bottom: 20px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        }
        h1 { color: #333; margin-bottom: 10px; }
        .subtitle { color: #666; }
        .grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
        }
        .card {
            background: white;
            border-radius: 10px;
            padding: 20px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        }
        h2 {
            color: #667eea;
            margin-bottom: 15px;
            padding-bottom: 10px;
            border-bottom: 2px solid #f0f0f0;
        }
        .status {
            display: inline-block;
            padding: 5px 10px;
            border-radius: 5px;
            font-size: 0.9em;
            margin-bottom: 15px;
        }
        .status.connected {
            background: #d4edda;
            color: #155724;
        }
        .status.disconnected {
            background: #f8d7da;
            color: #721c24;
        }
        .messages {
            height: 300px;
            overflow-y: auto;
            border: 1px solid #e0e0e0;
            border-radius: 5px;
            padding: 10px;
            margin-bottom: 15px;
            background: #f9f9f9;
        }
        .message {
            padding: 8px;
            margin-bottom: 5px;
            background: white;
            border-radius: 5px;
            border-left: 3px solid #667eea;
        }
        .message.notification {
            border-left-color: #ffc107;
        }
        .message.error {
            border-left-color: #dc3545;
        }
        .message.success {
            border-left-color: #28a745;
        }
        .controls {
            display: flex;
            gap: 10px;
            margin-bottom: 15px;
        }
        input[type="text"] {
            flex: 1;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
        }
        button {
            background: #667eea;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 5px;
            cursor: pointer;
        }
        button:hover {
            background: #5a67d8;
        }
        .progress-bar {
            width: 100%;
            height: 20px;
            background: #e0e0e0;
            border-radius: 10px;
            overflow: hidden;
            margin-top: 10px;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #667eea, #764ba2);
            transition: width 0.3s ease;
        }
        .user-list {
            list-style: none;
            padding: 0;
        }
        .user-list li {
            padding: 5px 10px;
            background: #f0f0f0;
            margin-bottom: 5px;
            border-radius: 5px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🌐 Gor Real-time Features Demo</h1>
            <p class="subtitle">WebSockets and Server-Sent Events in action</p>
        </div>

        <div class="grid">
            <!-- WebSocket Demo -->
            <div class="card">
                <h2>🔌 WebSocket Connection</h2>
                <div id="ws-status" class="status disconnected">Disconnected</div>
                
                <div class="controls">
                    <button onclick="connectWS()">Connect</button>
                    <button onclick="disconnectWS()">Disconnect</button>
                </div>

                <h3>Chat Channel</h3>
                <div id="ws-messages" class="messages"></div>
                <div class="controls">
                    <input type="text" id="ws-input" placeholder="Type a message..." onkeypress="if(event.key==='Enter') sendWSMessage()">
                    <button onclick="sendWSMessage()">Send</button>
                </div>

                <h3>Presence Channel</h3>
                <ul id="presence-list" class="user-list">
                    <li>No users online</li>
                </ul>
            </div>

            <!-- SSE Demo -->
            <div class="card">
                <h2>📡 Server-Sent Events</h2>
                <div id="sse-status" class="status disconnected">Disconnected</div>
                
                <div class="controls">
                    <button onclick="connectSSE()">Connect</button>
                    <button onclick="disconnectSSE()">Disconnect</button>
                </div>

                <h3>Event Stream</h3>
                <div id="sse-messages" class="messages"></div>

                <h3>Progress Tracking</h3>
                <div class="progress-bar">
                    <div id="progress-fill" class="progress-fill" style="width: 0%"></div>
                </div>
                <p id="progress-text">No active tasks</p>
            </div>
        </div>

        <!-- Control Panel -->
        <div class="card" style="margin-top: 20px;">
            <h2>🎮 Control Panel</h2>
            <div class="controls">
                <input type="text" id="broadcast-message" placeholder="Broadcast message...">
                <select id="broadcast-channel">
                    <option value="chat">Chat</option>
                    <option value="notifications">Notifications</option>
                    <option value="updates">Updates</option>
                </select>
                <button onclick="broadcast()">Broadcast</button>
            </div>
            <div class="controls" style="margin-top: 10px;">
                <input type="text" id="notify-title" placeholder="Notification title...">
                <input type="text" id="notify-message" placeholder="Notification message...">
                <select id="notify-level">
                    <option value="info">Info</option>
                    <option value="success">Success</option>
                    <option value="warning">Warning</option>
                    <option value="error">Error</option>
                </select>
                <button onclick="sendNotification()">Send Notification</button>
            </div>
        </div>
    </div>

    <script>
        let ws = null;
        let eventSource = null;

        // WebSocket functions
        function connectWS() {
            if (ws && ws.readyState === WebSocket.OPEN) {
                console.log('Already connected');
                return;
            }

            ws = new WebSocket('ws://localhost:8084/ws');

            ws.onopen = function() {
                document.getElementById('ws-status').className = 'status connected';
                document.getElementById('ws-status').textContent = 'Connected';
                addWSMessage('Connected to WebSocket server', 'success');

                // Subscribe to channels
                ws.send(JSON.stringify({
                    type: 'subscribe',
                    channel: 'chat'
                }));
                ws.send(JSON.stringify({
                    type: 'subscribe',
                    channel: 'presence'
                }));
                ws.send(JSON.stringify({
                    type: 'subscribe',
                    channel: 'updates'
                }));
            };

            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);
                console.log('WS message:', data);
                
                if (data.channel === 'presence' && data.message) {
                    updatePresence(data.message);
                } else if (data.type === 'success' || data.type === 'error') {
                    addWSMessage(JSON.stringify(data.data), data.type);
                } else {
                    addWSMessage(JSON.stringify(data), 'message');
                }
            };

            ws.onclose = function() {
                document.getElementById('ws-status').className = 'status disconnected';
                document.getElementById('ws-status').textContent = 'Disconnected';
                addWSMessage('Disconnected from WebSocket server', 'error');
            };

            ws.onerror = function(error) {
                addWSMessage('WebSocket error: ' + error, 'error');
            };
        }

        function disconnectWS() {
            if (ws) {
                ws.close();
                ws = null;
            }
        }

        function sendWSMessage() {
            const input = document.getElementById('ws-input');
            const message = input.value.trim();
            
            if (!message || !ws || ws.readyState !== WebSocket.OPEN) {
                return;
            }

            ws.send(JSON.stringify({
                type: 'message',
                channel: 'chat',
                data: {
                    message: message
                }
            }));

            input.value = '';
        }

        function addWSMessage(message, type = 'message') {
            const messagesDiv = document.getElementById('ws-messages');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + type;
            messageDiv.textContent = message;
            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        function updatePresence(data) {
            const list = document.getElementById('presence-list');
            list.innerHTML = '';
            
            if (data.users && Object.keys(data.users).length > 0) {
                for (const userId in data.users) {
                    const li = document.createElement('li');
                    li.textContent = data.users[userId].userName || userId;
                    list.appendChild(li);
                }
            } else {
                const li = document.createElement('li');
                li.textContent = 'No users online';
                list.appendChild(li);
            }
        }

        // SSE functions
        function connectSSE() {
            if (eventSource) {
                console.log('Already connected');
                return;
            }

            eventSource = new EventSource('/sse?channels=notifications,updates,progress');

            eventSource.onopen = function() {
                document.getElementById('sse-status').className = 'status connected';
                document.getElementById('sse-status').textContent = 'Connected';
                addSSEMessage('Connected to SSE server', 'success');
            };

            eventSource.onmessage = function(event) {
                const data = JSON.parse(event.data);
                addSSEMessage(JSON.stringify(data), 'message');
            };

            eventSource.addEventListener('notification', function(event) {
                const data = JSON.parse(event.data);
                addSSEMessage('Notification: ' + data.title + ' - ' + data.message, 'notification');
            });

            eventSource.addEventListener('progress', function(event) {
                const data = JSON.parse(event.data);
                updateProgress(data.progress, data.message);
            });

            eventSource.addEventListener('data_update', function(event) {
                const data = JSON.parse(event.data);
                addSSEMessage('Data Update: ' + data.entity + ' ' + data.action, 'message');
            });

            eventSource.onerror = function(error) {
                document.getElementById('sse-status').className = 'status disconnected';
                document.getElementById('sse-status').textContent = 'Disconnected';
                addSSEMessage('SSE connection error', 'error');
            };
        }

        function disconnectSSE() {
            if (eventSource) {
                eventSource.close();
                eventSource = null;
                document.getElementById('sse-status').className = 'status disconnected';
                document.getElementById('sse-status').textContent = 'Disconnected';
            }
        }

        function addSSEMessage(message, type = 'message') {
            const messagesDiv = document.getElementById('sse-messages');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + type;
            messageDiv.textContent = message;
            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        function updateProgress(progress, message) {
            document.getElementById('progress-fill').style.width = progress + '%';
            document.getElementById('progress-text').textContent = message || 'Processing...';
        }

        // Control panel functions
        function broadcast() {
            const message = document.getElementById('broadcast-message').value;
            const channel = document.getElementById('broadcast-channel').value;
            
            if (!message) return;

            fetch('/api/broadcast', {
                method: 'POST',
                headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                body: 'message=' + encodeURIComponent(message) + '&channel=' + channel
            }).then(response => response.text())
              .then(result => console.log(result));

            document.getElementById('broadcast-message').value = '';
        }

        function sendNotification() {
            const title = document.getElementById('notify-title').value;
            const message = document.getElementById('notify-message').value;
            const level = document.getElementById('notify-level').value;
            
            if (!title || !message) return;

            fetch('/api/notify', {
                method: 'POST',
                headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                body: 'title=' + encodeURIComponent(title) + 
                      '&message=' + encodeURIComponent(message) +
                      '&level=' + level
            }).then(response => response.text())
              .then(result => console.log(result));

            document.getElementById('notify-title').value = '';
            document.getElementById('notify-message').value = '';
        }

        // Auto-connect on load
        window.onload = function() {
            connectWS();
            connectSSE();
        };
    </script>
</body>
</html>
`
