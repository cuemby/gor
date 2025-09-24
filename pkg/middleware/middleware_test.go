package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cuemby/gor/pkg/gor"
)

// Helper function to create test context
func createTestContext(method, path string) *gor.Context {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	return &gor.Context{
		Request:  req,
		Response: w,
		Context:  context.Background(),
	}
}

func TestLogger(t *testing.T) {
	middleware := Logger()

	called := false
	handler := func(ctx *gor.Context) error {
		called = true
		return nil
	}

	ctx := createTestContext("GET", "/test")
	err := middleware(handler)(ctx)

	if err != nil {
		t.Errorf("Logger middleware returned error: %v", err)
	}

	if !called {
		t.Error("Handler was not called")
	}
}

func TestRecovery(t *testing.T) {
	middleware := Recovery()

	// Test panic recovery
	panicHandler := func(ctx *gor.Context) error {
		panic("test panic")
	}

	ctx := createTestContext("GET", "/test")
	err := middleware(panicHandler)(ctx)

	// Should not return error (panic is recovered)
	if err != nil {
		t.Errorf("Recovery middleware should not return error: %v", err)
	}

	// Check response
	w := ctx.Response.(*httptest.ResponseRecorder)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	// Test normal flow (no panic)
	normalHandler := func(ctx *gor.Context) error {
		ctx.Response.WriteHeader(http.StatusOK)
		return nil
	}

	ctx2 := createTestContext("GET", "/test")
	err = middleware(normalHandler)(ctx2)

	if err != nil {
		t.Errorf("Recovery middleware returned error for normal flow: %v", err)
	}

	w2 := ctx2.Response.(*httptest.ResponseRecorder)
	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name     string
		options  CORSOptions
		origin   string
		method   string
		expected int
	}{
		{
			name: "Allow all origins",
			options: CORSOptions{
				AllowOrigin:  "*",
				AllowMethods: "GET,POST",
				AllowHeaders: "Content-Type",
			},
			origin:   "http://example.com",
			method:   "GET",
			expected: http.StatusOK,
		},
		{
			name: "Specific origin allowed",
			options: CORSOptions{
				AllowOrigin: "http://allowed.com",
			},
			origin:   "http://allowed.com",
			method:   "GET",
			expected: http.StatusOK,
		},
		{
			name:     "Preflight request",
			options:  CORSOptions{AllowOrigin: "*"},
			origin:   "http://example.com",
			method:   "OPTIONS",
			expected: http.StatusNoContent,
		},
		{
			name: "With credentials",
			options: CORSOptions{
				AllowOrigin:      "*",
				AllowCredentials: true,
				MaxAge:           3600,
			},
			origin:   "http://example.com",
			method:   "GET",
			expected: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := CORS(tt.options)

			handler := func(ctx *gor.Context) error {
				ctx.Response.WriteHeader(http.StatusOK)
				return nil
			}

			ctx := createTestContext(tt.method, "/test")
			if tt.origin != "" {
				ctx.Request.Header.Set("Origin", tt.origin)
			}

			err := middleware(handler)(ctx)
			if err != nil {
				t.Errorf("CORS middleware returned error: %v", err)
			}

			w := ctx.Response.(*httptest.ResponseRecorder)
			if w.Code != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, w.Code)
			}

			// Check CORS headers
			if tt.options.AllowOrigin != "" {
				header := w.Header().Get("Access-Control-Allow-Origin")
				if header == "" {
					t.Error("Access-Control-Allow-Origin header not set")
				}
			}

			if tt.options.AllowCredentials {
				header := w.Header().Get("Access-Control-Allow-Credentials")
				if header != "true" {
					t.Error("Access-Control-Allow-Credentials should be true")
				}
			}
		})
	}
}

func TestCORS_OriginFunc(t *testing.T) {
	options := CORSOptions{
		AllowOriginFunc: func(origin string) bool {
			return strings.HasSuffix(origin, ".example.com")
		},
	}

	middleware := CORS(options)

	handler := func(ctx *gor.Context) error {
		return nil
	}

	// Test allowed origin
	ctx := createTestContext("GET", "/test")
	ctx.Request.Header.Set("Origin", "http://sub.example.com")

	err := middleware(handler)(ctx)
	if err != nil {
		t.Errorf("CORS middleware returned error: %v", err)
	}

	w := ctx.Response.(*httptest.ResponseRecorder)
	if w.Header().Get("Access-Control-Allow-Origin") != "http://sub.example.com" {
		t.Error("Origin should be allowed")
	}
}

func TestRequestID(t *testing.T) {
	middleware := RequestID()

	var capturedID string
	handler := func(ctx *gor.Context) error {
		if id, ok := ctx.Context.Value("request_id").(string); ok {
			capturedID = id
		}
		return nil
	}

	ctx := createTestContext("GET", "/test")
	err := middleware(handler)(ctx)

	if err != nil {
		t.Errorf("RequestID middleware returned error: %v", err)
	}

	if capturedID == "" {
		t.Error("Request ID was not set in context")
	}

	// Check response header
	w := ctx.Response.(*httptest.ResponseRecorder)
	headerID := w.Header().Get("X-Request-ID")
	if headerID == "" {
		t.Error("X-Request-ID header not set")
	}

	if headerID != capturedID {
		t.Error("Request ID in header doesn't match context")
	}
}

func TestRateLimit(t *testing.T) {
	middleware := RateLimit(2, 100*time.Millisecond)

	handler := func(ctx *gor.Context) error {
		ctx.Response.WriteHeader(http.StatusOK)
		return nil
	}

	// First request should pass
	ctx1 := createTestContext("GET", "/test")
	ctx1.Request.RemoteAddr = "192.168.1.1:1234"
	err := middleware(handler)(ctx1)
	if err != nil {
		t.Errorf("First request failed: %v", err)
	}

	// Second request should pass
	ctx2 := createTestContext("GET", "/test")
	ctx2.Request.RemoteAddr = "192.168.1.1:1234"
	err = middleware(handler)(ctx2)
	if err != nil {
		t.Errorf("Second request failed: %v", err)
	}

	// Third request should be rate limited
	ctx3 := createTestContext("GET", "/test")
	ctx3.Request.RemoteAddr = "192.168.1.1:1234"
	err = middleware(handler)(ctx3)

	w3 := ctx3.Response.(*httptest.ResponseRecorder)
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limit, got status %d", w3.Code)
	}

	// Wait for rate limit reset
	time.Sleep(150 * time.Millisecond)

	// Fourth request should pass after reset
	ctx4 := createTestContext("GET", "/test")
	ctx4.Request.RemoteAddr = "192.168.1.1:1234"
	err = middleware(handler)(ctx4)
	if err != nil {
		t.Errorf("Request after reset failed: %v", err)
	}

	w4 := ctx4.Response.(*httptest.ResponseRecorder)
	if w4.Code != http.StatusOK {
		t.Errorf("Expected success after reset, got status %d", w4.Code)
	}
}

func TestCompress(t *testing.T) {
	middleware := Compress()

	handler := func(ctx *gor.Context) error {
		ctx.Response.WriteHeader(http.StatusOK)
		return nil
	}

	// Test with gzip support
	ctx := createTestContext("GET", "/test")
	ctx.Request.Header.Set("Accept-Encoding", "gzip, deflate")

	err := middleware(handler)(ctx)
	if err != nil {
		t.Errorf("Compress middleware returned error: %v", err)
	}

	w := ctx.Response.(*httptest.ResponseRecorder)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Content-Encoding header should be gzip")
	}

	// Test without gzip support
	ctx2 := createTestContext("GET", "/test")
	ctx2.Request.Header.Set("Accept-Encoding", "deflate")

	err = middleware(handler)(ctx2)
	if err != nil {
		t.Errorf("Compress middleware returned error: %v", err)
	}

	w2 := ctx2.Response.(*httptest.ResponseRecorder)
	if w2.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Content-Encoding should not be gzip when client doesn't support it")
	}
}

func TestBasicAuth(t *testing.T) {
	users := map[string]string{
		"admin": "password123",
		"user":  "pass456",
	}

	middleware := BasicAuth("Test Realm", users)

	handler := func(ctx *gor.Context) error {
		ctx.Response.WriteHeader(http.StatusOK)
		return nil
	}

	// Test without credentials
	ctx1 := createTestContext("GET", "/test")
	err := middleware(handler)(ctx1)

	w1 := ctx1.Response.(*httptest.ResponseRecorder)
	if w1.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w1.Code)
	}
	if !strings.Contains(w1.Header().Get("WWW-Authenticate"), "Test Realm") {
		t.Error("WWW-Authenticate header should contain realm")
	}

	// Test with valid credentials
	ctx2 := createTestContext("GET", "/test")
	ctx2.Request.SetBasicAuth("admin", "password123")
	err = middleware(handler)(ctx2)

	if err != nil {
		t.Errorf("Valid auth failed: %v", err)
	}

	w2 := ctx2.Response.(*httptest.ResponseRecorder)
	if w2.Code != http.StatusOK {
		t.Errorf("Expected 200 with valid auth, got %d", w2.Code)
	}

	if ctx2.User != "admin" {
		t.Errorf("Expected user 'admin', got '%s'", ctx2.User)
	}

	// Test with invalid credentials
	ctx3 := createTestContext("GET", "/test")
	ctx3.Request.SetBasicAuth("admin", "wrongpass")
	err = middleware(handler)(ctx3)

	w3 := ctx3.Response.(*httptest.ResponseRecorder)
	if w3.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 with invalid auth, got %d", w3.Code)
	}
}

func TestCSRF(t *testing.T) {
	options := CSRFOptions{
		Secret:     "test-secret",
		HeaderName: "X-CSRF-Token",
		FieldName:  "csrf_token",
	}

	middleware := CSRF(options)

	handler := func(ctx *gor.Context) error {
		ctx.Response.WriteHeader(http.StatusOK)
		return nil
	}

	// Test GET request (should pass without token)
	ctx1 := createTestContext("GET", "/test")
	err := middleware(handler)(ctx1)
	if err != nil {
		t.Errorf("GET request failed: %v", err)
	}

	// Test POST without token (should fail)
	ctx2 := createTestContext("POST", "/test")
	err = middleware(handler)(ctx2)

	w2 := ctx2.Response.(*httptest.ResponseRecorder)
	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected 403 without CSRF token, got %d", w2.Code)
	}

	// Test POST with token in header
	ctx3 := createTestContext("POST", "/test")
	ctx3.Request.Header.Set("X-CSRF-Token", "valid-token-with-more-than-20-chars")
	err = middleware(handler)(ctx3)

	if err != nil {
		t.Errorf("POST with CSRF token failed: %v", err)
	}

	w3 := ctx3.Response.(*httptest.ResponseRecorder)
	if w3.Code != http.StatusOK {
		t.Errorf("Expected 200 with CSRF token, got %d", w3.Code)
	}
}

func TestTimeout(t *testing.T) {
	// Test successful request within timeout
	middleware := Timeout(100 * time.Millisecond)

	fastHandler := func(ctx *gor.Context) error {
		ctx.Response.WriteHeader(http.StatusOK)
		return nil
	}

	ctx1 := createTestContext("GET", "/test")
	err := middleware(fastHandler)(ctx1)

	if err != nil {
		t.Errorf("Fast handler failed: %v", err)
	}

	w1 := ctx1.Response.(*httptest.ResponseRecorder)
	if w1.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w1.Code)
	}

	// Test timeout
	slowHandler := func(ctx *gor.Context) error {
		time.Sleep(200 * time.Millisecond)
		ctx.Response.WriteHeader(http.StatusOK)
		return nil
	}

	ctx2 := createTestContext("GET", "/test")
	err = Timeout(50 * time.Millisecond)(slowHandler)(ctx2)

	w2 := ctx2.Response.(*httptest.ResponseRecorder)
	if w2.Code != http.StatusRequestTimeout {
		t.Errorf("Expected timeout status 408, got %d", w2.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 198.51.100.2"},
			remoteAddr: "192.168.1.1:1234",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "203.0.113.1"},
			remoteAddr: "192.168.1.1:1234",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:1234",
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("Request ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Request IDs should be unique")
	}
}

func TestValidateCSRFToken(t *testing.T) {
	tests := []struct {
		token    string
		secret   string
		expected bool
	}{
		{"", "secret", false},
		{"short", "secret", false},
		{"this-is-a-valid-token-longer-than-20", "secret", true},
		{"another-valid-token-with-more-chars", "", true},
	}

	for _, tt := range tests {
		result := validateCSRFToken(tt.token, tt.secret)
		if result != tt.expected {
			t.Errorf("validateCSRFToken(%s, %s) = %v, expected %v",
				tt.token, tt.secret, result, tt.expected)
		}
	}
}

// Benchmark tests
func BenchmarkLogger(b *testing.B) {
	middleware := Logger()
	handler := func(ctx *gor.Context) error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/test")
		middleware(handler)(ctx)
	}
}

func BenchmarkRequestID(b *testing.B) {
	middleware := RequestID()
	handler := func(ctx *gor.Context) error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/test")
		middleware(handler)(ctx)
	}
}

func BenchmarkRateLimit(b *testing.B) {
	middleware := RateLimit(1000, time.Second)
	handler := func(ctx *gor.Context) error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/test")
		ctx.Request.RemoteAddr = fmt.Sprintf("192.168.1.%d:1234", i%256)
		middleware(handler)(ctx)
	}
}
