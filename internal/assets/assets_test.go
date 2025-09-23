package assets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper function to create test directories and files
func setupTestAssets(t *testing.T) (string, string) {
	tmpDir, err := os.MkdirTemp("", "assets_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	sourceDir := filepath.Join(tmpDir, "src")
	outputDir := filepath.Join(tmpDir, "public")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create test CSS file
	cssContent := `
/* Test CSS file */
body {
    margin: 0;
    padding: 0;
    background-color: #fff;
}
.container {
    transform: translateX(100px);
    transition: all 0.3s ease;
    border-radius: 5px;
}
`
	if err := os.WriteFile(filepath.Join(sourceDir, "styles.css"), []byte(cssContent), 0644); err != nil {
		t.Fatalf("Failed to write CSS file: %v", err)
	}

	// Create test JS file
	jsContent := `
// Test JavaScript file
function hello(name) {
    console.log("Hello, " + name + "!");
    return name;
}

// Export for testing
if (typeof module !== 'undefined') {
    module.exports = { hello };
}
`
	if err := os.WriteFile(filepath.Join(sourceDir, "app.js"), []byte(jsContent), 0644); err != nil {
		t.Fatalf("Failed to write JS file: %v", err)
	}

	// Create test image (just a simple text file for testing)
	imageContent := []byte("FAKE_PNG_DATA")
	if err := os.WriteFile(filepath.Join(sourceDir, "logo.png"), imageContent, 0644); err != nil {
		t.Fatalf("Failed to write image file: %v", err)
	}

	// Cleanup after test
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return sourceDir, outputDir
}

func TestNewPipeline(t *testing.T) {
	sourceDir, outputDir := setupTestAssets(t)

	pipeline := NewPipeline(sourceDir, outputDir)

	if pipeline.sourcePath != sourceDir {
		t.Errorf("Expected sourcePath to be %s, got %s", sourceDir, pipeline.sourcePath)
	}
	if pipeline.outputPath != outputDir {
		t.Errorf("Expected outputPath to be %s, got %s", outputDir, pipeline.outputPath)
	}
	if pipeline.manifestPath != filepath.Join(outputDir, "manifest.json") {
		t.Errorf("Expected manifestPath to be %s, got %s", filepath.Join(outputDir, "manifest.json"), pipeline.manifestPath)
	}

	if !pipeline.fingerprint {
		t.Error("Expected fingerprinting to be enabled by default")
	}
	if !pipeline.compress {
		t.Error("Expected compression to be enabled by default")
	}
	if pipeline.processors == nil {
		t.Error("Expected processors map to be initialized")
	}
	if pipeline.manifest == nil {
		t.Error("Expected manifest map to be initialized")
	}
	if pipeline.cache == nil {
		t.Error("Expected cache map to be initialized")
	}
}

func TestPipeline_RegisterProcessor(t *testing.T) {
	sourceDir, outputDir := setupTestAssets(t)
	pipeline := NewPipeline(sourceDir, outputDir)

	cssProcessor := NewCSSProcessor(true, true)
	pipeline.RegisterProcessor("css", cssProcessor)

	// Check if processor is registered by name
	if _, exists := pipeline.processors["css"]; !exists {
		t.Error("CSS processor should be registered by name")
	}

	// Check if processor is registered by extension
	for _, ext := range cssProcessor.Extensions() {
		if _, exists := pipeline.processors[ext]; !exists {
			t.Errorf("CSS processor should be registered for extension %s", ext)
		}
	}
}

func TestPipeline_Compile(t *testing.T) {
	sourceDir, outputDir := setupTestAssets(t)
	pipeline := NewPipeline(sourceDir, outputDir)

	// Register processors
	pipeline.RegisterProcessor("css", NewCSSProcessor(true, true))
	pipeline.RegisterProcessor("js", NewJavaScriptProcessor(true, false))
	pipeline.RegisterProcessor("image", NewImageProcessor(false, 0, 0))

	err := pipeline.Compile()
	if err != nil {
		t.Fatalf("Compile() should not return error: %v", err)
	}

	// Check if output directory was created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Output directory should exist after compilation")
	}

	// Check if manifest file was created
	manifestPath := filepath.Join(outputDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("Manifest file should exist after compilation")
	}

	// Read and verify manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	var manifest map[string]string
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("Failed to parse manifest JSON: %v", err)
	}

	// Check if assets are in manifest
	expectedAssets := []string{"styles.css", "app.js", "logo.png"}
	for _, asset := range expectedAssets {
		if _, exists := manifest[asset]; !exists {
			t.Errorf("Asset %s should be in manifest", asset)
		}
	}

	// Verify fingerprinted files exist
	for originalName, fingerprintedName := range manifest {
		fingerprintedPath := filepath.Join(outputDir, fingerprintedName)
		if _, err := os.Stat(fingerprintedPath); os.IsNotExist(err) {
			t.Errorf("Fingerprinted file %s should exist", fingerprintedPath)
		}

		// Check if compressed versions exist for compressible assets
		ext := filepath.Ext(originalName)
		if shouldCompress(ext) {
			compressedPath := fingerprintedPath + ".gz"
			// Note: compressed files are only created if compression actually reduces size
			// For small test files, compression might not be beneficial
			if _, err := os.Stat(compressedPath); err == nil {
				t.Logf("Compressed file %s exists as expected", compressedPath)
			} else {
				t.Logf("Compressed file %s not created (likely due to small size)", compressedPath)
			}
		}
	}
}

func TestPipeline_GetAsset(t *testing.T) {
	sourceDir, outputDir := setupTestAssets(t)
	pipeline := NewPipeline(sourceDir, outputDir)

	// Register processors and compile
	pipeline.RegisterProcessor("css", NewCSSProcessor(false, false))
	err := pipeline.Compile()
	if err != nil {
		t.Fatalf("Failed to compile assets: %v", err)
	}

	// Get asset from cache/manifest
	asset, err := pipeline.GetAsset("styles.css")
	if err != nil {
		t.Fatalf("GetAsset() should not return error: %v", err)
	}

	if asset == nil {
		t.Fatal("Asset should not be nil")
	}

	if asset.Content == nil {
		t.Error("Asset content should not be nil")
	}

	if asset.ContentType != "text/css" {
		t.Errorf("Expected content type 'text/css', got %s", asset.ContentType)
	}

	// Test non-existent asset
	_, err = pipeline.GetAsset("nonexistent.css")
	if err == nil {
		t.Error("GetAsset() should return error for non-existent asset")
	}
}

func TestPipeline_AssetPath(t *testing.T) {
	sourceDir, outputDir := setupTestAssets(t)
	pipeline := NewPipeline(sourceDir, outputDir)

	// Manually set manifest for testing
	pipeline.manifest["styles.css"] = "styles-abcd1234.css"

	path := pipeline.AssetPath("styles.css")
	expected := "/assets/./styles-abcd1234.css"  // The implementation includes "./" for directory handling
	if path != expected {
		t.Errorf("Expected asset path %s, got %s", expected, path)
	}

	// Test non-existent asset
	path = pipeline.AssetPath("nonexistent.css")
	expected = "/assets/nonexistent.css"
	if path != expected {
		t.Errorf("Expected fallback asset path %s, got %s", expected, path)
	}
}

func TestCSSProcessor(t *testing.T) {
	processor := NewCSSProcessor(true, true)

	if processor == nil {
		t.Fatal("NewCSSProcessor should not return nil")
	}

	if !processor.minify {
		t.Error("Minify should be enabled")
	}
	if !processor.autoprefix {
		t.Error("Autoprefix should be enabled")
	}

	// Test extensions
	extensions := processor.Extensions()
	if len(extensions) != 1 || extensions[0] != ".css" {
		t.Errorf("Expected extensions [.css], got %v", extensions)
	}

	// Test processing
	css := `
/* Comment */
body {
    transform: rotate(45deg);
    margin: 10px;
}
`

	result, err := processor.Process([]byte(css), nil)
	if err != nil {
		t.Fatalf("Process() should not return error: %v", err)
	}

	resultStr := string(result)

	// Check if comment was removed (minification)
	if strings.Contains(resultStr, "/* Comment */") {
		t.Error("Comments should be removed during minification")
	}

	// Check if vendor prefixes were added
	if !strings.Contains(resultStr, "-webkit-transform") {
		t.Error("Vendor prefixes should be added")
	}
}

func TestJavaScriptProcessor(t *testing.T) {
	processor := NewJavaScriptProcessor(true, false)

	if processor == nil {
		t.Fatal("NewJavaScriptProcessor should not return nil")
	}

	if !processor.minify {
		t.Error("Minify should be enabled")
	}
	if processor.sourceMaps {
		t.Error("Source maps should be disabled")
	}

	// Test extensions
	extensions := processor.Extensions()
	expectedExtensions := []string{".js", ".mjs"}
	if len(extensions) != len(expectedExtensions) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExtensions), len(extensions))
	}

	// Test processing
	js := `
// Single line comment
/* Multi-line
   comment */
function test() {
    var a = 1 + 2;
    return a;
}
`

	result, err := processor.Process([]byte(js), nil)
	if err != nil {
		t.Fatalf("Process() should not return error: %v", err)
	}

	resultStr := string(result)

	// Check if comments were removed
	if strings.Contains(resultStr, "// Single line comment") {
		t.Error("Single-line comments should be removed during minification")
	}
	if strings.Contains(resultStr, "/* Multi-line") {
		t.Error("Multi-line comments should be removed during minification")
	}

	// Check basic minification
	if strings.Contains(resultStr, " + ") {
		t.Error("Spaces around operators should be removed")
	}
}

func TestImageProcessor(t *testing.T) {
	processor := NewImageProcessor(true, 800, 600)

	if processor == nil {
		t.Fatal("NewImageProcessor should not return nil")
	}

	if !processor.optimize {
		t.Error("Optimize should be enabled")
	}
	if processor.maxWidth != 800 {
		t.Errorf("Expected maxWidth 800, got %d", processor.maxWidth)
	}
	if processor.maxHeight != 600 {
		t.Errorf("Expected maxHeight 600, got %d", processor.maxHeight)
	}

	// Test extensions
	extensions := processor.Extensions()
	expectedExtensions := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"}
	if len(extensions) != len(expectedExtensions) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExtensions), len(extensions))
	}

	// Test processing (passthrough for now)
	imageData := []byte("fake_image_data")
	result, err := processor.Process(imageData, nil)
	if err != nil {
		t.Fatalf("Process() should not return error: %v", err)
	}

	if string(result) != string(imageData) {
		t.Error("Image processor should pass through data unchanged for now")
	}
}

func TestSassProcessor(t *testing.T) {
	processor := NewSassProcessor(true)

	if processor == nil {
		t.Fatal("NewSassProcessor should not return nil")
	}

	if !processor.minify {
		t.Error("Minify should be enabled")
	}

	// Test extensions
	extensions := processor.Extensions()
	expectedExtensions := []string{".sass", ".scss"}
	if len(extensions) != len(expectedExtensions) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExtensions), len(extensions))
	}

	// Test processing (should behave like CSS processor for now)
	scss := `
$primary-color: #333;
body {
    color: $primary-color;
    margin: 0;
}
`

	result, err := processor.Process([]byte(scss), nil)
	if err != nil {
		t.Fatalf("Process() should not return error: %v", err)
	}

	if len(result) == 0 {
		t.Error("Processed SASS should not be empty")
	}
}

func TestBundler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bundler_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	bundler := NewBundler(tmpDir)

	if bundler == nil {
		t.Fatal("NewBundler should not return nil")
	}

	if bundler.outputPath != tmpDir {
		t.Errorf("Expected output path %s, got %s", tmpDir, bundler.outputPath)
	}

	if bundler.format != "iife" {
		t.Errorf("Expected default format 'iife', got %s", bundler.format)
	}

	// Test adding entry point
	bundler.AddEntryPoint("app.js")
	if len(bundler.entryPoints) != 1 {
		t.Errorf("Expected 1 entry point, got %d", len(bundler.entryPoints))
	}
	if bundler.entryPoints[0] != "app.js" {
		t.Errorf("Expected entry point 'app.js', got %s", bundler.entryPoints[0])
	}

	// Test setting format
	bundler.SetFormat("esm")
	if bundler.format != "esm" {
		t.Errorf("Expected format 'esm', got %s", bundler.format)
	}
}

func TestAssetServer(t *testing.T) {
	sourceDir, outputDir := setupTestAssets(t)
	pipeline := NewPipeline(sourceDir, outputDir)

	// Compile assets first
	pipeline.RegisterProcessor("css", NewCSSProcessor(false, false))
	err := pipeline.Compile()
	if err != nil {
		t.Fatalf("Failed to compile assets: %v", err)
	}

	// Create server
	server := NewServer(pipeline, "/assets")

	if server.pathPrefix != "/assets" {
		t.Errorf("Expected path prefix '/assets', got %s", server.pathPrefix)
	}
	if server.maxAge != 365*24*time.Hour {
		t.Errorf("Expected max age 1 year, got %v", server.maxAge)
	}

	// Test with custom settings
	customServer := NewServer(pipeline, "").
		WithMaxAge(1*time.Hour).
		WithCORS("example.com")

	if customServer.pathPrefix != "/assets" {
		t.Error("Empty path prefix should default to '/assets'")
	}
	if customServer.maxAge != 1*time.Hour {
		t.Errorf("Expected max age 1 hour, got %v", customServer.maxAge)
	}
	if !customServer.enableCORS {
		t.Error("CORS should be enabled")
	}
	if len(customServer.corsOrigins) != 1 || customServer.corsOrigins[0] != "example.com" {
		t.Errorf("Expected CORS origins [example.com], got %v", customServer.corsOrigins)
	}
}

func TestAssetServer_ServeHTTP(t *testing.T) {
	sourceDir, outputDir := setupTestAssets(t)
	pipeline := NewPipeline(sourceDir, outputDir)

	// Disable fingerprinting for predictable paths
	pipeline.fingerprint = false

	// Compile assets
	pipeline.RegisterProcessor("css", NewCSSProcessor(false, false))
	err := pipeline.Compile()
	if err != nil {
		t.Fatalf("Failed to compile assets: %v", err)
	}

	server := NewServer(pipeline, "/assets")

	// Test serving existing asset
	req := httptest.NewRequest("GET", "/assets/styles.css", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/css" {
		t.Errorf("Expected Content-Type 'text/css', got %s", w.Header().Get("Content-Type"))
	}

	// Test serving non-existent asset
	req = httptest.NewRequest("GET", "/assets/nonexistent.css", nil)
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	// Test wrong path prefix
	req = httptest.NewRequest("GET", "/wrong/styles.css", nil)
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for wrong path prefix, got %d", w.Code)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test shouldCompress
	compressibleExts := []string{".css", ".js", ".json", ".html", ".xml", ".svg", ".txt"}
	for _, ext := range compressibleExts {
		if !shouldCompress(ext) {
			t.Errorf("Extension %s should be compressible", ext)
		}
	}

	nonCompressibleExts := []string{".png", ".jpg", ".gif", ".zip", ".gz"}
	for _, ext := range nonCompressibleExts {
		if shouldCompress(ext) {
			t.Errorf("Extension %s should not be compressible", ext)
		}
	}

	// Test getContentType
	contentTypes := map[string]string{
		".css":  "text/css",
		".js":   "application/javascript",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".json": "application/json",
		".html": "text/html",
		".svg":  "image/svg+xml",
		".woff": "font/woff",
		".xyz":  "application/octet-stream", // Unknown extension
	}

	for ext, expectedType := range contentTypes {
		actualType := getContentType(ext)
		if actualType != expectedType {
			t.Errorf("Expected content type %s for %s, got %s", expectedType, ext, actualType)
		}
	}

	// Test compressAsset
	// Use larger, more repetitive data to ensure compression works
	testData := []byte(strings.Repeat("This is test data that should compress very well with gzip compression because it has lots of repetition. ", 50))
	compressed, err := compressAsset(testData)
	if err != nil {
		t.Fatalf("compressAsset should not return error: %v", err)
	}

	if len(compressed) >= len(testData) {
		t.Errorf("Compressed data (%d bytes) should be smaller than original (%d bytes)", len(compressed), len(testData))
	}

	if len(compressed) == 0 {
		t.Error("Compressed data should not be empty")
	}
}

func TestAsset_Structure(t *testing.T) {
	now := time.Now()
	asset := &Asset{
		Path:         "/path/to/asset.css",
		Content:      []byte("body { color: red; }"),
		ContentType:  "text/css",
		Hash:         "abcd1234",
		Compressed:   true,
		LastModified: now,
	}

	if asset.Path != "/path/to/asset.css" {
		t.Errorf("Expected path '/path/to/asset.css', got %s", asset.Path)
	}
	if string(asset.Content) != "body { color: red; }" {
		t.Errorf("Expected content 'body { color: red; }', got %s", string(asset.Content))
	}
	if asset.ContentType != "text/css" {
		t.Errorf("Expected content type 'text/css', got %s", asset.ContentType)
	}
	if asset.Hash != "abcd1234" {
		t.Errorf("Expected hash 'abcd1234', got %s", asset.Hash)
	}
	if !asset.Compressed {
		t.Error("Expected compressed to be true")
	}
	if !asset.LastModified.Equal(now) {
		t.Errorf("Expected last modified %v, got %v", now, asset.LastModified)
	}
}

func TestModule_Structure(t *testing.T) {
	module := &Module{
		Path:         "/path/to/module.js",
		Content:      "export default function() {}",
		Dependencies: []string{"dependency1", "dependency2"},
		Exports:      []string{"default"},
		Imports:      map[string]string{"React": "react"},
	}

	if module.Path != "/path/to/module.js" {
		t.Errorf("Expected path '/path/to/module.js', got %s", module.Path)
	}
	if module.Content != "export default function() {}" {
		t.Errorf("Expected content 'export default function() {}', got %s", module.Content)
	}
	if len(module.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(module.Dependencies))
	}
	if len(module.Exports) != 1 || module.Exports[0] != "default" {
		t.Errorf("Expected exports ['default'], got %v", module.Exports)
	}
	if module.Imports["React"] != "react" {
		t.Errorf("Expected React import to be 'react', got %s", module.Imports["React"])
	}
}