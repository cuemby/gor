package views

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cuemby/gor/pkg/gor"
)

// Helper function to create test template directories
func setupTestViews(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "views_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create directory structure
	viewsDir := filepath.Join(tmpDir, "views")
	layoutsDir := filepath.Join(viewsDir, "layouts")
	sharedDir := filepath.Join(viewsDir, "shared")

	dirs := []string{viewsDir, layoutsDir, sharedDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create layout file
	layoutContent := `<!DOCTYPE html>
<html>
<head><title>Test Layout</title></head>
<body>
<header>Header</header>
{{template "content" .}}
<footer>Footer</footer>
</body>
</html>`
	layoutPath := filepath.Join(layoutsDir, "application.html")
	if err := os.WriteFile(layoutPath, []byte(layoutContent), 0644); err != nil {
		t.Fatalf("Failed to write layout file: %v", err)
	}

	// Create a simple view
	viewContent := `<h1>Welcome {{.Name}}</h1>
<p>{{.Message}}</p>`
	viewPath := filepath.Join(viewsDir, "index.html")
	if err := os.WriteFile(viewPath, []byte(viewContent), 0644); err != nil {
		t.Fatalf("Failed to write view file: %v", err)
	}

	// Create a controller/action view
	usersDir := filepath.Join(viewsDir, "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		t.Fatalf("Failed to create users directory: %v", err)
	}
	userIndexContent := `<h2>Users List</h2>
<ul>
{{range .Users}}<li>{{.}}</li>{{end}}
</ul>`
	userIndexPath := filepath.Join(usersDir, "index.html")
	if err := os.WriteFile(userIndexPath, []byte(userIndexContent), 0644); err != nil {
		t.Fatalf("Failed to write users index file: %v", err)
	}

	// Create a partial
	partialContent := `<div class="sidebar">{{.Title}}</div>`
	partialPath := filepath.Join(sharedDir, "_sidebar.html")
	if err := os.WriteFile(partialPath, []byte(partialContent), 0644); err != nil {
		t.Fatalf("Failed to write partial file: %v", err)
	}

	// Create a view without layout
	standaloneContent := `<h1>Standalone Page</h1>`
	standalonePath := filepath.Join(viewsDir, "standalone.html")
	if err := os.WriteFile(standalonePath, []byte(standaloneContent), 0644); err != nil {
		t.Fatalf("Failed to write standalone file: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return viewsDir
}

func TestNewTemplateEngine(t *testing.T) {
	viewsDir := setupTestViews(t)

	te := NewTemplateEngine(viewsDir, true)

	if te.viewsPath != viewsDir {
		t.Errorf("Expected viewsPath %s, got %s", viewsDir, te.viewsPath)
	}
	if te.layoutsPath != filepath.Join(viewsDir, "layouts") {
		t.Errorf("Expected layoutsPath %s, got %s", filepath.Join(viewsDir, "layouts"), te.layoutsPath)
	}
	if te.partialsPath != filepath.Join(viewsDir, "shared") {
		t.Errorf("Expected partialsPath %s, got %s", filepath.Join(viewsDir, "shared"), te.partialsPath)
	}
	if te.extension != ".html" {
		t.Errorf("Expected extension '.html', got %s", te.extension)
	}
	if !te.debug {
		t.Error("Expected debug mode to be enabled")
	}
	if te.cache == nil {
		t.Error("Cache should be initialized")
	}
	if te.funcs == nil {
		t.Error("Funcs should be initialized with default helpers")
	}
}

func TestTemplateEngine_Render(t *testing.T) {
	viewsDir := setupTestViews(t)
	te := NewTemplateEngine(viewsDir, true)

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Name":    "John",
		"Message": "Hello, World!",
	}

	err := te.Render(&buf, "index", data)
	if err != nil {
		t.Fatalf("Render() should not return error: %v", err)
	}

	output := buf.String()

	// Check for layout content
	if !strings.Contains(output, "<header>Header</header>") {
		t.Error("Output should contain layout header")
	}
	if !strings.Contains(output, "<footer>Footer</footer>") {
		t.Error("Output should contain layout footer")
	}

	// Check for view content
	if !strings.Contains(output, "<h1>Welcome John</h1>") {
		t.Error("Output should contain rendered view with data")
	}
	if !strings.Contains(output, "<p>Hello, World!</p>") {
		t.Error("Output should contain rendered message")
	}
}

func TestTemplateEngine_RenderWithLayout(t *testing.T) {
	viewsDir := setupTestViews(t)
	te := NewTemplateEngine(viewsDir, true)

	// Create a custom layout
	customLayoutContent := `<custom>{{template "content" .}}</custom>`
	customLayoutPath := filepath.Join(viewsDir, "layouts", "custom.html")
	if err := os.WriteFile(customLayoutPath, []byte(customLayoutContent), 0644); err != nil {
		t.Fatalf("Failed to write custom layout: %v", err)
	}

	var buf bytes.Buffer
	data := map[string]interface{}{"Name": "Test"}

	err := te.RenderWithLayout(&buf, "index", "custom", data)
	if err != nil {
		t.Fatalf("RenderWithLayout() should not return error: %v", err)
	}

	output := buf.String()

	// Check for custom layout
	if !strings.Contains(output, "<custom>") && !strings.Contains(output, "</custom>") {
		t.Error("Output should use custom layout")
	}

	// Should not have default layout content
	if strings.Contains(output, "<header>Header</header>") {
		t.Error("Output should not contain default layout header")
	}
}

func TestTemplateEngine_RenderPartial(t *testing.T) {
	viewsDir := setupTestViews(t)
	te := NewTemplateEngine(viewsDir, true)

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Title": "My Sidebar",
	}

	err := te.RenderPartial(&buf, "sidebar", data)
	if err != nil {
		t.Fatalf("RenderPartial() should not return error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, `<div class="sidebar">My Sidebar</div>`) {
		t.Errorf("Expected partial output, got: %s", output)
	}
}

func TestTemplateEngine_Cache(t *testing.T) {
	viewsDir := setupTestViews(t)

	t.Run("CachingEnabled", func(t *testing.T) {
		te := NewTemplateEngine(viewsDir, false) // debug = false enables caching

		// First render should compile and cache
		var buf1 bytes.Buffer
		data := map[string]interface{}{"Name": "Test1"}
		err := te.Render(&buf1, "index", data)
		if err != nil {
			t.Fatalf("First render should not error: %v", err)
		}

		// Check cache
		te.mu.RLock()
		cacheSize := len(te.cache)
		te.mu.RUnlock()

		if cacheSize == 0 {
			t.Error("Template should be cached after first render")
		}

		// Second render should use cache
		var buf2 bytes.Buffer
		data2 := map[string]interface{}{"Name": "Test2"}
		err = te.Render(&buf2, "index", data2)
		if err != nil {
			t.Fatalf("Second render should not error: %v", err)
		}

		// Cache size should not change
		te.mu.RLock()
		newCacheSize := len(te.cache)
		te.mu.RUnlock()

		if newCacheSize != cacheSize {
			t.Error("Cache size should not change on second render")
		}
	})

	t.Run("CachingDisabled", func(t *testing.T) {
		te := NewTemplateEngine(viewsDir, true) // debug = true disables caching

		var buf bytes.Buffer
		data := map[string]interface{}{"Name": "Test"}
		err := te.Render(&buf, "index", data)
		if err != nil {
			t.Fatalf("Render should not error: %v", err)
		}

		// Check cache should be empty in debug mode
		te.mu.RLock()
		cacheSize := len(te.cache)
		te.mu.RUnlock()

		if cacheSize != 0 {
			t.Error("Template should not be cached in debug mode")
		}
	})
}

func TestTemplateEngine_ControllerActionView(t *testing.T) {
	viewsDir := setupTestViews(t)
	te := NewTemplateEngine(viewsDir, true)

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Users": []string{"Alice", "Bob", "Charlie"},
	}

	err := te.Render(&buf, "users/index", data)
	if err != nil {
		t.Fatalf("Render() should not return error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "<h2>Users List</h2>") {
		t.Error("Output should contain users heading")
	}
	if !strings.Contains(output, "<li>Alice</li>") {
		t.Error("Output should contain Alice")
	}
	if !strings.Contains(output, "<li>Bob</li>") {
		t.Error("Output should contain Bob")
	}
	if !strings.Contains(output, "<li>Charlie</li>") {
		t.Error("Output should contain Charlie")
	}
}

func TestTemplateEngine_AddFunc(t *testing.T) {
	viewsDir := setupTestViews(t)
	te := NewTemplateEngine(viewsDir, true)

	// Add custom function
	te.AddFunc("double", func(n int) int {
		return n * 2
	})

	// Verify function was added
	if _, exists := te.funcs["double"]; !exists {
		t.Error("Custom function should be added to funcs map")
	}

	// Create a view that uses the custom function
	customViewContent := `<p>Double of 5 is {{double 5}}</p>`
	customViewPath := filepath.Join(viewsDir, "custom.html")
	if err := os.WriteFile(customViewPath, []byte(customViewContent), 0644); err != nil {
		t.Fatalf("Failed to write custom view: %v", err)
	}

	var buf bytes.Buffer
	err := te.Render(&buf, "custom", nil)
	if err != nil {
		t.Fatalf("Render() should not return error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Double of 5 is 10") {
		t.Error("Custom function should work in template")
	}
}

func TestTemplateEngine_MissingView(t *testing.T) {
	viewsDir := setupTestViews(t)
	te := NewTemplateEngine(viewsDir, true)

	var buf bytes.Buffer
	err := te.Render(&buf, "nonexistent", nil)

	if err == nil {
		t.Error("Render() should return error for missing view")
	}
}

func TestTemplateEngine_NoLayout(t *testing.T) {
	viewsDir := setupTestViews(t)
	te := NewTemplateEngine(viewsDir, true)

	var buf bytes.Buffer
	err := te.RenderWithLayout(&buf, "standalone", "nonexistent", nil)

	// Should not error - it should just render the view without layout
	if err != nil {
		t.Fatalf("RenderWithLayout() should not error when layout is missing: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<h1>Standalone Page</h1>") {
		t.Error("Should render view without layout when layout is missing")
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("StringHelpers", func(t *testing.T) {
		// Test capitalize
		if capitalize("hello") != "Hello" {
			t.Errorf("capitalize('hello') should return 'Hello'")
		}
		if capitalize("") != "" {
			t.Error("capitalize('') should return empty string")
		}

		// Test pluralize
		if pluralize("cat", 1) != "cat" {
			t.Error("pluralize('cat', 1) should return 'cat'")
		}
		if pluralize("cat", 2) != "cats" {
			t.Error("pluralize('cat', 2) should return 'cats'")
		}
		if pluralize("baby", 2) != "babies" {
			t.Error("pluralize('baby', 2) should return 'babies'")
		}
		if pluralize("box", 2) != "boxes" {
			t.Error("pluralize('box', 2) should return 'boxes'")
		}
		if pluralize("church", 2) != "churches" {
			t.Error("pluralize('church', 2) should return 'churches'")
		}

		// Test singularize
		if singularize("cats") != "cat" {
			t.Error("singularize('cats') should return 'cat'")
		}
		if singularize("babies") != "baby" {
			t.Error("singularize('babies') should return 'baby'")
		}
		if singularize("boxes") != "box" {
			t.Error("singularize('boxes') should return 'box'")
		}

		// Test truncate
		if truncate("Hello World", 5) != "Hello..." {
			t.Error("truncate should add ellipsis")
		}
		if truncate("Hi", 5) != "Hi" {
			t.Error("truncate should not modify short strings")
		}
	})

	t.Run("HTMLHelpers", func(t *testing.T) {
		// Test linkTo
		link := linkTo("Click me", "/path", `class="btn"`)
		expected := template.HTML(`<a href="/path" class="btn">Click me</a>`)
		if link != expected {
			t.Errorf("linkTo() returned %s, expected %s", link, expected)
		}

		// Test imageTag
		img := imageTag("/logo.png", `alt="Logo"`, `width="100"`)
		if !strings.Contains(string(img), `src="/logo.png"`) {
			t.Error("imageTag() should include src attribute")
		}
		if !strings.Contains(string(img), `alt="Logo"`) {
			t.Error("imageTag() should include alt attribute")
		}

		// Test scriptTag
		script := scriptTag("/app.js")
		expected = template.HTML(`<script src="/app.js"></script>`)
		if script != expected {
			t.Errorf("scriptTag() returned %s, expected %s", script, expected)
		}

		// Test styleTag
		style := styleTag("/app.css")
		expected = template.HTML(`<link rel="stylesheet" href="/app.css">`)
		if style != expected {
			t.Errorf("styleTag() returned %s, expected %s", style, expected)
		}
	})

	t.Run("FormHelpers", func(t *testing.T) {
		// Test formTag
		form := formTag("/submit")
		if !strings.Contains(string(form), `action="/submit"`) {
			t.Error("formTag() should include action")
		}
		if !strings.Contains(string(form), `method="POST"`) {
			t.Error("formTag() should default to POST method")
		}

		formGet := formTag("/search", "GET")
		if !strings.Contains(string(formGet), `method="GET"`) {
			t.Error("formTag() should accept custom method")
		}

		// Test textField
		field := textField("username", "john", `class="input"`)
		if !strings.Contains(string(field), `name="username"`) {
			t.Error("textField() should include name")
		}
		if !strings.Contains(string(field), `value="john"`) {
			t.Error("textField() should include value")
		}
		if !strings.Contains(string(field), `class="input"`) {
			t.Error("textField() should include attributes")
		}

		// Test textArea
		area := textArea("comment", "Hello", `rows="5"`)
		if !strings.Contains(string(area), `name="comment"`) {
			t.Error("textArea() should include name")
		}
		if !strings.Contains(string(area), `>Hello</textarea>`) {
			t.Error("textArea() should include value")
		}

		// Test selectTag
		options := []string{"Red", "Green", "Blue"}
		selectEl := selectTag("color", options, "Green")
		if !strings.Contains(string(selectEl), `<option value="Green" selected>`) {
			t.Error("selectTag() should mark selected option")
		}

		// Test submitTag
		submit := submitTag("Save", `class="btn"`)
		if !strings.Contains(string(submit), `value="Save"`) {
			t.Error("submitTag() should include value")
		}
		if !strings.Contains(string(submit), `type="submit"`) {
			t.Error("submitTag() should be submit type")
		}

		// Test hiddenField
		hidden := hiddenField("token", "abc123")
		if !strings.Contains(string(hidden), `type="hidden"`) {
			t.Error("hiddenField() should be hidden type")
		}
		if !strings.Contains(string(hidden), `value="abc123"`) {
			t.Error("hiddenField() should include value")
		}

		// Test csrfTag
		csrf := csrfTag("csrf123")
		if !strings.Contains(string(csrf), `name="csrf_token"`) {
			t.Error("csrfTag() should use csrf_token name")
		}
		if !strings.Contains(string(csrf), `value="csrf123"`) {
			t.Error("csrfTag() should include token value")
		}
	})

	t.Run("UtilityHelpers", func(t *testing.T) {
		// Test assetPath
		path := assetPath("app.js")
		if path != "/assets/app.js" {
			t.Errorf("assetPath() should prefix with /assets/, got %s", path)
		}

		// Test safeHTML and rawHTML
		unsafe := "<script>alert('xss')</script>"
		safe := safeHTML(unsafe)
		if safe != template.HTML(unsafe) {
			t.Error("safeHTML() should return HTML unescaped")
		}

		raw := rawHTML(unsafe)
		if raw != template.HTML(unsafe) {
			t.Error("rawHTML() should return HTML unescaped")
		}
	})
}

func TestViewRenderer(t *testing.T) {
	viewsDir := setupTestViews(t)
	vr := NewViewRenderer(viewsDir, true)

	if vr.engine == nil {
		t.Fatal("ViewRenderer should have engine initialized")
	}

	t.Run("Render", func(t *testing.T) {
		// Create mock Gor context
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		ctx := &gor.Context{
			Request:  req,
			Response: w,
		}

		data := map[string]interface{}{
			"Name":    "Test",
			"Message": "Hello",
		}

		err := vr.Render(ctx, "index", data)
		if err != nil {
			t.Fatalf("Render() should not return error: %v", err)
		}

		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/html; charset=utf-8" {
			t.Errorf("Expected Content-Type 'text/html; charset=utf-8', got %s", contentType)
		}

		body := w.Body.String()
		if !strings.Contains(body, "<h1>Welcome Test</h1>") {
			t.Error("Response should contain rendered content")
		}
	})

	t.Run("RenderWithLayout", func(t *testing.T) {
		// Create custom layout
		customLayoutContent := `<custom>{{template "content" .}}</custom>`
		customLayoutPath := filepath.Join(viewsDir, "layouts", "minimal.html")
		if err := os.WriteFile(customLayoutPath, []byte(customLayoutContent), 0644); err != nil {
			t.Fatalf("Failed to write custom layout: %v", err)
		}

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		ctx := &gor.Context{
			Request:  req,
			Response: w,
		}

		data := map[string]interface{}{"Name": "Test"}

		err := vr.RenderWithLayout(ctx, "index", "minimal", data)
		if err != nil {
			t.Fatalf("RenderWithLayout() should not return error: %v", err)
		}

		body := w.Body.String()
		if !strings.Contains(body, "<custom>") {
			t.Error("Response should use custom layout")
		}
	})
}
