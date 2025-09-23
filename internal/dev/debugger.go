package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"
)

// Debugger provides runtime debugging capabilities
type Debugger struct {
	server      *http.Server
	port        int
	breakpoints map[string]*Breakpoint
	logBuffer   *CircularBuffer
	metrics     *Metrics
	mu          sync.RWMutex
}

// Breakpoint represents a debug breakpoint
type Breakpoint struct {
	File      string                 `json:"file"`
	Line      int                    `json:"line"`
	Condition string                 `json:"condition"`
	HitCount  int                    `json:"hit_count"`
	Enabled   bool                   `json:"enabled"`
	Data      map[string]interface{} `json:"data"`
}

// CircularBuffer stores recent log entries
type CircularBuffer struct {
	entries []LogEntry
	size    int
	current int
	mu      sync.RWMutex
}

// LogEntry represents a log entry
type LogEntry struct {
	Time    time.Time   `json:"time"`
	Level   string      `json:"level"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	File    string      `json:"file"`
	Line    int         `json:"line"`
}

// Metrics tracks application metrics
type Metrics struct {
	RequestCount    int64                  `json:"request_count"`
	ErrorCount      int64                  `json:"error_count"`
	AvgResponseTime time.Duration          `json:"avg_response_time"`
	MemStats        runtime.MemStats       `json:"memory_stats"`
	Goroutines      int                    `json:"goroutines"`
	CustomMetrics   map[string]interface{} `json:"custom_metrics"`
	mu              sync.RWMutex
}

// NewDebugger creates a new debugger
func NewDebugger(port int) *Debugger {
	return &Debugger{
		port:        port,
		breakpoints: make(map[string]*Breakpoint),
		logBuffer:   NewCircularBuffer(1000),
		metrics:     NewMetrics(),
	}
}

// Start starts the debug server
func (d *Debugger) Start() error {
	mux := http.NewServeMux()

	// Debug UI
	mux.HandleFunc("/", d.handleDebugUI)

	// API endpoints
	mux.HandleFunc("/api/breakpoints", d.handleBreakpoints)
	mux.HandleFunc("/api/logs", d.handleLogs)
	mux.HandleFunc("/api/metrics", d.handleMetrics)
	mux.HandleFunc("/api/stack", d.handleStack)
	mux.HandleFunc("/api/heap", d.handleHeap)
	mux.HandleFunc("/api/goroutines", d.handleGoroutines)

	// Profiling endpoints
	mux.HandleFunc("/debug/pprof/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/debug/pprof/cmdline", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/debug/pprof/profile", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/debug/pprof/symbol", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/debug/pprof/trace", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	}))

	d.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", d.port),
		Handler: mux,
	}

	log.Printf("Debug server starting on http://localhost:%d\n", d.port)
	return d.server.ListenAndServe()
}

// Stop stops the debug server
func (d *Debugger) Stop() error {
	if d.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return d.server.Shutdown(ctx)
	}
	return nil
}

// handleDebugUI serves the debug UI
func (d *Debugger) handleDebugUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Gor Debugger</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
			margin: 0;
			padding: 20px;
			background: #1e1e1e;
			color: #d4d4d4;
		}
		.container {
			max-width: 1200px;
			margin: 0 auto;
		}
		h1 {
			color: #569cd6;
			border-bottom: 2px solid #3c3c3c;
			padding-bottom: 10px;
		}
		.tabs {
			display: flex;
			gap: 10px;
			margin-bottom: 20px;
			border-bottom: 1px solid #3c3c3c;
		}
		.tab {
			padding: 10px 20px;
			cursor: pointer;
			background: #2d2d2d;
			border: 1px solid #3c3c3c;
			border-bottom: none;
			color: #d4d4d4;
		}
		.tab.active {
			background: #1e1e1e;
			color: #569cd6;
		}
		.content {
			background: #2d2d2d;
			padding: 20px;
			border-radius: 4px;
		}
		.metric {
			display: inline-block;
			margin: 10px 20px 10px 0;
		}
		.metric-label {
			color: #9cdcfe;
			font-size: 12px;
			text-transform: uppercase;
		}
		.metric-value {
			font-size: 24px;
			font-weight: bold;
			color: #4ec9b0;
		}
		.log-entry {
			padding: 8px;
			border-bottom: 1px solid #3c3c3c;
			font-family: 'Monaco', 'Menlo', monospace;
			font-size: 12px;
		}
		.log-error {
			color: #f48771;
		}
		.log-info {
			color: #4ec9b0;
		}
		pre {
			background: #1e1e1e;
			padding: 10px;
			border-radius: 4px;
			overflow-x: auto;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>🐛 Gor Debugger</h1>
		
		<div class="tabs">
			<div class="tab active" onclick="showTab('metrics')">Metrics</div>
			<div class="tab" onclick="showTab('logs')">Logs</div>
			<div class="tab" onclick="showTab('stack')">Stack</div>
			<div class="tab" onclick="showTab('goroutines')">Goroutines</div>
			<div class="tab" onclick="showTab('profiling')">Profiling</div>
		</div>
		
		<div id="metrics" class="content">
			<div id="metrics-content">Loading...</div>
		</div>
		
		<div id="logs" class="content" style="display:none">
			<div id="logs-content">Loading...</div>
		</div>
		
		<div id="stack" class="content" style="display:none">
			<pre id="stack-content">Loading...</pre>
		</div>
		
		<div id="goroutines" class="content" style="display:none">
			<pre id="goroutines-content">Loading...</pre>
		</div>
		
		<div id="profiling" class="content" style="display:none">
			<h3>Profiling Tools</h3>
			<ul>
				<li><a href="/debug/pprof/heap" target="_blank">Heap Profile</a></li>
				<li><a href="/debug/pprof/goroutine" target="_blank">Goroutine Profile</a></li>
				<li><a href="/debug/pprof/profile?seconds=30" target="_blank">CPU Profile (30s)</a></li>
				<li><a href="/debug/pprof/trace?seconds=5" target="_blank">Trace (5s)</a></li>
			</ul>
		</div>
	</div>
	
	<script>
		function showTab(name) {
			document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
			document.querySelectorAll('.content').forEach(c => c.style.display = 'none');
			event.target.classList.add('active');
			document.getElementById(name).style.display = 'block';
			loadData(name);
		}
		
		function loadData(type) {
			fetch('/api/' + type)
				.then(r => r.json())
				.then(data => {
					const el = document.getElementById(type + '-content');
					if (type === 'metrics') {
						el.innerHTML = formatMetrics(data);
					} else if (type === 'logs') {
						el.innerHTML = formatLogs(data);
					} else {
						el.textContent = JSON.stringify(data, null, 2);
					}
				});
		}
		
		function formatMetrics(data) {
			return '' +
				'<div class="metric">' +
					'<div class="metric-label">Requests</div>' +
					'<div class="metric-value">' + data.request_count + '</div>' +
				'</div>' +
				'<div class="metric">' +
					'<div class="metric-label">Errors</div>' +
					'<div class="metric-value">' + data.error_count + '</div>' +
				'</div>' +
				'<div class="metric">' +
					'<div class="metric-label">Goroutines</div>' +
					'<div class="metric-value">' + data.goroutines + '</div>' +
				'</div>' +
				'<div class="metric">' +
					'<div class="metric-label">Memory (MB)</div>' +
					'<div class="metric-value">' + Math.round(data.memory_stats.Alloc / 1048576) + '</div>' +
				'</div>';
		}
		
		function formatLogs(data) {
			return data.map(function(log) {
				return '<div class="log-entry log-' + log.level + '">' +
					'<span>' + new Date(log.time).toISOString() + '</span> ' +
					'[' + log.level + '] ' + log.message +
				'</div>';
			}).join('');
		}
		
		// Auto-refresh
		setInterval(function() {
			const activeTab = document.querySelector('.tab.active').textContent.toLowerCase();
			loadData(activeTab);
		}, 2000);
		
		// Initial load
		loadData('metrics');
	</script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleBreakpoints handles breakpoint API
func (d *Debugger) handleBreakpoints(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	d.mu.RLock()
	defer d.mu.RUnlock()

	json.NewEncoder(w).Encode(d.breakpoints)
}

// handleLogs handles logs API
func (d *Debugger) handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	logs := d.logBuffer.GetAll()
	json.NewEncoder(w).Encode(logs)
}

// handleMetrics handles metrics API
func (d *Debugger) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	d.metrics.Update()
	json.NewEncoder(w).Encode(d.metrics)
}

// handleStack handles stack trace API
func (d *Debugger) handleStack(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stack := debug.Stack()
	json.NewEncoder(w).Encode(map[string]string{
		"stack": string(stack),
	})
}

// handleHeap handles heap dump API
func (d *Debugger) handleHeap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=heap.prof")

	err := pprof.WriteHeapProfile(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleGoroutines handles goroutines API
func (d *Debugger) handleGoroutines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	buf := make([]byte, 1<<20) // 1MB buffer
	n := runtime.Stack(buf, true)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      runtime.NumGoroutine(),
		"stacktrace": string(buf[:n]),
	})
}

// Log adds a log entry
func (d *Debugger) Log(level, message string, data interface{}) {
	entry := LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: message,
		Data:    data,
	}

	// Get caller info
	if _, file, line, ok := runtime.Caller(1); ok {
		entry.File = file
		entry.Line = line
	}

	d.logBuffer.Add(entry)
}

// RecordMetric records a custom metric
func (d *Debugger) RecordMetric(name string, value interface{}) {
	d.metrics.Record(name, value)
}

// NewCircularBuffer creates a new circular buffer
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		entries: make([]LogEntry, size),
		size:    size,
		current: 0,
	}
}

// Add adds an entry to the buffer
func (b *CircularBuffer) Add(entry LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.entries[b.current] = entry
	b.current = (b.current + 1) % b.size
}

// GetAll returns all non-empty entries
func (b *CircularBuffer) GetAll() []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []LogEntry
	for _, entry := range b.entries {
		if !entry.Time.IsZero() {
			result = append(result, entry)
		}
	}
	return result
}

// NewMetrics creates new metrics
func NewMetrics() *Metrics {
	return &Metrics{
		CustomMetrics: make(map[string]interface{}),
	}
}

// Update updates runtime metrics
func (m *Metrics) Update() {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime.ReadMemStats(&m.MemStats)
	m.Goroutines = runtime.NumGoroutine()
}

// Record records a custom metric
func (m *Metrics) Record(name string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CustomMetrics[name] = value
}

// IncrementRequests increments request count
func (m *Metrics) IncrementRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RequestCount++
}

// IncrementErrors increments error count
func (m *Metrics) IncrementErrors() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorCount++
}
