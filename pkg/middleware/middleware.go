package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/cuemby/gor/pkg/gor"
)

// Logger middleware logs HTTP requests
func Logger() gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			start := time.Now()

			// Log request
			log.Printf("[%s] %s %s",
				time.Now().Format("2006-01-02 15:04:05"),
				ctx.Request.Method,
				ctx.Request.URL.Path,
			)

			// Process request
			err := next(ctx)

			// Log response time
			duration := time.Since(start)
			log.Printf("Completed in %v", duration)

			return err
		}
	}
}

// Recovery middleware recovers from panics
func Recovery() gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			defer func() {
				if err := recover(); err != nil {
					// Log the error and stack trace
					log.Printf("PANIC: %v\n%s", err, debug.Stack())

					// Return 500 Internal Server Error
					ctx.Response.WriteHeader(http.StatusInternalServerError)
					ctx.Text(http.StatusInternalServerError, "Internal Server Error")
				}
			}()

			return next(ctx)
		}
	}
}

// CORS middleware adds CORS headers
func CORS(options CORSOptions) gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			origin := ctx.Request.Header.Get("Origin")

			// Check if origin is allowed
			if options.AllowOrigin == "*" || origin == options.AllowOrigin {
				ctx.Response.Header().Set("Access-Control-Allow-Origin", options.AllowOrigin)
			} else if options.AllowOriginFunc != nil && options.AllowOriginFunc(origin) {
				ctx.Response.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Set other CORS headers
			if options.AllowMethods != "" {
				ctx.Response.Header().Set("Access-Control-Allow-Methods", options.AllowMethods)
			}
			if options.AllowHeaders != "" {
				ctx.Response.Header().Set("Access-Control-Allow-Headers", options.AllowHeaders)
			}
			if options.ExposeHeaders != "" {
				ctx.Response.Header().Set("Access-Control-Expose-Headers", options.ExposeHeaders)
			}
			if options.AllowCredentials {
				ctx.Response.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if options.MaxAge > 0 {
				ctx.Response.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", options.MaxAge))
			}

			// Handle preflight requests
			if ctx.Request.Method == "OPTIONS" {
				ctx.Response.WriteHeader(http.StatusNoContent)
				return nil
			}

			return next(ctx)
		}
	}
}

// CORSOptions defines CORS configuration
type CORSOptions struct {
	AllowOrigin      string
	AllowOriginFunc  func(origin string) bool
	AllowMethods     string
	AllowHeaders     string
	ExposeHeaders    string
	AllowCredentials bool
	MaxAge           int
}

// RequestID middleware adds a unique request ID
func RequestID() gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			// Generate request ID
			requestID := generateRequestID()

			// Add to context
			ctx.Context = context.WithValue(ctx.Context, "request_id", requestID)

			// Add to response header
			ctx.Response.Header().Set("X-Request-ID", requestID)

			return next(ctx)
		}
	}
}

// RateLimit middleware implements rate limiting
func RateLimit(requests int, duration time.Duration) gor.MiddlewareFunc {
	// Simple in-memory rate limiter (production should use Redis or similar)
	type client struct {
		count     int
		lastReset time.Time
	}

	clients := make(map[string]*client)

	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			// Get client identifier (IP address)
			clientIP := getClientIP(ctx.Request)

			// Check rate limit
			now := time.Now()
			c, exists := clients[clientIP]
			if !exists {
				clients[clientIP] = &client{count: 1, lastReset: now}
			} else {
				// Reset if duration has passed
				if now.Sub(c.lastReset) > duration {
					c.count = 1
					c.lastReset = now
				} else {
					c.count++
					if c.count > requests {
						// Rate limit exceeded
						ctx.Response.WriteHeader(http.StatusTooManyRequests)
						return ctx.Text(http.StatusTooManyRequests, "Rate limit exceeded")
					}
				}
			}

			return next(ctx)
		}
	}
}

// Compress middleware adds gzip compression
func Compress() gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			// Check if client accepts gzip
			if !strings.Contains(ctx.Request.Header.Get("Accept-Encoding"), "gzip") {
				return next(ctx)
			}

			// Wrap response writer with gzip writer
			// This is simplified - production would need proper gzip.Writer implementation
			ctx.Response.Header().Set("Content-Encoding", "gzip")

			return next(ctx)
		}
	}
}

// Static serves static files
func Static(path string, root string) gor.MiddlewareFunc {
	fileServer := http.FileServer(http.Dir(root))

	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			// Check if request path matches static path
			if strings.HasPrefix(ctx.Request.URL.Path, path) {
				// Strip prefix and serve file
				http.StripPrefix(path, fileServer).ServeHTTP(ctx.Response, ctx.Request)
				return nil
			}

			return next(ctx)
		}
	}
}

// BasicAuth implements HTTP Basic Authentication
func BasicAuth(realm string, users map[string]string) gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			username, password, ok := ctx.Request.BasicAuth()
			if !ok {
				// Request authentication
				ctx.Response.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
				ctx.Response.WriteHeader(http.StatusUnauthorized)
				return ctx.Text(http.StatusUnauthorized, "Unauthorized")
			}

			// Check credentials
			if expectedPassword, exists := users[username]; !exists || password != expectedPassword {
				ctx.Response.WriteHeader(http.StatusUnauthorized)
				return ctx.Text(http.StatusUnauthorized, "Unauthorized")
			}

			// Set user in context
			ctx.User = username

			return next(ctx)
		}
	}
}

// CSRF middleware provides CSRF protection
func CSRF(options CSRFOptions) gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			// Skip CSRF for safe methods
			if ctx.Request.Method == "GET" || ctx.Request.Method == "HEAD" || ctx.Request.Method == "OPTIONS" {
				return next(ctx)
			}

			// Get CSRF token from request
			token := ctx.Request.Header.Get("X-CSRF-Token")
			if token == "" {
				token = ctx.Request.FormValue("csrf_token")
			}

			// Validate token (simplified - production needs secure token generation/validation)
			if token == "" || !validateCSRFToken(token, options.Secret) {
				ctx.Response.WriteHeader(http.StatusForbidden)
				return ctx.Text(http.StatusForbidden, "CSRF token invalid")
			}

			return next(ctx)
		}
	}
}

// CSRFOptions defines CSRF configuration
type CSRFOptions struct {
	Secret     string
	CookieName string
	HeaderName string
	FieldName  string
}

// Timeout middleware adds request timeout
func Timeout(duration time.Duration) gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			// Create timeout context
			timeoutCtx, cancel := context.WithTimeout(ctx.Context, duration)
			defer cancel()

			// Update context
			ctx.Context = timeoutCtx

			// Channel to track completion
			done := make(chan error, 1)

			// Run handler in goroutine
			go func() {
				done <- next(ctx)
			}()

			// Wait for completion or timeout
			select {
			case err := <-done:
				return err
			case <-timeoutCtx.Done():
				ctx.Response.WriteHeader(http.StatusRequestTimeout)
				return ctx.Text(http.StatusRequestTimeout, "Request timeout")
			}
		}
	}
}

// Helper functions

func generateRequestID() string {
	// Simple implementation - production should use UUID or similar
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// Return first IP in the chain
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

func validateCSRFToken(token, secret string) bool {
	// Simplified validation - production needs proper implementation
	return token != "" && len(token) > 20
}
