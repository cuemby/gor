package views

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cuemby/gor/pkg/gor"
)

// TemplateEngine provides Rails-style templating with layouts and partials
type TemplateEngine struct {
	viewsPath   string
	layoutsPath string
	partialsPath string
	extension   string
	cache       map[string]*template.Template
	mu          sync.RWMutex
	funcs       template.FuncMap
	debug       bool
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(viewsPath string, debug bool) *TemplateEngine {
	te := &TemplateEngine{
		viewsPath:    viewsPath,
		layoutsPath:  filepath.Join(viewsPath, "layouts"),
		partialsPath: filepath.Join(viewsPath, "shared"),
		extension:    ".html",
		cache:        make(map[string]*template.Template),
		debug:        debug,
		funcs:        defaultHelpers(),
	}

	// Load all templates on startup if not in debug mode
	if !debug {
		te.preloadTemplates()
	}

	return te
}

// Render renders a template with the given data
func (te *TemplateEngine) Render(w io.Writer, name string, data interface{}) error {
	return te.RenderWithLayout(w, name, "application", data)
}

// RenderWithLayout renders a template with a specific layout
func (te *TemplateEngine) RenderWithLayout(w io.Writer, name, layout string, data interface{}) error {
	// Get or compile template
	tmpl, err := te.getTemplate(name, layout)
	if err != nil {
		return fmt.Errorf("failed to get template %s: %w", name, err)
	}

	// Execute template
	return tmpl.Execute(w, data)
}

// RenderPartial renders a partial template
func (te *TemplateEngine) RenderPartial(w io.Writer, name string, data interface{}) error {
	partialPath := filepath.Join(te.partialsPath, "_"+name+te.extension)

	tmpl, err := te.getPartialTemplate(partialPath)
	if err != nil {
		return fmt.Errorf("failed to get partial %s: %w", name, err)
	}

	return tmpl.Execute(w, data)
}

// AddFunc adds a custom helper function
func (te *TemplateEngine) AddFunc(name string, fn interface{}) {
	te.funcs[name] = fn
}

// getTemplate retrieves or compiles a template with layout
func (te *TemplateEngine) getTemplate(name, layout string) (*template.Template, error) {
	cacheKey := fmt.Sprintf("%s:%s", layout, name)

	// Check cache if not in debug mode
	if !te.debug {
		te.mu.RLock()
		if tmpl, ok := te.cache[cacheKey]; ok {
			te.mu.RUnlock()
			return tmpl, nil
		}
		te.mu.RUnlock()
	}

	// Compile template
	tmpl, err := te.compileTemplate(name, layout)
	if err != nil {
		return nil, err
	}

	// Cache compiled template
	if !te.debug {
		te.mu.Lock()
		te.cache[cacheKey] = tmpl
		te.mu.Unlock()
	}

	return tmpl, nil
}

// getPartialTemplate retrieves or compiles a partial template
func (te *TemplateEngine) getPartialTemplate(path string) (*template.Template, error) {
	// Check cache if not in debug mode
	if !te.debug {
		te.mu.RLock()
		if tmpl, ok := te.cache[path]; ok {
			te.mu.RUnlock()
			return tmpl, nil
		}
		te.mu.RUnlock()
	}

	// Read partial file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse template
	tmpl, err := template.New(filepath.Base(path)).Funcs(te.funcs).Parse(string(content))
	if err != nil {
		return nil, err
	}

	// Cache compiled template
	if !te.debug {
		te.mu.Lock()
		te.cache[path] = tmpl
		te.mu.Unlock()
	}

	return tmpl, nil
}

// compileTemplate compiles a template with its layout
func (te *TemplateEngine) compileTemplate(name, layout string) (*template.Template, error) {
	// Build template paths
	viewPath := te.resolveViewPath(name)
	layoutPath := filepath.Join(te.layoutsPath, layout+te.extension)

	// Create base template
	tmpl := template.New(name).Funcs(te.funcs)

	// Parse layout first
	layoutContent, err := os.ReadFile(layoutPath)
	if err != nil {
		// If no layout found, just use the view
		viewContent, err := os.ReadFile(viewPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read view %s: %w", viewPath, err)
		}
		return tmpl.Parse(string(viewContent))
	}

	// Parse layout
	tmpl, err = tmpl.Parse(string(layoutContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse layout %s: %w", layout, err)
	}

	// Parse view
	viewContent, err := os.ReadFile(viewPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read view %s: %w", viewPath, err)
	}

	// Define the content block
	tmpl, err = tmpl.Parse(fmt.Sprintf(`{{define "content"}}%s{{end}}`, string(viewContent)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse view %s: %w", name, err)
	}

	// Load all partials
	if err := te.loadPartials(tmpl); err != nil {
		return nil, fmt.Errorf("failed to load partials: %w", err)
	}

	return tmpl, nil
}

// resolveViewPath resolves the full path for a view
func (te *TemplateEngine) resolveViewPath(name string) string {
	// Handle controller/action format (e.g., "users/index")
	if strings.Contains(name, "/") {
		return filepath.Join(te.viewsPath, name+te.extension)
	}
	// Simple view name
	return filepath.Join(te.viewsPath, name+te.extension)
}

// loadPartials loads all partial templates
func (te *TemplateEngine) loadPartials(tmpl *template.Template) error {
	// Check if partials directory exists
	if _, err := os.Stat(te.partialsPath); os.IsNotExist(err) {
		// No partials directory, that's okay
		return nil
	}

	// Walk through partials directory
	err := filepath.Walk(te.partialsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-template files
		if info.IsDir() || !strings.HasSuffix(path, te.extension) {
			return nil
		}

		// Skip non-partials (partials start with _)
		if !strings.HasPrefix(filepath.Base(path), "_") {
			return nil
		}

		// Read partial content
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Define partial template
		partialName := strings.TrimPrefix(filepath.Base(path), "_")
		partialName = strings.TrimSuffix(partialName, te.extension)

		_, err = tmpl.New(partialName).Parse(string(content))
		return err
	})

	return err
}

// preloadTemplates loads all templates into cache
func (te *TemplateEngine) preloadTemplates() {
	// This would walk through all view files and compile them
	// Implementation depends on specific needs
}


// defaultHelpers returns default template helper functions
func defaultHelpers() template.FuncMap {
	return template.FuncMap{
		// String helpers
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      strings.Title,
		"trim":       strings.TrimSpace,
		"capitalize": capitalize,
		"pluralize":  pluralize,
		"singularize": singularize,
		"truncate":   truncate,

		// URL helpers
		"url_for":     urlFor,
		"link_to":     linkTo,
		"asset_path":  assetPath,
		"image_tag":   imageTag,
		"script_tag":  scriptTag,
		"style_tag":   styleTag,

		// Form helpers
		"form_for":    formFor,
		"form_tag":    formTag,
		"text_field":  textField,
		"text_area":   textArea,
		"select_tag":  selectTag,
		"submit_tag":  submitTag,
		"hidden_field": hiddenField,
		"csrf_tag":    csrfTag,

		// Date/Time helpers
		"time_ago":    timeAgo,
		"date_format": dateFormat,
		"time_format": timeFormat,

		// Utility helpers
		"json":        jsonEncode,
		"safe":        safeHTML,
		"raw":         rawHTML,
		"partial":     partial,
		"yield":       templateYield,
	}
}

// String helper implementations
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
}

func pluralize(word string, count ...int) string {
	n := 2
	if len(count) > 0 {
		n = count[0]
	}

	if n == 1 {
		return word
	}

	// Simple pluralization rules
	if strings.HasSuffix(word, "y") {
		return word[:len(word)-1] + "ies"
	}
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
	   strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh") {
		return word + "es"
	}
	return word + "s"
}

func singularize(word string) string {
	// Simple singularization rules
	if strings.HasSuffix(word, "ies") {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "es") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") {
		return word[:len(word)-1]
	}
	return word
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// URL helper implementations
func urlFor(name string, params ...interface{}) template.HTML {
	// Implementation would use the router's URLFor method
	return template.HTML(fmt.Sprintf("/%s", name))
}

func linkTo(text, url string, attrs ...string) template.HTML {
	attributes := ""
	if len(attrs) > 0 {
		attributes = " " + strings.Join(attrs, " ")
	}
	return template.HTML(fmt.Sprintf(`<a href="%s"%s>%s</a>`, url, attributes, text))
}

func assetPath(path string) string {
	return "/assets/" + path
}

func imageTag(src string, attrs ...string) template.HTML {
	attributes := ""
	if len(attrs) > 0 {
		attributes = " " + strings.Join(attrs, " ")
	}
	return template.HTML(fmt.Sprintf(`<img src="%s"%s>`, src, attributes))
}

func scriptTag(src string) template.HTML {
	return template.HTML(fmt.Sprintf(`<script src="%s"></script>`, src))
}

func styleTag(href string) template.HTML {
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" href="%s">`, href))
}

// Form helper implementations
func formFor(model interface{}, action string) template.HTML {
	return template.HTML(fmt.Sprintf(`<form action="%s" method="POST">`, action))
}

func formTag(action string, method ...string) template.HTML {
	m := "POST"
	if len(method) > 0 {
		m = method[0]
	}
	return template.HTML(fmt.Sprintf(`<form action="%s" method="%s">`, action, m))
}

func textField(name, value string, attrs ...string) template.HTML {
	attributes := ""
	if len(attrs) > 0 {
		attributes = " " + strings.Join(attrs, " ")
	}
	return template.HTML(fmt.Sprintf(`<input type="text" name="%s" value="%s"%s>`, name, value, attributes))
}

func textArea(name, value string, attrs ...string) template.HTML {
	attributes := ""
	if len(attrs) > 0 {
		attributes = " " + strings.Join(attrs, " ")
	}
	return template.HTML(fmt.Sprintf(`<textarea name="%s"%s>%s</textarea>`, name, attributes, value))
}

func selectTag(name string, options []string, selected string) template.HTML {
	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<select name="%s">`, name))
	for _, opt := range options {
		if opt == selected {
			html.WriteString(fmt.Sprintf(`<option value="%s" selected>%s</option>`, opt, opt))
		} else {
			html.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, opt, opt))
		}
	}
	html.WriteString("</select>")
	return template.HTML(html.String())
}

func submitTag(value string, attrs ...string) template.HTML {
	attributes := ""
	if len(attrs) > 0 {
		attributes = " " + strings.Join(attrs, " ")
	}
	return template.HTML(fmt.Sprintf(`<input type="submit" value="%s"%s>`, value, attributes))
}

func hiddenField(name, value string) template.HTML {
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, name, value))
}

func csrfTag(token string) template.HTML {
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="csrf_token" value="%s">`, token))
}

// Date/Time helpers
func timeAgo(t interface{}) string {
	// Implementation would convert time to "2 hours ago" format
	return "recently"
}

func dateFormat(t interface{}, format string) string {
	// Implementation would format date
	return ""
}

func timeFormat(t interface{}, format string) string {
	// Implementation would format time
	return ""
}

// Utility helpers
func jsonEncode(v interface{}) template.JS {
	// Implementation would JSON encode the value
	return template.JS("{}")
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func rawHTML(s string) template.HTML {
	return template.HTML(s)
}

func partial(name string, data interface{}) template.HTML {
	// Implementation would render a partial
	return template.HTML("")
}

func templateYield() template.HTML {
	return template.HTML("{{template \"content\" .}}")
}

// ViewRenderer integrates with the Gor context
type ViewRenderer struct {
	engine *TemplateEngine
}

// NewViewRenderer creates a new view renderer
func NewViewRenderer(viewsPath string, debug bool) *ViewRenderer {
	return &ViewRenderer{
		engine: NewTemplateEngine(viewsPath, debug),
	}
}

// Render implements the rendering for Gor context
func (vr *ViewRenderer) Render(ctx *gor.Context, template string, data interface{}) error {
	ctx.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.Response.WriteHeader(http.StatusOK)

	var buf bytes.Buffer
	if err := vr.engine.Render(&buf, template, data); err != nil {
		return err
	}

	_, err := ctx.Response.Write(buf.Bytes())
	return err
}

// RenderWithLayout renders with a specific layout
func (vr *ViewRenderer) RenderWithLayout(ctx *gor.Context, template, layout string, data interface{}) error {
	ctx.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.Response.WriteHeader(http.StatusOK)

	var buf bytes.Buffer
	if err := vr.engine.RenderWithLayout(&buf, template, layout, data); err != nil {
		return err
	}

	_, err := ctx.Response.Write(buf.Bytes())
	return err
}