//go:build debug
// +build debug

package dev

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewDebugger(t *testing.T) {
	debugger := NewDebugger(8080)

	if debugger == nil {
		t.Fatal("NewDebugger returned nil")
		return
	}

	if debugger.port != 8080 {
		t.Errorf("Expected port 8080, got %d", debugger.port)
	}

	if debugger.breakpoints == nil {
		t.Error("Breakpoints map not initialized")
	}

	if debugger.logBuffer == nil {
		t.Error("LogBuffer not initialized")
	}

	if debugger.metrics == nil {
		t.Error("Metrics not initialized")
	}
}

func TestDebugger_Log(t *testing.T) {
	debugger := NewDebugger(8080)

	debugger.Log("INFO", "Test message", map[string]string{"key": "value"})

	// Since GetRecentLogs doesn't exist, we'll test through the buffer directly
	logs := debugger.logBuffer.GetAll()

	if len(logs) == 0 {
		t.Error("Expected at least one log entry")
	}

	found := false
	for _, log := range logs {
		if log.Message == "Test message" && log.Level == "INFO" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Log entry not found")
	}
}

func TestDebugger_RecordMetric(t *testing.T) {
	debugger := NewDebugger(8080)

	// Record a metric
	debugger.RecordMetric("test_metric", 42)

	// Since we can't directly access metrics, we'll test through the handler
	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()

	debugger.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the response contains metrics
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if customMetrics, ok := response["custom_metrics"].(map[string]interface{}); ok {
		if val, exists := customMetrics["test_metric"]; !exists || val != 42.0 {
			t.Error("Custom metric not found or incorrect value")
		}
	}
}

func TestDebugger_HandleLogs(t *testing.T) {
	debugger := NewDebugger(8080)

	// Add some logs
	debugger.Log("INFO", "Test 1", nil)
	debugger.Log("ERROR", "Test 2", nil)

	req := httptest.NewRequest("GET", "/api/logs", nil)
	w := httptest.NewRecorder()

	debugger.handleLogs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", ct)
	}
}

func TestDebugger_HandleStack(t *testing.T) {
	debugger := NewDebugger(8080)

	req := httptest.NewRequest("GET", "/api/stack", nil)
	w := httptest.NewRecorder()

	debugger.handleStack(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return stack trace
	body := w.Body.String()
	if !strings.Contains(body, "goroutine") {
		t.Error("Stack trace should contain 'goroutine'")
	}
}

func TestDebugger_HandleGoroutines(t *testing.T) {
	debugger := NewDebugger(8080)

	req := httptest.NewRequest("GET", "/api/goroutines", nil)
	w := httptest.NewRecorder()

	debugger.handleGoroutines(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return goroutine info
	body := w.Body.String()
	if body == "" {
		t.Error("Goroutines response should not be empty")
	}
}

func TestCircularBuffer(t *testing.T) {
	buffer := NewCircularBuffer(3)

	if buffer == nil {
		t.Fatal("NewCircularBuffer returned nil")
		return
	}

	if buffer.size != 3 {
		t.Errorf("Expected size 3, got %d", buffer.size)
	}

	// Add entries
	for i := 1; i <= 5; i++ {
		buffer.Add(LogEntry{
			Time:    time.Now(),
			Message: strings.Repeat("a", i),
		})
	}

	// Should have entries (circular buffer wraps around)
	entries := buffer.GetAll()
	if len(entries) == 0 {
		t.Error("Buffer should contain entries")
	}

	// Count non-zero time entries
	count := 0
	for _, entry := range entries {
		if !entry.Time.IsZero() {
			count++
		}
	}

	if count != 3 {
		t.Errorf("Expected 3 valid entries, got %d", count)
	}
}

func TestCircularBuffer_Add(t *testing.T) {
	buffer := NewCircularBuffer(10)

	entry := LogEntry{
		Time:    time.Now(),
		Level:   "INFO",
		Message: "Test entry",
	}

	buffer.Add(entry)

	entries := buffer.GetAll()
	found := false
	for _, e := range entries {
		if e.Message == "Test entry" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Added entry not found in buffer")
	}
}

func TestMetrics(t *testing.T) {
	metrics := NewMetrics()

	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
		return
	}

	if metrics.CustomMetrics == nil {
		t.Error("CustomMetrics not initialized")
	}

	// Test recording a metric
	metrics.Record("test", 123)

	if val, exists := metrics.CustomMetrics["test"]; !exists || val != 123 {
		t.Error("Metric not recorded correctly")
	}
}

func TestMetrics_IncrementRequests(t *testing.T) {
	metrics := NewMetrics()

	initial := metrics.RequestCount
	metrics.IncrementRequests()

	if metrics.RequestCount != initial+1 {
		t.Errorf("Expected RequestCount %d, got %d", initial+1, metrics.RequestCount)
	}
}

func TestMetrics_IncrementErrors(t *testing.T) {
	metrics := NewMetrics()

	initial := metrics.ErrorCount
	metrics.IncrementErrors()

	if metrics.ErrorCount != initial+1 {
		t.Errorf("Expected ErrorCount %d, got %d", initial+1, metrics.ErrorCount)
	}
}

func TestMetrics_Update(t *testing.T) {
	metrics := NewMetrics()

	metrics.Update()

	// After update, should have current goroutine count
	if metrics.Goroutines <= 0 {
		t.Error("Goroutines count should be positive after Update")
	}

	// Memory stats should be populated
	if metrics.MemStats.Alloc == 0 {
		t.Error("Memory stats should be populated after Update")
	}
}

// Benchmark tests
func BenchmarkDebugger_Log(b *testing.B) {
	debugger := NewDebugger(8080)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		debugger.Log("INFO", "Benchmark message", nil)
	}
}

func BenchmarkDebugger_RecordMetric(b *testing.B) {
	debugger := NewDebugger(8080)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		debugger.RecordMetric("bench_metric", i)
	}
}

func BenchmarkCircularBuffer_Add(b *testing.B) {
	buffer := NewCircularBuffer(1000)

	entry := LogEntry{
		Time:    time.Now(),
		Message: "Benchmark entry",
		Level:   "INFO",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Add(entry)
	}
}

func BenchmarkMetrics_Update(b *testing.B) {
	metrics := NewMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.Update()
	}
}
