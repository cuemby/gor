package assets

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Server serves compiled assets
type Server struct {
	pipeline    *Pipeline
	pathPrefix  string
	maxAge      time.Duration
	enableGzip  bool
	enableCORS  bool
	corsOrigins []string
}

// NewServer creates a new asset server
func NewServer(pipeline *Pipeline, pathPrefix string) *Server {
	if pathPrefix == "" {
		pathPrefix = "/assets"
	}

	return &Server{
		pipeline:    pipeline,
		pathPrefix:  pathPrefix,
		maxAge:      365 * 24 * time.Hour, // 1 year for fingerprinted assets
		enableGzip:  true,
		enableCORS:  false,
		corsOrigins: []string{"*"},
	}
}

// WithMaxAge sets the max-age for cache control
func (s *Server) WithMaxAge(maxAge time.Duration) *Server {
	s.maxAge = maxAge
	return s
}

// WithCORS enables CORS headers
func (s *Server) WithCORS(origins ...string) *Server {
	s.enableCORS = true
	if len(origins) > 0 {
		s.corsOrigins = origins
	}
	return s
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if this is an asset request
	if !strings.HasPrefix(r.URL.Path, s.pathPrefix) {
		http.NotFound(w, r)
		return
	}

	// Get asset path
	assetPath := strings.TrimPrefix(r.URL.Path, s.pathPrefix)
	assetPath = strings.TrimPrefix(assetPath, "/")

	// Get the asset
	asset, err := s.pipeline.GetAsset(assetPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set headers
	s.setHeaders(w, asset)

	// Handle conditional requests
	if s.handleConditional(w, r, asset) {
		return
	}

	// Check if client accepts gzip
	if s.enableGzip && asset.Compressed && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		// Try to serve compressed version
		compressedPath := asset.Path + ".gz"
		if compressedContent, err := os.ReadFile(compressedPath); err == nil {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(compressedContent)))
			_, _ = w.Write(compressedContent)
			return
		}
	}

	// Serve the asset
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(asset.Content)))
	_, _ = w.Write(asset.Content)
}

// setHeaders sets response headers
func (s *Server) setHeaders(w http.ResponseWriter, asset *Asset) {
	// Content-Type
	w.Header().Set("Content-Type", asset.ContentType)

	// Cache-Control
	if asset.Hash != "" {
		// Fingerprinted assets can be cached forever
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(s.maxAge.Seconds())))
	} else {
		// Non-fingerprinted assets should revalidate
		w.Header().Set("Cache-Control", "public, must-revalidate")
	}

	// ETag
	if asset.Hash != "" {
		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, asset.Hash))
	}

	// Last-Modified
	w.Header().Set("Last-Modified", asset.LastModified.UTC().Format(http.TimeFormat))

	// CORS headers
	if s.enableCORS {
		origin := "*"
		if len(s.corsOrigins) > 0 {
			origin = strings.Join(s.corsOrigins, ", ")
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Max-Age", "3600")
	}

	// Security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// handleConditional handles conditional requests
func (s *Server) handleConditional(w http.ResponseWriter, r *http.Request, asset *Asset) bool {
	// Check If-None-Match (ETag)
	if asset.Hash != "" {
		ifNoneMatch := r.Header.Get("If-None-Match")
		if ifNoneMatch == fmt.Sprintf(`"%s"`, asset.Hash) {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	// Check If-Modified-Since
	ifModifiedSince := r.Header.Get("If-Modified-Since")
	if ifModifiedSince != "" {
		t, err := http.ParseTime(ifModifiedSince)
		if err == nil && !asset.LastModified.After(t) {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	return false
}

// Middleware returns HTTP middleware for serving assets
func (s *Server) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, s.pathPrefix) {
			s.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Helpers for templates

// AssetHelpers provides template helpers for assets
type AssetHelpers struct {
	pipeline *Pipeline
}

// NewAssetHelpers creates new asset helpers
func NewAssetHelpers(pipeline *Pipeline) *AssetHelpers {
	return &AssetHelpers{
		pipeline: pipeline,
	}
}

// StylesheetLinkTag generates a stylesheet link tag
func (h *AssetHelpers) StylesheetLinkTag(path string, options ...map[string]string) string {
	assetPath := h.pipeline.AssetPath(path)

	tag := fmt.Sprintf(`<link rel="stylesheet" href="%s"`, assetPath)

	if len(options) > 0 {
		for key, value := range options[0] {
			tag += fmt.Sprintf(` %s="%s"`, key, value)
		}
	}

	tag += " />"
	return tag
}

// JavaScriptIncludeTag generates a script tag
func (h *AssetHelpers) JavaScriptIncludeTag(path string, options ...map[string]string) string {
	assetPath := h.pipeline.AssetPath(path)

	tag := fmt.Sprintf(`<script src="%s"`, assetPath)

	if len(options) > 0 {
		for key, value := range options[0] {
			tag += fmt.Sprintf(` %s="%s"`, key, value)
		}
	}

	tag += "></script>"
	return tag
}

// ImageTag generates an image tag
func (h *AssetHelpers) ImageTag(path string, options ...map[string]string) string {
	assetPath := h.pipeline.AssetPath(path)

	tag := fmt.Sprintf(`<img src="%s"`, assetPath)

	if len(options) > 0 {
		for key, value := range options[0] {
			tag += fmt.Sprintf(` %s="%s"`, key, value)
		}
	}

	tag += " />"
	return tag
}

// AssetPath returns the path to an asset
func (h *AssetHelpers) AssetPath(path string) string {
	return h.pipeline.AssetPath(path)
}

// InlineAsset returns the content of an asset inline
func (h *AssetHelpers) InlineAsset(path string) string {
	asset, err := h.pipeline.GetAsset(path)
	if err != nil {
		return fmt.Sprintf("<!-- Asset not found: %s -->", path)
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".css":
		return fmt.Sprintf("<style>%s</style>", string(asset.Content))
	case ".js":
		return fmt.Sprintf("<script>%s</script>", string(asset.Content))
	case ".svg":
		return string(asset.Content)
	default:
		return fmt.Sprintf("<!-- Cannot inline %s files -->", ext)
	}
}

// PreloadTag generates a preload link tag
func (h *AssetHelpers) PreloadTag(path string, asType string) string {
	assetPath := h.pipeline.AssetPath(path)
	return fmt.Sprintf(`<link rel="preload" href="%s" as="%s" />`, assetPath, asType)
}
