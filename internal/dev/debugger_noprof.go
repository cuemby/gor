//go:build !debug
// +build !debug

package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	// pprof import excluded in non-debug builds for security
	"runtime"
	"runtime/debug"
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
	File      string                `json:"file"`
	Line      int                   `json:"line"`
	Condition string                `json:"condition"`
	HitCount  int                   `json:"hit_count"`
	Handler   func(context.Context) `json:"-"`
	Enabled   bool                  `json:"enabled"`
}

// CircularBuffer for log storage
type CircularBuffer struct {
	data []string
	size int
	head int
	mu   sync.RWMutex
}

// Metrics tracks runtime metrics
type Metrics struct {
	StartTime      time.Time
	RequestCount   int64
	ErrorCount     int64
	AverageLatency time.Duration
	mu             sync.RWMutex
}

// NewDebugger creates a new debugger instance
func NewDebugger(port int) *Debugger {
	return &Debugger{
		port:        port,
		breakpoints: make(map[string]*Breakpoint),
		logBuffer:   NewCircularBuffer(1000),
		metrics:     &Metrics{StartTime: time.Now()},
	}
}

// NewCircularBuffer creates a new circular buffer
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		data: make([]string, size),
		size: size,
	}
}

// Start starts the debug server
func (d *Debugger) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Debug endpoints (pprof endpoints NOT registered in non-debug builds)
	mux.HandleFunc("/debug/breakpoints", d.handleBreakpoints)
	mux.HandleFunc("/debug/logs", d.handleLogs)
	mux.HandleFunc("/debug/metrics", d.handleMetrics)
	mux.HandleFunc("/debug/gc", d.handleGC)
	mux.HandleFunc("/debug/stack", d.handleStack)

	d.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", d.port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Debug server starting on port %d (profiling disabled in non-debug build)", d.port)
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Debug server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return d.server.Shutdown(shutdownCtx)
}

// SetBreakpoint sets a breakpoint
func (d *Debugger) SetBreakpoint(file string, line int, condition string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := fmt.Sprintf("%s:%d", file, line)
	d.breakpoints[key] = &Breakpoint{
		File:      file,
		Line:      line,
		Condition: condition,
		Enabled:   true,
	}
}

// RemoveBreakpoint removes a breakpoint
func (d *Debugger) RemoveBreakpoint(file string, line int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := fmt.Sprintf("%s:%d", file, line)
	delete(d.breakpoints, key)
}

// CheckBreakpoint checks if execution should pause at a breakpoint
func (d *Debugger) CheckBreakpoint(file string, line int) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", file, line)
	bp, exists := d.breakpoints[key]
	if !exists || !bp.Enabled {
		return false
	}

	bp.HitCount++
	return true
}

// Log adds a message to the debug log
func (d *Debugger) Log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	d.logBuffer.Add(msg)
	log.Print(msg)
}

// Add adds a message to the circular buffer
func (cb *CircularBuffer) Add(msg string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.data[cb.head] = fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05.000"), msg)
	cb.head = (cb.head + 1) % cb.size
}

// GetAll returns all messages in the buffer
func (cb *CircularBuffer) GetAll() []string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	result := make([]string, 0, cb.size)
	for i := 0; i < cb.size; i++ {
		idx := (cb.head + i) % cb.size
		if cb.data[idx] != "" {
			result = append(result, cb.data[idx])
		}
	}
	return result
}

// HTTP handlers
func (d *Debugger) handleBreakpoints(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(d.breakpoints)
}

func (d *Debugger) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs := d.logBuffer.GetAll()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}

func (d *Debugger) handleMetrics(w http.ResponseWriter, r *http.Request) {
	d.metrics.mu.RLock()
	defer d.metrics.mu.RUnlock()

	metrics := map[string]interface{}{
		"uptime":          time.Since(d.metrics.StartTime).String(),
		"request_count":   d.metrics.RequestCount,
		"error_count":     d.metrics.ErrorCount,
		"average_latency": d.metrics.AverageLatency.String(),
		"goroutines":      runtime.NumGoroutine(),
		"memory":          getMemStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metrics)
}

func (d *Debugger) handleGC(w http.ResponseWriter, r *http.Request) {
	runtime.GC()
	debug.FreeOSMemory()

	fmt.Fprintf(w, "Garbage collection completed")
}

func (d *Debugger) handleStack(w http.ResponseWriter, r *http.Request) {
	buf := make([]byte, 1<<20) // 1MB buffer
	n := runtime.Stack(buf, true)

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(buf[:n])
}

func getMemStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc":       m.Alloc,
		"total_alloc": m.TotalAlloc,
		"sys":         m.Sys,
		"num_gc":      m.NumGC,
		"heap_alloc":  m.HeapAlloc,
		"heap_sys":    m.HeapSys,
	}
}

// CPUProfile is disabled in non-debug builds
func (d *Debugger) CPUProfile(duration time.Duration) error {
	return fmt.Errorf("CPU profiling is disabled in non-debug builds")
}

// HeapProfile is disabled in non-debug builds
func (d *Debugger) HeapProfile() error {
	return fmt.Errorf("heap profiling is disabled in non-debug builds")
}
